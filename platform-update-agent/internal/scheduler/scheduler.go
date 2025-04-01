// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/comms"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/downloader"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/metadata"
	pb "github.com/open-edge-platform/infra-managers/maintenance/pkg/api/maintmgr/v1"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

const repeatedScheduleTag = "RepeatedSchedule"
const singleScheduleTag = "SingleSchedule"

type PuaScheduler struct {
	scheduler    *gocron.Scheduler
	client       *comms.Client
	nodeGuid     string
	updater      Updater
	log          *logrus.Entry
	updateLocker UpdateLocker
}

type Updater interface {
	StartUpdate(durationSeconds int64)
}

type UpdateLocker interface {
	LockForUpdate() // see description in downloader -> LockForUpdate
	Unlock()
}

type DoNothingUpdater struct{}

func (m DoNothingUpdater) StartUpdate(durationSeconds int64) {
	// Do nothing
}

var doNothingUpdater = DoNothingUpdater{}

func NewPuaScheduler(client *comms.Client, nodeGuid string, updater Updater, updateLocker UpdateLocker, log *logrus.Entry) (*PuaScheduler, error) {
	if updater == nil {
		return nil, errors.New("nil updater passed to NewPuaScheduler")
	}
	if updateLocker == nil {
		return nil, errors.New("nil updateLocker passed to NewPuaScheduler")
	}
	scheduler := gocron.NewScheduler(time.UTC)
	scheduler.SingletonModeAll()
	scheduler.StartAsync()
	scheduler.SetMaxConcurrentJobs(1, gocron.WaitMode)

	return &PuaScheduler{
		scheduler:    scheduler,
		client:       client,
		nodeGuid:     nodeGuid,
		updater:      updater,
		log:          log,
		updateLocker: updateLocker,
	}, nil
}

func (p *PuaScheduler) removeRepeatedSchedules() {
	err := p.scheduler.RemoveByTag(repeatedScheduleTag)
	if err != nil {
		p.log.Debugf("scheduler failed to remove cron jobs by '%v' tag - %v", repeatedScheduleTag, err)
	}
}

func (p *PuaScheduler) scheduleRepeatedSchedule(schedule *pb.RepeatedSchedule, osType string) {

	_, err := p.scheduler.Tag(repeatedScheduleTag).Cron(CronScheduleToString(schedule)).
		Do(func() {
			endTime := time.Now().Add(time.Duration(schedule.DurationSeconds) * time.Second)
			p.triggerUpdate(repeatedScheduleTag, endTime, p.IsUpdateAlreadyApplied, osType)
		})
	if err != nil {
		p.log.Errorf("failed to schedule cron job - %v", err)
		innerErr := metadata.SetMetaUpdateStatus(pb.UpdateStatus_STATUS_TYPE_FAILED)
		if innerErr != nil {
			p.log.Errorf("failed to set metadata - %v", innerErr)
			return
		}
	}
}

// triggerUpdate is called by gocron to start an update
// it is responsible for coordinating with downloader's exclusion lock
// and aborting the update if we can't get a lock in time
// For Edge Microvisor Toolkit, if the user applies the same image update, the update will be skipped.
func (p *PuaScheduler) triggerUpdate(tag string, endTime time.Time, updateAlreadyApplied func(osType string) bool, osType string) {
	if updateAlreadyApplied(osType) {
		p.log.Infof("UPDATE already applied. Skipping update.")
		return
	}
	p.log.Debugf("update is triggered by %v", tag)
	p.log.Infof("UPDATE: attempting to acquire update/download lock")
	p.updateLocker.LockForUpdate()
	defer p.updateLocker.Unlock()
	p.log.Debugf("UPDATE: lock acquired, checking if we have time to start the update")
	if time.Now().Before(endTime) {
		p.log.Infof("UPDATE: update started")
		p.updater.StartUpdate(int64(math.Round(time.Until(endTime).Seconds())))
	} else {
		p.log.Infof("UPDATE: maintenance window expired; not running update")
	}
}

func (p *PuaScheduler) scheduleSingleRun(schedule *pb.SingleSchedule, osType string) {
	err := p.scheduler.RemoveByTag(singleScheduleTag)
	if err != nil {
		p.log.Debugf("scheduler failed to remove cron job by '%v' tag - %v", singleScheduleTag, err)
	}
	p.log.Debugf("Check schedule.StartSeconds: %v\n", schedule.StartSeconds)
	startTime := time.Unix(int64(schedule.StartSeconds), 0)
	endTime := time.Unix(int64(schedule.EndSeconds), 0)
	now := time.Now()

	durationSeconds := int64(0)
	if schedule.EndSeconds != 0 {
		// special case: startTime is in the past, but endTime is at least 10 minutes in the future;
		// then bring startTime a little bit into the future so the schedule will still run
		if startTime.Before(now) && endTime.After(now.Add(10*time.Minute)) {
			startTime = now.Add(1 * time.Second)
		}

		durationSeconds = int64(endTime.Sub(startTime).Seconds())
	}

	// Define a human-readable layout
	layout := "Mon, 02 Jan 2006 15:04:05 MST"
	// Format and print the times
	p.log.Infof("Start Time: %s\n", startTime.Format(layout))
	p.log.Infof("End Time: %s\n", endTime.Format(layout))
	p.log.Infof("Current Time: %s\n", now.Format(layout))

	// if, even after above adjustment, startTime is still in the past, skip scheduling
	if startTime.Before(now) {
		p.log.Infof("startTime is before now. It is still in the past, skip scheduling")
		return
	}

	_, err = p.scheduler.Every(1).Day().Tag(singleScheduleTag).StartAt(startTime).LimitRunsTo(1).
		Do(p.triggerUpdate, singleScheduleTag, startTime.Add(time.Duration(durationSeconds)*time.Second), p.IsUpdateAlreadyApplied, osType)
	if err != nil {
		p.log.Errorf("failed to schedule single schedule job - %v", err)
		innerErr := metadata.SetMetaUpdateStatus(pb.UpdateStatus_STATUS_TYPE_FAILED)
		if innerErr != nil {
			p.log.Errorf("failed to set metadata - %v", innerErr)
		}
	}
}

func (p *PuaScheduler) HandleRepeatedSchedule(schedules []*pb.RepeatedSchedule, meta *metadata.Meta, osType string) {

	jobs, _ := p.scheduler.FindJobsByTag(repeatedScheduleTag)
	if len(schedules) == 0 {
		err := p.scheduler.RemoveByTag(repeatedScheduleTag)
		if err != nil {
			p.log.Debugf("scheduler failed to remove cron job by '%v' tag - %v", repeatedScheduleTag, err)
		}
	}
	var exists = false
	if len(jobs) != 0 && (len(schedules) == len(meta.RepeatedSchedules)) {
		for _, schedule := range schedules {
			exists = false
			for _, metaSchedule := range meta.RepeatedSchedules {
				if proto.Equal(schedule, metaSchedule) {
					exists = true
					break
				}
			}
			if exists {
				continue
			}
			break
		}
	}

	if !exists {
		if len(jobs) != 0 {
			p.removeRepeatedSchedules()
		}
		meta.RepeatedSchedules = []*pb.RepeatedSchedule{}
		for i, schedule := range schedules {
			p.scheduleRepeatedSchedule(schedules[i], osType)
			meta.RepeatedSchedules = append(meta.RepeatedSchedules, schedule)
		}
	}
}

func (p *PuaScheduler) HandleSingleSchedule(schedule *pb.SingleSchedule, meta *metadata.Meta, osType string) {
	switch {
	case schedule == nil:
		err := p.scheduler.RemoveByTag(singleScheduleTag)
		if err != nil {
			p.log.Debugf("scheduler failed to remove cron job by '%v' tag - %v", singleScheduleTag, err)
		}
		meta.SingleScheduleFinished = true
	case schedule != nil && !(proto.Equal(meta.SingleSchedule, schedule) && meta.SingleScheduleFinished):
		jobs, _ := p.scheduler.FindJobsByTag(singleScheduleTag) // error is ignored intentionally as it only returns ErrJobNotFoundWithTag
		switch {
		// Ensure there is only one job (update) running each time.
		case len(jobs) == 0:
			{
				p.scheduleSingleRun(schedule, osType)
				meta.SingleSchedule = schedule
				meta.SingleScheduleFinished = false
			}
		default:
			if jobs[0].FinishedRunCount() == 1 {
				meta.SingleScheduleFinished = true
				p.log.Info("marking single schedule job as done")
			}
		}
	}
}

func (p *PuaScheduler) CleanupSchedule() {
	err := p.scheduler.RemoveByTag(singleScheduleTag)
	if err != nil {
		p.log.Debugf("scheduler failed to remove cron job by '%v' tag - %v", singleScheduleTag, err)
	}
	err = p.scheduler.RemoveByTag(repeatedScheduleTag)
	if err != nil {
		p.log.Debugf("scheduler failed to remove cron job by '%v' tag - %v", repeatedScheduleTag, err)
	}
}

func (p *PuaScheduler) GetJobs() []*gocron.Job {
	return p.scheduler.Jobs()
}

// GetNextJob returns the first job that will run out of the list of single and repeated jobs
func (p *PuaScheduler) GetNextJob() *gocron.Job {
	jobs := p.GetJobs()

	if len(jobs) == 0 {
		return nil
	}

	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].NextRun().Before(jobs[j].NextRun())
	})

	return jobs[0]
}

func CronScheduleToString(schedule *pb.RepeatedSchedule) string {
	return fmt.Sprintf("%v %v %v %v %v", schedule.GetCronMinutes(), schedule.GetCronHours(),
		schedule.GetCronDayMonth(), schedule.GetCronMonth(), schedule.GetCronDayWeek())
}

func (p *PuaScheduler) IsUpdateAlreadyApplied(osType string) bool {
	// Proceed if it's Edge Microvisor Toolkit.
	p.log.Debugf("Check if update already applied in %v", osType)
	if osType == "emt" {
		actualOSSource, err := metadata.GetMetaOSProfileUpdateSourceActual()
		if err != nil {
			p.log.Errorf("WARNING: Cannot retrieve OS source already on system.")
			return false
		}
		updateOSSource, get_err := metadata.GetMetaOSProfileUpdateSourceDesired()
		if get_err != nil {
			p.log.Errorf("WARNING: Cannot retrieve desired OS source from metadata file.")
			return false
		}
		p.log.Debugf("actualOSSource= %v", actualOSSource)
		p.log.Debugf("updateOSSource= %v", updateOSSource)
		return downloader.AreOsImagesEqual(updateOSSource, actualOSSource)
	}

	return false
}
