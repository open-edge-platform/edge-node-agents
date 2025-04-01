// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package downloader

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/metadata"
	pb "github.com/open-edge-platform/infra-managers/maintenance/pkg/api/maintmgr/v1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func createTestMetadata(t *testing.T) *metadata.MetaController {
	metadataController := metadata.NewController()
	file, err := os.CreateTemp("/tmp", "metadata-")
	assert.NoError(t, err)
	defer file.Close()
	metadata.MetaPath = file.Name()
	err = metadata.InitMetadata()
	assert.NoError(t, err)
	return metadataController
}

func createNullLogger() *logrus.Entry {
	logger := logrus.New()
	logger.Out = io.Discard
	return logger.WithFields(logrus.Fields{"test": true})
}

func TestCalculateDownloadStart(t *testing.T) {
	type testCase struct {
		name            string
		now             time.Time
		updateStart     time.Time
		downloadWindow  time.Duration
		immediateWindow time.Duration
		expectNow       bool      // If true, expect downloadStart == now
		expectInRange   bool      // If true, expect downloadStart within a range
		rangeStart      time.Time // Start of expected range
		rangeEnd        time.Time // End of expected range
	}

	// Initialize common time variables
	baseDate := time.Date(2024, time.March, 15, 12, 0, 0, 0, time.UTC)
	updateStart := baseDate.Add(1 * time.Hour)
	downloadWindow := 6 * time.Hour
	immediateWindow := 10 * time.Minute
	immediateWindowStart := updateStart.Add(-immediateWindow)
	downloadWindowStart := immediateWindowStart.Add(-downloadWindow)

	testCases := []testCase{
		{
			name:            "Now well before download window",
			now:             downloadWindowStart.Add(-1 * time.Hour),
			updateStart:     updateStart,
			downloadWindow:  downloadWindow,
			immediateWindow: immediateWindow,
			expectNow:       false,
			expectInRange:   true,
			rangeStart:      downloadWindowStart,
			rangeEnd:        immediateWindowStart,
		},
		{
			name:            "Now at the start of download window",
			now:             downloadWindowStart,
			updateStart:     updateStart,
			downloadWindow:  downloadWindow,
			immediateWindow: immediateWindow,
			expectNow:       false,
			expectInRange:   true,
			rangeStart:      downloadWindowStart,
			rangeEnd:        immediateWindowStart,
		},
		{
			name:            "Now within download window but not in immediate window",
			now:             downloadWindowStart.Add(2 * time.Hour),
			updateStart:     updateStart,
			downloadWindow:  downloadWindow,
			immediateWindow: immediateWindow,
			expectNow:       false,
			expectInRange:   true,
			rangeStart:      downloadWindowStart.Add(2 * time.Hour),
			rangeEnd:        immediateWindowStart,
		},
		{
			name:            "Now within immediate window before updateStart",
			now:             immediateWindowStart.Add(2 * time.Minute),
			updateStart:     updateStart,
			downloadWindow:  downloadWindow,
			immediateWindow: immediateWindow,
			expectNow:       true,
			expectInRange:   false,
		},
		{
			name:            "Now exactly at immediate window start",
			now:             immediateWindowStart,
			updateStart:     updateStart,
			downloadWindow:  downloadWindow,
			immediateWindow: immediateWindow,
			expectNow:       true,
			expectInRange:   false,
		},
		{
			name:            "Now exactly at updateStart",
			now:             updateStart,
			updateStart:     updateStart,
			downloadWindow:  downloadWindow,
			immediateWindow: immediateWindow,
			expectNow:       true,
			expectInRange:   false,
		},
		{
			name:            "Now after updateStart",
			now:             updateStart.Add(30 * time.Minute),
			updateStart:     updateStart,
			downloadWindow:  downloadWindow,
			immediateWindow: immediateWindow,
			expectNow:       true,
			expectInRange:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testDownloader := NewDownloader(tc.immediateWindow, tc.downloadWindow, &FakeDownloadExecutor{}, createNullLogger(), metadata.NewController())
			gotRangeStart, gotRangeEnd := testDownloader.CalculateDownloadStart(tc.now, tc.updateStart)

			if tc.expectNow {
				assert.Equal(t, tc.now, gotRangeStart, "start of range should be now")
				assert.Equal(t, tc.now, gotRangeEnd, "end of range should be now")
			} else if tc.expectInRange {
				assert.Equal(t, tc.rangeStart, gotRangeStart)
				assert.Equal(t, tc.rangeEnd, gotRangeEnd)
			} else {
				t.Error("Invalid test case (needs to be 'now' or in a range): ", tc.name)
			}
		})
	}
}

// TestGetRandomTimeInRange tests the GetRandomTimeInRange function.
func TestGetRandomTimeInRange(t *testing.T) {
	startTime := time.Date(2023, time.January, 1, 12, 0, 0, 0, time.UTC)
	endTime := time.Date(2023, time.January, 2, 12, 0, 0, 0, time.UTC)
	sameTime := startTime

	testCases := []struct {
		name          string
		start         time.Time
		end           time.Time
		expectedExact time.Time // Used only when start == end
	}{
		{
			name:  "Start before End",
			start: startTime,
			end:   endTime,
		},
		{
			name:          "Start equals End",
			start:         sameTime,
			end:           sameTime,
			expectedExact: sameTime,
		},
		{
			name:  "Start after End",
			start: endTime,
			end:   startTime,
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			result := GetRandomTimeInRange(tc.start, tc.end)

			if tc.start.Equal(tc.end) {
				// When start == end, the result should exactly match
				if !result.Equal(tc.expectedExact) {
					t.Errorf("expected exactly %v, got %v", tc.expectedExact, result)
				}
				return
			}

			// Determine the effective start and end after potential swapping
			effectiveStart, effectiveEnd := tc.start, tc.end
			if effectiveStart.After(effectiveEnd) {
				effectiveStart, effectiveEnd = effectiveEnd, effectiveStart
			}

			// Ensure the result is within the range [effectiveStart, effectiveEnd)
			if result.Before(effectiveStart) || !result.Before(effectiveEnd) {
				t.Errorf("result %v not within range [%v, %v)", result, effectiveStart, effectiveEnd)
			}
		})
	}
}

func Test_NonBlockingDownloadAttempt_DoesNotBlockUpdates(t *testing.T) {
	testDownloader := NewDownloader(10*time.Minute, 6*time.Hour, &FakeDownloadExecutor{}, createNullLogger(), metadata.NewController())

	testDownloader.LockForUpdate()
	defer testDownloader.Unlock()

	if testDownloader.tryLockForDownload() {
		t.Error("Expected the download attempt to fail due to an active update, but it succeeded")
	}
}

// FakeDownloadExecutor is a mock implementation of the DownloadExecutor interface.
type FakeDownloadExecutor struct {
	downloadCalled bool
	downloadArgs   *pb.OSProfileUpdateSource
	downloadChan   chan bool
}

func (f *FakeDownloadExecutor) Download(ctx context.Context, prependToImageURL string, source *pb.OSProfileUpdateSource) error {
	f.downloadCalled = true
	f.downloadArgs = source
	if f.downloadChan != nil {
		f.downloadChan <- true
	}
	return nil
}

// Test for osImageParamsChanged function
func TestDownloader_osImageParamsChanged(t *testing.T) {
	assert := assert.New(t)

	assert.True(AreOsImagesEqual(nil, nil), "Both sources nil should return false")
	assert.False(AreOsImagesEqual(nil, &pb.OSProfileUpdateSource{}), "Nil lastUpdateSource and non-nil newSource should return true")

	// Both not nil, same values
	a := &pb.OSProfileUpdateSource{
		OsImageUrl: "url",
		OsImageId:  "id",
		OsImageSha: "sha",
	}
	b := &pb.OSProfileUpdateSource{
		OsImageUrl: "url",
		OsImageId:  "id",
		OsImageSha: "sha",
	}
	assert.True(AreOsImagesEqual(a, b), "Same OsImage parameters should return false")
}

func TestDownloader_startDownload(t *testing.T) {
	assert := assert.New(t)

	// Create a fake DownloadExecutor
	fakeExecutor := &FakeDownloadExecutor{}

	// Create a Downloader with the fake executor
	metadataController := createTestMetadata(t)
	d := NewDownloader(0, 0, fakeExecutor, logrus.NewEntry(logrus.New()), metadataController)

	// Define an updateSource
	updateSource := &pb.OSProfileUpdateSource{
		OsImageUrl: "http://example.com/image",
		OsImageId:  "image-id",
		OsImageSha: "image-sha",
	}

	// Case 1: Lock is available
	downloadResult := d.startDownload(context.Background(), "", updateSource)

	// Check that Download was called and succeeded
	assert.True(downloadResult, "download result should be true")
	assert.True(fakeExecutor.downloadCalled, "Expected Download to be called")
	assert.Equal(updateSource, fakeExecutor.downloadArgs, "Expected updateSource to be passed")

	// Check that status is DOWNLOADED
	status, err := metadataController.GetMetaUpdateStatus()
	assert.NoError(err)
	assert.Equal(status, pb.UpdateStatus_STATUS_TYPE_DOWNLOADED)

	// Reset
	fakeExecutor.downloadCalled = false
	fakeExecutor.downloadArgs = nil

	// Case 2: Lock is already held, so TryLockForDownload should fail
	// Simulate by locking it before calling startDownload
	d.LockForUpdate()
	defer d.Unlock() // Ensure we unlock after test

	downloadResult = d.startDownload(context.Background(), "", updateSource)

	// Download should not be called
	assert.False(downloadResult, "Download should not have been called")
	assert.False(fakeExecutor.downloadCalled, "Expected Download not to be called since lock is held")
}

func TestDownloader_startDownload_shouldSetLastDownloaded(t *testing.T) {
	assert := assert.New(t)

	// Create a fake DownloadExecutor
	fakeExecutor := &FakeDownloadExecutor{downloadChan: make(chan bool, 1)}

	// Create a Downloader with the fake executor
	metadataController := createTestMetadata(t)
	d := NewDownloader(0, 0, fakeExecutor, logrus.NewEntry(logrus.New()), metadataController)

	// Define desired and booted update sources
	bootedUpdateSource := &pb.OSProfileUpdateSource{
		OsImageUrl: "http://example.com/image-booted",
		OsImageId:  "image-booted-id",
		OsImageSha: "image-booted-sha",
	}

	desiredUpdateSource := &pb.OSProfileUpdateSource{
		OsImageUrl: "http://example.com/image-desired",
		OsImageId:  "image-desired-id",
		OsImageSha: "image-desired-sha",
	}

	lastDownloaded := d.GetLastDownloaded()
	assert.Nil(lastDownloaded)

	d.Notify("", desiredUpdateSource, time.Now(), bootedUpdateSource)

	// Wait for Download to be called
	select {
	case <-fakeExecutor.downloadChan:
		// Download was called
	case <-time.After(1 * time.Second):
		t.Fatal("Download was not called within 1 second")
	}

	// TODO: is there a way to do this without a short sleep? need to give post-download time to update status on disk
	time.Sleep(10 * time.Millisecond)

	// Check that status is DOWNLOADED
	status, err := metadataController.GetMetaUpdateStatus()
	assert.NoError(err)
	assert.Equal(status, pb.UpdateStatus_STATUS_TYPE_DOWNLOADED)

	// Check that last downloaded was updated correctly
	lastDownloaded = d.GetLastDownloaded()
	assert.True(AreOsImagesEqual(desiredUpdateSource, lastDownloaded), "last downloaded should be set to desired update source")
}

func TestDownloader_Notify_blankSystem(t *testing.T) {
	assert := assert.New(t)

	// Create a fake DownloadExecutor with a channel to signal when Download is called
	fakeExecutor := &FakeDownloadExecutor{
		downloadChan: make(chan bool, 1),
	}

	// Create a Downloader
	metadataController := createTestMetadata(t)
	d := NewDownloader(0, 0, fakeExecutor, createNullLogger(), metadataController)

	// Provide a notification with new parameters
	newUpdateSource := &pb.OSProfileUpdateSource{
		OsImageUrl: "http://example.com/image",
		OsImageId:  "image-id",
		OsImageSha: "image-sha",
	}
	now := time.Now()
	d.Notify("", newUpdateSource, now, &pb.OSProfileUpdateSource{})

	// Wait for Download to be called
	select {
	case <-fakeExecutor.downloadChan:
		// Download was called
	case <-time.After(1 * time.Second):
		t.Fatal("Download was not called within 1 second")
	}

	// Check that Download was called
	assert.True(fakeExecutor.downloadCalled, "Expected Download to be called")
	assert.Equal(newUpdateSource, fakeExecutor.downloadArgs, "Expected updateSource to be passed to Download")

	// reset executor
	fakeExecutor.downloadCalled = false

	// Now, notify with the same source/update start
	d.Notify("", newUpdateSource, now, &pb.OSProfileUpdateSource{})

	// Wait to ensure Download is not called
	select {
	case <-fakeExecutor.downloadChan:
		t.Fatal("Download should not have been called")
	case <-time.After(100 * time.Millisecond):
		// No download as expected
	}

	// Check that Download was not called
	assert.False(fakeExecutor.downloadCalled, "Expected Download not to be called")
}

func TestDownloader_Notify_sameAsOnSystem(t *testing.T) {
	assert := assert.New(t)

	// Create a fake DownloadExecutor with a channel to signal when Download is called
	fakeExecutor := &FakeDownloadExecutor{
		downloadChan: make(chan bool, 1),
	}

	// Create a Downloader
	metadataController := createTestMetadata(t)
	d := NewDownloader(0, 0, fakeExecutor, createNullLogger(), metadataController)

	// Provide a notification with new parameters
	newUpdateSource := &pb.OSProfileUpdateSource{
		OsImageUrl: "http://example.com/image",
		OsImageId:  "image-id",
		OsImageSha: "image-sha",
	}
	now := time.Now()
	// blank system
	d.Notify("", newUpdateSource, now, &pb.OSProfileUpdateSource{})

	// Wait for Download to be called
	select {
	case <-fakeExecutor.downloadChan:
		// Download was called
	case <-time.After(1 * time.Second):
		t.Fatal("Download was not called within 1 second")
	}

	// Check that Download was called
	assert.True(fakeExecutor.downloadCalled, "Expected Download to be called")
	assert.Equal(newUpdateSource, fakeExecutor.downloadArgs, "Expected updateSource to be passed to Download")

	// reset executor
	fakeExecutor.downloadCalled = false

	// Now, notify assuming the system has been updated
	d.Notify("", newUpdateSource, now, newUpdateSource)

	// Wait to ensure Download is not called
	select {
	case <-fakeExecutor.downloadChan:
		t.Fatal("Download should not have been called")
	case <-time.After(100 * time.Millisecond):
		// No download as expected
	}

	// Check that Download was not called
	assert.False(fakeExecutor.downloadCalled, "Expected Download not to be called")
}
