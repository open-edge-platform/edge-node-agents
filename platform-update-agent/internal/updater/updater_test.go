// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package updater

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"reflect"
	"testing"
	"time"

	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/aptmirror"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/metadata"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/utils"
	pb "github.com/open-edge-platform/infra-managers/maintenance/pkg/api/maintmgr/v1"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockCleaner struct{}

func (m *MockCleaner) CleanupAfterUpdate(granularLogPath string) error {
	// Do nothing, simulate successful cleanup
	return nil
}

func Test_VerifyDefaultConstructorOfUpdateControllerUbuntu(t *testing.T) {
	up, err := NewUpdateController("", "ubuntu", func() bool { return true })
	assert.Nil(t, err)
	assert.NotNil(t, up.metaController)
	assert.NotNil(t, up.timeNow)
	assert.NotNil(t, up.edgeNodeUpdater)

	fv := reflect.ValueOf(up.edgeNodeUpdater).Elem().FieldByName("subsystemUpdaters")
	assert.Equal(t, 6, fv.Len())
}

func Test_VerifyDefaultConstructorOfUpdateControllerEmt(t *testing.T) {
	up, err := NewUpdateController("", "emt", func() bool { return true })
	assert.Nil(t, err)
	assert.NotNil(t, up.metaController)
	assert.NotNil(t, up.timeNow)
	assert.NotNil(t, up.edgeNodeUpdater)

	fv := reflect.ValueOf(up.edgeNodeUpdater).Elem().FieldByName("subsystemUpdaters")
	assert.Equal(t, 1, fv.Len())
}

func Test_VerifyDefaultConstructorOfUpdateControllerInvalidOs(t *testing.T) {
	_, err := NewUpdateController("", "invalidos", func() bool { return true })
	assert.Equal(t, err, fmt.Errorf("unsupported os type: invalidos"))
}

func TestUpdater_StartUpdate_handleErrorThrownBySetMetaUpdateStatus(t *testing.T) {
	var interceptedStatusType pb.UpdateStatus_StatusType

	u := &UpdateController{
		metaController: &metadata.MetaController{
			SetMetaUpdateStatus: func(s pb.UpdateStatus_StatusType) error {
				interceptedStatusType = s
				return fmt.Errorf("SetMetaUpdateStatusError")
			},
		},
	}

	assert.NotEqual(t, pb.UpdateStatus_STATUS_TYPE_STARTED, interceptedStatusType)

	u.StartUpdate(1)

	assert.Equal(t, pb.UpdateStatus_STATUS_TYPE_STARTED, interceptedStatusType)
}
func TestUpdater_StartUpdate_handleErrorThrownBySetMetaUpdateTime(t *testing.T) {
	var interceptedStatusType pb.UpdateStatus_StatusType
	var interceptedUpdateTime time.Time

	now := time.Now()
	u := &UpdateController{
		metaController: &metadata.MetaController{
			SetMetaUpdateStatus: func(s pb.UpdateStatus_StatusType) error {
				interceptedStatusType = s
				return nil
			},
			SetMetaUpdateTime: func(updateTime time.Time) error {
				interceptedUpdateTime = updateTime
				return fmt.Errorf("SetMetaUpdateTimeError")
			},
			SetMetaUpdateDuration: func(updateDuration int64) error {
				require.Fail(t, "SetMetaUpdateDuration function shouldn't be called")
				return nil
			},
		},
		edgeNodeUpdater: testUpdater{
			updateFn: func() error {
				require.Fail(t, "updateAll function shall not be called")
				return nil
			},
		},
		timeNow: func() time.Time {
			return now
		},
	}

	u.StartUpdate(1)

	assert.Equal(t, pb.UpdateStatus_STATUS_TYPE_STARTED, interceptedStatusType)
	assert.Equal(t, now, interceptedUpdateTime)
}

type testUpdater struct {
	updateFn func() error
}

func (t testUpdater) update() error {
	return t.updateFn()
}

func TestUpdater_StartUpdate_handleErrorThrownBySetMetaUpdateDuration(t *testing.T) {
	var interceptedStatusType pb.UpdateStatus_StatusType
	var interceptedUpdateTime time.Time
	var interceptedUpdateDuration int64

	now := time.Now()
	u := &UpdateController{
		metaController: &metadata.MetaController{
			SetMetaUpdateStatus: func(s pb.UpdateStatus_StatusType) error {
				interceptedStatusType = s
				return nil
			},
			SetMetaUpdateTime: func(updateTime time.Time) error {
				interceptedUpdateTime = updateTime
				return nil
			},
			SetMetaUpdateDuration: func(updateDuration int64) error {
				interceptedUpdateDuration = updateDuration
				return fmt.Errorf("SetMetaUpdateDurationError")
			},
		},
		edgeNodeUpdater: testUpdater{
			updateFn: func() error {
				require.Fail(t, "updateAll function shall not be called")
				return nil
			},
		},
		timeNow: func() time.Time {
			return now
		},
	}

	u.StartUpdate(1)

	assert.Equal(t, pb.UpdateStatus_STATUS_TYPE_STARTED, interceptedStatusType)
	assert.Equal(t, now, interceptedUpdateTime)
	assert.EqualValues(t, 1, interceptedUpdateDuration)
}

type InMemoryFileSystem struct {
	fs afero.Fs
}

func (fs *InMemoryFileSystem) Read(path string) ([]byte, error) {
	return afero.ReadFile(fs.fs, path)
}

func TestUpdater_StartUpdate_handleErrorThrownDuringEdgeNodeUpdate(t *testing.T) {
	var interceptedStatusType []pb.UpdateStatus_StatusType
	var interceptedUpdateTime time.Time
	var interceptedUpdateDuration int64
	var interceptedUpdateAllExecution bool

	memFs := afero.NewMemMapFs()
	_, err := memFs.Create("/tmp/dummy.log")
	if err != nil {
		t.Errorf("Failed to create dummy log: %v", err)
	}
	fs := &InMemoryFileSystem{fs: memFs}

	now := time.Now()

	u := &UpdateController{
		metaController: &metadata.MetaController{
			SetMetaUpdateStatus: func(s pb.UpdateStatus_StatusType) error {
				interceptedStatusType = append(interceptedStatusType, s)
				return nil
			},
			SetMetaUpdateTime: func(updateTime time.Time) error {
				interceptedUpdateTime = updateTime
				return nil
			},
			SetMetaUpdateDuration: func(updateDuration int64) error {
				interceptedUpdateDuration = updateDuration
				return nil
			},
			SetMetaUpdateLog: func(s string) error {
				return nil
			},
		},
		fileSystem:      fs,
		granularLogPath: "/tmp/dummy.log",
		edgeNodeUpdater: testUpdater{updateFn: func() error {
			interceptedUpdateAllExecution = true
			return fmt.Errorf("updateAllError")
		}},
		timeNow: func() time.Time {
			return now
		},
		cleaner: &MockCleaner{},
	}

	u.StartUpdate(1)

	assert.Equal(t, now, interceptedUpdateTime)
	assert.EqualValues(t, 1, interceptedUpdateDuration)
	assert.True(t, interceptedUpdateAllExecution, "updateAll functional shall be called")
	assert.Equal(t, []pb.UpdateStatus_StatusType{pb.UpdateStatus_STATUS_TYPE_STARTED, pb.UpdateStatus_STATUS_TYPE_FAILED}, interceptedStatusType)
}

func TestUpdater_StartUpdate_happyPath(t *testing.T) {
	var interceptedStatusType []pb.UpdateStatus_StatusType
	var interceptedUpdateTime time.Time
	var interceptedUpdateDuration int64
	var interceptedUpdateAllExecution bool

	now := time.Now()

	u := &UpdateController{
		metaController: &metadata.MetaController{
			SetMetaUpdateStatus: func(s pb.UpdateStatus_StatusType) error {
				interceptedStatusType = append(interceptedStatusType, s)
				return nil
			},
			SetMetaUpdateTime: func(updateTime time.Time) error {
				interceptedUpdateTime = updateTime
				return nil
			},
			SetMetaUpdateDuration: func(updateDuration int64) error {
				interceptedUpdateDuration = updateDuration
				return nil
			},
		},
		edgeNodeUpdater: testUpdater{updateFn: func() error {
			interceptedUpdateAllExecution = true
			return nil
		}},
		timeNow: func() time.Time {
			return now
		},
	}

	u.StartUpdate(1)

	assert.Equal(t, []pb.UpdateStatus_StatusType{pb.UpdateStatus_STATUS_TYPE_STARTED}, interceptedStatusType)
	assert.Equal(t, now, interceptedUpdateTime)
	assert.EqualValues(t, 1, interceptedUpdateDuration)
	assert.True(t, interceptedUpdateAllExecution, "updateAll functional shall be called")
}

func TestUpdater_ContinueUpdate_happyPath(t *testing.T) {
	var interceptedUpdateAllExecution bool
	var interceptedStatusType []pb.UpdateStatus_StatusType

	u := &UpdateController{
		edgeNodeUpdater: testUpdater{updateFn: func() error {
			interceptedUpdateAllExecution = true
			return fmt.Errorf("updateAllError")
		}},
		metaController: &metadata.MetaController{
			SetMetaUpdateStatus: func(s pb.UpdateStatus_StatusType) error {
				interceptedStatusType = append(interceptedStatusType, s)
				return nil
			},
		},
	}

	u.ContinueUpdate()
	assert.True(t, interceptedUpdateAllExecution, "updateAll functional shall be called")
	assert.Equal(t, []pb.UpdateStatus_StatusType{pb.UpdateStatus_STATUS_TYPE_FAILED}, interceptedStatusType)
}

func Test_VerifyUpdate_handleLogFileDoesNotExistError(t *testing.T) {

	u := &UpdateController{}
	sut := u.VerifyUpdate
	status, log, time, err := sut("not/existing/file", "not/existing/log/file")

	assert.Equal(t, pb.UpdateStatus_STATUS_TYPE_FAILED, status)
	assert.Equal(t, log, "")
	assert.Empty(t, time)
	assert.ErrorContains(t, err, "reading INBC logs failed")
}

func Test_VerifyUpdate_handleLogFileIsEmptyError(t *testing.T) {
	sut := (&UpdateController{}).VerifyUpdate
	logFile, err := os.CreateTemp("/tmp", "inbc.log")
	require.NoError(t, err)
	defer logFile.Close()

	status, log, time, err := sut(logFile.Name(), "not/existing/log/file")

	assert.Equal(t, pb.UpdateStatus_STATUS_TYPE_FAILED, status)
	assert.Equal(t, log, "")
	assert.Empty(t, time)
	assert.ErrorContains(t, err, "INBC log file is empty")
}

func Test_VerifyUpdate_handleDeserializationError(t *testing.T) {

	logFile, err := os.CreateTemp("/tmp", "inbc.log")
	require.NoError(t, err)

	defer logFile.Close()

	_, err = logFile.WriteString("{")
	require.NoError(t, err)

	sut := (&UpdateController{}).VerifyUpdate
	status, log, time, err := sut(logFile.Name(), "not/existing/log/file")

	assert.Equal(t, pb.UpdateStatus_STATUS_TYPE_FAILED, status)
	assert.Equal(t, log, "")
	assert.Empty(t, time)
	assert.ErrorContains(t, err, "unmarshalling INBC update status failed")
}

func Test_VerifyUpdate_handleStatusSuccess(t *testing.T) {

	logFile, err := os.CreateTemp("/tmp", "inbc.log")
	require.NoError(t, err)

	defer logFile.Close()

	_, err = logFile.WriteString(`{"Status":"SUCCESS", "Time":"anyTime"}`)
	require.NoError(t, err)

	granularLogFile, logErr := os.CreateTemp("/tmp", "inbc-log.log")
	require.NoError(t, logErr)

	defer granularLogFile.Close()

	_, logErr = granularLogFile.WriteString(`{"UpdateLog":[{"update_type":"os","package_name":"emacs","update_time":"2024-07-03T01:50:55.935223","action":"install","status":"SUCCESS","version":"1:26.3+1-1ubuntu2\n"},{"update_type":"os","package_name":"wcalc","update_time":"2024-07-03T01:50:55.935223","action":"install","status":"SUCCESS","version":"2.5-3build1\n"}]}`)
	require.NoError(t, logErr)

	sut := (&UpdateController{}).VerifyUpdate
	status, log, time, err := sut(logFile.Name(), granularLogFile.Name())

	assert.Equal(t, pb.UpdateStatus_STATUS_TYPE_UPDATED, status)
	assert.Equal(t, `{"UpdateLog":[{"update_type":"os","package_name":"emacs","update_time":"2024-07-03T01:50:55.935223","action":"install","status":"SUCCESS","version":"1:26.3+1-1ubuntu2\n"},{"update_type":"os","package_name":"wcalc","update_time":"2024-07-03T01:50:55.935223","action":"install","status":"SUCCESS","version":"2.5-3build1\n"}]}`, log)
	assert.Equal(t, "anyTime", time)
	assert.NoError(t, err)
}

func Test_VerifyUpdate_handleStatusFailed(t *testing.T) {

	logFile, err := os.CreateTemp("/tmp", "inbc.log")
	require.NoError(t, err)

	defer logFile.Close()

	_, err = logFile.WriteString(`{"Status":"FAIL", "Time":"anyTime", "Error":"anyError"}`)
	require.NoError(t, err)

	granularLogFile, logErr := os.CreateTemp("/tmp", "inbc-log.log")
	require.NoError(t, logErr)

	defer granularLogFile.Close()

	_, logErr = granularLogFile.WriteString(`{"UpdateLog":[{"update_type":"os","package_name":"emacs","update_time":"2024-07-03T01:50:55.935223","action":"install","status":"FAIL","version":"1:26.3+1-1ubuntu2\n"},{"update_type":"os","package_name":"wcalc","update_time":"2024-07-03T01:50:55.935223","action":"install","status":"SUCCESS","version":"2.5-3build1\n"}]}`)
	require.NoError(t, logErr)

	sut := (&UpdateController{}).VerifyUpdate
	status, log, time, err := sut(logFile.Name(), granularLogFile.Name())

	assert.Equal(t, pb.UpdateStatus_STATUS_TYPE_FAILED, status)
	assert.Equal(t, `{"UpdateLog":[{"update_type":"os","package_name":"emacs","update_time":"2024-07-03T01:50:55.935223","action":"install","status":"FAIL","version":"1:26.3+1-1ubuntu2\n"},{"update_type":"os","package_name":"wcalc","update_time":"2024-07-03T01:50:55.935223","action":"install","status":"SUCCESS","version":"2.5-3build1\n"}]}`, log)
	assert.Equal(t, "anyTime", time)
	assert.ErrorContains(t, err, "anyError")
}

func Test_VerifyUpdate_handleStatusPending(t *testing.T) {

	logFile, err := os.CreateTemp("/tmp", "inbc.log")
	require.NoError(t, err)

	defer logFile.Close()

	_, err = logFile.WriteString(`{"Status":"PENDING", "Time":"anyTime", "Error":"anyError"}`)
	require.NoError(t, err)

	granularLogFile, logErr := os.CreateTemp("/tmp", "inbc-log.log")
	require.NoError(t, logErr)

	defer granularLogFile.Close()

	_, logErr = granularLogFile.WriteString(`{"UpdateLog":[{"update_type":"os","package_name":"emacs","update_time":"2024-07-03T01:50:55.935223","action":"install","status":"SUCCESS","version":"1:26.3+1-1ubuntu2\n"}]}`)
	require.NoError(t, logErr)

	sut := (&UpdateController{}).VerifyUpdate
	status, log, time, err := sut(logFile.Name(), granularLogFile.Name())

	assert.Equal(t, pb.UpdateStatus_STATUS_TYPE_STARTED, status)
	assert.Equal(t, `{"UpdateLog":[{"update_type":"os","package_name":"emacs","update_time":"2024-07-03T01:50:55.935223","action":"install","status":"SUCCESS","version":"1:26.3+1-1ubuntu2\n"}]}`, log)
	assert.Equal(t, "anyTime", time)
	assert.NoError(t, err)
}

func Test_VerifyUpdate_handleUnknownStatus(t *testing.T) {

	statusFile, err := os.CreateTemp("/tmp", "inbc.log")
	require.NoError(t, err)

	defer statusFile.Close()

	_, err = statusFile.WriteString(`{"Status":"UNKNOWN"}`)
	require.NoError(t, err)

	granularLogFile, logErr := os.CreateTemp("/tmp", "inbc-log.log")
	require.NoError(t, logErr)

	defer granularLogFile.Close()

	_, logErr = granularLogFile.WriteString(`{"UpdateLog":[]}`)
	require.NoError(t, logErr)

	sut := (&UpdateController{}).VerifyUpdate
	status, log, time, err := sut(statusFile.Name(), granularLogFile.Name())

	assert.Equal(t, pb.UpdateStatus_STATUS_TYPE_FAILED, status)
	assert.Equal(t, `{"UpdateLog":[]}`, log)
	assert.Empty(t, time)
	assert.ErrorContains(t, err, "status of the last OS update is unknown")
}

func Test_updateKernel_whenKernelParamsAreEmptyFunctionShouldDoNothing(t *testing.T) {
	testLogger, hook := test.NewNullLogger()
	log = testLogger.WithField("test", "test")

	kernelUpdater := kernelUpdater{
		Executor:   nil,
		kernelFile: "",
		MetaController: &metadata.MetaController{
			GetMetaUpdateSource: func() (*pb.UpdateSource, error) {
				return &pb.UpdateSource{
					KernelCommand: "",
				}, nil
			},
		},
	}
	sut := kernelUpdater.update
	err := sut()

	assert.NoError(t, err)
	assert.Equal(t, hook.LastEntry().Message, "update source or provided kernel is empty - skipping kernel update")
}

func Test_updateKernel_shouldFailAfterSymlinkIsInputted(t *testing.T) {
	symLinkPath := "/tmp/symlink_temp.txt"
	file, _ := os.CreateTemp("", "kernel_temp")
	defer file.Close()
	err := os.Symlink(file.Name(), symLinkPath)
	assert.Nil(t, err)
	defer os.Remove(symLinkPath)
	defer os.Remove(file.Name())
	kernelUpdater := kernelUpdater{
		Executor:   nil,
		kernelFile: symLinkPath,
		MetaController: &metadata.MetaController{
			GetMetaUpdateSource: func() (*pb.UpdateSource, error) {
				return &pb.UpdateSource{
					KernelCommand: "foo=bar",
				}, nil
			},
		},
	}

	sut := kernelUpdater.update

	assert.ErrorContains(t, sut(), fmt.Sprintf("loading metadata failed- %v is a symlink", symLinkPath))
}

func Test_updateKernel_shouldReturnErrorWhenDontHaveAccessToFile(t *testing.T) {
	socketPath := "/tmp/mysocket.sock"
	_ = os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	assert.NoError(t, err)
	defer listener.Close()
	kernelUpdater := kernelUpdater{
		Executor:   nil,
		kernelFile: socketPath,
		MetaController: &metadata.MetaController{
			GetMetaUpdateSource: func() (*pb.UpdateSource, error) {
				return &pb.UpdateSource{
					KernelCommand: "foo=bar",
				}, nil
			},
		},
	}

	sut := kernelUpdater.update

	assert.ErrorContains(t, sut(), fmt.Sprintf("failed to write modified kernel params to %v file", socketPath))
}

func Test_updateKernel_shouldReturnErrorWhenMetadataFileDoesntExist(t *testing.T) {
	metadata.MetaPath = ""
	kernelUpdater := kernelUpdater{
		Executor:       nil,
		kernelFile:     "/etc/default/grub.d/123",
		MetaController: metadata.NewController(),
	}

	err := kernelUpdater.update()

	assert.ErrorContains(t, err, "open : no such file or directory")
}

func Test_updateKernel_happyPath(t *testing.T) {
	var interceptedCommands [][]string
	kernelFile, _ := os.CreateTemp("", "kernel_temp")
	defer kernelFile.Close()
	defer os.Remove(kernelFile.Name())

	kernelUpdater := kernelUpdater{
		Executor: utils.NewExecutor[[]string](
			func(name string, args ...string) *[]string {
				command := append([]string{name}, args...)
				interceptedCommands = append(interceptedCommands, command)
				return new([]string)
			}, func(in *[]string) ([]byte, error) {
				return nil, nil
			}),
		MetaController: &metadata.MetaController{
			GetMetaUpdateSource: func() (*pb.UpdateSource, error) {
				return &pb.UpdateSource{
					KernelCommand: "foo=bar",
				}, nil
			},
			SetMetaUpdateInProgress: func(updateType metadata.UpdateType) error {
				return nil
			},
			GetInstallPackageList: func() (string, error) {
				return "", nil // No additional packages to install
			},
		},
		kernelFile: kernelFile.Name(),
	}

	sut := kernelUpdater.update

	require.NoError(t, sut())

	// Verify that only update-grub command was executed (no reboot command)
	require.Len(t, interceptedCommands, 1)
	require.Equal(t, upgradeGrubCommand, interceptedCommands[0])

	file, err := os.ReadFile(kernelFile.Name())
	require.NoError(t, err)
	require.Equal(t, `GRUB_CMDLINE_LINUX_DEFAULT="foo=bar"`, string(file))
}

func Test_updateKernel_withCustomReposConfigured(t *testing.T) {
	var interceptedCommands [][]string
	kernelFile, _ := os.CreateTemp("", "kernel_temp")
	defer kernelFile.Close()
	defer os.Remove(kernelFile.Name())

	kernelUpdater := kernelUpdater{
		Executor: utils.NewExecutor[[]string](
			func(name string, args ...string) *[]string {
				command := append([]string{name}, args...)
				interceptedCommands = append(interceptedCommands, command)
				return new([]string)
			}, func(in *[]string) ([]byte, error) {
				return nil, nil
			}),
		MetaController: &metadata.MetaController{
			GetMetaUpdateSource: func() (*pb.UpdateSource, error) {
				return &pb.UpdateSource{
					KernelCommand: "foo=bar",
					CustomRepos:   []string{"Types: deb\nURIs: https://test.repo.com\nSuites: example\nComponents: release"}, // Package installation will happen
				}, nil
			},
			SetMetaUpdateInProgress: func(updateType metadata.UpdateType) error {
				return nil
			},
			GetInstallPackageList: func() (string, error) {
				return "", nil
			},
		},
		kernelFile: kernelFile.Name(),
	}

	sut := kernelUpdater.update

	require.NoError(t, sut())

	// Verify that only update-grub command was executed (reboot handled by osAndPackagesUpdater)
	require.Len(t, interceptedCommands, 1)
	require.Equal(t, upgradeGrubCommand, interceptedCommands[0])

	file, err := os.ReadFile(kernelFile.Name())
	require.NoError(t, err)
	require.Equal(t, `GRUB_CMDLINE_LINUX_DEFAULT="foo=bar"`, string(file))
}

func Test_updateKernel_withAdditionalPackagesConfigured(t *testing.T) {
	var interceptedCommands [][]string
	kernelFile, _ := os.CreateTemp("", "kernel_temp")
	defer kernelFile.Close()
	defer os.Remove(kernelFile.Name())

	kernelUpdater := kernelUpdater{
		Executor: utils.NewExecutor[[]string](
			func(name string, args ...string) *[]string {
				command := append([]string{name}, args...)
				interceptedCommands = append(interceptedCommands, command)
				return new([]string)
			}, func(in *[]string) ([]byte, error) {
				return nil, nil
			}),
		MetaController: &metadata.MetaController{
			GetMetaUpdateSource: func() (*pb.UpdateSource, error) {
				return &pb.UpdateSource{
					KernelCommand: "foo=bar",
					// No custom repos, but additional packages to install
				}, nil
			},
			SetMetaUpdateInProgress: func(updateType metadata.UpdateType) error {
				return nil
			},
			GetInstallPackageList: func() (string, error) {
				return "package1 package2", nil // Additional packages will be installed
			},
		},
		kernelFile: kernelFile.Name(),
	}

	sut := kernelUpdater.update

	require.NoError(t, sut())

	// Verify that only update-grub command was executed (reboot handled by osAndPackagesUpdater)
	require.Len(t, interceptedCommands, 1)
	require.Equal(t, upgradeGrubCommand, interceptedCommands[0])

	file, err := os.ReadFile(kernelFile.Name())
	require.NoError(t, err)
	require.Equal(t, `GRUB_CMDLINE_LINUX_DEFAULT="foo=bar"`, string(file))
}

func Test_updateKernel_handleUpdateGrubCommandExecutionError(t *testing.T) {
	var interceptedCommand *[]string
	kernelFile, _ := os.CreateTemp("", "kernel_temp")
	defer kernelFile.Close()
	defer os.Remove(kernelFile.Name())
	kernelUpdater := kernelUpdater{
		Executor: utils.NewExecutor[[]string](
			func(cmd string, args ...string) *[]string {
				result := append([]string{cmd}, args...)
				return &result
			},
			func(cmd *[]string) ([]byte, error) {
				interceptedCommand = cmd
				return nil, fmt.Errorf("error")
			},
		),
		MetaController: &metadata.MetaController{
			GetMetaUpdateSource: func() (*pb.UpdateSource, error) {
				return &pb.UpdateSource{
					KernelCommand: "foo=bar",
				}, nil
			},
		},
		kernelFile: kernelFile.Name(),
	}

	sut := kernelUpdater.update

	require.ErrorContains(t, sut(), _ERR_GRUB_UPDATE_FAILED)
	assert.Equal(t, &upgradeGrubCommand, interceptedCommand)
}

func Test_osAndPackagesUpdater_happyPath(t *testing.T) {
	var interceptedCommand [][]string
	var interceptedUpdateInProgressCall metadata.UpdateType

	commandExecutor := utils.NewExecutor[[]string](
		func(name string, args ...string) *[]string {
			interceptedCommand = append(interceptedCommand, append([]string{name}, args...))
			return new([]string)
		}, func(in *[]string) ([]byte, error) {
			return nil, nil
		})

	updater := osAndAgentsUpdater{
		Executor: commandExecutor,
		MetaController: &metadata.MetaController{
			SetMetaUpdateInProgress: func(updateType metadata.UpdateType) error {
				interceptedUpdateInProgressCall = updateType
				return nil
			},
			GetMetaUpdateSource: func() (*pb.UpdateSource, error) {
				return &pb.UpdateSource{
					OsRepoUrl: "http://linux-ftp.fi.intel.com/pub/mirrors/ubuntu",
				}, nil
			},
		},

		AptController: &aptmirror.AptController{
			ConfigureOsAptRepo: func(osRepoURL string) error {
				return nil
			},
		},
	}
	sut := (&updater).update

	require.NoError(t, sut())
	assert.Equal(t, metadata.OS, interceptedUpdateInProgressCall)
	assert.Equal(t, [][]string{inbcSotaDownloadOnlyCommand, inbcSotaNoDownloadCommand}, interceptedCommand)
}

func Test_osAndPackagesUpdater_handleInbcSotaCommandExecutionError(t *testing.T) {

	commandExecutor := utils.NewExecutor[[]string](
		func(string, ...string) *[]string { return nil },
		func(*[]string) ([]byte, error) { return nil, fmt.Errorf("error") })

	updater := osAndAgentsUpdater{
		AptController: &aptmirror.AptController{
			ConfigureOsAptRepo: func(osRepoURL string) error {
				return nil
			},
		},
		MetaController: &metadata.MetaController{
			GetMetaUpdateSource: func() (*pb.UpdateSource, error) {
				return &pb.UpdateSource{
					OsRepoUrl: "http://test-os-repo-url.com",
				}, nil
			},
		},
		Executor: commandExecutor,
	}
	sut := updater.update

	require.ErrorContains(t, sut(), "failed to execute shell command([sudo inbc sota --mode download-only --reboot no])- error")
}

func Test_osAndPackagesUpdater_handleErrorThrownBySetMetaUpdateInProgress(t *testing.T) {
	var interceptedUpdateInProgressCall metadata.UpdateType

	commandExecutor := utils.NewExecutor[[]string](
		func(string, ...string) *[]string { return nil },
		func(*[]string) ([]byte, error) { return nil, nil })

	updater := osAndAgentsUpdater{
		AptController: &aptmirror.AptController{
			ConfigureOsAptRepo: func(osRepoURL string) error {
				return nil
			},
		},
		Executor: commandExecutor,
		MetaController: &metadata.MetaController{
			SetMetaUpdateInProgress: func(updateType metadata.UpdateType) error {
				interceptedUpdateInProgressCall = updateType
				return fmt.Errorf("error")
			},
			GetMetaUpdateSource: func() (*pb.UpdateSource, error) {
				return &pb.UpdateSource{
					OsRepoUrl: "http://test-os-repo-url.com",
				}, nil
			},
		},
	}
	sut := updater.update

	require.ErrorContains(t, sut(), "cannot write metafile: error")
	assert.Equal(t, metadata.OS, interceptedUpdateInProgressCall)
}

func Test_osAndPackagesUpdater_handleInbcSotaNoDownloadCommandExecutionError(t *testing.T) {

	commandExecutor := utils.NewExecutor[[]string](
		func(cmd string, args ...string) *[]string {
			result := append([]string{cmd}, args...)
			return &result
		},
		func(command *[]string) ([]byte, error) {
			switch {
			case reflect.DeepEqual(command, &inbcSotaNoDownloadCommand):
				return nil, fmt.Errorf("error")
			default:
				return []byte{}, nil
			}
		},
	)

	updater := osAndAgentsUpdater{
		AptController: &aptmirror.AptController{
			ConfigureOsAptRepo: func(osRepoURL string) error {
				return nil
			},
		},
		Executor: commandExecutor,
		MetaController: &metadata.MetaController{
			SetMetaUpdateInProgress: func(updateType metadata.UpdateType) error {
				return nil
			},
			GetMetaUpdateSource: func() (*pb.UpdateSource, error) {
				return &pb.UpdateSource{
					OsRepoUrl: "http://test-os-repo-url.com",
				}, nil
			},
		},
	}
	sut := updater.update

	require.ErrorContains(t, sut(), "failed to execute shell command([sudo inbc sota --mode no-download])- error")
}

func Test_edgeNodeUpdater_handleErrorThrownByGetMetaUpdateTime(t *testing.T) {
	enu := edgeNodeUpdater{
		MetaController: &metadata.MetaController{
			GetMetaUpdateTime: func() (time.Time, error) {
				return time.Time{}, fmt.Errorf("GetMetaUpdateTimeError")
			},
		},
		subsystemUpdaters: nil,
		timeNow:           nil,
	}

	sut := enu.checkTimeout
	assert.ErrorContains(t, sut(), "error reading metadata file: GetMetaUpdateTimeError")
}

func Test_edgeNodeUpdater_ShouldFailIfMetadataFileDoesntExist(t *testing.T) {
	metadata.MetaPath = ""
	enu := edgeNodeUpdater{
		MetaController:    metadata.NewController(),
		subsystemUpdaters: []SubsystemUpdater{&kernelUpdater{}},
	}

	err := enu.update()

	assert.ErrorContains(t, err, "error reading metadata file: open : no such file or directory")
}

func Test_edgeNodeUpdater_handleErrorThrownByGetMetaUpdateDuration(t *testing.T) {
	enu := edgeNodeUpdater{
		MetaController: &metadata.MetaController{
			GetMetaUpdateTime: func() (time.Time, error) {
				return time.Time{}, nil
			},
			GetMetaUpdateDuration: func() (int64, error) {
				return 1, fmt.Errorf("GetMetaUpdateDurationError")
			},
		},
		subsystemUpdaters: nil,
		timeNow:           nil,
	}

	sut := enu.checkTimeout
	assert.ErrorContains(t, sut(), "error reading metadata file: GetMetaUpdateDurationError")
}

func Test_edgeNodeUpdater_handleTimeoutError(t *testing.T) {
	fakeTime := time.Now()

	enu := edgeNodeUpdater{
		MetaController: &metadata.MetaController{
			GetMetaUpdateTime: func() (time.Time, error) {
				return fakeTime.Add(-10 * time.Minute), nil
			},
			GetMetaUpdateDuration: func() (int64, error) {
				return 1, nil
			},
		},
		timeNow: func() time.Time {
			return fakeTime
		},
	}

	sut := enu.checkTimeout
	assert.ErrorContains(t, sut(), "partial success - timed out before was able to perform full update")
}

func Test_edgeNodeUpdater_checkTimeout_happyPath(t *testing.T) {
	fakeTime := time.Now()

	enu := edgeNodeUpdater{
		MetaController: &metadata.MetaController{
			GetMetaUpdateTime: func() (time.Time, error) {
				return fakeTime.Add(0 * time.Minute), nil
			},
			GetMetaUpdateDuration: func() (int64, error) {
				return 1, nil
			},
		},
		timeNow: func() time.Time {
			return fakeTime
		},
	}

	sut := enu.checkTimeout
	assert.NoError(t, sut())
}

func Test_edgeNodeUpdater_update_happyPath(t *testing.T) {

	subsystemUpdaterExecuted := false
	numberOfGetMetaUpdateTimeExecutions := 0
	numberOfGetMetaUpdateDurationExecutions := 0

	fakeTime := time.Now()

	enu := edgeNodeUpdater{
		timeNow: func() time.Time {
			return fakeTime
		},
		MetaController: &metadata.MetaController{
			GetMetaUpdateTime: func() (time.Time, error) {
				numberOfGetMetaUpdateTimeExecutions++
				return fakeTime, nil
			},
			GetMetaUpdateDuration: func() (int64, error) {
				numberOfGetMetaUpdateDurationExecutions++
				return 1, nil
			},
		},
		subsystemUpdaters: []SubsystemUpdater{
			testUpdater{updateFn: func() error {
				subsystemUpdaterExecuted = true
				return nil
			}},
		},
	}

	sut := enu.update

	assert.NoError(t, sut())
	assert.True(t, subsystemUpdaterExecuted)
	assert.Equal(t, 1, numberOfGetMetaUpdateTimeExecutions)
	assert.Equal(t, 1, numberOfGetMetaUpdateDurationExecutions)
}

func Test_edgeNodeUpdater_update_handleErrorThrownByFirstUpdater(t *testing.T) {

	subsystemUpdater1Executed := false
	subsystemUpdater2Executed := false
	numberOfGetMetaUpdateTimeExecutions := 0
	numberOfGetMetaUpdateDurationExecutions := 0

	fakeTime := time.Now()

	enu := edgeNodeUpdater{
		timeNow: func() time.Time {
			return fakeTime
		},
		MetaController: &metadata.MetaController{
			GetMetaUpdateTime: func() (time.Time, error) {
				numberOfGetMetaUpdateTimeExecutions++
				return fakeTime, nil
			},
			GetMetaUpdateDuration: func() (int64, error) {
				numberOfGetMetaUpdateDurationExecutions++
				return 1, nil
			},
		},
		subsystemUpdaters: []SubsystemUpdater{
			testUpdater{updateFn: func() error {
				subsystemUpdater1Executed = true
				return fmt.Errorf("firstUpdaterError")
			}},
			testUpdater{updateFn: func() error {
				subsystemUpdater2Executed = true
				return nil
			}},
		},
	}

	sut := enu.update

	assert.ErrorContains(t, sut(), "firstUpdaterError")
	assert.True(t, subsystemUpdater1Executed)
	assert.False(t, subsystemUpdater2Executed)
	assert.Equal(t, 1, numberOfGetMetaUpdateTimeExecutions)
	assert.Equal(t, 1, numberOfGetMetaUpdateDurationExecutions)
}

func Test_selfUpdater_update_happyPath(t *testing.T) {

	var interceptedSetMetaUpdateInProgressCall metadata.UpdateType
	var interceptedCommandCall *[]string

	commandExecutor := utils.NewExecutor[[]string](
		stringCommand,
		func(command *[]string) ([]byte, error) {
			interceptedCommandCall = command
			return []byte{}, nil
		},
	)

	selfUpdater := selfUpdater{
		MetaController: &metadata.MetaController{
			SetMetaUpdateInProgress: func(updateType metadata.UpdateType) error {
				interceptedSetMetaUpdateInProgressCall = updateType
				return nil
			},
		},
		Executor: commandExecutor,
	}
	sut := selfUpdater.update

	assert.NoError(t, sut())
	assert.Equal(t, metadata.SELF, interceptedSetMetaUpdateInProgressCall)
	assert.EqualValues(t, &installPlatformUpdateAgentCommand, interceptedCommandCall)
}

func stringCommand(cmd string, args ...string) *[]string {
	result := append([]string{cmd}, args...)
	return &result
}

func Test_inbmUpdater_update_shouldRunUpdateAndSetCorrectUpdateInMetadata(t *testing.T) {
	metadataController := metadata.NewController()
	executor := utils.NewExecutor(func(name string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}, utils.ExecuteAndReadOutput)
	iUpdater := &inbmUpdater{
		MetaController: metadataController,
		Executor:       executor,
	}
	file, err := os.CreateTemp("/tmp", "metadata-")
	assert.NoError(t, err)
	defer file.Close()
	metadata.MetaPath = file.Name()
	err = metadata.InitMetadata()
	assert.NoError(t, err)
	status, err := metadata.GetMetaUpdateInProgress()
	assert.NoError(t, err)
	assert.Equal(t, "", status)

	err = iUpdater.update()

	assert.NoError(t, err)
	status, err = metadata.GetMetaUpdateInProgress()
	assert.NoError(t, err)
	assert.Equal(t, metadata.INBM, metadata.UpdateType(status))
}
func Test_inbmUpdater_update_shouldReturnErrorIfUnableToAccessMetadataFile(t *testing.T) {
	metadataController := metadata.NewController()
	metadata.MetaPath = ""
	executor := utils.NewExecutor(exec.Command, utils.ExecuteAndReadOutput)
	iUpdater := &inbmUpdater{
		MetaController: metadataController,
		Executor:       executor,
	}

	err := iUpdater.update()

	assert.ErrorContains(t, err, "open : no such file or directory")
}

func Test_inbmUpdater_update_shouldRunEvenForKernelOnlyUpdate(t *testing.T) {
	// Set up a temporary metadata file
	file, err := os.CreateTemp("/tmp", "metadata-")
	assert.NoError(t, err)
	defer os.Remove(file.Name())
	metadata.MetaPath = file.Name()

	// Initialize the metadata file first
	err = metadata.InitMetadata()
	assert.NoError(t, err)

	// Set up metadata with kernel-only update source
	err = metadata.SetMetaUpdateSource(&pb.UpdateSource{
		KernelCommand: "some.kernel.param=value",
		CustomRepos:   []string{}, // Empty repos indicates kernel-only update
	})
	assert.NoError(t, err)

	metadataController := metadata.NewController()
	executor := utils.NewExecutor(func(name string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}, utils.ExecuteAndReadOutput)

	iUpdater := &inbmUpdater{
		MetaController: metadataController,
		Executor:       executor,
	}

	err = iUpdater.update()

	// Should not error and should run INBM update even for kernel-only updates
	assert.NoError(t, err)

	// Verify that INBM update was set in progress (should be "INBM")
	updateInProgress, err := metadata.GetMetaUpdateInProgress()
	assert.NoError(t, err)
	assert.Equal(t, "INBM", updateInProgress)
}

func Test_selfUpdater_update_shouldRunUpdateAndSetCorrectUpdateInMetadata(t *testing.T) {
	metadataController := metadata.NewController()
	executor := utils.NewExecutor(func(name string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}, utils.ExecuteAndReadOutput)
	file, err := os.CreateTemp("/tmp", "metadata-")
	assert.NoError(t, err)
	defer file.Close()
	metadata.MetaPath = file.Name()
	err = metadata.InitMetadata()
	assert.NoError(t, err)
	status, err := metadata.GetMetaUpdateInProgress()
	assert.NoError(t, err)
	assert.Equal(t, "", status)
	selfUpdater := &selfUpdater{
		MetaController: metadataController,
		Executor:       executor,
	}

	err = selfUpdater.update()

	assert.NoError(t, err)
	status, err = metadata.GetMetaUpdateInProgress()
	assert.NoError(t, err)
	assert.Equal(t, metadata.SELF, metadata.UpdateType(status))
}

func Test_selfUpdater_update_shouldReturnErrorIfUnableToAccessMetadataFile(t *testing.T) {
	metadataController := metadata.NewController()
	executor := utils.NewExecutor(exec.Command, utils.ExecuteAndReadOutput)
	metadata.MetaPath = ""
	selfUpdater := &selfUpdater{
		MetaController: metadataController,
		Executor:       executor,
	}

	err := selfUpdater.update()

	assert.ErrorContains(t, err, "cannot write metafile: open : no such file or directory")
}

func Test_selfUpdater_update_shouldReturnErrorIfCmdReturnedError(t *testing.T) {
	metadataController := metadata.NewController()
	executor := utils.NewExecutor(func(name string, args ...string) *exec.Cmd {
		return exec.Command("binary-that-doesnt-exist")
	}, utils.ExecuteAndReadOutput)
	file, err := os.CreateTemp("/tmp", "metadata-")
	assert.NoError(t, err)
	defer file.Close()
	metadata.MetaPath = file.Name()
	err = metadata.InitMetadata()
	assert.NoError(t, err)
	status, err := metadata.GetMetaUpdateInProgress()
	assert.NoError(t, err)
	assert.Equal(t, "", status)
	selfUpdater := &selfUpdater{
		MetaController: metadataController,
		Executor:       executor,
	}

	err = selfUpdater.update()

	assert.ErrorContains(t, err, "PUA installation failed: failed to run 'binary-that-doesnt-exist' command")
}

// TODO - remove this commented out test completely if we remove ConfigureOsAptRepo
// (as OsRepoUrl in MM proto has been deprecated and we don't call ConfigureOsAptRepo anywhere else)
// func Test_packagesUpdate_shouldReturnErrorIfRepoConfigurationFailed(t *testing.T) {
// 	aptMirrorController := aptmirror.NewController()
// 	aptMirrorController.UpdatePackages = func() error {
// 		return nil
// 	}
// 	aptMirrorController.CleanupCustomRepos = func() error {
// 		return nil
// 	}
// 	aptMirrorController.ConfigureCustomAptRepos = func(customRepos []string) error {
// 		return nil
// 	}
// 	aptMirrorController.ConfigureOsAptRepo = func(osRepoURL string) error {
// 		return fmt.Errorf("test error")
// 	}

// 	metadataController := &metadata.MetaController{
// 		GetMetaUpdateSource: func() (*pb.UpdateSource, error) {
// 			return &pb.UpdateSource{
// 				OsRepoUrl: "http://test-os-repo-url.com",
// 				CustomRepos: []string{
// 					"Types: deb\nURIs: https://test1.com\nSuites: edge-node\nComponents: release\nSigned-By:\npublic GPG key",
// 					"Types: deb\nURIs: https://test2.com\nSuites: edge-node\nComponents: release\nSigned-By:\npublic GPG key",
// 				},
// 			}, nil
// 		},
// 	}

// 	pUpdater := &packagesUpdater{
// 		MetaController: metadataController,
// 		AptController:  aptMirrorController,
// 	}

// 	err := pUpdater.update()

// 	assert.ErrorContains(t, err, "failed to execute shell command - test error")
// }

func Test_packagesUpdater_handleErrorThrownByGetMetaUpdateSource(t *testing.T) {
	updater := packagesUpdater{
		MetaController: &metadata.MetaController{
			GetMetaUpdateSource: func() (*pb.UpdateSource, error) {
				return &pb.UpdateSource{}, fmt.Errorf("failed to read meta update source")
			},
		},
	}
	sut := updater.update

	require.ErrorContains(t, sut(), "failed to read meta update source")
}

func Test_packagesUpdater_update_shouldRunSuccessfully(t *testing.T) {
	metadataController := metadata.NewController()
	aptMirrorController := aptmirror.NewController()
	aptMirrorController.UpdatePackages = func() error {
		return nil
	}
	aptMirrorController.CleanupCustomRepos = func() error {
		return nil
	}
	aptMirrorController.ConfigureCustomAptRepos = func(customRepos []string) error {
		return nil
	}
	aptMirrorController.ConfigureOsAptRepo = func(osRepoURL string) error {
		return nil
	}
	file, err := os.CreateTemp("/tmp", "metadata-")
	assert.NoError(t, err)
	defer file.Close()
	metadata.MetaPath = file.Name()
	err = metadata.InitMetadata()
	assert.NoError(t, err)
	status, err := metadata.GetMetaUpdateInProgress()
	assert.NoError(t, err)
	assert.Equal(t, "", status)
	updateSource := &pb.UpdateSource{
		CustomRepos: []string{
			"Types: deb\nURIs: https://test1.com\nSuites: edge-node\nComponents: release\nSigned-By:\npublic GPG key",
			"Types: deb\nURIs: https://test2.com\nSuites: edge-node\nComponents: release\nSigned-By:\npublic GPG key",
		},
		OsRepoUrl: "http://linux-ftp.fi.intel.com/pub/mirrors/ubuntu",
	}
	err = metadata.SetMetaUpdateSource(updateSource)
	assert.NoError(t, err)
	pUpdater := &packagesUpdater{
		MetaController: metadataController,
		AptController:  aptMirrorController,
	}

	err = pUpdater.update()

	assert.NoError(t, err)
}

func Test_packagesUpdater_update_shouldRunSuccessfullyDeprecatedRepo(t *testing.T) {
	file, _ := os.CreateTemp("", "temp.list")
	defer file.Close()
	defer os.Remove(file.Name())
	metadataController := metadata.NewController()
	aptMirrorController := aptmirror.NewController()
	aptMirrorController.AptRepoFile = file.Name()
	aptMirrorController.UpdatePackages = func() error {
		return nil
	}
	aptMirrorController.CleanupCustomRepos = func() error {
		return nil
	}
	aptMirrorController.ConfigureCustomAptRepos = func(customRepos []string) error {
		return nil
	}
	file, err := os.CreateTemp("/tmp", "metadata-")
	assert.NoError(t, err)
	defer file.Close()
	metadata.MetaPath = file.Name()
	err = metadata.InitMetadata()
	assert.NoError(t, err)
	status, err := metadata.GetMetaUpdateInProgress()
	assert.NoError(t, err)
	assert.Equal(t, "", status)
	updateSource := &pb.UpdateSource{
		CustomRepos: []string{
			"deb [signed-by=/tmp/key.gpg] http://new.apt.repo.com/ jammy-backports main",
		},
	}
	err = metadata.SetMetaUpdateSource(updateSource)
	assert.NoError(t, err)
	pUpdater := &packagesUpdater{
		MetaController: metadataController,
		AptController:  aptMirrorController,
	}

	err = pUpdater.update()

	assert.NoError(t, err)
}

func Test_packagesUpdater_update_shouldFailWhenNotSignedRepo(t *testing.T) {
	metadataController := metadata.NewController()
	aptMirrorController := aptmirror.NewController()
	file, err := os.CreateTemp("/tmp", "metadata-")
	assert.NoError(t, err)
	defer file.Close()
	metadata.MetaPath = file.Name()
	err = metadata.InitMetadata()
	assert.NoError(t, err)
	status, err := metadata.GetMetaUpdateInProgress()
	assert.NoError(t, err)
	assert.Equal(t, "", status)
	updateSource := &pb.UpdateSource{
		CustomRepos: []string{
			"deb http://new.apt.repo.com/ jammy-backports main",
		},
	}
	err = metadata.SetMetaUpdateSource(updateSource)
	assert.NoError(t, err)
	pUpdater := &packagesUpdater{
		MetaController: metadataController,
		AptController:  aptMirrorController,
	}

	err = pUpdater.update()

	assert.ErrorContains(t, err, "deprecated custom apt repo configuration failed. Error")
}

func Test_packagesUpdater_update_shouldSkipIfReposAreEmpty(t *testing.T) {
	metadataController := metadata.NewController()

	file, err := os.CreateTemp("/tmp", "metadata-")
	assert.NoError(t, err)
	defer file.Close()
	metadata.MetaPath = file.Name()
	err = metadata.InitMetadata()
	assert.NoError(t, err)
	status, err := metadata.GetMetaUpdateInProgress()
	assert.NoError(t, err)
	assert.Equal(t, "", status)
	err = metadata.SetMetaUpdateSource(&pb.UpdateSource{CustomRepos: []string{}})
	assert.NoError(t, err)
	pUpdater := &packagesUpdater{
		MetaController: metadataController,
		AptController:  nil,
	}

	err = pUpdater.update()

	assert.NoError(t, err)
}

func Test_packagesUpdater_update_shouldFailIfMetadataFileDoesntExist(t *testing.T) {
	metadataController := metadata.NewController()
	aptMirrorController := aptmirror.NewController()
	pUpdater := &packagesUpdater{
		MetaController: metadataController,
		AptController:  aptMirrorController,
	}
	metadata.MetaPath = ""

	err := pUpdater.update()

	assert.ErrorContains(t, err, "error reading metadata file - open : no such file or directory")
}

func Test_newPackageInstaller_happyPath(t *testing.T) {
	metadataController := &metadata.MetaController{
		GetInstallPackageList: func() (string, error) {
			return "tree traceroute", nil
		},
		SetMetaUpdateInProgress: func(updateType metadata.UpdateType) error {
			return nil
		},
	}
	commandExecutor := utils.NewExecutor[[]string](
		stringCommand,
		func(command *[]string) ([]byte, error) {
			return []byte{}, nil
		},
	)

	newPackageInstaller := &newPackageInstaller{
		MetaController: metadataController,
		Executor:       commandExecutor,
	}

	sut := newPackageInstaller.update

	assert.NoError(t, sut())
	require.NoError(t, sut())
}

func Test_newPackageInstaller_emptyInstallPackagesListShouldNotReturnError(t *testing.T) {
	metadataController := metadata.NewController()
	commandExecutor := utils.NewExecutor[[]string](
		stringCommand,
		func(command *[]string) ([]byte, error) {
			return []byte{}, nil
		},
	)
	var packages = ""

	newPackageInstaller := &newPackageInstaller{
		MetaController: metadataController,
		Executor:       commandExecutor,
	}
	file, err := os.CreateTemp("/tmp", "metadata-")
	assert.NoError(t, err)
	defer file.Close()

	metadata.MetaPath = file.Name()
	err = metadata.InitMetadata()
	assert.NoError(t, err)
	err = metadata.SetInstalledPackages(packages)
	assert.NoError(t, err)

	sut := newPackageInstaller.update
	assert.NoError(t, sut())
	require.NoError(t, sut())
}

func Test_newPackageInstaller_shouldReturnErrorIfUnableToAccessMetadataFile(t *testing.T) {
	metadataController := metadata.NewController()
	commandExecutor := utils.NewExecutor[[]string](
		stringCommand,
		func(command *[]string) ([]byte, error) {
			return []byte{}, nil
		},
	)
	newPackageInstaller := &newPackageInstaller{
		MetaController: metadataController,
		Executor:       commandExecutor,
	}

	metadata.MetaPath = ""
	err := newPackageInstaller.update()

	assert.ErrorContains(t, err, "error reading metadata file: open : no such file or directory")
}

type mockExecutor struct {
	executeFunc func([]string) ([]byte, error)
}

func (m *mockExecutor) Execute(args []string) ([]byte, error) {
	return m.executeFunc(args)
}

func Test_emtUpdater_update(t *testing.T) {
	tests := []struct {
		name                string
		setMetaUpdateError  error
		executeError        error
		expectedError       string
		expectedExecuteCall bool
	}{
		{
			name:                "successful update",
			setMetaUpdateError:  nil,
			executeError:        nil,
			expectedError:       "",
			expectedExecuteCall: true,
		},
		{
			name:                "SetMetaUpdateInProgress error",
			setMetaUpdateError:  fmt.Errorf("meta update error"),
			executeError:        nil,
			expectedError:       "cannot write metafile: meta update error",
			expectedExecuteCall: false,
		},
		{
			name:                "Execute error",
			setMetaUpdateError:  nil,
			executeError:        fmt.Errorf("execute error"),
			expectedError:       "failed to execute shell command([sudo inbc sota --mode no-download])- execute error",
			expectedExecuteCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executeCalled := false
			updater := &emtUpdater{
				Executor: &mockExecutor{
					executeFunc: func(args []string) ([]byte, error) {
						executeCalled = true
						assert.Equal(t, []string{"sudo", "inbc", "sota", "--mode", "no-download"}, args)
						return nil, tt.executeError
					},
				},
				MetaController: &metadata.MetaController{
					SetMetaUpdateInProgress: func(updateType metadata.UpdateType) error {
						assert.Equal(t, metadata.OS, updateType)
						return tt.setMetaUpdateError
					},
				},
				DownloadChecker: func() bool { return true },
			}

			err := updater.update()

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.expectedExecuteCall, executeCalled)
		})
	}
}

func Test_emtUpdater_shouldNotUpdateIfDownloadNotDone(t *testing.T) {
	updater := &emtUpdater{
		DownloadChecker: func() bool { return false },
	}

	err := updater.update()
	assert.Equal(t, "cannot execute Edge Microvisor Toolkit update as download has not taken place",
		err.Error(),
		"update error should be set if download has not been done")
}

func TestCleaner_CleanupAfterUpdate_NoGranularFile(t *testing.T) {
	var interceptedCommands [][]string

	executor := utils.NewExecutor(
		func(name string, args ...string) *[]string {
			cmd := append([]string{name}, args...)
			interceptedCommands = append(interceptedCommands, cmd)
			return new([]string)
		},
		func(in *[]string) ([]byte, error) {
			return nil, nil
		},
	)

	cleaner := NewCleaner(executor, "ubuntu")

	err := cleaner.CleanupAfterUpdate("not/existing/log/file")

	assert.NoError(t, err)
	require.Len(t, interceptedCommands, 1)
	require.Equal(t, aptCleanCommand, interceptedCommands[0])
}

func TestCleaner_CleanupAfterUpdate_happyPath_GranularFileExists_Ubuntu(t *testing.T) {
	var interceptedCommands [][]string

	executor := utils.NewExecutor(
		func(name string, args ...string) *[]string {
			cmd := append([]string{name}, args...)
			interceptedCommands = append(interceptedCommands, cmd)
			return new([]string)
		},
		func(in *[]string) ([]byte, error) {
			return nil, nil
		},
	)

	granularLogFile, fileErr := os.CreateTemp("/tmp", "inbm-log-*.log")
	require.NoError(t, fileErr)
	defer os.Remove(granularLogFile.Name())

	cleaner := NewCleaner(executor, "ubuntu")

	err := cleaner.CleanupAfterUpdate(granularLogFile.Name())

	assert.NoError(t, err)
	require.Len(t, interceptedCommands, 2)
	require.Equal(t, aptCleanCommand, interceptedCommands[0])
	require.Equal(t, granularLogTruncateCommand, interceptedCommands[1])
}

func TestCleaner_CleanupAfterUpdate_happyPath_GranularFileExists_Emt(t *testing.T) {
	var interceptedCommands [][]string

	executor := utils.NewExecutor(
		func(name string, args ...string) *[]string {
			cmd := append([]string{name}, args...)
			interceptedCommands = append(interceptedCommands, cmd)
			return new([]string)
		},
		func(in *[]string) ([]byte, error) {
			return nil, nil
		},
	)

	granularLogFile, fileErr := os.CreateTemp("/tmp", "inbm-log-*.log")
	require.NoError(t, fileErr)
	defer os.Remove(granularLogFile.Name())

	cleaner := NewCleaner(executor, "emt")

	err := cleaner.CleanupAfterUpdate(granularLogFile.Name())

	assert.NoError(t, err)
	require.Len(t, interceptedCommands, 1)
	require.Equal(t, granularLogTruncateCommand, interceptedCommands[0])
}

func TestCleaner_CleanupAfterUpdate_UnsupportedOSType(t *testing.T) {
	executor := utils.NewExecutor(
		func(name string, args ...string) *[]string {
			return &[]string{}
		},
		func(in *[]string) ([]byte, error) {
			return nil, nil
		},
	)

	cleaner := NewCleaner(executor, "unsupported-os")

	err := cleaner.CleanupAfterUpdate("/var/log/inbm-update-log.log")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported os type")
}

func TestGranularLogShowsUpdate_SpecificCases(t *testing.T) {
	tests := []struct {
		name     string
		log      string
		expected bool
	}{
		{
			name:     "Empty JSON object",
			log:      "{}",
			expected: false,
		},
		{
			name: "JSON object with SUCCESS status",
			log: `{
				"UpdateLog": [
					{
						"update_type": "application",
						"package_name": "intel-opencl-icd",
						"update_time": "2024-12-10T01:23:11.123758",
						"action": "install",
						"status": "SUCCESS",
						"version": "22.14.22890-1"
					}
				]
			}`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := granularLogShowsUpdate(tt.log)
			assert.Equal(t, tt.expected, result)
		})
	}
}
