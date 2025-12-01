// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package installer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"testing"
	"time"

	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/metadata"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/utils"

	"github.com/stretchr/testify/require"
)

func TestNewInstaller(t *testing.T) {
	commandExecutor := utils.NewExecutor[exec.Cmd](nil, nil)

	sut := New(commandExecutor)

	require.Equal(t, reflect.ValueOf(commandExecutor).Pointer(), reflect.ValueOf(sut.commandExecutor).Pointer())
}

func Test_execute(t *testing.T) {

	var interceptedCommand string
	var interceptedCommandArgs []string
	var expectedOutputOfExecutedCommand = []byte("ok")
	var expectedErrorThrownByExecutedCommand = fmt.Errorf("error")

	executor := utils.NewExecutor[[]string](
		asStringArray,
		func(i *[]string) (out []byte, e error) {
			interceptedCommand = (*i)[0]
			interceptedCommandArgs = (*i)[1:]
			return expectedOutputOfExecutedCommand, expectedErrorThrownByExecutedCommand
		})

	installer := New(executor)

	out, err := installer.execute([]string{"foo", "bar"})

	require.Equal(t, "foo", interceptedCommand)
	require.Equal(t, []string{"bar"}, interceptedCommandArgs)
	require.Equal(t, expectedOutputOfExecutedCommand, out)
	require.Equal(t, expectedErrorThrownByExecutedCommand, err)
}

func asStringArray(name string, args ...string) *[]string {
	r := append([]string{name}, args...)
	return &r
}

func TestInbm_removeDocker_handleRandomErrorReturnedByRemoveDockerCommand(t *testing.T) {

	var interceptedCommand []string
	var expectedError = fmt.Errorf("error")

	executor := utils.NewExecutor[[]string](
		asStringArray,
		func(args *[]string) (out []byte, e error) {
			interceptedCommand = *args
			return nil, expectedError
		})

	installer := New(executor)
	// Override the sleep function to avoid the delay
	installer.inbcStabilizingSleepFunc = func(ctx context.Context) error {
		return nil
	}

	sut := installer.removeDocker
	done, err := sut(context.TODO())

	require.Equal(t, removeDockerCommand, interceptedCommand)
	require.False(t, done, "returned `done` has unexpected value")
	require.NoError(t, err, "error shall not be returned ")

}

func TestInbm_removeDocker_handleNothingToRemoveErrorReturnedByRemoveDockerCommand(t *testing.T) {

	var interceptedCommand []string

	executor := utils.NewExecutor[[]string](
		asStringArray,
		func(args *[]string) (out []byte, e error) {
			interceptedCommand = *args
			return nil, fmt.Errorf(_ERR_PATH_DOES_NOT_CONTAIN_VALUE)
		})

	installer := New(executor)
	// Override the sleep function to avoid the delay
	installer.inbcStabilizingSleepFunc = func(ctx context.Context) error {
		return nil
	}

	sut := installer.removeDocker
	done, err := sut(context.TODO())

	require.Equal(t, removeDockerCommand, interceptedCommand)
	require.True(t, done, "returned `done` has unexpected value")
	require.NoError(t, err, "error shall not be returned ")

}

func TestInbm_removeDocker_handleNoErrorReturnedByRemoveDockerCommand(t *testing.T) {

	var interceptedCommand []string
	executor := utils.NewExecutor[[]string](
		asStringArray,
		func(args *[]string) (out []byte, e error) {
			interceptedCommand = *args
			return nil, nil
		})

	installer := New(executor)
	// Override the sleep function to avoid the delay
	installer.inbcStabilizingSleepFunc = func(ctx context.Context) error {
		return nil
	}

	sut := installer.removeDocker
	done, err := sut(context.TODO())

	require.Equal(t, removeDockerCommand, interceptedCommand)
	require.True(t, done, "returned `done` has unexpected value")
	require.NoError(t, err, "error shall not be returned ")

}

func TestInbm_ModifyConfiguration_happy_path(t *testing.T) {

	executor := utils.NewExecutor[[]string](
		asStringArray,
		func(args *[]string) (out []byte, e error) {
			return []byte{}, nil
		})

	installer := New(executor)
	// Override the sleep function to avoid the delay
	installer.inbcStabilizingSleepFunc = func(ctx context.Context) error {
		return nil
	}

	sut := installer.modifyConfiguration

	require.NoError(t, sut(context.TODO()))
}

func TestInbm_ModifyConfiguration_handleErrorReturnedByRemoveDocker(t *testing.T) {
	expectedError := fmt.Errorf("error")

	// Mock executor to simulate a failure in the Docker removal process
	executor := utils.NewExecutor[[]string](
		asStringArray,
		func(args *[]string) (out []byte, e error) {
			return []byte{}, expectedError
		},
	)

	ctx, cancel := context.WithTimeout(context.TODO(), time.Millisecond)
	defer cancel()

	// Verify that the function properly handles and propagates the error
	installer := New(executor)
	err := installer.modifyConfiguration(ctx)
	require.ErrorContains(t, err, "failed to modify INBM configuration")
}

func TestInbm_ModifyConfiguration_handleErrorReturnedByRestartInbmConfigurationCommand(t *testing.T) {

	expectedError := fmt.Errorf("error")

	executor := utils.NewExecutor[[]string](
		asStringArray,
		func(args *[]string) (out []byte, e error) {
			switch {
			case reflect.DeepEqual(*args, removeDockerCommand):
				return nil, nil
			case reflect.DeepEqual(*args, restartInbmConfigurationCommand):
				return nil, expectedError
			}
			return nil, nil
		})

	installer := New(executor)
	// Override the sleep function to avoid the delay
	installer.inbcStabilizingSleepFunc = func(ctx context.Context) error {
		return nil
	}

	sut := installer.modifyConfiguration

	require.ErrorContains(t, sut(context.TODO()), "failed to execute shell command ")
}

func TestInbm_ProvisionInbm_IfAlreadyProvisionedShouldDoNothing(t *testing.T) {
	inbmConfigSuccessPath = "testdata/.inbm-config-success"

	executor := utils.NewExecutor[[]string](
		asStringArray,
		func(args *[]string) (out []byte, e error) {
			require.Fail(t, "executor function shall not be called")
			return nil, e
		})

	installer := New(executor)
	// Override the sleep function to avoid the delay
	installer.inbcStabilizingSleepFunc = func(ctx context.Context) error {
		return nil
	}

	sut := installer.ProvisionInbm

	require.Nil(t, sut(context.TODO()))
}

func TestInbm_ProvisionInbm_HappyPath(t *testing.T) {
	inbmConfigSuccessPath = "testdata/.notconfigured"
	defer os.Remove(inbmConfigSuccessPath)
	executor := utils.NewExecutor[[]string](
		asStringArray,
		func(args *[]string) (out []byte, e error) {
			// No commands should be executed for debian package provisioning
			return nil, nil
		})

	installer := New(executor)

	sut := installer.ProvisionInbm

	require.NoError(t, sut(context.TODO()))
	// Verify that the config success file was created
	_, err := os.Stat(inbmConfigSuccessPath)
	require.NoError(t, err)
}

func TestInbm_UpdatePackages_ZeroUpgradedHappyPath(t *testing.T) {
	var interceptedCommand []string

	executor := utils.NewExecutor(asStringArray, func(command *[]string) (out []byte, e error) {
		interceptedCommand = *command
		return []byte("0 upgraded"), nil
	})
	installer := New(executor)

	sut := installer.UpgradeInbmPackages
	require.NoError(t, sut(context.TODO()))
	require.Equal(t, upgradeDependenciesCommand, interceptedCommand)
}

func TestInbm_UpdatePackages_NonZeroUpgradedHappyPath(t *testing.T) {
	upgardeDependenciesCommandExecuted := false
	removeDockerCommandExecuted := false
	restartInbmConfigurationCommandExecuted := false

	executor := utils.NewExecutor(asStringArray, func(command *[]string) (out []byte, e error) {
		switch {
		case reflect.DeepEqual(upgradeDependenciesCommand, *command):
			upgardeDependenciesCommandExecuted = true
			return []byte{}, nil
		case reflect.DeepEqual(removeDockerCommand, *command):
			removeDockerCommandExecuted = true
		case reflect.DeepEqual(restartInbmConfigurationCommand, *command):
			restartInbmConfigurationCommandExecuted = true
		default:
			require.Failf(t, "unexpected command executed:", "%v", command)
		}
		return []byte{}, nil
	})
	installer := New(executor)

	sut := installer.UpgradeInbmPackages
	require.NoError(t, sut(context.TODO()))
	require.True(t, upgardeDependenciesCommandExecuted)
	require.True(t, removeDockerCommandExecuted)
	require.True(t, restartInbmConfigurationCommandExecuted)
}

func Test_newPackageInstaller_happyPath(t *testing.T) {
	var interceptedSetMetaUpdateInProgressCall metadata.UpdateType

	inbcSotaDownloadOnlyInstallPackagesCmdExecuted := false
	inbcSotaNoDownloadInstallPackagesCmdExecuted := false
	packages := "tree traceroute"
	inbcSotaDownloadOnlyInstallPackagesCmd := append(inbcSotaDownloadOnlyInstallPackagesCommand, packages)
	inbcSotaNoDownloadInstallPackagesCmd := append(inbcSotaNoDownloadInstallPackagesCommand, packages)

	executor := utils.NewExecutor(asStringArray, func(command *[]string) (out []byte, e error) {
		switch {
		case reflect.DeepEqual(inbcSotaDownloadOnlyInstallPackagesCmd, *command):
			inbcSotaDownloadOnlyInstallPackagesCmdExecuted = true
			return []byte{}, nil
		case reflect.DeepEqual(inbcSotaNoDownloadInstallPackagesCmd, *command):
			inbcSotaNoDownloadInstallPackagesCmdExecuted = true
			return []byte{}, nil
		}
		return []byte{}, nil
	})

	metadataController := &metadata.MetaController{
		SetMetaUpdateInProgress: func(updateType metadata.UpdateType) error {
			interceptedSetMetaUpdateInProgressCall = updateType
			return nil
		},
		GetInstallPackageList: func() (string, error) {
			return "tree", nil
		},
	}
	installer := New(executor)
	installer.MetaController = metadataController

	err := installer.InstallAdditionalPackages(packages)
	require.NoError(t, err)
	require.True(t, inbcSotaDownloadOnlyInstallPackagesCmdExecuted)
	require.True(t, inbcSotaNoDownloadInstallPackagesCmdExecuted)
	require.Equal(t, metadata.OS, interceptedSetMetaUpdateInProgressCall)
}
