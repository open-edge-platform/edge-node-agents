// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package installer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/logger"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/metadata"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/utils"
)

var log = logger.Logger()

const (
	_ERR_PATH_DOES_NOT_CONTAIN_VALUE = "The following element path doesn't contain the value to remove"
	_ERR_CANNOT_SET_METAFILE         = "cannot write metafile"
)

var (
	inbmConfigSuccessPath = "/var/edge-node/pua/.inbm-config-success"

	restartInbmConfigurationCommand = []string{
		"sudo", "systemctl", "restart", "inbm-configuration",
	}

	removeDockerCommand = []string{
		"sudo", "inbc", "remove", "--path", "sotaSW:docker",
	}

	provisionTcCommand = []string{
		"sudo", "SKIP_DOCKER_CONFIGURATION=x", "NO_CLOUD=x", "NO_OTA_CERT=x", "PROVISION_TPM=auto", "provision-tc",
	}

	upgradeDependenciesCommand = []string{
		"sudo", "apt", "install", "--only-upgrade", "-y", "inbc-program", "trtl", "inbm-cloudadapter-agent",
		"inbm-dispatcher-agent", "inbm-configuration-agent", "inbm-telemetry-agent", "inbm-diagnostic-agent",
		"mqtt", "tpm-provision",
	}

	inbcSotaDownloadOnlyInstallPackagesCommand = []string{
		"sudo", "inbc", "sota", "--mode", "download-only", "-rb", "no", "--package-list",
	}

	inbcSotaNoDownloadInstallPackagesCommand = []string{
		"sudo", "inbc", "sota", "--mode", "no-download", "-rb", "no", "--package-list",
	}
)

type Installer struct {
	commandExecutor          utils.Executor
	MetaController           *metadata.MetaController
	inbcStabilizingSleepFunc func(ctx context.Context) error // will be called when we need to sleep to stabilize INBC
}

func defaultSleeper(ctx context.Context) error {
	select {
	case <-time.After(time.Second):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func New(commandExecutor utils.Executor) *Installer {
	return &Installer{
		commandExecutor:          commandExecutor,
		inbcStabilizingSleepFunc: defaultSleeper,
	}
}

func NewWithDefaults() *Installer {
	return &Installer{
		commandExecutor:          utils.NewExecutor[exec.Cmd](exec.Command, utils.ExecuteAndReadOutput),
		inbcStabilizingSleepFunc: defaultSleeper,
	}
}

func (i *Installer) ProvisionInbm(_ context.Context) error {
	if fileOrDirExists(inbmConfigSuccessPath) {
		log.Debugf("INBC is already provisioned")
		return nil
	}

	log.Info("running `provision-tc` script - it may take a while")
	if _, err := i.execute(provisionTcCommand); err != nil {
		return fmt.Errorf("failed to execute shell command - %v", err)
	}

	file, err := os.Create(inbmConfigSuccessPath)
	if err != nil {
		log.Errorf("Creating file failed: %s", inbmConfigSuccessPath)
		return err
	}
	defer file.Close()

	log.Info("INBM provisioning finished")

	return nil
}

func (i *Installer) UpgradeInbmPackages(ctx context.Context) error {
	out, err := i.execute(upgradeDependenciesCommand)
	if err != nil {
		return fmt.Errorf("failed to execute shell command - %v", err)
	}
	log.Info("Ran `apt install` command")

	if isUpdated(string(out)) {
		log.Info("running `provision-tc` script - it may take a while")

		if _, err = i.execute(provisionTcCommand); err != nil {
			return fmt.Errorf("failed to execute shell command - %v", err)
		}

		if err := i.modifyConfiguration(ctx); err != nil {
			return fmt.Errorf("failed to modify INBC configuration - %v", err)
		}
	}
	return nil
}

// modifyConfiguration adjusts the INBM configuration by removing Docker-related settings
// and restarting the INBM configuration service. It attempts to remove Docker configuration
// repeatedly until successful or a timeout occurs.
//
// The function performs the following steps:
// 1. Attempts to remove Docker configuration using the removeDocker method.
// 2. If successful, restarts the INBM configuration service.
//
// Parameters:
//   - ctx: A context.Context for handling timeouts and cancellation.
//
// Returns:
//   - error: An error if any step fails, or nil if the configuration is successfully modified.
//
// The function will retry the Docker removal step for up to 5 minutes, with 30-second intervals
// between attempts. If the Docker removal is not successful within this time, an error is returned.
func (i *Installer) modifyConfiguration(ctx context.Context) error {
	if err := wait.PollUntilContextTimeout(ctx, time.Second*30, time.Minute*5, true, i.removeDocker); err != nil {
		return fmt.Errorf("failed to modify INBM configuration - %v", err)
	}

	if _, err := i.execute(restartInbmConfigurationCommand); err != nil {
		return fmt.Errorf("failed to execute shell command - %v", err)
	}

	return nil
}

func isUpdated(output string) bool {
	return !strings.Contains(output, "0 upgraded")
}

func (i *Installer) execute(args []string) ([]byte, error) {
	return i.commandExecutor.Execute(args)
}

func (i *Installer) removeDocker(ctx context.Context) (done bool, err error) {
	// after provisioning INBC needs some time to start up.
	// Overall it is extremely small period of time (<1 second), but it could depend on many factors, so that's why we are retrying command.

	if err := i.inbcStabilizingSleepFunc(ctx); err != nil {
		return false, err
	}

	_, err = i.execute(removeDockerCommand)
	if err != nil {
		if strings.Contains(err.Error(), _ERR_PATH_DOES_NOT_CONTAIN_VALUE) {
			return true, nil
		}
		log.Warnf("error during execution of shell command - %v. Retrying.", err)
		return false, nil
	}
	return true, nil
}

func fileOrDirExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func (i *Installer) InstallAdditionalPackages(packages string) error {

	if packages == "" {
		log.Info("list of additional packages for installation is empty.")
		return nil
	}

	packages = strings.ReplaceAll(packages, "\n", ",")

	log.Infof("Installing additional packages: %v", packages)

	_, err := i.execute(append(inbcSotaDownloadOnlyInstallPackagesCommand, packages))

	if err != nil {
		return fmt.Errorf("failed to execute shell command(%v)- %v", inbcSotaDownloadOnlyInstallPackagesCommand, err)
	}

	if err := i.MetaController.SetMetaUpdateInProgress(metadata.NEW); err != nil {
		return fmt.Errorf("%s", fmt.Sprintf("%s: %v", _ERR_CANNOT_SET_METAFILE, err))
	}

	if _, err := i.execute(append(inbcSotaNoDownloadInstallPackagesCommand, packages)); err != nil {
		return fmt.Errorf("failed to execute shell command(%v)- %v", inbcSotaNoDownloadInstallPackagesCommand, err)
	}
	return nil
}
