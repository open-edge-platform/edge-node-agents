package emt

import (
    "errors"
    "testing"

    "github.com/spf13/afero"
    "github.com/stretchr/testify/assert"
    pb "github.com/intel/intel-inb-manageability/pkg/api/inbd/v1"
)
type mockExecutor struct {
	commands [][]string
	stdout   []string
	stderr   []string
	errors   []error
}

func (m *mockExecutor) Execute(command []string) ([]byte, []byte, error) {
	m.commands = append(m.commands, command)
	var stdout, stderr string
	if len(m.stderr) > 0 {
		stderr = m.stderr[0]
		m.stderr = m.stderr[1:]
	}
	if len(m.stdout) > 0 {
		stdout = m.stdout[0]
		m.stdout = m.stdout[1:]
	}
	return []byte(stdout), []byte(stderr), m.errors[0]
}

func TestReboot_Success(t *testing.T) {
    mockExec := &mockExecutor{
		stdout: []string{""},
		errors: []error{nil},
	}
    mockFs := afero.NewMemMapFs()
    mockWriteUpdateStatus := func(fs afero.Fs, status, request, errorMsg string) {}
    mockWriteGranularLog := func(status, reason string) {}

    rebooter := &Rebooter{
        commandExecutor:   mockExec,
        request:           &pb.UpdateSystemSoftwareRequest{DoNotReboot: false},
        writeUpdateStatus: mockWriteUpdateStatus,
        writeGranularLog:  mockWriteGranularLog,
        fs:                mockFs,
    }

    err := rebooter.Reboot()
    assert.NoError(t, err)
    assert.Equal(t, [][]string{{"/usr/sbin/reboot"}}, mockExec.commands, "Reboot command should be executed")
}

func TestReboot_CommandExecutionFailure(t *testing.T) {
	mockExec := &mockExecutor{
		stdout: []string{""},
		errors: []error{errors.New("mock command execution error")},
	}
    mockFs := afero.NewMemMapFs()
    mockWriteUpdateStatusCalled := false
    mockWriteUpdateStatus := func(fs afero.Fs, status, request, errorMsg string) {
        mockWriteUpdateStatusCalled = true
        assert.Equal(t, FAIL, status)
        assert.Contains(t, errorMsg, "mock command execution error")
    }
    mockWriteGranularLogCalled := false
    mockWriteGranularLog := func(status, reason string) {
        mockWriteGranularLogCalled = true
        assert.Equal(t, FAIL, status)
        assert.Equal(t, FAILURE_REASON_UNSPECIFIED, reason)
    }

    rebooter := &Rebooter{
        commandExecutor:   mockExec,
        request:           &pb.UpdateSystemSoftwareRequest{},
        writeUpdateStatus: mockWriteUpdateStatus,
        writeGranularLog:  mockWriteGranularLog,
        fs:                mockFs,
    }

    err := rebooter.Reboot()
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "mock command execution error")
    assert.True(t, mockWriteUpdateStatusCalled, "writeUpdateStatus should be called on failure")
    assert.True(t, mockWriteGranularLogCalled, "writeGranularLog should be called on failure")
}
