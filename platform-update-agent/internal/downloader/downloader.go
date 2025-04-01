// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package downloader

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/metadata"
	pb "github.com/open-edge-platform/infra-managers/maintenance/pkg/api/maintmgr/v1"
	"github.com/sirupsen/logrus"
)

// Downloader handles pre-downloading files before updates.
type Downloader struct {
	immediateWindow    time.Duration // length of immediate download window
	downloadWindow     time.Duration // length of randomized download window
	operationLock      *sync.Mutex   // exclusion lock to coordinate downloads and lastUpdateSource
	stateLock          *sync.Mutex   // state lock used to protect data inside struct
	downloadExecutor   DownloadExecutor
	notificationChan   chan notification // used to notify internal goroutine that something has changed
	log                *logrus.Entry
	downloadTimer      *time.Timer               // timer for next download
	lastUpdateStart    time.Time                 // last update start time we received
	lastUpdateSource   *pb.OSProfileUpdateSource // last update source we received
	lastDownloaded     *pb.OSProfileUpdateSource // last successful download
	metadataController *metadata.MetaController  // metadata controller to make testing easier
}

type DownloadExecutor interface {
	// Download starts a download. It can optionally be canceled with the given context. If there
	// is an error, it will be returned.
	// prependToImageURL will be prepended to the image URL on download
	Download(ctx context.Context, prependToImageURL string, source *pb.OSProfileUpdateSource) error
}

type notification struct {
	UpdateSource          *pb.OSProfileUpdateSource
	NextRunTime           time.Time
	BootedOsProfileSource *pb.OSProfileUpdateSource
}

// Create a new downloader
// immediateWindow sets the length of the immediate download window relative to the update start time
// downloadWindow sets the length of the randomized download window relative to the immediate window start
// downloadExecutor is an interface that can perform the actual download, with ability to cancel and return errors
func NewDownloader(immediateWindow time.Duration,
	downloadWindow time.Duration,
	downloadExecutor DownloadExecutor,
	log *logrus.Entry,
	metadataController *metadata.MetaController) *Downloader {
	d := &Downloader{
		immediateWindow:    immediateWindow,
		downloadWindow:     downloadWindow,
		downloadExecutor:   downloadExecutor,
		operationLock:      &sync.Mutex{},
		stateLock:          &sync.Mutex{},
		notificationChan:   make(chan notification),
		log:                log,
		metadataController: metadataController,
	}

	return d
}

// GetRandomTimeInRange returns a random time between start and end.
// If start is after end, they are swapped.
func GetRandomTimeInRange(start, end time.Time) time.Time {
	if start.After(end) {
		start, end = end, start
	}

	// Calculate the duration between start and end
	duration := end.Sub(start)

	// If duration is zero, return the start time
	if duration == 0 {
		return start
	}

	// Generate a random duration within the range [0, duration)
	randomDuration := time.Duration(rand.Int63n(duration.Nanoseconds()))

	// Add the random duration to the start time
	return start.Add(randomDuration)
}

// CalculateDownloadStart calculates a download start time range given:
//   - now: what time it is now
//   - updateStart: when the update is supposed to start
//   - immediateWindow: how long before updateStart defines the immediate window
//   - downloadWindow: how long before immediate window defines the download window
//
// See function definition and unit tests for algorithm and examples.
func (d *Downloader) CalculateDownloadStart(now time.Time,
	updateStart time.Time) (rangeStart time.Time, rangeEnd time.Time) {
	immediateWindowStart := updateStart.Add(-d.immediateWindow)
	downloadWindowStart := immediateWindowStart.Add(-d.downloadWindow)

	switch {
	case now.Equal(downloadWindowStart) || now.Before(downloadWindowStart):
		return downloadWindowStart, immediateWindowStart
	case now.After(downloadWindowStart) && now.Before(immediateWindowStart):
		return now, immediateWindowStart
	default: // now.Equal(immediateWindowStart) || now.After(immediateWindowStart)
		return now, now
	}
}

// LockForUpdate will acquire an update lock. Updates are expected to attempt to
// acquire this lock before starting, and then check that we are still in update
// window once the lock is acquired.  Unlock() should be called when the update is done.
func (d *Downloader) LockForUpdate() {
	d.operationLock.Lock()
}

// tryLockForDownload will attempt to acquire a lock to perform a download.
// If successful, the function will immediately return true and the downloader
// can assume it's safe to download (no updates will happen).  Unlock() should be
// called when the download is done.
// If not successful, the function will immediately return false and the downloader
// should skip the download as update is already underway.
func (d *Downloader) tryLockForDownload() bool {
	return d.operationLock.TryLock()
}

// Unlock will unlock the update/download lock and return immediately
func (d *Downloader) Unlock() {
	d.operationLock.Unlock()
}

// Notify is used to notify the downloader of the latest update source, next update run time, and currently booted os profile.
// NOTE: currently we are only looking at the OS profile used for Edge Microvisor Toolkit, the OSProfileUpdateSource, as this is the only OS
// that has downloads.
// prependToImageURL will be prepended to the image URL before being parsed as a URL
// This may be the same as the previous source/run time in which case there will be no change,
// but this could require canceling/rescheduling a download.
// This could also be the same as what's already on the system, in which case we should cancel any existing
// scheduled download and return.
// nextRunTime may be time.Time{} (the zero value) which means -do not download-; cancel any existing downloads
func (d *Downloader) Notify(prependToImageURL string, updateSource *pb.OSProfileUpdateSource, nextRunTime time.Time, bootedOsProfileSource *pb.OSProfileUpdateSource) {
	d.stateLock.Lock()
	defer d.stateLock.Unlock()
	// If update source matches what is already on OS, cancel any existing timer and return; do not set new timer

	if AreOsImagesEqual(updateSource, bootedOsProfileSource) {
		// Cancel existing timer if it exists
		if d.downloadTimer != nil {
			d.downloadTimer.Stop()
		}

		d.log.Debugf("DOWNLOADER: Not downloading as update source matches booted OS")
		return
	}

	// Check if update time has changed or if OsImage params have changed
	if !d.lastUpdateStart.Equal(nextRunTime) || !AreOsImagesEqual(updateSource, d.lastUpdateSource) {
		// Cancel existing timer if it exists
		if d.downloadTimer != nil {
			d.downloadTimer.Stop()
		}

		// If the 'next update time' is actually valid (and not time's zero value), calculate new download time
		if !nextRunTime.IsZero() {
			rangeStart, rangeEnd := d.CalculateDownloadStart(time.Now(), nextRunTime)
			downloadTime := GetRandomTimeInRange(rangeStart, rangeEnd)

			delay := time.Until(downloadTime)
			d.downloadTimer = time.AfterFunc(delay, func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute) // INBC specific timeout
				defer cancel()

				if d.startDownload(ctx, prependToImageURL, updateSource) {
					d.setLastDownloaded(updateSource)
				}
			})
		}

		// Update last known values
		d.lastUpdateStart = nextRunTime
		d.lastUpdateSource = updateSource
	}
}

// Helper function to compare OsImage* params
// return true if identical
func AreOsImagesEqual(a *pb.OSProfileUpdateSource, b *pb.OSProfileUpdateSource) bool {
	if a == nil || b == nil {
		return a == b
	}

	return a.OsImageSha == b.OsImageSha
}

// startDownload attempts to run the download; if there is a failure or if it
// cannot get a lock on first attempt, it returns false; otherwise true
// prependToImageURL will be prepended to the image URL on download
func (d *Downloader) startDownload(ctx context.Context, prependToImageURL string, updateSource *pb.OSProfileUpdateSource) bool {
	if d.tryLockForDownload() {
		defer d.Unlock()
		d.log.Infof("Executing download")

		if err := d.setStatus(pb.UpdateStatus_STATUS_TYPE_DOWNLOADING); err != nil {
			return false
		}

		err := d.downloadExecutor.Download(ctx, prependToImageURL, updateSource)
		if err != nil {
			if err := d.setStatus(pb.UpdateStatus_STATUS_TYPE_FAILED); err != nil {
				d.log.Errorf("DOWNLOAD: could not set status to FAILED after failed download: %v", err)
			}

			if err := d.metadataController.SetMetaUpdateLog(err.Error()); err != nil {
				d.log.Errorf("DOWNLOAD: could not set meta update log after failed download: %v", err)
			}

			d.log.Errorf("Error in download: %v", err)
			return false
		}

		if err := d.setStatus(pb.UpdateStatus_STATUS_TYPE_DOWNLOADED); err != nil {
			d.log.Errorf("DOWNLOAD: could not set status to DOWNLOADED after successful download: %v", err)
			return false
		}

		return true

	} else {
		d.log.Infof("Download skipped due to active update or active other download")
		return false
	}
}

func (d *Downloader) setLastDownloaded(lastDownloaded *pb.OSProfileUpdateSource) {
	d.stateLock.Lock()
	defer d.stateLock.Unlock()

	d.lastDownloaded = lastDownloaded
}

// GetLastDownloaded returns the last update source that has been successfully
// downloaded, or `nil` if there has been no successful download yet.
func (d *Downloader) GetLastDownloaded() *pb.OSProfileUpdateSource {
	d.stateLock.Lock()
	defer d.stateLock.Unlock()

	return d.lastDownloaded
}
