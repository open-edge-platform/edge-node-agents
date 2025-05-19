package utils

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockExecutor is a mock implementation of the Executor interface
type MockExecutor struct {
    mock.Mock
}

func (m *MockExecutor) Execute(command []string) ([]byte, []byte, error) {
    args := m.Called(command)
    return []byte(args.String(0)), []byte(args.String(1)), args.Error(2)
}

func TestClearStateFile_Success(t *testing.T) {
    mockExecutor := new(MockExecutor)
    stateFilePath := "/path/to/state/file"

    // Mock the truncate command to succeed
    truncateCommand := []string{"truncate", "-s", "0", stateFilePath}
    mockExecutor.On("Execute", truncateCommand).Return("", "", nil)

    // Call ClearStateFile
    err := ClearStateFile(mockExecutor, stateFilePath)

    // Assertions
    assert.NoError(t, err)
    mockExecutor.AssertCalled(t, "Execute", truncateCommand)
}

func TestClearStateFile_CommandError(t *testing.T) {
    mockExecutor := new(MockExecutor)
    stateFilePath := "/path/to/state/file"

    // Mock the truncate command to fail
    truncateCommand := []string{"truncate", "-s", "0", stateFilePath}
    mockExecutor.On("Execute", truncateCommand).Return("", "mock stderr", errors.New("mock error"))

    // Call ClearStateFile
    err := ClearStateFile(mockExecutor, stateFilePath)

    // Assertions
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "failed to truncate inbd state file")
    mockExecutor.AssertCalled(t, "Execute", truncateCommand)
}

func TestClearStateFile_EmptyStatePath(t *testing.T) {
    mockExecutor := new(MockExecutor)
    stateFilePath := ""

    // Mock the truncate command with an empty path
    emptyPathCommand := []string{"truncate", "-s", "0", stateFilePath}
    mockExecutor.On("Execute", emptyPathCommand).Return("", "mock stderr", errors.New("mock error"))

    // Call ClearStateFile
    err := ClearStateFile(mockExecutor, stateFilePath)

    // Assertions
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "failed to truncate inbd state file")
    mockExecutor.AssertCalled(t, "Execute", emptyPathCommand)
}

func TestReadStateFile_Success(t *testing.T) {
    // Create an in-memory filesystem
    fs := afero.NewOsFs()
	filePath := "/tmp/testfile.txt"

    // Define the expected state
    expectedState := INBDState{
        RestartReason:  "update",
        SnapshotNumber: 42,
        TiberVersion:   "1.0.0",
    }

    // Write the JSON content to the state file
    fileContent, err := json.Marshal(expectedState)
    assert.NoError(t, err)
    err = afero.WriteFile(fs, filePath, fileContent, 0644)
    assert.NoError(t, err)

    // Call ReadStateFile
    actualState, err := ReadStateFile(fs, filePath)

    // Assertions
    assert.NoError(t, err)
    assert.Equal(t, expectedState, actualState)
}

func TestReadStateFile_FileReadError(t *testing.T) {
	// Save the original Open function
    originalOpen := Open

    // Mock the Open function to simulate a file read error
    Open = func(fs afero.Fs, filePath string) (afero.File, error) {
        return nil, errors.New("mock file read error")
    }
    defer func() { Open = originalOpen }() // Restore the original function after the test

	// Create an in-memory filesystem
    fs := afero.NewMemMapFs()
    filePath := "/tmp/testfile.txt"

    // Create a file with no read permissions
    err := afero.WriteFile(fs, filePath, []byte{}, 0000)
    assert.NoError(t, err)

    // Call ReadStateFile
    _, err = ReadStateFile(fs, filePath)

    // Assertions
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "mock file read error")
}

func TestReadStateFile_JSONParseError(t *testing.T) {
    // Create an in-memory filesystem
    fs := afero.NewOsFs()
	filePath := "/tmp/testfile.txt"


    // Write invalid JSON content to the state file
    err := afero.WriteFile(fs, filePath, []byte("invalid-json"), 0644)
    assert.NoError(t, err)

    // Call ReadStateFile
    _, err = ReadStateFile(fs, filePath)

    // Assertions
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "error parsing JSON")
}

// TODO:  Add tests for WriteToStateFile