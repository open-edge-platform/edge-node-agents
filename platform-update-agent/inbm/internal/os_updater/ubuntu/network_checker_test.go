package ubuntu

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckNetworkConnection_Success(t *testing.T) {
	mockExecutor := new(MockExecutor)

	// Mock the "ip route show default" command to succeed with valid output
	mockExecutor.On("Execute", []string{"ip", "route", "show", "default"}).Return("default via 192.168.1.1 dev eth0", "", nil)

	// Call CheckNetworkConnection
	result := CheckNetworkConnection(mockExecutor)

	// Assertions
	assert.True(t, result, "Expected network connection to be active")
	mockExecutor.AssertCalled(t, "Execute", []string{"ip", "route", "show", "default"})
}

func TestCheckNetworkConnection_NoDefaultGateway(t *testing.T) {
	mockExecutor := new(MockExecutor)

	// Mock the "ip route show default" command to succeed with no output
	mockExecutor.On("Execute", []string{"ip", "route", "show", "default"}).Return("", "", nil)

	// Call CheckNetworkConnection
	result := CheckNetworkConnection(mockExecutor)

	// Assertions
	assert.False(t, result, "Expected network connection to be inactive due to no default gateway")
	mockExecutor.AssertCalled(t, "Execute", []string{"ip", "route", "show", "default"})
}

func TestCheckNetworkConnection_CommandError(t *testing.T) {
	mockExecutor := new(MockExecutor)

	// Mock the "ip route show default" command to fail
	mockExecutor.On("Execute", []string{"ip", "route", "show", "default"}).Return("", "mock stderr", fmt.Errorf("mock command error"))

	// Call CheckNetworkConnection
	result := CheckNetworkConnection(mockExecutor)

	// Assertions
	assert.False(t, result, "Expected network connection to be inactive due to command error")
	mockExecutor.AssertCalled(t, "Execute", []string{"ip", "route", "show", "default"})
}
