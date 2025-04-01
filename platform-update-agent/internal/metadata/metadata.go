// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/logger"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/utils"
	pb "github.com/open-edge-platform/infra-managers/maintenance/pkg/api/maintmgr/v1"
)

type UpdateType string

const (
	NONE UpdateType = "NONE"
	SELF UpdateType = "SELF"
	INBM UpdateType = "INBM"
	OS   UpdateType = "OS"
	NEW  UpdateType = "NEW"
)

var (
	MetaPath          string
	log               = logger.Logger()
	metadataFileMutex = sync.Mutex{}
)

type Meta struct {
	UpdateTime       string             `json:"updateTime"`
	UpdateDuration   int64              `json:"updateDuration"`
	UpdateInProgress string             `json:"updateInProgress"`
	UpdateStatus     string             `json:"updateStatus"`
	UpdateLog        string             `json:"updateLog"`
	SingleSchedule   *pb.SingleSchedule `json:"singleSchedule"`
	// Deprecated: Do not use.
	RepeatedSchedule       *pb.RepeatedSchedule   `json:"repeatedSchedule"`
	RepeatedSchedules      []*pb.RepeatedSchedule `json:"repeatedSchedules"`
	UpdateSource           *pb.UpdateSource       `json:"updateSource"`
	SingleScheduleFinished bool                   `json:"singleScheduleFinished"`
	InstalledPackages      string                 `json:"installedPackages"`
	// Actual is what is reported to MM and gets set after a successful update
	OSProfileUpdateSourceActual *pb.OSProfileUpdateSource `json:"osProfileUpdateSourceActual,omitempty"`
	// Desired is what MM says we should be updating to
	OSProfileUpdateSourceDesired *pb.OSProfileUpdateSource `json:"osProfileUpdateSourceDesired,omitempty"`
}

func InitMetadata() error {

	fmt.Printf("MetaPath: %v", MetaPath)

	exists, err := fileExists(MetaPath)
	if err != nil {
		log.Errorf("File exists check failed; %v", err)
	}

	if !exists {
		f, err := os.Create(MetaPath)
		if err != nil {
			log.Errorf("Creating file failed: %s", MetaPath)
			return err
		}
		defer f.Close()
		log.Infof("New metadata file created: %s", MetaPath)
		// Set file permission
		err = os.Chmod(MetaPath, os.FileMode(0600))
		if err != nil {
			log.Errorf("File permission set failed: %s", MetaPath)
			return err
		}
		log.Infof("New metadata file permission set.")
	}

	notEmpty, err := fileHasContent()
	if err != nil {
		log.Errorf("File content check failed: %v", err)
	}

	if notEmpty {
		return nil
	}

	log.Infoln("Initializing metadata file ", MetaPath)

	status := pb.UpdateStatus_STATUS_TYPE_UP_TO_DATE.String()

	m := Meta{
		UpdateStatus:                 status,
		UpdateInProgress:             "",
		OSProfileUpdateSourceActual:  nil,
		OSProfileUpdateSourceDesired: nil,
	}
	err = writeMeta(m)
	if err != nil {
		log.Error("Error initializing metadata")
	}
	return nil
}

func fileExists(f string) (bool, error) {
	_, err := os.Stat(f)
	if err != nil {
		if os.IsNotExist(err) {
			log.Warnf("Metadata file does not exist: %v", f)
			return false, nil
		} else {
			log.Error("Checking if file exists failed")
			return false, err
		}
	} else {
		return true, nil
	}
}

func fileHasContent() (bool, error) {
	info, err := os.Stat(MetaPath)
	if err != nil {
		log.Fatal(err)
		return false, err
	}

	s := info.Size()
	if s == 0 {
		log.Warnln("Metadata file is empty")
		return false, nil
	}
	return true, nil
}

func ReadMeta() (Meta, error) {
	meta := Meta{}
	content, err := os.ReadFile(MetaPath)
	if err != nil {
		log.Errorf("Reading metadata failed: %v %v", MetaPath, err)
		return Meta{}, err
	}

	err = utils.IsSymlink(MetaPath)
	if err != nil {
		return Meta{}, err
	}

	if len(content) == 0 {
		log.Errorln("no content found in metadata file")
		return Meta{}, fmt.Errorf("no content found in metadata file")
	}

	err = json.Unmarshal(content, &meta)
	if err != nil {
		log.Errorf("Unmarshaling metadata failed: %v", err)
		return Meta{}, err
	}
	return meta, nil
}

func writeMeta(m Meta) error {

	err := utils.IsSymlink(MetaPath)
	if err != nil {
		return err
	}

	content, err := json.Marshal(m)
	if err != nil {
		log.Errorf("Writing metadata failed: %v", err)
		return err
	}
	err = os.WriteFile(MetaPath, content, 0600)
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}

func SetMetaUpdateStatus(s pb.UpdateStatus_StatusType) error {
	metadataFileMutex.Lock()
	defer metadataFileMutex.Unlock()

	meta, err := ReadMeta()
	if err != nil {
		return err
	}

	meta.UpdateStatus = s.String()
	err = writeMeta(meta)
	if err != nil {
		return err
	}

	return nil
}

func SetMetaUpdateLog(s string) error {
	metadataFileMutex.Lock()
	defer metadataFileMutex.Unlock()

	meta, err := ReadMeta()
	if err != nil {
		return err
	}

	meta.UpdateLog = s
	err = writeMeta(meta)
	if err != nil {
		return err
	}

	return nil
}

func SetMetaUpdateSource(updateSource *pb.UpdateSource) error {
	metadataFileMutex.Lock()
	defer metadataFileMutex.Unlock()

	meta, err := ReadMeta()
	if err != nil {
		return err
	}

	meta.UpdateSource = updateSource
	err = writeMeta(meta)
	if err != nil {
		return err
	}

	return nil
}

func SetMetaSchedules(singleSchedule *pb.SingleSchedule, repeatedSchedules []*pb.RepeatedSchedule, singleScheduleFinished bool) error {
	metadataFileMutex.Lock()
	defer metadataFileMutex.Unlock()

	meta, err := ReadMeta()
	if err != nil {
		return err
	}

	if singleSchedule != nil {
		meta.SingleSchedule = singleSchedule
	}

	if repeatedSchedules != nil {
		meta.RepeatedSchedules = repeatedSchedules
	}

	meta.SingleScheduleFinished = singleScheduleFinished
	err = writeMeta(meta)
	if err != nil {
		return err
	}

	return nil
}

func SetSingleScheduleFinished(singleScheduleFinished bool) error {
	metadataFileMutex.Lock()
	defer metadataFileMutex.Unlock()

	meta, err := ReadMeta()
	if err != nil {
		return err
	}

	meta.SingleScheduleFinished = singleScheduleFinished
	err = writeMeta(meta)
	log.Info("marking single schedule job as done")
	if err != nil {
		return err
	}
	return nil
}

func IsInsideSingleScheduleWindow(currentTime time.Time) (bool, error) {

	meta, err := ReadMeta()
	if err != nil {
		log.Errorln("no content found in metadata file")
		return false, err
	}

	// Skip if no singleSchedule
	if meta.SingleSchedule == nil {
		return false, nil
	}
	startTime := time.Unix(int64(meta.SingleSchedule.StartSeconds), 0)
	EndTime := time.Unix(int64(meta.SingleSchedule.EndSeconds), 0)

	if currentTime.After(startTime) && currentTime.Before(EndTime) {
		log.Info("PUA is within single schedule window.")
		return true, nil
	}

	return false, nil
}

func SetMetaUpdateInProgress(updateType UpdateType) error {
	metadataFileMutex.Lock()
	defer metadataFileMutex.Unlock()

	meta, err := ReadMeta()
	if err != nil {
		return err
	}

	meta.UpdateInProgress = string(updateType)
	err = writeMeta(meta)
	if err != nil {
		return err
	}

	return nil
}

func SetMetaUpdateDuration(updateDuration int64) error {
	metadataFileMutex.Lock()
	defer metadataFileMutex.Unlock()

	meta, err := ReadMeta()
	if err != nil {
		return err
	}

	meta.UpdateDuration = updateDuration
	err = writeMeta(meta)
	if err != nil {
		return err
	}

	return nil
}

func SetMetaUpdateTime(updateTime time.Time) error {
	metadataFileMutex.Lock()
	defer metadataFileMutex.Unlock()

	meta, err := ReadMeta()
	if err != nil {
		return err
	}

	meta.UpdateTime = updateTime.Format(time.RFC3339)
	err = writeMeta(meta)
	if err != nil {
		return err
	}

	return nil
}

func SetInstalledPackages(packages string) error {
	metadataFileMutex.Lock()
	defer metadataFileMutex.Unlock()

	meta, err := ReadMeta()
	if err != nil {
		return err
	}

	meta.InstalledPackages = packages
	err = writeMeta(meta)
	if err != nil {
		return err
	}

	return nil
}

func SetMetaOSProfileUpdateSourceActual(osProfileUpdateSource *pb.OSProfileUpdateSource) error {
	metadataFileMutex.Lock()
	defer metadataFileMutex.Unlock()

	meta, err := ReadMeta()
	if err != nil {
		return err
	}

	meta.OSProfileUpdateSourceActual = osProfileUpdateSource
	err = writeMeta(meta)
	if err != nil {
		return err
	}

	return nil
}

func SetMetaOSProfileUpdateSourceDesired(osProfileUpdateSource *pb.OSProfileUpdateSource) error {
	metadataFileMutex.Lock()
	defer metadataFileMutex.Unlock()

	meta, err := ReadMeta()
	if err != nil {
		return err
	}

	meta.OSProfileUpdateSourceDesired = osProfileUpdateSource
	err = writeMeta(meta)
	if err != nil {
		return err
	}

	return nil
}

func GetMetaUpdateSource() (*pb.UpdateSource, error) {
	meta, err := ReadMeta()

	if err != nil {
		return &pb.UpdateSource{}, err
	}

	return meta.UpdateSource, nil
}

func GetMetaUpdateStatus() (pb.UpdateStatus_StatusType, error) {
	meta, err := ReadMeta()

	if err != nil {
		return pb.UpdateStatus_STATUS_TYPE_FAILED, err
	}

	return convertStatusType(meta.UpdateStatus), nil
}

func GetMetaUpdateLog() (string, error) {
	meta, err := ReadMeta()

	if err != nil {
		return "", err
	}

	return meta.UpdateLog, nil
}

func GetMetaUpdateInProgress() (string, error) {
	meta, err := ReadMeta()

	if err != nil {
		return "", err
	}

	return meta.UpdateInProgress, nil
}

func GetMetaUpdateDuration() (int64, error) {
	meta, err := ReadMeta()

	if err != nil {
		return 0, err
	}

	return meta.UpdateDuration, nil
}

func GetMetaUpdateTime() (time.Time, error) {
	meta, err := ReadMeta()

	if err != nil {
		return time.Time{}, err
	}

	updateTime, err := time.Parse(time.RFC3339, meta.UpdateTime)
	if err != nil {
		return time.Time{}, err
	}

	return updateTime, nil
}

func GetInstalledPackages() (string, error) {
	meta, err := ReadMeta()

	if err != nil {
		return "", err
	}

	return meta.InstalledPackages, nil
}

func GetMetaOSProfileUpdateSourceActual() (*pb.OSProfileUpdateSource, error) {
	meta, err := ReadMeta()
	if err != nil {
		return &pb.OSProfileUpdateSource{}, err
	}

	return meta.OSProfileUpdateSourceActual, nil
}

func GetMetaOSProfileUpdateSourceDesired() (*pb.OSProfileUpdateSource, error) {
	meta, err := ReadMeta()
	if err != nil {
		return &pb.OSProfileUpdateSource{}, err
	}

	return meta.OSProfileUpdateSourceDesired, nil
}

func convertStatusType(s string) pb.UpdateStatus_StatusType {
	m := map[string]pb.UpdateStatus_StatusType{
		pb.UpdateStatus_STATUS_TYPE_UP_TO_DATE.String():  pb.UpdateStatus_STATUS_TYPE_UP_TO_DATE,
		pb.UpdateStatus_STATUS_TYPE_STARTED.String():     pb.UpdateStatus_STATUS_TYPE_STARTED,
		pb.UpdateStatus_STATUS_TYPE_UPDATED.String():     pb.UpdateStatus_STATUS_TYPE_UPDATED,
		pb.UpdateStatus_STATUS_TYPE_FAILED.String():      pb.UpdateStatus_STATUS_TYPE_FAILED,
		pb.UpdateStatus_STATUS_TYPE_DOWNLOADING.String(): pb.UpdateStatus_STATUS_TYPE_DOWNLOADING,
		pb.UpdateStatus_STATUS_TYPE_DOWNLOADED.String():  pb.UpdateStatus_STATUS_TYPE_DOWNLOADED,
		pb.UpdateStatus_STATUS_TYPE_UNSPECIFIED.String(): pb.UpdateStatus_STATUS_TYPE_UNSPECIFIED,
	}
	return m[s]
}

type MetaController struct {
	SetMetaUpdateStatus                 func(s pb.UpdateStatus_StatusType) error
	GetMetaUpdateStatus                 func() (s pb.UpdateStatus_StatusType, err error)
	SetMetaUpdateLog                    func(s string) error
	SetMetaUpdateTime                   func(updateTime time.Time) error
	SetMetaUpdateDuration               func(updateDuration int64) error
	SetMetaUpdateInProgress             func(updateType UpdateType) error
	SetInstallPackageList               func(packages string) error
	GetMetaUpdateTime                   func() (time.Time, error)
	GetMetaUpdateDuration               func() (int64, error)
	GetMetaUpdateSource                 func() (*pb.UpdateSource, error)
	GetInstallPackageList               func() (string, error)
	SetMetaProfileName                  func(name string) error
	SetMetaProfileVersion               func(version string) error
	SetMetaOSImageID                    func(osImageID string) error
	SetMetaOSType                       func(osType pb.PlatformUpdateStatusResponse_OSType) error
	SetMetaOSProfileUpdateSourceActual  func(osProfileUpdateSource *pb.OSProfileUpdateSource) error
	SetMetaOSProfileUpdateSourceDesired func(osProfileUpdateSource *pb.OSProfileUpdateSource) error
	GetMetaProfileName                  func() (string, error)
	GetMetaProfileVersion               func() (string, error)
	GetMetaOSImageID                    func() (string, error)
	GetMetaOSType                       func() (pb.PlatformUpdateStatusResponse_OSType, error)
	GetMetaOSProfileUpdateSourceActual  func() (*pb.OSProfileUpdateSource, error)
	GetMetaOSProfileUpdateSourceDesired func() (*pb.OSProfileUpdateSource, error)
	SetSingleScheduleFinished           func(singleScheduleFinished bool) error
	IsInsideSingleScheduleWindow        func(currentTime time.Time) (bool, error)
}

func NewController() *MetaController {
	return &MetaController{
		SetMetaUpdateStatus:                 SetMetaUpdateStatus,
		GetMetaUpdateStatus:                 GetMetaUpdateStatus,
		SetMetaUpdateLog:                    SetMetaUpdateLog,
		SetMetaUpdateTime:                   SetMetaUpdateTime,
		SetMetaUpdateDuration:               SetMetaUpdateDuration,
		SetInstallPackageList:               SetInstalledPackages,
		GetMetaUpdateSource:                 GetMetaUpdateSource,
		SetMetaUpdateInProgress:             SetMetaUpdateInProgress,
		GetMetaUpdateTime:                   GetMetaUpdateTime,
		GetMetaUpdateDuration:               GetMetaUpdateDuration,
		GetInstallPackageList:               GetInstalledPackages,
		SetMetaOSProfileUpdateSourceActual:  SetMetaOSProfileUpdateSourceActual,
		SetMetaOSProfileUpdateSourceDesired: SetMetaOSProfileUpdateSourceDesired,
		GetMetaOSProfileUpdateSourceActual:  GetMetaOSProfileUpdateSourceActual,
		GetMetaOSProfileUpdateSourceDesired: GetMetaOSProfileUpdateSourceDesired,
		SetSingleScheduleFinished:           SetSingleScheduleFinished,
		IsInsideSingleScheduleWindow:        IsInsideSingleScheduleWindow,
	}
}
