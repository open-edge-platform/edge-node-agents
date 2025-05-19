// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package protovalidator

import (
	"fmt"
	"os"

	protovalidate "github.com/bufbuild/protovalidate-go"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	pb "github.com/open-edge-platform/edge-node-agents/common/pkg/api/status/proto"
)

var (
	protovalidator  *protovalidate.Validator
	preWarmMessages = make([]proto.Message, 0)
)

var statusMessages = []proto.Message{
	&pb.GetStatusIntervalRequest{},
	&pb.ReportStatusRequest{},
}

func init() {
	MustInit(statusMessages)
}

func startProtovalidate(preWarmMsg ...proto.Message) (*protovalidate.Validator, error) {
	validator, err := protovalidate.New(
		// this warms up validator - pre-uploads message's validation constraints
		protovalidate.WithMessages(
			preWarmMsg...,
		),
	)
	if err != nil {
		fmt.Println("Error starting validator")
		return nil, errors.Wrap(err, "Error starting validator")
	}

	return &validator, nil
}

// MustInit initializes protovalidate and pre-warms it with provided preWarmMsgs.
// Note that this function does fatal in the case of error.
func MustInit(preWarmMsgs []proto.Message) {
	preWarmMessages = append(preWarmMessages, preWarmMsgs...)
	_validator, err := startProtovalidate(preWarmMessages...)
	if err != nil {
		fmt.Printf("Failed to initialize proto validate: %s", err)
		os.Exit(1)
	}
	protovalidator = _validator
}

func ValidateMessage(message proto.Message) error {
	if err := (*protovalidator).Validate(message); err != nil {
		fmt.Printf("Error validating input data: %v", message)
		return errors.Wrap(err, "Error validating input data")
	}

	return nil
}
