// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	auth "github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
	mock_server "github.com/open-edge-platform/edge-node-agents/platform-update-agent/cmd/mock-server/mock-server"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/comms"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/downloader"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/logger"
	pua_logger "github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/logger"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/metadata"
	pb "github.com/open-edge-platform/infra-managers/maintenance/pkg/api/maintmgr/v1"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

type MockDownloadExecutor struct {
}

func (m *MockDownloadExecutor) Download(
	ctx context.Context,
	prependToImageURL string,
	source *pb.OSProfileUpdateSource,
) error {
	return nil
}

var client *comms.Client
var hook *test.Hook
var mockDownloadExecutor *MockDownloadExecutor = &MockDownloadExecutor{}
var log = pua_logger.Logger()

func init() {
	var testMetaFile *os.File
	grpcAddress := "localhost:8089"
	server, listener := mock_server.NewGrpcServer(grpcAddress, "../../mocks", mock_server.UBUNTU)
	go func() {
		if err := mock_server.RunGrpcServer(server, listener); err != nil {
			log.Println("Failed to run maintenance gRPC server")
		}
	}()

	tlsConfig, err := auth.GetAuthConfig(context.TODO(), nil)
	if err != nil {
		log.Fatal("failed to get TLS config")
	}
	client = comms.NewClient(grpcAddress, tlsConfig)

	if client == nil {
		log.Fatal("failed to create GRPC client for tests")
	}
	err = client.Connect(context.TODO())
	if err != nil {
		log.Fatalf("failed to create test GRPC client - %v", err)
	}
	testMetaFile, err = os.CreateTemp("/tmp", "pua-test-")
	if err != nil {
		log.Fatal("failed to create metadata file for test")
	}
	defer testMetaFile.Close()
	_, err = testMetaFile.WriteString("{}")
	if err != nil {
		log.Errorf("failed to write to metadata file for test")
	}
	metadata.MetaPath = testMetaFile.Name()

	testLog, testHook := test.NewNullLogger()
	hook = testHook
	logger.SetLogger(testLog.WithField("test", "test"))
}

func TestPuaScheduler_NewPuaSchedule_returns_error_if_nil_updater(t *testing.T) {
	_, err := NewPuaScheduler(
		client,
		"abcd",
		nil,
		downloader.NewDownloader(
			10*time.Minute,
			6*time.Hour,
			mockDownloadExecutor,
			pua_logger.Logger(),
			metadata.NewController(),
		),
		pua_logger.Logger(),
	)

	assert.NotNil(t, err, "NewPuaScheduler should return error if Updater is nil")
}

func TestPuaScheduler_NewPuaSchedule_returns_error_if_nil_updatelocker(t *testing.T) {
	_, err := NewPuaScheduler(client, "abcd", doNothingUpdater, nil, pua_logger.Logger())

	assert.NotNil(t, err, "NewPuaScheduler should return error if updateLocker is nil")
}

func TestPuaScheduler_scheduleRepeatedSchedule_shouldLogErrorIfScheduleIsIncorrect(t *testing.T) {
	puaScheduler, _ := NewPuaScheduler(
		client,
		"abcd",
		doNothingUpdater,
		downloader.NewDownloader(
			10*time.Minute,
			6*time.Hour,
			mockDownloadExecutor,
			pua_logger.Logger(),
			metadata.NewController(),
		),
		pua_logger.Logger(),
	)

	puaScheduler.scheduleRepeatedSchedule(&pb.RepeatedSchedule{
		CronMinutes:  "a",
		CronHours:    "2",
		CronDayMonth: "3",
		CronMonth:    "4",
		CronDayWeek:  "5",
	}, "ubuntu")

	assert.Contains(
		t,
		hook.LastEntry().Message,
		"failed to schedule cron job - gocron: cron expression failed to be parsed: failed to parse int from a",
	)
}

func TestPuaScheduler_scheduleRepeatedSchedule_shouldScheduleIfCronFormatIsCorrect(t *testing.T) {
	puaScheduler, _ := NewPuaScheduler(
		client,
		"abcd",
		doNothingUpdater,
		downloader.NewDownloader(
			10*time.Minute,
			6*time.Hour,
			mockDownloadExecutor,
			pua_logger.Logger(),
			metadata.NewController(),
		),
		pua_logger.Logger(),
	)

	puaScheduler.scheduleRepeatedSchedule(&pb.RepeatedSchedule{
		CronMinutes:  "2",
		CronHours:    "3",
		CronDayMonth: "4",
		CronMonth:    "5",
		CronDayWeek:  "6",
	}, "ubuntu")

	assert.Equal(t, puaScheduler.scheduler.Len(), 1)
	job := puaScheduler.scheduler.Jobs()[0]
	assert.Equal(t, len(job.Tags()), 1)
	assert.Equal(t, job.Tags()[0], repeatedScheduleTag)
}

func TestPuaScheduler_HandleSingleSchedule_shouldScheduleIfFormatIsCorrect(t *testing.T) {
	puaScheduler, _ := NewPuaScheduler(
		client,
		"abcd",
		doNothingUpdater,
		downloader.NewDownloader(
			10*time.Minute,
			6*time.Hour,
			mockDownloadExecutor,
			pua_logger.Logger(),
			metadata.NewController(),
		),
		pua_logger.Logger(),
	)

	puaScheduler.scheduleSingleRun(&pb.SingleSchedule{
		StartSeconds: uint64(time.Now().Add(time.Second * 2).Unix()),
		EndSeconds:   uint64(time.Now().Add(time.Second * 10).Unix()),
	}, "ubuntu")

	assert.Equal(t, puaScheduler.scheduler.Len(), 1)
	job := puaScheduler.scheduler.Jobs()[0]
	assert.Equal(t, len(job.Tags()), 1)
	assert.Equal(t, job.Tags()[0], singleScheduleTag)
}

func TestPuaScheduler_HandleSingleSchedule_shouldScheduleIfStartIsPastAndEndIs11MinFuture(
	t *testing.T,
) {
	puaScheduler, _ := NewPuaScheduler(
		client,
		"abcd",
		doNothingUpdater,
		downloader.NewDownloader(
			10*time.Minute,
			6*time.Hour,
			mockDownloadExecutor,
			pua_logger.Logger(),
			metadata.NewController(),
		),
		pua_logger.Logger(),
	)

	now := time.Now()
	puaScheduler.scheduleSingleRun(&pb.SingleSchedule{
		StartSeconds: uint64(now.Add(-time.Second * 10).Unix()),
		EndSeconds:   uint64(now.Add(time.Minute * 11).Unix()),
	}, "ubuntu")

	assert.Equal(t, puaScheduler.scheduler.Len(), 1)
	job := puaScheduler.scheduler.Jobs()[0]
	delta := job.NextRun().Sub(now)
	assert.Less(
		t,
		delta,
		2.0*time.Second,
		"Next run should be less than 2 seconds from the current time",
	)
	assert.Greater(
		t,
		delta,
		0.0*time.Second,
		"Next run should be greater than 0 seconds from the current time",
	)
	assert.Equal(t, len(job.Tags()), 1)
	assert.Equal(t, job.Tags()[0], singleScheduleTag)
}

func TestPuaScheduler_HandleSingleSchedule_shouldNotScheduleIfStartIsPastAndEndIs5MinFuture(
	t *testing.T,
) {
	puaScheduler, _ := NewPuaScheduler(
		client,
		"abcd",
		doNothingUpdater,
		downloader.NewDownloader(
			10*time.Minute,
			6*time.Hour,
			mockDownloadExecutor,
			pua_logger.Logger(),
			metadata.NewController(),
		),
		pua_logger.Logger(),
	)

	now := time.Now()
	puaScheduler.scheduleSingleRun(&pb.SingleSchedule{
		StartSeconds: uint64(now.Add(-time.Second * 10).Unix()),
		EndSeconds:   uint64(now.Add(time.Minute * 5).Unix()),
	}, "ubuntu")

	assert.Equal(t, puaScheduler.scheduler.Len(), 0)
}

func TestPuaScheduler_HandleRepeatedSchedule_shouldCleanUpIfNilScheduleIsReceived(t *testing.T) {
	puaScheduler, _ := NewPuaScheduler(
		client,
		"abcd",
		doNothingUpdater,
		downloader.NewDownloader(
			10*time.Minute,
			6*time.Hour,
			mockDownloadExecutor,
			pua_logger.Logger(),
			metadata.NewController(),
		),
		pua_logger.Logger(),
	)
	puaScheduler.scheduleRepeatedSchedule(&pb.RepeatedSchedule{
		CronMinutes:  "2",
		CronHours:    "3",
		CronDayMonth: "4",
		CronMonth:    "5",
		CronDayWeek:  "6",
	}, "ubuntu")
	assert.Equal(t, puaScheduler.scheduler.Len(), 1)
	testMeta := &metadata.Meta{
		RepeatedSchedules: []*pb.RepeatedSchedule{
			{
				CronMinutes:  "this",
				CronHours:    "schedule",
				CronDayMonth: "will",
				CronMonth:    "be",
				CronDayWeek:  "removed",
			},
		},
	}

	puaScheduler.HandleRepeatedSchedule(nil, testMeta, "ubuntu")

	assert.Equal(t, puaScheduler.scheduler.Len(), 0)
}

func TestPuaScheduler_HandleRepeatedSchedule_shouldRecreateJobAfterRestart(t *testing.T) {
	puaScheduler, _ := NewPuaScheduler(
		client,
		"abcd",
		doNothingUpdater,
		downloader.NewDownloader(
			10*time.Minute,
			6*time.Hour,
			mockDownloadExecutor,
			pua_logger.Logger(),
			metadata.NewController(),
		),
		pua_logger.Logger(),
	)
	schedule := []*pb.RepeatedSchedule{
		{
			CronMinutes:  "2",
			CronHours:    "3",
			CronDayMonth: "4",
			CronMonth:    "5",
			CronDayWeek:  "6",
		},
	}
	testMeta := &metadata.Meta{
		RepeatedSchedules: schedule,
	}
	assert.Equal(t, puaScheduler.scheduler.Len(), 0)

	puaScheduler.HandleRepeatedSchedule(schedule, testMeta, "ubuntu")

	assert.Equal(t, puaScheduler.scheduler.Len(), 1)
	assert.Equal(t, testMeta.RepeatedSchedules[0], schedule[0])
}

func TestPuaScheduler_HandleRepeatedSchedule_shouldOverwriteCurrentJobIfNewScheduleReceived(
	t *testing.T,
) {
	puaScheduler, _ := NewPuaScheduler(
		client,
		"abcd",
		doNothingUpdater,
		downloader.NewDownloader(
			10*time.Minute,
			6*time.Hour,
			mockDownloadExecutor,
			pua_logger.Logger(),
			metadata.NewController(),
		),
		pua_logger.Logger(),
	)
	schedule := []*pb.RepeatedSchedule{
		{
			CronMinutes:  "2",
			CronHours:    "3",
			CronDayMonth: "4",
			CronMonth:    "5",
			CronDayWeek:  "6",
		},
	}
	testMeta := &metadata.Meta{
		RepeatedSchedules: schedule,
	}

	puaScheduler.HandleRepeatedSchedule(schedule, testMeta, "ubuntu")

	assert.Equal(t, puaScheduler.scheduler.Len(), 1)
	newSchedule := []*pb.RepeatedSchedule{
		{
			CronMinutes:  "8",
			CronHours:    "7",
			CronDayMonth: "6",
			CronMonth:    "5",
			CronDayWeek:  "4",
		},
	}

	puaScheduler.HandleRepeatedSchedule(newSchedule, testMeta, "ubuntu")

	assert.Equal(t, puaScheduler.scheduler.Len(), 1)
	assert.Equal(t, testMeta.RepeatedSchedules[0], newSchedule[0])
}

func TestPuaScheduler_HandleRepeatedSchedule_shouldOverwriteCurrentJobIfNewMultipleSchedulesReceived(
	t *testing.T,
) {
	puaScheduler, _ := NewPuaScheduler(
		client,
		"abcd",
		doNothingUpdater,
		downloader.NewDownloader(
			10*time.Minute,
			6*time.Hour,
			mockDownloadExecutor,
			pua_logger.Logger(),
			metadata.NewController(),
		),
		pua_logger.Logger(),
	)
	schedule := []*pb.RepeatedSchedule{
		{
			CronMinutes:  "2",
			CronHours:    "3",
			CronDayMonth: "4",
			CronMonth:    "5",
			CronDayWeek:  "6",
		},
	}
	testMeta := &metadata.Meta{
		RepeatedSchedules: schedule,
	}

	puaScheduler.HandleRepeatedSchedule(schedule, testMeta, "ubuntu")

	assert.Equal(t, puaScheduler.scheduler.Len(), 1)
	newSchedule := []*pb.RepeatedSchedule{
		{
			CronMinutes:  "8",
			CronHours:    "7",
			CronDayMonth: "6",
			CronMonth:    "5",
			CronDayWeek:  "4",
		},
		{
			CronMinutes:  "7",
			CronHours:    "10",
			CronDayMonth: "7",
			CronMonth:    "10",
			CronDayWeek:  "5",
		},
	}

	puaScheduler.HandleRepeatedSchedule(newSchedule, testMeta, "ubuntu")

	assert.Equal(t, puaScheduler.scheduler.Len(), 2)
	assert.Equal(t, testMeta.RepeatedSchedules[0], newSchedule[0])
	assert.Equal(t, testMeta.RepeatedSchedules[1], newSchedule[1])
}

func TestPuaScheduler_HandleRepeatedSchedule_shouldOverwriteCurrentJobWithEmptySchedulesReceived(
	t *testing.T,
) {
	puaScheduler, _ := NewPuaScheduler(
		client,
		"abcd",
		doNothingUpdater,
		downloader.NewDownloader(
			10*time.Minute,
			6*time.Hour,
			mockDownloadExecutor,
			pua_logger.Logger(),
			metadata.NewController(),
		),
		pua_logger.Logger(),
	)
	schedule := []*pb.RepeatedSchedule{
		{
			CronMinutes:  "2",
			CronHours:    "3",
			CronDayMonth: "4",
			CronMonth:    "5",
			CronDayWeek:  "6",
		},
	}
	testMeta := &metadata.Meta{
		RepeatedSchedules: schedule,
	}

	puaScheduler.HandleRepeatedSchedule(schedule, testMeta, "ubuntu")

	assert.Equal(t, puaScheduler.scheduler.Len(), 1)
	newSchedule := []*pb.RepeatedSchedule{}

	puaScheduler.HandleRepeatedSchedule(newSchedule, testMeta, "ubuntu")

	assert.Equal(t, puaScheduler.scheduler.Len(), 0)
}

func TestPuaScheduler_RemoveRepeatedSchedule_shouldRemoveExistingJobsByTag(t *testing.T) {
	puaScheduler, _ := NewPuaScheduler(
		client,
		"abcd",
		doNothingUpdater,
		downloader.NewDownloader(
			10*time.Minute,
			6*time.Hour,
			mockDownloadExecutor,
			pua_logger.Logger(),
			metadata.NewController(),
		),
		pua_logger.Logger(),
	)
	schedule := []*pb.RepeatedSchedule{
		{
			CronMinutes:  "2",
			CronHours:    "3",
			CronDayMonth: "4",
			CronMonth:    "5",
			CronDayWeek:  "6",
		},
	}
	testMeta := &metadata.Meta{
		RepeatedSchedules: schedule,
	}
	puaScheduler.HandleRepeatedSchedule(schedule, testMeta, "ubuntu")
	assert.Equal(t, puaScheduler.scheduler.Len(), 1)
	puaScheduler.removeRepeatedSchedules()
	assert.Equal(t, puaScheduler.scheduler.Len(), 0)

}

func TestPuaScheduler_HandleRepeatedSchedule_shouldntRescheduleIfJobWithScheduleAlreadyExists(
	t *testing.T,
) {
	puaScheduler, _ := NewPuaScheduler(
		client,
		"abcd",
		doNothingUpdater,
		downloader.NewDownloader(
			10*time.Minute,
			6*time.Hour,
			mockDownloadExecutor,
			pua_logger.Logger(),
			metadata.NewController(),
		),
		pua_logger.Logger(),
	)
	schedule := []*pb.RepeatedSchedule{
		{
			CronMinutes:  "2",
			CronHours:    "3",
			CronDayMonth: "4",
			CronMonth:    "5",
			CronDayWeek:  "6",
		},
		{
			CronMinutes:  "8",
			CronHours:    "7",
			CronDayMonth: "6",
			CronMonth:    "5",
			CronDayWeek:  "4",
		},
	}
	testMeta := &metadata.Meta{
		RepeatedSchedules: schedule,
	}

	puaScheduler.HandleRepeatedSchedule(schedule, testMeta, "ubuntu")

	assert.Equal(t, puaScheduler.scheduler.Len(), 2)
	assert.Equal(t, len(puaScheduler.scheduler.Jobs()[0].Tags()), 1)
	puaScheduler.scheduler.Jobs()[0].Tag("test-tag")
	// The jobs could be reordered on invocation of HandleRepeatedSchedule, hence checking all
	assert.Condition(t, func() bool {
		for _, job := range puaScheduler.scheduler.Jobs() {
			if len(job.Tags()) == 2 {
				return true
			}
		}
		return false
	}, "Expected to find a job with 2 tags")

	puaScheduler.HandleRepeatedSchedule(schedule, testMeta, "ubuntu")

	assert.Equal(t, puaScheduler.scheduler.Len(), 2)
	assert.Condition(t, func() bool {
		for _, job := range puaScheduler.scheduler.Jobs() {
			if len(job.Tags()) == 2 {
				return true
			}
		}
		return false
	}, "Expected to find a job with 2 tags")
	assert.Equal(t, testMeta.RepeatedSchedules[0], schedule[0])
	assert.Equal(t, testMeta.RepeatedSchedules[1], schedule[1])

}

func TestPuaScheduler_HandleSingleSchedule_shouldCleanupIfReceivedNil(t *testing.T) {
	puaScheduler, _ := NewPuaScheduler(
		client,
		"abcd",
		doNothingUpdater,
		downloader.NewDownloader(
			10*time.Minute,
			6*time.Hour,
			mockDownloadExecutor,
			pua_logger.Logger(),
			metadata.NewController(),
		),
		pua_logger.Logger(),
	)

	now := time.Now()
	schedule := &pb.SingleSchedule{
		StartSeconds: uint64(now.Add(time.Minute * 30).Unix()),
		EndSeconds:   uint64(now.Add(time.Minute * 40).Unix()),
	}
	testMeta := &metadata.Meta{
		SingleSchedule:         schedule,
		SingleScheduleFinished: false,
	}

	puaScheduler.scheduleSingleRun(schedule, "ubuntu")

	assert.Equal(t, len(puaScheduler.scheduler.Jobs()), 1)

	puaScheduler.HandleSingleSchedule(nil, testMeta, "ubuntu")

	assert.True(t, testMeta.SingleScheduleFinished)
	assert.Equal(t, len(puaScheduler.scheduler.Jobs()), 0)
}

func TestPuaScheduler_HandleSingleSchedule(t *testing.T) {

	tests := []struct {
		name                          string
		singleScheduleFinished        bool
		expectedNumberOfScheduledJobs int
		customAssert                  func(*metadata.Meta)
	}{
		{
			"JobHasntFinishedAndRestartOccuredThenJobBeRescheduled",
			false,
			1,
			func(meta *metadata.Meta) {
				assert.False(t, meta.SingleScheduleFinished)
			},
		},
		{
			"JobFinishedAndRestartOccuredThenJobShouldntBeRescheduled",
			true,
			0,
			func(meta *metadata.Meta) {
				assert.True(t, meta.SingleScheduleFinished)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			puaScheduler, _ := NewPuaScheduler(
				client,
				"abcd",
				doNothingUpdater,
				downloader.NewDownloader(
					10*time.Minute,
					6*time.Hour,
					mockDownloadExecutor,
					pua_logger.Logger(),
					metadata.NewController(),
				),
				pua_logger.Logger(),
			)
			now := time.Now()
			schedule := &pb.SingleSchedule{
				StartSeconds: uint64(now.Add(time.Minute * 30).Unix()),
				EndSeconds:   uint64(now.Add(time.Minute * 40).Unix()),
			}
			testMeta := &metadata.Meta{
				SingleSchedule:         schedule,
				SingleScheduleFinished: tt.singleScheduleFinished,
			}
			assert.Equal(t, len(puaScheduler.scheduler.Jobs()), 0)

			puaScheduler.HandleSingleSchedule(schedule, testMeta, "ubuntu")

			assert.Equal(t, testMeta.SingleSchedule, schedule)
			assert.Equal(t, len(puaScheduler.scheduler.Jobs()), tt.expectedNumberOfScheduledJobs)
			tt.customAssert(testMeta)
		})
	}
}

func TestPuaScheduler_CleanupSchedule_ShouldRemoveAllJobs(t *testing.T) {
	puaScheduler, _ := NewPuaScheduler(
		client,
		"abcd",
		doNothingUpdater,
		downloader.NewDownloader(
			10*time.Minute,
			6*time.Hour,
			mockDownloadExecutor,
			pua_logger.Logger(),
			metadata.NewController(),
		),
		pua_logger.Logger(),
	)
	now := time.Now()
	sSchedule := &pb.SingleSchedule{
		StartSeconds: uint64(now.Add(time.Minute * 30).Unix()),
		EndSeconds:   uint64(now.Add(time.Minute * 40).Unix()),
	}
	rSchedule := &pb.RepeatedSchedule{
		CronMinutes:  "2",
		CronHours:    "3",
		CronDayMonth: "4",
		CronMonth:    "5",
		CronDayWeek:  "6",
	}
	puaScheduler.scheduleRepeatedSchedule(rSchedule, "ubuntu")
	puaScheduler.scheduleSingleRun(sSchedule, "ubuntu")

	assert.Len(t, puaScheduler.scheduler.Jobs(), 2)

	puaScheduler.CleanupSchedule()

	assert.Len(t, puaScheduler.scheduler.Jobs(), 0)
}

func TestPuaScheduler_GetJobs_ShouldReturnCorrectAmountOfJobs(t *testing.T) {
	puaScheduler, _ := NewPuaScheduler(
		client,
		"abcd",
		doNothingUpdater,
		downloader.NewDownloader(
			10*time.Minute,
			6*time.Hour,
			mockDownloadExecutor,
			pua_logger.Logger(),
			metadata.NewController(),
		),
		pua_logger.Logger(),
	)

	puaScheduler.scheduleSingleRun(&pb.SingleSchedule{
		StartSeconds: uint64(time.Now().Add(time.Second * 123).Unix()),
	}, "ubuntu")

	jobs := puaScheduler.GetJobs()

	assert.Len(t, jobs, 1)
}

func generateDailyRepeatedSchedule(
	startTime time.Time,
	duration time.Duration,
) *pb.RepeatedSchedule {
	return &pb.RepeatedSchedule{
		DurationSeconds: uint32(math.Round(duration.Seconds())),
		CronMinutes:     fmt.Sprintf("%d", startTime.UTC().Minute()),
		CronHours:       fmt.Sprintf("%d", startTime.UTC().Hour()),
		CronDayMonth:    "*",
		CronMonth:       "*",
		CronDayWeek:     "*",
	}
}

func TestPuaScheduler_GetNextJob_ShouldReturnNextJob(t *testing.T) {
	puaScheduler, _ := NewPuaScheduler(
		client,
		"abcd",
		doNothingUpdater,
		downloader.NewDownloader(
			10*time.Minute,
			6*time.Hour,
			mockDownloadExecutor,
			pua_logger.Logger(),
			metadata.NewController(),
		),
		pua_logger.Logger(),
	)

	now := time.Now().Truncate(time.Second)
	singleStart := now.Add(time.Second * 123)
	repeatedStart := now.Add(time.Second * 200)

	puaScheduler.scheduleSingleRun(&pb.SingleSchedule{
		StartSeconds: uint64(singleStart.Unix()),
	}, "ubuntu")
	puaScheduler.scheduleRepeatedSchedule(
		generateDailyRepeatedSchedule(repeatedStart, 30*time.Second), "ubuntu",
	)

	job := puaScheduler.GetNextJob()
	assert.Equal(t, singleStart, job.NextRun())
}

func TestPuaScheduler_CleanupSchedule_ShouldLogErrorIfNoJobsArePresent(t *testing.T) {
	logger, hook := test.NewNullLogger()
	pua_logger.SetLogger(logger.WithField("a", "b"))
	logger.SetLevel(logrus.DebugLevel)
	puaScheduler, _ := NewPuaScheduler(
		client,
		"abcd",
		doNothingUpdater,
		downloader.NewDownloader(
			10*time.Minute,
			6*time.Hour,
			mockDownloadExecutor,
			pua_logger.Logger(),
			metadata.NewController(),
		),
		pua_logger.Logger(),
	)

	puaScheduler.CleanupSchedule()

	assert.Contains(
		t,
		hook.LastEntry().Message,
		"scheduler failed to remove cron job by 'RepeatedSchedule' tag - gocron: no jobs found with given tag",
	)
	assert.Contains(
		t,
		hook.AllEntries()[len(hook.AllEntries())-2].Message,
		"scheduler failed to remove cron job by 'SingleSchedule' tag - gocron: no jobs found with given tag",
	)
}

func TestDoNothingUpdater_DoesNothing(t *testing.T) {
	u := DoNothingUpdater{}
	u.StartUpdate(1)
}

func TestPuaScheduler_IsUpdateAlreadyApplied(t *testing.T) {
	puaScheduler, _ := NewPuaScheduler(
		client,
		"abcd",
		doNothingUpdater,
		downloader.NewDownloader(
			10*time.Minute,
			6*time.Hour,
			mockDownloadExecutor,
			pua_logger.Logger(),
			metadata.NewController(),
		),
		pua_logger.Logger(),
	)

	t.Run("[Edge Microvisor Toolkit] IsUpdateAlreadyApplied return true when same SHA detected.", func(t *testing.T) {
		actualOSSource := &pb.OSProfileUpdateSource{
			OsImageUrl:     "https://www.example.com/",
			OsImageId:      "example-image-id",
			OsImageSha:     "example-image-sha",
			ProfileName:    "example-profile-name",
			ProfileVersion: "example-profile-version",
		}

		desiredOSSource := &pb.OSProfileUpdateSource{
			OsImageUrl:     "https://www.example.com/2",
			OsImageId:      "example-image-id-2",
			OsImageSha:     "example-image-sha",
			ProfileName:    "example-profile-name-2",
			ProfileVersion: "example-profile-version-2",
		}

		metadata.SetMetaOSProfileUpdateSourceActual(actualOSSource)
		metadata.SetMetaOSProfileUpdateSourceDesired(desiredOSSource)

		assert.True(t, puaScheduler.IsUpdateAlreadyApplied("emt"))
	})

	t.Run("[Edge Microvisor Toolkit] IsUpdateAlreadyApplied return false when different SHA detected.", func(t *testing.T) {
		actualOSSource := &pb.OSProfileUpdateSource{
			OsImageUrl:     "https://www.example.com/",
			OsImageId:      "example-image-id",
			OsImageSha:     "example-image-sha",
			ProfileName:    "example-profile-name",
			ProfileVersion: "example-profile-version",
		}

		desiredOSSource := &pb.OSProfileUpdateSource{
			OsImageUrl:     "https://www.example.com/2",
			OsImageId:      "example-image-id-2",
			OsImageSha:     "example-image-sha-2",
			ProfileName:    "example-profile-name-2",
			ProfileVersion: "example-profile-version-2",
		}

		metadata.SetMetaOSProfileUpdateSourceActual(actualOSSource)
		metadata.SetMetaOSProfileUpdateSourceDesired(desiredOSSource)

		assert.False(t, puaScheduler.IsUpdateAlreadyApplied("emt"))
	})

}
