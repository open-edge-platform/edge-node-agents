package ubuntu

import (
	"errors"
	//"os/exec"
	"testing"

	//"github.com/intel/intel-inb-manageability/internal/inbd/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/sys/unix"
	//"github.com/intel/intel-inb-manageability/internal/inbd/utils"
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

	// Mock isSnapperInstalled to return true
	mockExecutor.On("Execute", []string{"which", "snapper"}).Return("/usr/bin/snapper", "", nil)

	// Mock ClearStateFile to succeed
	mockExecutor.On("Execute", mock.Anything).Return("", "", nil)

	// Mock ensureSnapperConfig to succeed
	mockExecutor.On("Execute", []string{"snapper", "-c", "rootConfig", "list-configs"}).Return("rootConfig", "", nil)

	// Mock snapshot creation to succeed
	mockExecutor.On("Execute", []string{"snapper", "-c", "rootConfig", "create", "-p", "--description", "sota_update"}).Return("", "", nil)

	snapshotter := Snapshotter{
		CommandExecutor: mockExecutor,
		IsBTRFSFileSystemFunc: func(path string, statfsFunc func(string, *unix.Statfs_t) error) (bool, error) {
			return true, nil
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

	// Mock isSnapperInstalled to return false
	mockExecutor.On("Execute", []string{"which", "snapper"}).Return("", "", nil)

	snapshotter := Snapshotter{
		CommandExecutor: mockExecutor,
		IsBTRFSFileSystemFunc: func(path string, statfsFunc func(string, *unix.Statfs_t) error) (bool, error) {
			return true, nil
		},
	}

	// Call Snapshot
	err := snapshotter.Snapshot()

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "snapper is not installed")
	mockExecutor.AssertExpectations(t)
}

func TestSnapshot_WarningInStderr(t *testing.T) {
	mockExecutor := new(MockExecutor)

	// Mock IsBTRFSFileSystem to return true
	isBtrfsFunc := func(path string, statfsFunc func(string, *unix.Statfs_t) error) (bool, error) {
		return true, nil
	}

	// Mock isSnapperInstalled to return true
	mockExecutor.On("Execute", []string{"which", "snapper"}).Return("/usr/bin/snapper", "", nil)

	// Mock ClearStateFile to succeed
	mockExecutor.On("Execute", mock.Anything).Return("", "", nil)

	// Mock ensureSnapperConfig to succeed
	mockExecutor.On("Execute", []string{"snapper", "-c", "rootConfig", "list-configs"}).Return("rootConfig", "", nil)

	// Mock snapshot creation to produce a warning in stderr
	mockExecutor.On("Execute", []string{"snapper", "-c", "rootConfig", "create", "-p", "--description", "sota_update"}).Return("Snapshot created", "Warning: minor issue", nil)

	snapshotter := Snapshotter{
		CommandExecutor:       mockExecutor,
		IsBTRFSFileSystemFunc: isBtrfsFunc,
	}

	// Call Snapshot
	err := snapshotter.Snapshot()

	// Assertions
	assert.NoError(t, err)
	mockExecutor.AssertExpectations(t)
}

func TestIsSnapperInstalled_Success(t *testing.T) {
	mockExecutor := new(MockExecutor)

	// Mock the "which snapper" command to simulate snapper being installed
	mockExecutor.On("Execute", []string{"which", "snapper"}).Return("/usr/bin/snapper", "", nil)

	// Call isSnapperInstalled
	isInstalled, err := isSnapperInstalled(mockExecutor)

	// Assertions
	assert.NoError(t, err)
	assert.True(t, isInstalled, "Snapper should be detected as installed")
	mockExecutor.AssertCalled(t, "Execute", []string{"which", "snapper"})
}

// TODO: Create test that hits the false, no error case

func TestIsSnapperInstalled_CommandError(t *testing.T) {
	mockExecutor := new(MockExecutor)

	// Mock the "which snapper" command to simulate a command execution error
	mockExecutor.On("Execute", []string{"which", "snapper"}).Return(string([]byte("")), string([]byte("mock stderr")), errors.New("mock command error"))

	// Call isSnapperInstalled
	isInstalled, err := isSnapperInstalled(mockExecutor)

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "snapper is not installed")
	assert.False(t, isInstalled, "Snapper should not be detected as installed on error")
	mockExecutor.AssertCalled(t, "Execute", []string{"which", "snapper"})
}

func TestEnsureSnapperConfig_ConfigExists(t *testing.T) {
	mockExecutor := new(MockExecutor)

	// Mock the "snapper -c rootConfig list-configs" command to simulate the config already exists
	mockExecutor.On("Execute", []string{"snapper", "-c", "rootConfig", "list-configs"}).Return("rootConfig", "", nil)

	// Call ensureSnapperConfig
	err := ensureSnapperConfig(mockExecutor, "rootConfig")

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
	err := ensureSnapperConfig(mockExecutor, "rootConfig")

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
	err := ensureSnapperConfig(mockExecutor, "rootConfig")

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
	err := ensureSnapperConfig(mockExecutor, "rootConfig")

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create snapper config")
	mockExecutor.AssertCalled(t, "Execute", []string{"snapper", "-c", "rootConfig", "list-configs"})
	mockExecutor.AssertCalled(t, "Execute", []string{"snapper", "-c", "rootConfig", "create-config", "/"})
}
