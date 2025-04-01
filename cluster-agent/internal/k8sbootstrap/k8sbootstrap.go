// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// package k8sbootstrap provides facilities to bootstrap k8s on the node on which is running
package k8sbootstrap

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/open-edge-platform/edge-node-agents/cluster-agent/internal/logger"
	"github.com/sirupsen/logrus"
)

var (
	execCommand = exec.CommandContext
	log         = logger.Logger
)

func Execute(ctx context.Context, command string) error {
	args := []string{"-o", "pipefail", "-c", command}
	log.Infof("Executing: bash %v", strings.Join(args, " "))
	cmd := execCommand(ctx, "bash", args...)
	cmd.WaitDelay = 10 * time.Millisecond
	cmd.Stdout = log.Logger.WriterLevel(logrus.InfoLevel)
	cmd.Stderr = log.Logger.WriterLevel(logrus.ErrorLevel)

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
