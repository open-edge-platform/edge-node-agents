// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package protovalidator

import (
	"testing"

	"google.golang.org/protobuf/proto"

	pb "github.com/open-edge-platform/edge-node-agents/common/pkg/api/status/proto"
)

func TestMustInit(t *testing.T) {
	tests := []struct {
		name        string
		preWarmMsgs []proto.Message
		wantErr     bool // wantErr means: expect panic
	}{
		{
			name:        "Valid initialization",
			preWarmMsgs: []proto.Message{&pb.GetStatusIntervalRequest{}, &pb.ReportStatusRequest{}},
			wantErr:     false,
		},
		{
			name:        "Empty initialization",
			preWarmMsgs: []proto.Message{},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if tt.wantErr && r == nil {
					t.Errorf("MustInit() expected panic but did not panic")
				}
				if !tt.wantErr && r != nil {
					t.Errorf("MustInit() panicked unexpectedly: %v", r)
				}
			}()

			MustInit(tt.preWarmMsgs)

			if protovalidator == nil {
				t.Errorf("MustInit() failed to initialize protovalidator")
			}
		})
	}
}

func TestValidateMessage(t *testing.T) {
	tests := []struct {
		name    string
		message proto.Message
		wantErr bool
	}{
		{
			name:    "Valid message",
			message: &pb.GetStatusIntervalRequest{AgentName: "test-agent"},
			wantErr: false,
		},
		{
			name:    "Invalid message",
			message: &pb.ReportStatusRequest{}, // No agent name
			wantErr: true,
		},
	}

	MustInit(statusMessages)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMessage(tt.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
