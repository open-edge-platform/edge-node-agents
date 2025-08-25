package ubuntu

import (
	"errors"
	"fmt"
	"testing"

	utils "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/internal/inbd/utils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/sys/unix"
)

// MockExecutor is a mock implementation of utils.Executor
type MockExecutor struct {
	mock.Mock
}

func (m *MockExecutor) Execute(command []string) ([]byte, []byte, error) {
	args := m.Called(command)
	return []byte(args.String(0)), []byte(args.String(1)), args.Error(2)
}

// MockExitError is a custom implementation of exec.ExitError
type MockExitError struct{}

func (m *MockExitError) Error() string {
	return "mock exit error"
}

func (m *MockExitError) ExitCode() int {
	return 1
}

func TestSnapshot_Success(t *testing.T) {
	mockExecutor := new(MockExecutor)

	// Mock snapshot creation to succeed
	mockExecutor.On("Execute", []string{"snapper", "-c", "rootConfig", "create", "-p", "--description", "sota_update"}).Return("1", "", nil)

	snapshotter := Snapshotter{
		CommandExecutor: mockExecutor,
		IsBTRFSFileSystemFunc: func(path string, statfsFunc func(string, *unix.Statfs_t) error) (bool, error) {
			return true, nil
		},
		IsSnapperInstalledFunc: func(cmdExecutor utils.Executor) (bool, error) {
			return true, nil
		},
		EnsureSnapperConfigFunc: func(cmdExecutor utils.Executor, configName string) error {
			return nil
		},
		ClearStateFileFunc: func(cmdExecutor utils.Executor, stateFilePath string) error {
			return nil
		},
		WriteToStateFileFunc: func(fs afero.Fs, stateFilePath string, content string) error {
			return nil
		},
	}

	// Call Snapshot
	err := snapshotter.Snapshot()

	// Assertions
	assert.NoError(t, err)
	mockExecutor.AssertExpectations(t)
}

func TestSnapshot_NotBTRFS(t *testing.T) {
	mockExecutor := new(MockExecutor)

	snapshotter := Snapshotter{
		CommandExecutor: mockExecutor,
		IsBTRFSFileSystemFunc: func(path string, statfsFunc func(string, *unix.Statfs_t) error) (bool, error) {
			return false, nil
		},
	}

	// Call Snapshot
	err := snapshotter.Snapshot()

	// Assertions
	assert.NoError(t, err)
}

func TestSnapshot_SnapperNotInstalled(t *testing.T) {
	mockExecutor := new(MockExecutor)

	snapshotter := Snapshotter{
		CommandExecutor: mockExecutor,
		IsBTRFSFileSystemFunc: func(path string, statfsFunc func(string, *unix.Statfs_t) error) (bool, error) {
			return true, nil
		},
		IsSnapperInstalledFunc: func(cmdExecutor utils.Executor) (bool, error) {
			return false, nil
		},
	}

	// Call Snapshot
	err := snapshotter.Snapshot()

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "snapper is not installed")
	mockExecutor.AssertExpectations(t)
}

func TestSnapshot_SnapshotIDNotValidInteger(t *testing.T) {
	mockExecutor := new(MockExecutor)

	// Mock IsBTRFSFileSystem to return true
	isBtrfsFunc := func(path string, statfsFunc func(string, *unix.Statfs_t) error) (bool, error) {
		return true, nil
	}

	// Mock snapshot creation to produce a warning in stderr
	mockExecutor.On("Execute", []string{"snapper", "-c", "rootConfig", "create", "-p", "--description", "sota_update"}).Return("Snapshot created", "Warning: minor issue", nil)

	snapshotter := Snapshotter{
		CommandExecutor:       mockExecutor,
		IsBTRFSFileSystemFunc: isBtrfsFunc,
		IsSnapperInstalledFunc: func(cmdExecutor utils.Executor) (bool, error) {
			return true, nil
		},
		ClearStateFileFunc: func(cmdExecutor utils.Executor, stateFilePath string) error {
			return nil
		},
		EnsureSnapperConfigFunc: func(cmdExecutor utils.Executor, configName string) error {
			return nil
		},
		WriteToStateFileFunc: func(fs afero.Fs, stateFilePath string, content string) error {
			return nil
		},
	}

	// Call Snapshot
	err := snapshotter.Snapshot()

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "snapshot ID is not a valid integer")
	mockExecutor.AssertExpectations(t)
}

func TestSnapshot_ClearStateFileError(t *testing.T) {
	mockExecutor := new(MockExecutor)

	// Mock the "snapper create" command to succeed
	mockExecutor.On("Execute", []string{"snapper", "-c", "rootConfig", "create", "-p", "--description", "sota_update"}).Return("42", "", nil)

	// Mock other dependencies to succeed
	snapshotter := Snapshotter{
		CommandExecutor: mockExecutor,
		IsBTRFSFileSystemFunc: func(path string, statfsFunc func(string, *unix.Statfs_t) error) (bool, error) {
			return true, nil
		},
		IsSnapperInstalledFunc: func(cmdExecutor utils.Executor) (bool, error) {
			return true, nil
		},
		ClearStateFileFunc: func(cmdExecutor utils.Executor, stateFilePath string) error {
			return fmt.Errorf("mock clear state file error")
		},
		Fs: afero.NewMemMapFs(),
	}

	// Call Snapshot
	err := snapshotter.Snapshot()

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to clear dispatcher state file")
	assert.Contains(t, err.Error(), "mock clear state file error")
}

func TestSnapshot_EnsureSnapperConfigError(t *testing.T) {
	mockExecutor := new(MockExecutor)

	// Mock the "snapper create" command to succeed
	mockExecutor.On("Execute", []string{"snapper", "-c", "rootConfig", "create", "-p", "--description", "sota_update"}).Return("42", "", nil)

	// Mock other dependencies to succeed
	snapshotter := Snapshotter{
		CommandExecutor: mockExecutor,
		IsBTRFSFileSystemFunc: func(path string, statfsFunc func(string, *unix.Statfs_t) error) (bool, error) {
			return true, nil
		},
		IsSnapperInstalledFunc: func(cmdExecutor utils.Executor) (bool, error) {
			return true, nil
		},
		ClearStateFileFunc: func(cmdExecutor utils.Executor, stateFilePath string) error {
			return nil
		},
		EnsureSnapperConfigFunc: func(cmdExecutor utils.Executor, configName string) error {
			return fmt.Errorf("mock ensure snapper config error")
		},
		Fs: afero.NewMemMapFs(),
	}

	// Call Snapshot
	err := snapshotter.Snapshot()

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to ensure snapper config exists")
	assert.Contains(t, err.Error(), "mock ensure snapper config error")
}

func TestUndoChange_Success(t *testing.T) {
	mockExecutor := new(MockExecutor)

	// Mock the "snapper undochange" command to succeed
	mockExecutor.On("Execute", []string{"snapper", "-c", "rootConfig", "undochange", "42..0"}).Return("Undo successful", "", nil)

	// Call UndoChange
	err := UndoChange(mockExecutor, 42)

	// Assertions
	assert.NoError(t, err)
	mockExecutor.AssertCalled(t, "Execute", []string{"snapper", "-c", "rootConfig", "undochange", "42..0"})
}

func TestUndoChange_CommandError(t *testing.T) {
	mockExecutor := new(MockExecutor)

	// Mock the "snapper undochange" command to fail
	mockExecutor.On("Execute", []string{"snapper", "-c", "rootConfig", "undochange", "42..0"}).Return("", "mock stderr", fmt.Errorf("mock command error"))

	// Call UndoChange
	err := UndoChange(mockExecutor, 42)

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error executing command")
	assert.Contains(t, err.Error(), "mock stderr")
	assert.Contains(t, err.Error(), "mock command error")
	mockExecutor.AssertCalled(t, "Execute", []string{"snapper", "-c", "rootConfig", "undochange", "42..0"})
}

func TestUndoChange_StderrWarning(t *testing.T) {
	mockExecutor := new(MockExecutor)

	// Mock the "snapper undochange" command to succeed with a warning in stderr
	mockExecutor.On("Execute", []string{"snapper", "-c", "rootConfig", "undochange", "42..0"}).Return("Undo successful", "mock warning", nil)

	// Call UndoChange
	err := UndoChange(mockExecutor, 42)

	// Assertions
	assert.NoError(t, err)
	mockExecutor.AssertCalled(t, "Execute", []string{"snapper", "-c", "rootConfig", "undochange", "42..0"})
}

func TestUndoChange_SnapshotNumberZero(t *testing.T) {
	mockExecutor := new(MockExecutor)

	// Call UndoChange with snapshotNumber set to 0
	err := UndoChange(mockExecutor, 0)

	// Assertions
	assert.NoError(t, err)
	mockExecutor.AssertNotCalled(t, "Execute") // Ensure no command is executed
}

func TestDeleteSnapshot_Success(t *testing.T) {
	mockExecutor := new(MockExecutor)

	// Mock the "snapper delete" command to succeed
	mockExecutor.On("Execute", []string{"snapper", "-c", "rootConfig", "delete", "42"}).Return("Delete successful", "", nil)

	// Call DeleteSnapshot
	err := DeleteSnapshot(mockExecutor, 42)

	// Assertions
	assert.NoError(t, err)
	mockExecutor.AssertCalled(t, "Execute", []string{"snapper", "-c", "rootConfig", "delete", "42"})
}

func TestDeleteSnapshot_SnapshotNumberZero(t *testing.T) {
	mockExecutor := new(MockExecutor)

	// Call DeleteSnapshot with snapshotNumber set to 0
	err := DeleteSnapshot(mockExecutor, 0)

	// Assertions
	assert.NoError(t, err)
	mockExecutor.AssertNotCalled(t, "Execute") // Ensure no command is executed
}

func TestDeleteSnapshot_CommandError(t *testing.T) {
	mockExecutor := new(MockExecutor)

	// Mock the "snapper delete" command to fail
	mockExecutor.On("Execute", []string{"snapper", "-c", "rootConfig", "delete", "42"}).Return("", "mock stderr", fmt.Errorf("mock command error"))

	// Call DeleteSnapshot
	err := DeleteSnapshot(mockExecutor, 42)

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error executing command")
	assert.Contains(t, err.Error(), "mock stderr")
	assert.Contains(t, err.Error(), "mock command error")
	mockExecutor.AssertCalled(t, "Execute", []string{"snapper", "-c", "rootConfig", "delete", "42"})
}

func TestDeleteSnapshot_StderrWarning(t *testing.T) {
	mockExecutor := new(MockExecutor)

	// Mock the "snapper delete" command to succeed with a warning in stderr
	mockExecutor.On("Execute", []string{"snapper", "-c", "rootConfig", "delete", "42"}).Return("Delete successful", "mock warning", nil)

	// Call DeleteSnapshot
	err := DeleteSnapshot(mockExecutor, 42)

	// Assertions
	assert.NoError(t, err)
	mockExecutor.AssertCalled(t, "Execute", []string{"snapper", "-c", "rootConfig", "delete", "42"})
}

func TestIsSnapperInstalled_Success(t *testing.T) {
	mockExecutor := new(MockExecutor)

	// Mock the "which snapper" command to simulate snapper being installed
	mockExecutor.On("Execute", []string{"snapper", "--version"}).Return("/usr/bin/snapper", "", nil)

	// Call isSnapperInstalled
	isInstalled, err := IsSnapperInstalled(mockExecutor)

	// Assertions
	assert.NoError(t, err)
	assert.True(t, isInstalled, "Snapper should be detected as installed")
	mockExecutor.AssertCalled(t, "Execute", []string{"snapper", "--version"})
}

// TODO: Create test that hits the false, no error case

func TestIsSnapperInstalled_CommandError(t *testing.T) {
	mockExecutor := new(MockExecutor)

	// Mock the "snapper --version" command to simulate a command execution error
	mockExecutor.On("Execute", []string{"snapper", "--version"}).Return(string([]byte("")), string([]byte("mock stderr")), errors.New("mock command error"))

	// Call isSnapperInstalled
	isInstalled, err := IsSnapperInstalled(mockExecutor)

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "snapper is not installed")
	assert.False(t, isInstalled, "Snapper should not be detected as installed on error")
	mockExecutor.AssertCalled(t, "Execute", []string{"snapper", "--version"})
}

func TestEnsureSnapperConfig_ConfigExists(t *testing.T) {
	mockExecutor := new(MockExecutor)

	// Mock the "snapper -c rootConfig list-configs" command to simulate the config already exists
	mockExecutor.On("Execute", []string{"snapper", "-c", "rootConfig", "list-configs"}).Return("rootConfig", "", nil)

	// Call ensureSnapperConfig
	err := EnsureSnapperConfig(mockExecutor, "rootConfig")

	// Assertions
	assert.NoError(t, err)
	mockExecutor.AssertCalled(t, "Execute", []string{"snapper", "-c", "rootConfig", "list-configs"})
}

func TestEnsureSnapperConfig_CreateConfig(t *testing.T) {
	mockExecutor := new(MockExecutor)

	// Mock the "snapper -c rootConfig list-configs" command to simulate the config does not exist
	mockExecutor.On("Execute", []string{"snapper", "-c", "rootConfig", "list-configs"}).Return("", "", nil)

	// Mock the "snapper -c rootConfig create-config /" command to simulate successful creation
	mockExecutor.On("Execute", []string{"snapper", "-c", "rootConfig", "create-config", "/"}).Return("", "", nil)

	// Call ensureSnapperConfig
	err := EnsureSnapperConfig(mockExecutor, "rootConfig")

	// Assertions
	assert.NoError(t, err)
	mockExecutor.AssertCalled(t, "Execute", []string{"snapper", "-c", "rootConfig", "list-configs"})
	mockExecutor.AssertCalled(t, "Execute", []string{"snapper", "-c", "rootConfig", "create-config", "/"})
}

func TestEnsureSnapperConfig_ListConfigsError(t *testing.T) {
	mockExecutor := new(MockExecutor)

	// Mock the "snapper -c rootConfig list-configs" command to simulate an error
	mockExecutor.On("Execute", []string{"snapper", "-c", "rootConfig", "list-configs"}).Return("", "mock stderr", errors.New("mock error"))

	// Call ensureSnapperConfig
	err := EnsureSnapperConfig(mockExecutor, "rootConfig")

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check snapper config")
	mockExecutor.AssertCalled(t, "Execute", []string{"snapper", "-c", "rootConfig", "list-configs"})
}

func TestEnsureSnapperConfig_CreateConfigError(t *testing.T) {
	mockExecutor := new(MockExecutor)

	// Mock the "snapper -c rootConfig list-configs" command to simulate the config does not exist
	mockExecutor.On("Execute", []string{"snapper", "-c", "rootConfig", "list-configs"}).Return("", "", nil)

	// Mock the "snapper -c rootConfig create-config /" command to simulate an error
	mockExecutor.On("Execute", []string{"snapper", "-c", "rootConfig", "create-config", "/"}).Return("", "mock stderr", errors.New("mock error"))

	// Call ensureSnapperConfig
	err := EnsureSnapperConfig(mockExecutor, "rootConfig")

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create snapper config")
	mockExecutor.AssertCalled(t, "Execute", []string{"snapper", "-c", "rootConfig", "list-configs"})
	mockExecutor.AssertCalled(t, "Execute", []string{"snapper", "-c", "rootConfig", "create-config", "/"})
}

func TestSnapshot_WriteToStateFileError(t *testing.T) {
	mockExecutor := new(MockExecutor)

	// Mock the "snapper create" command to succeed
	mockExecutor.On("Execute", []string{"snapper", "-c", "rootConfig", "create", "-p", "--description", "sota_update"}).Return("42", "", nil)

	// Mock the other dependencies to succeed
	snapshotter := Snapshotter{
		CommandExecutor: mockExecutor,
		IsBTRFSFileSystemFunc: func(path string, statfsFunc func(string, *unix.Statfs_t) error) (bool, error) {
			return true, nil
		},
		IsSnapperInstalledFunc: func(cmdExecutor utils.Executor) (bool, error) {
			return true, nil
		},
		EnsureSnapperConfigFunc: func(cmdExecutor utils.Executor, configName string) error {
			return nil
		},
		ClearStateFileFunc: func(cmdExecutor utils.Executor, stateFilePath string) error {
			return nil
		},
		WriteToStateFileFunc: func(fs afero.Fs, stateFilePath string, content string) error {
			return fmt.Errorf("mock write to state file error")
		},
		Fs: afero.NewMemMapFs(),
	}

	// Call Snapshot
	err := snapshotter.Snapshot()

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write to state file")
	assert.Contains(t, err.Error(), "mock write to state file error")
}
