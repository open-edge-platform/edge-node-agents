// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/aptmirror"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/installer"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/logger"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/metadata"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/utils"
	pb "github.com/open-edge-platform/infra-managers/maintenance/pkg/api/maintmgr/v1"
)

const (
	// more update types to be added later
	kernelPath      = "/etc/default/grub.d/90-platform-update-agent.cfg"
	kernelParamName = "GRUB_CMDLINE_LINUX_DEFAULT"
)

type UpdateStatus struct {
	Status   string `json:"Status"`
	Type     string `json:"Type"`
	Time     string `json:"Time"`
	Metadata string `json:"Metadata"`
	Error    string `json:"Error,omitempty"`
	Version  string `json:"Version"`
}

var log = logger.Logger()

var (
	installPlatformUpdateAgentCommand = []string{
		"sudo", "NEEDRESTART_MODE=a", "apt", "install", "--only-upgrade", "-y", "platform-update-agent",
	}

	// this is where reboot happens in Edge Microvisor Toolkit
	inbcEmtUpdateCommand = []string{
		"sudo", "inbc", "sota", "--mode", "no-download",
	}

	// this is where reboot happens in Ubuntu
	inbcSotaNoDownloadCommand = []string{
		"sudo", "inbc", "sota", "--mode", "no-download",
	}

	inbcSotaDownloadOnlyCommand = []string{
		"sudo", "inbc", "sota", "--mode", "download-only", "--reboot", "no",
	}

	upgradeGrubCommand = []string{
		"sudo", "update-grub",
	}

	rebootCommand = []string{
		"sudo", "reboot",
	}

	aptCleanCommand = []string{
		"sudo", "apt", "clean",
	}

	// we use truncate rather than remove here as some OSes like Edge Microvisor Toolkit require files that need to persist
	// between reboots to not be removed
	granularLogTruncateCommand = []string{
		"sudo", "truncate", "-s", "0", "/var/log/inbm-update-log.log",
	}
)

const (
	_ERR_CANNOT_SET_METAFILE     = "cannot write metafile"
	_ERR_PUA_INSTALLATION_FAILED = "PUA installation failed"

	_ERR_GRUB_UPDATE_FAILED = "GRUB update failed"
)

// NewUpdateController sets up an UpdateController with the necessary dependencies
// osType is "ubuntu" or "emt" or "debian" (enic)
// downloadChecker is a function that should return 'true' if it's OK to update; it can be
// used in cases where the update requires a download to complete beforehand
// it may also be completely ignored in OSes such as Ubuntu where updates do not require downloads
func NewUpdateController(granularPath string, osType string, downloadChecker func() bool) (*UpdateController, error) {
	if osType != "ubuntu" && osType != "emt" && osType != "debian" {
		return nil, fmt.Errorf("unsupported os type: %s", osType)
	}

	metadataController := metadata.NewController()
	aptMirrorController := aptmirror.NewController()
	executor := utils.NewExecutor(exec.Command, utils.ExecuteAndReadOutput)

	var enUpdater SubsystemUpdater

	if osType == "ubuntu" || osType == "debian" {
		packagesUpdater := &packagesUpdater{
			MetaController: metadataController,
			AptController:  aptMirrorController,
		}

		selfUpdater := &selfUpdater{
			MetaController: metadataController,
			Executor:       executor,
		}

		inbmUpdater := &inbmUpdater{
			MetaController: metadataController,
			Executor:       executor,
		}

		kernelUpdater := &kernelUpdater{
			Executor:       executor,
			MetaController: metadataController,
			kernelFile:     kernelPath,
		}

		newPackageInstaller := &newPackageInstaller{
			Executor:       executor,
			MetaController: metadataController,
		}

		osAndPackagesUpdater := &osAndAgentsUpdater{
			Executor:       executor,
			MetaController: metadataController,
			AptController:  aptMirrorController,
		}

		enUpdater = &edgeNodeUpdater{
			MetaController:    metadata.NewController(),
			subsystemUpdaters: []SubsystemUpdater{packagesUpdater, selfUpdater, inbmUpdater, kernelUpdater, newPackageInstaller, osAndPackagesUpdater},
			timeNow:           time.Now,
		}
	} else {
		emtUpdater := &emtUpdater{
			Executor:        executor,
			MetaController:  metadataController,
			DownloadChecker: downloadChecker,
		}

		enUpdater = &edgeNodeUpdater{
			MetaController:    metadata.NewController(),
			subsystemUpdaters: []SubsystemUpdater{emtUpdater},
			timeNow:           time.Now,
		}
	}

	return &UpdateController{
		metaController:  metadata.NewController(),
		fileSystem:      &RealFileSystem{},
		granularLogPath: granularPath,
		edgeNodeUpdater: enUpdater,
		timeNow:         time.Now,
		cleaner:         NewCleanerWithDefaults(osType),
	}, nil
}

type UpdateController struct {
	metaController  *metadata.MetaController
	fileSystem      FileSystem
	granularLogPath string
	edgeNodeUpdater SubsystemUpdater
	timeNow         func() time.Time
	cleaner         CleanerInterface
}

type FileSystem interface {
	Read(path string) ([]byte, error)
}

type RealFileSystem struct{}

func (fs *RealFileSystem) Read(path string) ([]byte, error) {
	return os.ReadFile(path)
}

var edgeNodeUpdateMutex sync.Mutex

func (u *UpdateController) StartUpdate(durationSeconds int64) {
	if !edgeNodeUpdateMutex.TryLock() {
		log.Errorf("StartUpdate failed: Edge Node Update is already in progress.")
		return
	}
	defer edgeNodeUpdateMutex.Unlock()

	log.Infof("Starting Edge Node Update.")

	err := u.metaController.SetMetaUpdateStatus(pb.UpdateStatus_STATUS_TYPE_STARTED)
	if err != nil {
		log.Errorf("failed to set metadata - %v", err)
		return
	}

	startTime := u.timeNow()

	err = u.metaController.SetMetaUpdateTime(startTime)
	if err != nil {
		log.Errorf("failed to set metadata - %v", err)
		return
	}

	err = u.metaController.SetMetaUpdateDuration(durationSeconds)
	if err != nil {
		log.Errorf("failed to set metadata - %v", err)
		return
	}

	err = u.edgeNodeUpdater.update()
	if err != nil {
		log.Errorf("Update error: %v", err)
		innerErr := u.metaController.SetMetaUpdateStatus(pb.UpdateStatus_STATUS_TYPE_FAILED)
		if innerErr != nil {
			log.Errorf("failed to set metadata - %v", innerErr)
		}
		// Read granular log file
		updateLog := ""
		if !strings.Contains(err.Error(), aptmirror.ERR_INVALID_SIGNATURE) {
			logContent, logErr := u.fileSystem.Read(u.granularLogPath)
			if logErr != nil {
				fmt.Printf("reading INBC logs failed: %v", logErr)
				updateLog = ""
			} else {
				updateLog = string(logContent)
			}
		}

		setErr := u.metaController.SetMetaUpdateLog(updateLog)

		if setErr != nil {
			log.Errorf("failed to set metadata - %v", setErr)
		}
		// Remove the log file
		err = u.cleaner.CleanupAfterUpdate(u.granularLogPath)
		if err != nil {
			log.Warnf("Cleanup failed: %v", err)
		}

		return
	}
}

func (u *UpdateController) ContinueUpdate() {
	log.Infof("Continuing Edge Node Update.")

	err := u.edgeNodeUpdater.update()
	if err != nil {
		log.Errorf("Update error: %v", err)
		innerErr := u.metaController.SetMetaUpdateStatus(pb.UpdateStatus_STATUS_TYPE_FAILED)
		if innerErr != nil {
			log.Errorf("failed to set metadata - %v", innerErr)
		}
		return
	}
}

// VerifyUpdate determines status of executed update and records the granular log
func (u *UpdateController) VerifyUpdate(logPath string, granularLogPath string) (status pb.UpdateStatus_StatusType, granularLog string, time string, err error) {
	content, err := os.ReadFile(logPath)
	if err != nil {
		// Check if this is a kernel-only update that doesn't generate INBC logs
		updateInProgress, metaErr := metadata.GetMetaUpdateInProgress()
		if metaErr == nil && updateInProgress == string(metadata.OS) {
			// For kernel-only updates, check if kernel command was actually updated
			updateSource, sourceErr := metadata.GetMetaUpdateSource()
			if sourceErr == nil && updateSource != nil && updateSource.KernelCommand != "" {
				log.Info("Kernel-only update detected - verifying kernel command line update")
				// If we have a kernel command and the system rebooted (which is why we're here),
				// the kernel update was successful
				return pb.UpdateStatus_STATUS_TYPE_UPDATED, "Kernel command line parameters updated successfully", "", nil
			}
		}
		return pb.UpdateStatus_STATUS_TYPE_FAILED, "", "", fmt.Errorf("reading INBC logs failed: %v", err)
	}

	if len(content) == 0 {
		// Check if this is a kernel-only update that doesn't generate INBC logs
		updateInProgress, metaErr := metadata.GetMetaUpdateInProgress()
		if metaErr == nil && updateInProgress == string(metadata.OS) {
			// For kernel-only updates, check if kernel command was actually updated
			updateSource, sourceErr := metadata.GetMetaUpdateSource()
			if sourceErr == nil && updateSource != nil && updateSource.KernelCommand != "" {
				return pb.UpdateStatus_STATUS_TYPE_UPDATED, "Kernel command line parameters updated successfully", "", nil
			}
		}
		return pb.UpdateStatus_STATUS_TYPE_FAILED, "", "", fmt.Errorf("INBC log file is empty")
	}

	updateStatus := UpdateStatus{}
	err = json.Unmarshal(content, &updateStatus)
	if err != nil {
		return pb.UpdateStatus_STATUS_TYPE_FAILED, "", "", fmt.Errorf("unmarshalling INBC update status failed: %v", err)
	}

	// Read granular log file
	updateLog := ""
	logContent, logErr := os.ReadFile(granularLogPath)
	if logErr != nil {
		fmt.Printf("reading INBC logs failed: %v", logErr)
	} else {
		updateLog = string(logContent)
	}

	switch updateStatus.Status {
	case "SUCCESS":
		log.Infof("OS update status %s, update time: %s", updateStatus.Status, updateStatus.Time)
		return pb.UpdateStatus_STATUS_TYPE_UPDATED, updateLog, updateStatus.Time, nil
	case "NO_UPDATE_AVAILABLE":
		log.Infof("OS update status %s, update time: %s", updateStatus.Status, updateStatus.Time)
		if granularLogShowsUpdate(updateLog) {
			// this can happen if a package was installed but there were no other packages to update
			// in this case, we need to return UPDATED indicating an update took place
			return pb.UpdateStatus_STATUS_TYPE_UPDATED, updateLog, updateStatus.Time, nil
		}
		// here there was no update, so not UPDATED, just UP_TO_DATE
		return pb.UpdateStatus_STATUS_TYPE_UP_TO_DATE, updateLog, updateStatus.Time, nil
	case "FAIL":
		log.Infof("OS update status %s, update time: %s", updateStatus.Status, updateStatus.Time)
		return pb.UpdateStatus_STATUS_TYPE_FAILED, updateLog, updateStatus.Time, fmt.Errorf("%s", updateStatus.Error)
	case "PENDING":
		log.Infof("OS update status %s, update time: %s", updateStatus.Status, updateStatus.Time)
		return pb.UpdateStatus_STATUS_TYPE_STARTED, updateLog, updateStatus.Time, nil
	}

	return pb.UpdateStatus_STATUS_TYPE_FAILED, updateLog, updateStatus.Time, fmt.Errorf("status of the last OS update is unknown. Please verify logs")
}

// UpdateLogEntry represents an entry in the update log
type UpdateLogEntry struct {
	UpdateType  string `json:"update_type"`
	PackageName string `json:"package_name"`
	UpdateTime  string `json:"update_time"`
	Action      string `json:"action"`
	Status      string `json:"status"`
	Version     string `json:"version"`
}

// UpdateLog represents the structure of the update log JSON
type UpdateLog struct {
	UpdateLog []UpdateLogEntry `json:"UpdateLog"`
}

// granularLogShowsUpdate checks if the granular log shows that an update with status "SUCCESS" was performed
func granularLogShowsUpdate(log string) bool {
	var updateLog UpdateLog
	err := json.Unmarshal([]byte(log), &updateLog)
	if err != nil {
		return false
	}

	for _, entry := range updateLog.UpdateLog {
		if entry.Status == "SUCCESS" {
			return true
		}
	}

	return false
}

type SubsystemUpdater interface {
	update() error
}

// this will update 'OS and agents' and then reboot
type osAndAgentsUpdater struct {
	utils.Executor
	*metadata.MetaController
	*aptmirror.AptController
}

func (o *osAndAgentsUpdater) update() error {
	log.Info("Executing os and agents update")

	_, err := o.Execute(inbcSotaDownloadOnlyCommand)
	if err != nil {
		return fmt.Errorf("failed to execute shell command(%v)- %v", inbcSotaDownloadOnlyCommand, err)
	}

	if err := o.SetMetaUpdateInProgress(metadata.OS); err != nil {
		return fmt.Errorf("%s", fmt.Sprintf("%s: %v", _ERR_CANNOT_SET_METAFILE, err))
	}

	if _, err := o.Execute(inbcSotaNoDownloadCommand); err != nil {
		return fmt.Errorf("failed to execute shell command(%v)- %v", inbcSotaNoDownloadCommand, err)
	}
	return nil
}

// this perform the switch partition & reboot on Edge Microvisor Toolkit
type emtUpdater struct {
	utils.Executor
	*metadata.MetaController
	DownloadChecker func() bool // checks whether download is done; this must return true for update to start
}

func (o *emtUpdater) update() error {
	log.Info("Executing Edge Microvisor Toolkit A/B update")

	if !o.DownloadChecker() {
		return fmt.Errorf("cannot execute Edge Microvisor Toolkit update as download has not taken place")
	}

	if err := o.SetMetaUpdateInProgress(metadata.OS); err != nil {
		return fmt.Errorf("%s", fmt.Sprintf("%s: %v", _ERR_CANNOT_SET_METAFILE, err))
	}

	_, err := o.Execute(inbcEmtUpdateCommand)
	if err != nil {
		return fmt.Errorf("failed to execute shell command(%v)- %v", inbcEmtUpdateCommand, err)
	}

	return nil
}

type kernelUpdater struct {
	utils.Executor
	*metadata.MetaController
	kernelFile string
}

func (k *kernelUpdater) update() error {
	log.Info("Executing kernel update")
	metadataUpdateSource, err := k.GetMetaUpdateSource()
	if err != nil {
		return err
	}

	if metadataUpdateSource == nil || metadataUpdateSource.KernelCommand == "" {
		log.Infof("update source or provided kernel is empty - skipping kernel update")
		return nil
	}

	err = utils.IsSymlink(k.kernelFile)
	if err != nil {
		return err
	}

	err = os.WriteFile(k.kernelFile, []byte(kernelParamName+`="`+metadataUpdateSource.KernelCommand+`"`), 0o600)
	if err != nil {
		return fmt.Errorf("failed to write modified kernel params to %v file - %v", k.kernelFile, err)
	}

	if _, err = k.Execute(upgradeGrubCommand); err != nil {
		return fmt.Errorf("%s: %v", _ERR_GRUB_UPDATE_FAILED, err)
	}

	// Set metadata to indicate OS update is in progress so the system continues after reboot
	if err := k.SetMetaUpdateInProgress(metadata.OS); err != nil {
		return fmt.Errorf("%s: %v", _ERR_CANNOT_SET_METAFILE, err)
	}

	// Reboot is required for kernel command line changes to take effect
	log.Info("Kernel update completed. Rebooting system for changes to take effect.")
	if _, err = k.Execute(rebootCommand); err != nil {
		return fmt.Errorf("failed to execute reboot command: %v", err)
	}

	return nil
}

type packagesUpdater struct {
	*metadata.MetaController
	*aptmirror.AptController
}

func (p *packagesUpdater) update() error {
	log.Info("Executing packages update")
	updateSource, err := p.GetMetaUpdateSource()
	if err != nil {
		return fmt.Errorf("error reading metadata file - %v", err)
	}

	if updateSource == nil || len(updateSource.CustomRepos) == 0 {
		log.Info("No custom apt repositories configured - skipping package updates")
		return nil
	}

	isDeprecated := p.IsDeprecatedFormat(updateSource.CustomRepos)

	if isDeprecated {
		err = p.ConfigureDeprecatedCustomAptRepos(updateSource.CustomRepos)
		if err != nil {
			return fmt.Errorf("deprecated custom apt repo configuration failed. Error - %v", err)
		}
		return p.UpdatePackages()
	}

	err = p.CleanupCustomRepos()
	if err != nil {
		return fmt.Errorf("failed to cleanup custom repos - %v", err)
	}

	err = p.ConfigureForwardProxy(updateSource.CustomRepos)
	if err != nil {
		return fmt.Errorf("failed to configure forward proxy - %v", err)
	}

	err = p.ConfigureCustomAptRepos(updateSource.CustomRepos)
	if err != nil {
		return fmt.Errorf("custom apt repo configuration failed. Error - %v", err)
	}

	// We used to configure OS apt repo here, but OsRepoUrl has been deprecated
	// and is no longer used.

	// TODO: in future, decide whether to completely remove ConfigureOsAptRepo function
	// err = p.ConfigureOsAptRepo(updateSource.OsRepoUrl)
	// if err != nil {
	// 	return fmt.Errorf("failed to execute shell command - %v", err)
	// }

	return p.UpdatePackages()
}

type selfUpdater struct {
	*metadata.MetaController
	utils.Executor
}

func (s *selfUpdater) update() error {
	log.Info("Executing PUA package update")

	err := s.SetMetaUpdateInProgress(metadata.SELF)
	if err != nil {
		return fmt.Errorf("%s: %v", _ERR_CANNOT_SET_METAFILE, err)
	}

	if _, err = s.Execute(installPlatformUpdateAgentCommand); err != nil {
		return fmt.Errorf("%s: %v", _ERR_PUA_INSTALLATION_FAILED, err)
	}

	return nil
}

type inbmUpdater struct {
	*metadata.MetaController
	utils.Executor
}

func (i *inbmUpdater) update() error {
	log.Info("Executing INBM update")

	// Check if there's an update source
	updateSource, err := i.GetMetaUpdateSource()
	if err != nil {
		return err
	}

	// If this is a kernel-only update (has KernelCommand but no CustomRepos), skip INBM
	if updateSource != nil && updateSource.KernelCommand != "" && len(updateSource.CustomRepos) == 0 {
		log.Info("Kernel-only update detected - skipping INBM package upgrade")
		return nil
	}

	if err := i.SetMetaUpdateInProgress(metadata.INBM); err != nil {
		return fmt.Errorf("%s: %v", _ERR_CANNOT_SET_METAFILE, err)
	}

	installer := installer.New(i.Executor)
	return installer.UpgradeInbmPackages(context.TODO())
}

type newPackageInstaller struct {
	utils.Executor
	*metadata.MetaController
}

func (i *newPackageInstaller) update() error {
	log.Info("Executing installation of additional packages")

	packages, err := i.GetInstallPackageList()
	if err != nil {
		return fmt.Errorf("error reading metadata file: %v", err)
	}

	installer := installer.New(i.Executor)
	installer.MetaController = i.MetaController
	return installer.InstallAdditionalPackages(packages)
}

type edgeNodeUpdater struct {
	*metadata.MetaController
	subsystemUpdaters []SubsystemUpdater
	timeNow           func() time.Time
}

func (e *edgeNodeUpdater) update() error {
	for _, updater := range e.subsystemUpdaters {
		if err := e.checkTimeout(); err != nil {
			return err
		}

		if err := updater.update(); err != nil {
			return err
		}

	}
	log.Info("Pre-boot OS Update process completed")
	return nil
}

func (e *edgeNodeUpdater) checkTimeout() error {
	updateStartTime, err := e.GetMetaUpdateTime()
	if err != nil {
		return fmt.Errorf("error reading metadata file: %v", err)
	}

	updateDuration, err := e.GetMetaUpdateDuration()
	if err != nil {
		return fmt.Errorf("error reading metadata file: %v", err)
	}

	if updateDuration != 0 && (e.timeNow().Unix()-updateStartTime.Unix()) >= updateDuration {
		return fmt.Errorf("partial success - timed out before was able to perform full update")
	}
	return nil
}

type CleanerInterface interface {
	CleanupAfterUpdate(granularLogPath string) error
}

type Cleaner struct {
	commandExecutor utils.Executor
	osType          string // "ubuntu" or "emt"
}

func NewCleaner(commandExecutor utils.Executor, osType string) *Cleaner {
	return &Cleaner{
		commandExecutor: commandExecutor,
		osType:          osType,
	}
}

func NewCleanerWithDefaults(osType string) *Cleaner {
	return &Cleaner{
		commandExecutor: utils.NewExecutor(exec.Command, utils.ExecuteAndReadOutput),
		osType:          osType,
	}
}

func (c *Cleaner) CleanupAfterUpdate(granularLogPath string) error {
	log.Infof("Cleanup apt artifacts after update.")

	if c.osType != "ubuntu" && c.osType != "emt" && c.osType != "debian" {
		return fmt.Errorf("unsupported os type: %s", c.osType)
	}

	if c.osType == "ubuntu" || c.osType == "debian" {
		if _, err := c.commandExecutor.Execute(aptCleanCommand); err != nil {
			return fmt.Errorf("failed to execute shell command(%v)- %v", aptCleanCommand, err)
		}
	}

	// Common to all OSes

	log.Infof("Cleanup granular log file after update.")
	_, err := os.Stat(granularLogPath)
	if err == nil {
		if _, err := c.commandExecutor.Execute(granularLogTruncateCommand); err != nil {
			return fmt.Errorf("failed to execute shell command(%v)- %v", granularLogTruncateCommand, err)
		}
	} else {
		log.Debugf("Granular log file not exist.")
	}

	return nil
}
