// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/sirupsen/logrus"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/metrics"
	"github.com/open-edge-platform/edge-node-agents/common/pkg/status"
	auth "github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/info"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/comms"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/config"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/downloader"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/installer"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/logger"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/metadata"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/scheduler"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/updater"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/utils"
	pb "github.com/open-edge-platform/infra-managers/maintenance/pkg/api/maintmgr/v1"
)

const (
	AGENT_NAME           = "platform-update-agent"
	UPDATE_READ_INTERVAL = 10 * time.Second
	RETRIES              = 5
	NUM_RETRIES          = 3
)

var (
	log                 = logger.Logger()
	kernelRegexp        = regexp.MustCompile(`(^[A-Za-z0-9-_=.,/ ]*$)`)
	lastUpdateTimestamp int64
)

func init() {
	flag.String("config", "", "Config file path")
	flag.String("force-os", "", "Force OS detection to 'ubuntu' or 'emt' for testing")
}

func main() {
	log.Infof("Args: %v\n", os.Args[1:])
	log.Infof("Starting %s - %s\n", info.Component, info.Version)

	flag.Parse()
	configPath := flag.Lookup("config").Value.String()
	puaConfig, err := config.New(configPath)
	if err != nil {
		log.Fatal("Unable to initialize configuration. Platform update agent will terminate")
	}

	logLevel := puaConfig.LogLevel

	setLogLevel(logLevel)

	metadata.MetaPath = puaConfig.MetadataPath

	err = metadata.InitMetadata()
	if err != nil {
		log.Fatalf("Error initializing metadata: %v", err)
	}

	installer := installer.NewWithDefaults()
	if err := installer.ProvisionInbm(context.TODO()); err != nil {
		log.Errorf("failed to provision INBM - %v", err)
	}

	ctx := context.Background()

	// metrics
	shutdown, err := metrics.Init(ctx, puaConfig.MetricsEndpoint, puaConfig.MetricsInterval, info.Component, info.Version)
	if err != nil {
		log.Errorf("Initialization of metrics failed: %v", err)
	} else {
		log.Info("Metrics collection started")
		defer func() {
			err = shutdown(ctx)
			if err != nil && !errors.Is(err, context.Canceled) {
				log.Errorf("Shutting down metrics failed! Error: %v", err)
			}
		}()
	}
	tlsConfig, err := auth.GetAuthConfig(auth.GetAuthContext(ctx, puaConfig.JWT.AccessTokenPath), nil)
	if err != nil {
		log.Fatalf("TLS configuration creation failed! Error: %v", err)
	}

	maintenanceManager := comms.ConnectToEdgeInfrastructureManager(auth.GetAuthContext(ctx, puaConfig.JWT.AccessTokenPath), puaConfig.UpdateServiceURL, tlsConfig)
	osType, err := utils.DetectOS(&utils.RealFileReader{}, flag.Lookup("force-os").Value.String())
	if err != nil {
		log.Fatalf("Terminating: unable to detect OS: %v", err)
	}

	log.Debugf("Detected OS: %s", osType)

	cleaner := updater.NewCleanerWithDefaults(osType)
	downloadExecutor := downloader.NewDownloadExecutor(log)
	puaDownloader := downloader.NewDownloader(puaConfig.ImmediateDownloadWindow,
		puaConfig.DownloadWindow,
		downloadExecutor,
		log,
		metadata.NewController())

	downloadChecker := func() bool {
		// this function should return true if download has downloaded desired update, false otherwise
		// it may return true on OSes that don't require downloads; or update controller may ignore it completely
		// on those OSes (such as Ubuntu)
		log.Debug("Checking to see if download has occurred before starting update")
		lastDownloadedOS := puaDownloader.GetLastDownloaded()
		desiredDownloadedOS, err := metadata.GetMetaOSProfileUpdateSourceDesired()
		if err != nil {
			log.Error("Error checking desired OS before update")
			log.Debugf("Error: %v", err)
			return false
		}
		return downloader.AreOsImagesEqual(lastDownloadedOS, desiredDownloadedOS)
	}
	updateController, err := updater.NewUpdateController(puaConfig.INBCGranularLogsPath, osType, downloadChecker)
	if err != nil {
		log.Fatalf("Terminating: unable to initialize update controller: %v", err)
	}
	puaScheduler, err := scheduler.NewPuaScheduler(maintenanceManager, puaConfig.GUID, updateController, puaDownloader, log)
	if err != nil {
		log.Fatalf("Terminating: unable to initialize PUA scheduler: %v", err)
	}

	log.Infoln("Checking update status")
	if updateStatus, err := metadata.GetMetaUpdateStatus(); err != nil {
		log.Errorf("Error reading metadata file: %v", err)
	} else if err == nil && updateStatus == pb.UpdateStatus_STATUS_TYPE_STARTED {

		updateInProgress, err := metadata.GetMetaUpdateInProgress()
		if err != nil {
			log.Fatalf("Error reading status from metadata file: %v", err)
		}
		if updateInProgress == string(metadata.SELF) {
			continueUpdateAfterPuaRestart(updateController)
		} else if updateInProgress == string(metadata.OS) {
			continueUpdateAfterOsReboot(updateController, puaConfig, cleaner, osType)
		}
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	updateResChan := make(chan *pb.PlatformUpdateStatusResponse)
	go handleEdgeInfrastructureManagerRequest(wg, puaConfig, ctx, maintenanceManager, updateResChan, osType)

	// Sending health status (Ready/NotReady) periodically
	wg.Add(1)
	go SendHealthStatus(wg, ctx, puaConfig.StatusEndpoint, puaConfig.TickerInterval)

	wg.Add(1)

	// Read the latest metadata
	meta, err := metadata.ReadMeta()
	if err != nil {
		log.Fatalf("Failed to read metadata: %v", err)
	}

	go func() {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				return

			case updateRes := <-updateResChan:
				handleUpdateRes(updateRes, puaScheduler, meta, puaDownloader, osType, puaConfig.ReleaseServiceFQDN)
			}
		}
	}()

	wg.Wait()
	log.Info("Exiting Platform Update Agent")
}

func setLogLevel(logLevel string) {
	switch strings.ToLower(logLevel) {
	case "debug":
		log.Logger.SetLevel(logrus.DebugLevel)
	case "error":
		log.Logger.SetLevel(logrus.ErrorLevel)
	default:
		log.Logger.SetLevel(logrus.InfoLevel)
	}
}

func handleEdgeInfrastructureManagerRequest(wg *sync.WaitGroup,
	puaConfig *config.Config,
	ctx context.Context,
	maintenanceManager *comms.Client,
	updateResChan chan *pb.PlatformUpdateStatusResponse,
	osType string,
) {
	defer wg.Done()

	if osType != "ubuntu" && osType != "emt" && osType != "debian" {
		log.Fatalf("Unsupported OS: %s", osType)
	}

	cyclicalTicker := time.NewTicker(puaConfig.TickerInterval)
	for {
		select {
		case <-ctx.Done():
			return

		case <-cyclicalTicker.C:
			log.Info("Checking for new update...")

			updateStatusType, err := metadata.GetMetaUpdateStatus()
			if err != nil {
				log.Errorf("Error reading status from metadata file: %v", err)
			}
			updateLog, getLogErr := metadata.GetMetaUpdateLog()
			if err != nil {
				log.Errorf("Error reading granular log from metadata file: %v", getLogErr)
			}
			osProfileUpdateSourceActual, err := metadata.GetMetaOSProfileUpdateSourceActual()
			if err != nil {
				log.Errorf("Error reading os profile update source from metadata file: %v", err)
			}
			if osProfileUpdateSourceActual == nil {
				osProfileUpdateSourceActual = &pb.OSProfileUpdateSource{}
			}

			status := &pb.UpdateStatus{
				StatusType:     updateStatusType,
				StatusDetail:   updateLog,
				ProfileName:    osProfileUpdateSourceActual.ProfileName,
				ProfileVersion: osProfileUpdateSourceActual.ProfileVersion,
				OsImageId:      osProfileUpdateSourceActual.OsImageId,
			}

			var resp *pb.PlatformUpdateStatusResponse
			op := func() error {
				platformUpdateStatusResponse, reqErr := maintenanceManager.PlatformUpdateStatus(auth.GetAuthContext(ctx, puaConfig.JWT.AccessTokenPath), status, puaConfig.GUID)
				if reqErr != nil {
					log.Errorf("Failed to send update status: %v", reqErr)
				}
				resp = platformUpdateStatusResponse
				return nil
			}
			err = backoff.Retry(op, backoff.WithMaxRetries(backoff.WithContext(backoff.NewExponentialBackOff(), ctx), NUM_RETRIES))
			if err != nil {
				log.Errorf("will try to reconnect because of failure: %v", err)
				conn_err := maintenanceManager.GrpcConn.Close()
				if conn_err != nil {
					return
				}
				conn_err = maintenanceManager.Connect(ctx)
				if conn_err != nil {
					return
				}
				return
			}

			// Store the last successful response timestamp from orchestrator
			atomic.StoreInt64(&lastUpdateTimestamp, time.Now().Unix())

			switch osType {
			case "ubuntu":
				handleUbuntuResponse(resp, updateResChan)
			case "debian":
				handleUbuntuResponse(resp, updateResChan)
			case "emt":
				handleEmtResponse(resp, updateResChan)
			}

			if updateStatusType == pb.UpdateStatus_STATUS_TYPE_UPDATED {
				err = metadata.SetMetaUpdateStatus(pb.UpdateStatus_STATUS_TYPE_UP_TO_DATE)
				if err != nil {
					log.Errorf("failed to set metadata - %v", err)
				}
			}

		}
	}
}

func handleUbuntuResponse(res *pb.PlatformUpdateStatusResponse, updateResChan chan *pb.PlatformUpdateStatusResponse) {
	switch {
	case res.GetUpdateSource() == nil || res.GetUpdateSchedule() == nil:
		log.Warnf("skipping response as it is missing one of the fields")
		log.Debugf("skipped response - %v", res)
	case res.GetOsType() == pb.PlatformUpdateStatusResponse_OS_TYPE_IMMUTABLE:
		log.Errorf("skipping IMMUTABLE update request on Ubuntu node")
		_ = metadata.SetMetaUpdateLog("skipping IMMUTABLE update request on Ubuntu node")
	case kernelRegexp.MatchString(res.UpdateSource.KernelCommand):
		trySendUpdateStatus(res, updateResChan)
	case !kernelRegexp.MatchString(res.UpdateSource.KernelCommand):
		log.Errorf("skipping request, as '%v' kernel command violates safe kernel settings", res.UpdateSource.KernelCommand)
	default:
		log.Debugf("unexpected case of default in switch - %v", res)
	}
}

func handleEmtResponse(res *pb.PlatformUpdateStatusResponse, updateResChan chan *pb.PlatformUpdateStatusResponse) {
	switch {
	case res.GetOsProfileUpdateSource() == nil ||
		res.GetUpdateSchedule() == nil ||
		res.GetOsType() == *pb.PlatformUpdateStatusResponse_OS_TYPE_UNSPECIFIED.Enum():
		log.Warnf("skipping response as it is missing one of the fields")
		log.Debugf("skipped response - %v", res)
	case res.GetOsType() == pb.PlatformUpdateStatusResponse_OS_TYPE_MUTABLE:
		log.Errorf("skipping MUTABLE update request on Edge Microvisor Toolkit node")
		_ = metadata.SetMetaUpdateLog("skipping MUTABLE update request on Edge Microvisor Toolkit node")
	default:
		trySendUpdateStatus(res, updateResChan)
	}
}

func trySendUpdateStatus(res *pb.PlatformUpdateStatusResponse, updateResChan chan *pb.PlatformUpdateStatusResponse) {
	select {
	// If receiver is reading from channel then this goroutine will write to it.
	case updateResChan <- res:
	// Receiver not reading from channel, skipping to select.
	default:
		log.Debugf("skipping writing to channel, as it is busy or already closed")
	}
}

/*
After the PUA receives a response from MM, it will check the UpdateSchedule in the response.
It could be a SingleSchedule and nil RepeatedSchedules, or nil SingleSchedule and RepeatedSchedules.
The PUA will record the information of the SingleSchedule and RepeatedSchedules in metadata.json and compare this information when there is an incoming schedule, ensuring that the schedule only executes once.
PUA must also communicate the next expected update and the download source to the Downloader.
*/
func handleUpdateRes(updateRes *pb.PlatformUpdateStatusResponse, puaScheduler *scheduler.PuaScheduler, meta metadata.Meta, puaDownloader *downloader.Downloader, osType, rsFQDN string) {
	err := metadata.SetMetaOSProfileUpdateSourceDesired(updateRes.GetOsProfileUpdateSource())
	if err != nil {
		log.Warnf("failed to update metadata - %v", err)
	}

	err = metadata.SetMetaUpdateSource(updateRes.GetUpdateSource())
	if err != nil {
		log.Warnf("failed to update metadata - %v", err)
	}

	if updateRes.UpdateSchedule != nil {
		puaScheduler.HandleSingleSchedule(updateRes.UpdateSchedule.SingleSchedule, &meta, osType)
		puaScheduler.HandleRepeatedSchedule(updateRes.UpdateSchedule.RepeatedSchedules, &meta, osType)

		err = metadata.SetMetaSchedules(updateRes.UpdateSchedule.SingleSchedule, updateRes.UpdateSchedule.RepeatedSchedules, meta.SingleScheduleFinished)
		if err != nil {
			log.Warnf("failed to update metadata - %v", err)
		}
	} else {
		log.Warnf("got nil UpdateSchedule - skipping scheduling")
		puaScheduler.CleanupSchedule()
	}

	err = metadata.SetInstalledPackages(updateRes.GetInstalledPackages())
	if err != nil {
		log.Warnf("failed to update metadata - %v", err)
	}

	// For now, to avoid any chance of breaking Ubuntu, only do this notification part on Edge Microvisor Toolkit
	if osType == "emt" {
		// notify downloader of next update time and current update source
		nextJob := puaScheduler.GetNextJob()
		var nextRunTime time.Time
		if nextJob != nil {
			nextRunTime = nextJob.NextRun()
		} else {
			nextRunTime = time.Time{} // this means 'no time'; we should NOT download. See Notify docstring
		}

		actualOSSource, err := metadata.GetMetaOSProfileUpdateSourceActual()
		if err != nil {
			log.Errorf("Cannot retrieve OS source already on system; skipping download schedule")
			return
		}

		updateSource := updateRes.GetOsProfileUpdateSource()

		puaDownloader.Notify(rsFQDN+"/", updateSource, nextRunTime, actualOSSource)
	}
}

func continueUpdateAfterPuaRestart(updateController *updater.UpdateController) {
	log.Infoln("Detected node update in progress")
	updateController.ContinueUpdate()
}

func continueUpdateAfterOsReboot(updateController *updater.UpdateController, puaConfig *config.Config, cleaner *updater.Cleaner, osType string) {
	log.Infoln("Detected node update in progress")
	var status pb.UpdateStatus_StatusType
	var granularLog string
	var err error

	for i := 0; i < RETRIES; i++ {
		status, granularLog, _, err = updateController.VerifyUpdate(puaConfig.INBCLogsPath, puaConfig.INBCGranularLogsPath)
		if err != nil {
			log.Errorf("Update verification failed: %v", err)
			break
		}
		if status != pb.UpdateStatus_STATUS_TYPE_STARTED {
			break
		}
		time.Sleep(UPDATE_READ_INTERVAL)
	}

	if status == pb.UpdateStatus_STATUS_TYPE_STARTED {
		log.Error("Update still in progress. Please check the logs.")
	}

	// if we are inside the metadata's single schedule window, since we just finished an update we
	// can safely set single schedule finished to true
	inWindow, err := metadata.IsInsideSingleScheduleWindow(time.Now())
	if err != nil {
		log.Error("Cannot check whether we are inside the single schedule window; update might repeat until end of single schedule maintenance window")
	} else {
		if inWindow {
			err := metadata.SetSingleScheduleFinished(true)
			if err != nil {
				log.Error("Cannot set single schedule finished; update might repeat until end of single schedule maintenance window")
			}
		}
	}

	// on Edge Microvisor Toolkit, if the update was successful, we need to record the desired OS as actual
	if osType == "emt" && status == pb.UpdateStatus_STATUS_TYPE_UPDATED {
		desiredOSProfile, err := metadata.GetMetaOSProfileUpdateSourceDesired()
		if err != nil {
			log.Errorf("Failed to get desired OS profile from metadata: %v", err)
		} else {
			err = metadata.SetMetaOSProfileUpdateSourceActual(desiredOSProfile)
			if err != nil {
				log.Debugf("Failed to set actual OS profile in metadata: %v", err)
			}
		}
	}

	err = cleaner.CleanupAfterUpdate(puaConfig.INBCGranularLogsPath)
	if err != nil {
		log.Warnf("Post-update cleanup failed: %v", err)
	}

	err = metadata.SetMetaUpdateStatus(status)
	if err != nil {
		log.Fatalf("Metadata update failed: %v", err)
	}

	err = metadata.SetMetaUpdateLog(granularLog)
	if err != nil {
		log.Fatalf("Metadata update failed: %v", err)
	}

	err = metadata.SetMetaUpdateInProgress(metadata.NONE)
	if err != nil {
		log.Fatalf("Metadata update failed: %v", err)
	}
}

func SendHealthStatus(wg *sync.WaitGroup, ctx context.Context, statusServerEndpoint string, tickerInterval time.Duration) {
	defer wg.Done()
	context, cancel := context.WithCancel(ctx)
	statusClient, statusInterval := initStatusClientAndTicker(context, cancel, statusServerEndpoint)
	compareInterval := max(int64(statusInterval.Seconds()), int64(tickerInterval.Seconds()))
	ticker := time.NewTicker(1 * time.Nanosecond)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ticker.Stop()

			lastCheck := atomic.LoadInt64(&lastUpdateTimestamp)
			now := time.Now().Unix()
			log.Debugf("now=%v, lastCheck=%v \n", now, lastCheck)
			log.Debugf("now-lastCheck=%v \n", now-lastCheck)

			// To ensure that the agent is not marked as not ready due to a functional delay, a
			// check of 2x of interval is considered. This should prevent false negatives.
			if now-lastCheck <= 2*compareInterval {
				if err := statusClient.SendStatusReady(ctx, AGENT_NAME); err != nil {
					log.Errorf("Failed to send status Ready: %v", err)
				}
				log.Infoln("Status Ready")
			} else {
				if err := statusClient.SendStatusNotReady(ctx, AGENT_NAME); err != nil {
					log.Errorf("Failed to send status Not Ready: %v", err)
				}
				log.Infoln("Status Not Ready")
			}

			ticker.Reset(statusInterval)
		}
	}
}

func initStatusClientAndTicker(ctx context.Context, cancel context.CancelFunc, statusServer string) (*status.StatusClient, time.Duration) {
	statusClient, err := status.InitClient(statusServer)
	if err != nil {
		log.Errorf("Failed to initialize status client: %v", err)
		cancel()
	}

	var interval time.Duration
	op := func() error {
		interval, err = statusClient.GetStatusInterval(ctx, AGENT_NAME)
		if err != nil {
			log.Errorf("Failed to get status interval: %v", err)
		}
		return err
	}

	// High number of retries as retries would mostly indicate a problem with the status server
	err = backoff.Retry(op, backoff.WithContext(backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 30), ctx))
	if err != nil {
		log.Warnf("Defaulting to 10 seconds")
		interval = 10 * time.Second
	}

	return statusClient, interval
}
