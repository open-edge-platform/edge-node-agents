// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus/hooks/test"

	pb "github.com/open-edge-platform/infra-managers/maintenance/pkg/api/maintmgr/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper function to init metadata
func initMetaDataHelper(t *testing.T) (*os.File, error) {
	file, err := os.CreateTemp("", "json")
	require.Nil(t, err)
	defer file.Close()
	MetaPath = file.Name()
	err = InitMetadata()
	return file, err

}

func Test_InitMetadata_NoErrorIfPathValid(t *testing.T) {
	file, err := initMetaDataHelper(t)
	defer os.Remove(file.Name())
	assert.Nil(t, err)

}

func Test_InitMetadata_CreatesFileIfPathDoesntExist(t *testing.T) {
	MetaPath = "/tmp/newFile.json"
	err := InitMetadata()
	assert.Nil(t, err)
	assert.FileExists(t, "/tmp/newFile.json")
	os.Remove("/tmp/newFile.json")
}

func Test_InitMetadata_AssignsStatusCorrectlyToJSONFile(t *testing.T) {
	file, err := initMetaDataHelper(t)
	defer os.Remove(file.Name())
	assert.Nil(t, err)

	meta, err := ReadMeta()
	require.Nil(t, err)

	assert.Equal(t, meta.UpdateStatus, pb.UpdateStatus_STATUS_TYPE_UP_TO_DATE.String())
}

func Test_ReadMetadata_NoMetadataReturnedWhenSymlinkIsInputted(t *testing.T) {

	symLinkPath := "/tmp/symlink_temp.txt"

	file, _ := os.CreateTemp("", "metadata_temp.txt")
	defer file.Close()
	err := os.Symlink(file.Name(), symLinkPath)

	defer os.Remove(symLinkPath)
	defer os.Remove(file.Name())

	require.Nil(t, err)

	MetaPath = symLinkPath
	err = InitMetadata()

	assert.Nil(t, err)

	meta, err := ReadMeta()
	assert.Equal(t, Meta{}, meta)
	assert.NotNil(t, err)

}

func Test_ReadMeta_NoMetadataReturnedWhenInvalidFileInputted(t *testing.T) {
	file, err := os.CreateTemp("", "metadata_temp.txt")
	require.Nil(t, err)
	defer file.Close()
	defer os.Remove(file.Name())

	_, err = file.WriteString("invalid metadata format")
	require.Nil(t, err)

	MetaPath = file.Name()
	err = InitMetadata()
	assert.Nil(t, err)
	meta, err := ReadMeta()
	assert.NotNil(t, err)
	assert.Equal(t, Meta{}, meta)

}

func Test_MetaUpdateStatus_TestGettersAndSetters(t *testing.T) {
	file, err := initMetaDataHelper(t)
	defer os.Remove(file.Name())
	require.Nil(t, err)

	err = SetMetaUpdateStatus(pb.UpdateStatus_STATUS_TYPE_DOWNLOADED)
	assert.Nil(t, err)

	status, err := GetMetaUpdateStatus()
	assert.Nil(t, err)
	assert.Equal(t, status, pb.UpdateStatus_STATUS_TYPE_DOWNLOADED)
}

func Test_GetMetaUpdateStatus_ShouldReturnErrorIfMetadataFileDoesntExists(t *testing.T) {
	MetaPath = "/tmp/path-that-doesnt-exists"

	status, err := GetMetaUpdateStatus()

	assert.Equal(t, pb.UpdateStatus_STATUS_TYPE_FAILED, status)
	assert.ErrorContains(t, err, "open /tmp/path-that-doesnt-exists: no such file or directory")
}

func Test_SetMetaUpdateStatus_ShouldReturnErrorIfMetadataFileDoesntExists(t *testing.T) {
	MetaPath = "/tmp/path-that-doesnt-exists"

	err := SetMetaUpdateStatus(pb.UpdateStatus_STATUS_TYPE_UPDATED)

	assert.ErrorContains(t, err, "open /tmp/path-that-doesnt-exists: no such file or directory")
}

func Test_MetaUpdateSource_GettersandSetters(t *testing.T) {
	file, err := initMetaDataHelper(t)
	defer os.Remove(file.Name())
	require.Nil(t, err)

	updateRes := pb.UpdateSource{}
	updateRes.CustomRepos = []string{"Types: deb\nURIs: https://files.internal.example.intel.com\nSuites: example\nComponents: release\nSigned-By:\npublic GPG key"}

	err = SetMetaUpdateSource(&updateRes)
	assert.Nil(t, err)

	source, err := GetMetaUpdateSource()
	assert.Nil(t, err)
	assert.Equal(t, source, &updateRes)
}

func Test_GetMetaUpdateSource_ShouldReturnErrorIfMetadataFileDoesntExists(t *testing.T) {
	MetaPath = "/tmp/path-that-doesnt-exists"

	source, err := GetMetaUpdateSource()

	assert.Zero(t, source.OsRepoUrl)
	assert.Zero(t, source.KernelCommand)
	assert.Zero(t, source.CustomRepos)
	assert.ErrorContains(t, err, "open /tmp/path-that-doesnt-exists: no such file or directory")
}

func Test_SetMetaUpdateSource_ShouldReturnErrorIfMetadataFileDoesntExists(t *testing.T) {
	MetaPath = "/tmp/path-that-doesnt-exists"

	err := SetMetaUpdateSource(nil)

	assert.ErrorContains(t, err, "open /tmp/path-that-doesnt-exists: no such file or directory")
}

func Test_InstalledPackages_GettersandSetters(t *testing.T) {
	file, err := initMetaDataHelper(t)
	defer os.Remove(file.Name())
	require.Nil(t, err)

	installedPackages := "intel-opencl-icd\ncluster-agent"

	err = SetInstalledPackages(installedPackages)
	assert.Nil(t, err)

	source, err := GetInstalledPackages()
	assert.Nil(t, err)
	assert.Equal(t, source, installedPackages)
}

func Test_GetInstalledPackages_ShouldReturnErrorIfMetaFileDoesntExists(t *testing.T) {
	MetaPath = "/tmp/path-that-doesnt-exists"

	packages, err := GetInstalledPackages()

	assert.Zero(t, packages)
	assert.ErrorContains(t, err, "open /tmp/path-that-doesnt-exists: no such file or directory")
}

func Test_SetInstalledPackages_ShouldReturnErrorIfMetaFileDoesntExists(t *testing.T) {
	MetaPath = "/tmp/path-that-doesnt-exists"

	err := SetInstalledPackages("")

	assert.ErrorContains(t, err, "open /tmp/path-that-doesnt-exists: no such file or directory")
}

func Test_MetaSchedules_TestGettersAndSetters(t *testing.T) {
	file, err := initMetaDataHelper(t)
	defer os.Remove(file.Name())
	require.Nil(t, err)

	singleschedule := pb.SingleSchedule{
		StartSeconds: 50,
		EndSeconds:   60,
	}
	repeatedschedule := []*pb.RepeatedSchedule{
		{
			DurationSeconds: 1,
			CronMinutes:     "3",
			CronHours:       "4",
			CronDayWeek:     "5",
			CronDayMonth:    "6",
			CronMonth:       "7",
		},
	}
	isFinished := true
	err = SetMetaSchedules(&singleschedule, repeatedschedule, isFinished)
	assert.Nil(t, err)

	meta, err := ReadMeta()
	require.Nil(t, err)

	assert.Equal(t, meta.SingleSchedule, &singleschedule)
	assert.Equal(t, meta.RepeatedSchedules, repeatedschedule)
	assert.Equal(t, meta.SingleScheduleFinished, isFinished)
}

func Test_GetMetaSchedules_shouldReturnErrorIfMetadataFileDoesntExists(t *testing.T) {
	MetaPath = "/tmp/path-that-doesnt-exists"

	err := SetMetaSchedules(nil, nil, true)

	assert.ErrorContains(t, err, "open /tmp/path-that-doesnt-exists: no such file or directory")
}

func Test_MetaUpdateInProgress_TestGettersAndSetters(t *testing.T) {
	file, err := initMetaDataHelper(t)
	defer os.Remove(file.Name())
	require.Nil(t, err)
	err = SetMetaUpdateInProgress(INBM)
	assert.Nil(t, err)

	response, err := GetMetaUpdateInProgress()

	assert.Nil(t, err)
	assert.Equal(t, string(INBM), response)
}

func Test_GetMetaUpdateInProgress_shouldReturnErrorIfMetadataFileDoesntExists(t *testing.T) {
	MetaPath = "/tmp/path-that-doesnt-exists"

	response, err := GetMetaUpdateInProgress()

	assert.Zero(t, response)
	assert.ErrorContains(t, err, "open /tmp/path-that-doesnt-exists: no such file or directory")
}

func Test_SetMetaUpdateInProgress_shouldReturnErrorIfMetadataFileDoesntExists(t *testing.T) {
	MetaPath = "/tmp/path-that-doesnt-exists"

	err := SetMetaUpdateInProgress(NONE)

	assert.ErrorContains(t, err, "open /tmp/path-that-doesnt-exists: no such file or directory")
}

func Test_MetaUpdateDuration_TestGettersAndSetters(t *testing.T) {
	file, err := initMetaDataHelper(t)
	defer os.Remove(file.Name())
	require.Nil(t, err)

	var testInt int64 = 256
	err = SetMetaUpdateDuration(testInt)
	assert.Nil(t, err)

	response, err := GetMetaUpdateDuration()
	assert.Nil(t, err)
	assert.Equal(t, response, testInt)
}
func Test_GetMetaUpdateDuration_shouldReturnErrorIfMetadataFileDoesntExists(t *testing.T) {
	MetaPath = "/tmp/path-that-doesnt-exists"

	response, err := GetMetaUpdateDuration()

	assert.Zero(t, response)
	assert.ErrorContains(t, err, "open /tmp/path-that-doesnt-exists: no such file or directory")
}

func Test_SetMetaUpdateDuration_shouldReturnErrorIfMetadataFileDoesntExists(t *testing.T) {
	MetaPath = "/tmp/path-that-doesnt-exists"

	err := SetMetaUpdateDuration(123)

	assert.ErrorContains(t, err, "open /tmp/path-that-doesnt-exists: no such file or directory")
}

func Test_MetaUpdateTime_TestGettersAndSetters(t *testing.T) {
	file, err := initMetaDataHelper(t)
	defer os.Remove(file.Name())
	require.Nil(t, err)
	location, err := time.LoadLocation("UTC")
	require.Nil(t, err)
	timeStruct := time.Date(2000, 1, 1, 12, 00, 00, 00, location)

	err = SetMetaUpdateTime(timeStruct)
	assert.Nil(t, err)

	timeResponse, err := GetMetaUpdateTime()
	assert.Nil(t, err)
	assert.Equal(t, timeResponse, timeStruct)
}

func Test_SetMetaUpdateTime_ShouldFailIfMetadataFileDoesntExists(t *testing.T) {
	MetaPath = "/tmp/path-that-doesnt-exists"

	err := SetMetaUpdateTime(time.Now())

	assert.EqualError(t, err, "open /tmp/path-that-doesnt-exists: no such file or directory")
}
func Test_MetaUpdateTime_ShouldFailIfMetadataFileDoesntExists(t *testing.T) {
	MetaPath = "/tmp/path-that-doesnt-exists"

	time, err := GetMetaUpdateTime()

	assert.Zero(t, time)
	assert.EqualError(t, err, "open /tmp/path-that-doesnt-exists: no such file or directory")
}

func Test_MetaUpdateTime_ShouldFailIfDurationIsRandomString(t *testing.T) {
	file, err := os.CreateTemp("/tmp", "metadata-")
	assert.Nil(t, err)
	defer file.Close()
	MetaPath = file.Name()
	err = InitMetadata()
	assert.NoError(t, err)
	modifyMetadataFile(t, file, "updateTime", "zxcvb")

	time, err := GetMetaUpdateTime()

	assert.Zero(t, time)
	assert.ErrorContains(t, err, `parsing time "zxcvb" as`)
}

func modifyMetadataFile(t *testing.T, file *os.File, fieldName string, fieldValue any) {
	fileContent, err := os.ReadFile(file.Name())
	assert.NoError(t, err)
	fileAsMap := map[string]any{}
	err = json.Unmarshal(fileContent, &fileAsMap)
	assert.NoError(t, err)
	fileAsMap[fieldName] = fieldValue
	fileContent, err = json.Marshal(fileAsMap)
	assert.NoError(t, err)
	err = os.WriteFile(file.Name(), fileContent, 0600)
	assert.NoError(t, err)
}

func TestNewController_ShouldReturnNonNilFunctions(t *testing.T) {
	controller := NewController()

	assert.NotNil(t, controller.SetMetaUpdateInProgress)
	assert.NotNil(t, controller.SetMetaUpdateStatus)
	assert.NotNil(t, controller.GetMetaUpdateDuration)
	assert.NotNil(t, controller.SetMetaUpdateDuration)
	assert.NotNil(t, controller.GetMetaUpdateSource)
	assert.NotNil(t, controller.GetMetaUpdateTime)
	assert.NotNil(t, controller.SetMetaUpdateTime)
}

func Test_writeMeta_shouldReturnErrorIfFileDoesntExists(t *testing.T) {
	MetaPath = "/tmp/path-that-doesnt-exists"

	err := writeMeta(Meta{})

	assert.ErrorContains(t, err, "lstat command failed: lstat /tmp/path-that-doesnt-exists: no such file or directory")
}

func TestReadMeta_shouldReturnErrorIfFileIsEmpty(t *testing.T) {
	file, err := os.CreateTemp("/tmp", "metadata-")
	assert.NoError(t, err)
	defer file.Close()
	MetaPath = file.Name()

	meta, err := ReadMeta()

	assert.Zero(t, meta)
	assert.ErrorContains(t, err, "no content found in metadata file")
}

func Test_fileHasContent_shouldReturnErrorIfUnableToReadFile(t *testing.T) {
	testLogger, _ := test.NewNullLogger()
	log = testLogger.WithField("test", "test")
	testLogger.ExitFunc = func(i int) {}
	MetaPath = "/tmp/path-that-doesnt-exists"

	hasContent, err := fileHasContent()

	assert.False(t, hasContent)
	assert.ErrorContains(t, err, "stat /tmp/path-that-doesnt-exists: no such file or directory")
}

// New helper to create old-format metadata
func initOldMetaDataHelper(t *testing.T) (*os.File, error) {
	oldMeta := Meta{
		UpdateTime:             "2023-08-01T12:00:00Z",
		UpdateDuration:         3600,
		UpdateInProgress:       "NONE",
		UpdateStatus:           pb.UpdateStatus_STATUS_TYPE_UP_TO_DATE.String(),
		UpdateLog:              "Initial update log",
		SingleSchedule:         &pb.SingleSchedule{StartSeconds: 100, EndSeconds: 200},
		RepeatedSchedules:      []*pb.RepeatedSchedule{},
		UpdateSource:           &pb.UpdateSource{KernelCommand: "command"},
		SingleScheduleFinished: false,
		InstalledPackages:      "package1\npackage2",
		// New fields are omitted to simulate old metadata format
	}

	file, err := os.CreateTemp("", "old_metadata.json")
	require.Nil(t, err)

	content, err := json.Marshal(oldMeta)
	require.Nil(t, err)

	_, err = file.Write(content)
	require.Nil(t, err)
	file.Close()

	MetaPath = file.Name()
	return file, err
}

func Test_ReadMeta_OldFormat(t *testing.T) {
	file, err := initOldMetaDataHelper(t)
	defer os.Remove(file.Name())
	require.Nil(t, err)

	meta, err := ReadMeta()
	assert.Nil(t, err)

	// Verify existing fields
	assert.Equal(t, "2023-08-01T12:00:00Z", meta.UpdateTime)
	assert.Equal(t, int64(3600), meta.UpdateDuration)
	assert.Equal(t, "NONE", meta.UpdateInProgress)
	assert.Equal(t, pb.UpdateStatus_STATUS_TYPE_UP_TO_DATE.String(), meta.UpdateStatus)
	assert.Equal(t, "Initial update log", meta.UpdateLog)
	assert.Equal(t, &pb.SingleSchedule{StartSeconds: 100, EndSeconds: 200}, meta.SingleSchedule)
	assert.Empty(t, meta.RepeatedSchedules)
	assert.Equal(t, &pb.UpdateSource{KernelCommand: "command"}, meta.UpdateSource)
	assert.False(t, meta.SingleScheduleFinished)
	assert.Equal(t, "package1\npackage2", meta.InstalledPackages)

	// Verify new fields have zero values
	assert.Nil(t, meta.OSProfileUpdateSourceActual)
	assert.Nil(t, meta.OSProfileUpdateSourceDesired)
}

func Test_InitMetadata_WithOldFormat(t *testing.T) {
	file, err := initOldMetaDataHelper(t)
	defer os.Remove(file.Name())
	require.Nil(t, err)

	// Re-initialize metadata should not overwrite existing data
	err = InitMetadata()
	assert.Nil(t, err)

	meta, err := ReadMeta()
	assert.Nil(t, err)

	// Verify existing fields
	assert.Equal(t, "2023-08-01T12:00:00Z", meta.UpdateTime)
	assert.Equal(t, int64(3600), meta.UpdateDuration)
	assert.Equal(t, "NONE", meta.UpdateInProgress)
	assert.Equal(t, pb.UpdateStatus_STATUS_TYPE_UP_TO_DATE.String(), meta.UpdateStatus)
	assert.Equal(t, "Initial update log", meta.UpdateLog)
	assert.Equal(t, &pb.SingleSchedule{StartSeconds: 100, EndSeconds: 200}, meta.SingleSchedule)
	assert.Empty(t, meta.RepeatedSchedules)
	assert.Equal(t, &pb.UpdateSource{KernelCommand: "command"}, meta.UpdateSource)
	assert.False(t, meta.SingleScheduleFinished)
	assert.Equal(t, "package1\npackage2", meta.InstalledPackages)

	// Verify new fields have zero values
	assert.Nil(t, meta.OSProfileUpdateSourceActual)
	assert.Nil(t, meta.OSProfileUpdateSourceDesired)
}

func Test_MetaSettersAndGetters_NewFields(t *testing.T) {
	file, err := initMetaDataHelper(t)
	defer os.Remove(file.Name())
	require.Nil(t, err)

	// Set new fields
	osProfileUpdateSourceActual := &pb.OSProfileUpdateSource{
		OsImageUrl:     "https://example.com/os-image.img",
		OsImageId:      "os-image-12345",
		OsImageSha:     "sha256:abcdef123456",
		ProfileName:    "EnterpriseProfile",
		ProfileVersion: "v1.2.3",
	}
	err = SetMetaOSProfileUpdateSourceActual(osProfileUpdateSourceActual)
	assert.Nil(t, err)

	osProfileUpdateSourceDesired := &pb.OSProfileUpdateSource{
		OsImageUrl:     "https://example.com/os-image-2.img",
		OsImageId:      "os-image-12346",
		OsImageSha:     "sha256:abcdef123457",
		ProfileName:    "EnterpriseProfile",
		ProfileVersion: "v1.2.4",
	}
	err = SetMetaOSProfileUpdateSourceDesired(osProfileUpdateSourceDesired)
	assert.Nil(t, err)

	// Get and verify new fields
	osProfileSourceActual, err := GetMetaOSProfileUpdateSourceActual()
	assert.Nil(t, err)
	assert.Equal(t, osProfileUpdateSourceActual, osProfileSourceActual)

	osProfileSourceDesired, err := GetMetaOSProfileUpdateSourceDesired()
	assert.Nil(t, err)
	assert.Equal(t, osProfileUpdateSourceDesired, osProfileSourceDesired)
}

func Test_ReadMeta_ShouldHandleMissingNewFields(t *testing.T) {
	file, err := initOldMetaDataHelper(t)
	defer os.Remove(file.Name())
	require.Nil(t, err)

	meta, err := ReadMeta()
	assert.Nil(t, err)

	// New fields should have zero values
	assert.Nil(t, meta.OSProfileUpdateSourceActual)
	assert.Nil(t, meta.OSProfileUpdateSourceDesired)
}

func Test_WriteMeta_WithNewFields(t *testing.T) {
	file, err := os.CreateTemp("", "metadata_with_new_fields.json")
	defer os.Remove(file.Name())
	require.Nil(t, err)
	MetaPath = file.Name()

	// Initialize with default values
	err = InitMetadata()
	assert.Nil(t, err)

	// Set new fields
	osProfileUpdateSourceActual := &pb.OSProfileUpdateSource{
		OsImageUrl:     "https://example.com/os-image-new.img",
		OsImageId:      "os-image-67890",
		OsImageSha:     "sha256:123456abcdef",
		ProfileName:    "StandardProfile",
		ProfileVersion: "v2.0.0",
	}
	err = SetMetaOSProfileUpdateSourceActual(osProfileUpdateSourceActual)
	assert.Nil(t, err)

	osProfileUpdateSourceDesired := &pb.OSProfileUpdateSource{
		OsImageUrl:     "https://example.com/os-image-new-2.img",
		OsImageId:      "os-image-67891",
		OsImageSha:     "sha256:123456abcde1",
		ProfileName:    "StandardProfile",
		ProfileVersion: "v2.0.1",
	}
	err = SetMetaOSProfileUpdateSourceDesired(osProfileUpdateSourceDesired)
	assert.Nil(t, err)

	// Read back and verify
	meta, err := ReadMeta()
	assert.Nil(t, err)

	assert.Equal(t, osProfileUpdateSourceActual, meta.OSProfileUpdateSourceActual)
	assert.Equal(t, osProfileUpdateSourceDesired, meta.OSProfileUpdateSourceDesired)
}

func Test_readMeta_OldFormat_OtherFields(t *testing.T) {
	file, err := initOldMetaDataHelper(t)
	defer os.Remove(file.Name())
	require.Nil(t, err)

	meta, err := ReadMeta()
	assert.Nil(t, err)

	// Existing fields should be correctly populated
	assert.Equal(t, "2023-08-01T12:00:00Z", meta.UpdateTime)
	assert.Equal(t, int64(3600), meta.UpdateDuration)
	assert.Equal(t, "NONE", meta.UpdateInProgress)
	assert.Equal(t, pb.UpdateStatus_STATUS_TYPE_UP_TO_DATE.String(), meta.UpdateStatus)
	assert.Equal(t, "Initial update log", meta.UpdateLog)
	assert.Equal(t, &pb.SingleSchedule{StartSeconds: 100, EndSeconds: 200}, meta.SingleSchedule)
	assert.Empty(t, meta.RepeatedSchedules)
	assert.Equal(t, &pb.UpdateSource{KernelCommand: "command"}, meta.UpdateSource)
	assert.False(t, meta.SingleScheduleFinished)
	assert.Equal(t, "package1\npackage2", meta.InstalledPackages)

	// New fields should be zero-valued
	assert.Nil(t, meta.OSProfileUpdateSourceActual)
	assert.Nil(t, meta.OSProfileUpdateSourceDesired)
}
