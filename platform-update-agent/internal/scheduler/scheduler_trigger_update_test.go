// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"math"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// MockUpdateLocker is a manual mock for the UpdateLocker interface.
type MockUpdateLocker struct {
	LockCalled   bool
	UnlockCalled bool
	mu           sync.Mutex // To protect concurrent access if needed
}

func (m *MockUpdateLocker) LockForUpdate() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.LockCalled = true
}

func (m *MockUpdateLocker) Unlock() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.UnlockCalled = true
}

// MockUpdater is a manual mock for the Updater interface.
type MockUpdater struct {
	StartUpdateCalled  bool
	StartUpdateParam   int64
	StartUpdateParamMu sync.Mutex
}

func (m *MockUpdater) StartUpdate(durationSeconds int64) {
	m.StartUpdateParamMu.Lock()
	defer m.StartUpdateParamMu.Unlock()
	m.StartUpdateCalled = true
	m.StartUpdateParam = durationSeconds
}

func TestPuaScheduler_TriggerUpdate(t *testing.T) {
	// Initialize the manual mocks
	mockLocker := &MockUpdateLocker{}
	mockUpdater := &MockUpdater{}

	// Initialize a dummy logger
	logger := logrus.NewEntry(logrus.New())

	// Create the PuaScheduler instance with mocks
	scheduler := &PuaScheduler{
		updater:      mockUpdater,
		log:          logger,
		updateLocker: mockLocker,
	}

	// Define a tag for the update
	updateTag := "test-trigger"

	t.Run("StartUpdate is called when endTime is in the future", func(t *testing.T) {
		// Reset mock states
		mockLocker.LockCalled = false
		mockLocker.UnlockCalled = false
		mockUpdater.StartUpdateCalled = false
		mockUpdater.StartUpdateParam = 0

		// Arrange
		endTime := time.Now().Add(10 * time.Second) // endTime 10 seconds in the future

		// Act
		scheduler.triggerUpdate(updateTag, endTime, scheduler.IsUpdateAlreadyApplied, "ubuntu")

		// Assert
		assert.True(t, mockLocker.LockCalled, "LockForUpdate should be called")
		assert.True(t, mockLocker.UnlockCalled, "Unlock should be called")
		assert.True(t, mockUpdater.StartUpdateCalled, "StartUpdate should be called")

		expectedDuration := int64(math.Round(time.Until(endTime).Seconds()))
		assert.Equal(t, expectedDuration, mockUpdater.StartUpdateParam, "StartUpdate should be called with correctly rounded duration")
	})

	t.Run("StartUpdate is not called when endTime has passed", func(t *testing.T) {
		// Reset mock states
		mockLocker.LockCalled = false
		mockLocker.UnlockCalled = false
		mockUpdater.StartUpdateCalled = false
		mockUpdater.StartUpdateParam = 0

		// Arrange
		endTime := time.Now().Add(-10 * time.Second) // endTime 10 seconds in the past

		// Act
		scheduler.triggerUpdate(updateTag, endTime, scheduler.IsUpdateAlreadyApplied, "ubuntu")

		// Assert
		assert.True(t, mockLocker.LockCalled, "LockForUpdate should be called")
		assert.True(t, mockLocker.UnlockCalled, "Unlock should be called")
		assert.False(t, mockUpdater.StartUpdateCalled, "StartUpdate should not be called")
	})

	t.Run("LockForUpdate and Unlock are always called", func(t *testing.T) {
		// First Part: endTime in the future
		// Reset mock states
		mockLocker.LockCalled = false
		mockLocker.UnlockCalled = false
		mockUpdater.StartUpdateCalled = false
		mockUpdater.StartUpdateParam = 0

		endTimeFuture := time.Now().Add(5 * time.Second)

		// Act
		scheduler.triggerUpdate(updateTag, endTimeFuture, scheduler.IsUpdateAlreadyApplied, "ubuntu")

		// Assert
		assert.True(t, mockLocker.LockCalled, "LockForUpdate should be called")
		assert.True(t, mockLocker.UnlockCalled, "Unlock should be called")
		assert.True(t, mockUpdater.StartUpdateCalled, "StartUpdate should be called")

		expectedDurationFuture := int64(math.Round(time.Until(endTimeFuture).Seconds()))
		assert.Equal(t, expectedDurationFuture, mockUpdater.StartUpdateParam, "StartUpdate should be called with correctly rounded duration")

		// Second Part: endTime in the past
		// Reset mock states
		mockLocker.LockCalled = false
		mockLocker.UnlockCalled = false
		mockUpdater.StartUpdateCalled = false
		mockUpdater.StartUpdateParam = 0

		endTimePast := time.Now().Add(-5 * time.Second)

		// Act
		scheduler.triggerUpdate(updateTag, endTimePast, scheduler.IsUpdateAlreadyApplied, "ubuntu")

		// Assert
		assert.True(t, mockLocker.LockCalled, "LockForUpdate should be called")
		assert.True(t, mockLocker.UnlockCalled, "Unlock should be called")
		assert.False(t, mockUpdater.StartUpdateCalled, "StartUpdate should not be called")
	})

	t.Run("StartUpdate receives correctly rounded duration", func(t *testing.T) {
		// Reset mock states
		mockLocker.LockCalled = false
		mockLocker.UnlockCalled = false
		mockUpdater.StartUpdateCalled = false
		mockUpdater.StartUpdateParam = 0

		// Arrange
		// Set endTime to 4.6 seconds in the future; rounded to 5
		endTime := time.Now().Add(4*time.Second + 600*time.Millisecond)

		// Act
		scheduler.triggerUpdate(updateTag, endTime, scheduler.IsUpdateAlreadyApplied, "ubuntu")

		// Assert
		assert.True(t, mockLocker.LockCalled, "LockForUpdate should be called")
		assert.True(t, mockLocker.UnlockCalled, "Unlock should be called")
		assert.True(t, mockUpdater.StartUpdateCalled, "StartUpdate should be called")

		expectedDuration := int64(math.Round(time.Until(endTime).Seconds()))
		assert.Equal(t, expectedDuration, mockUpdater.StartUpdateParam, "StartUpdate should be called with correctly rounded duration")
	})
}
