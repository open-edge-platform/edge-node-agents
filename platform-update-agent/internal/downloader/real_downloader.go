// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package downloader

import (
	"context"
	"fmt"
	"os/exec"

	pb "github.com/open-edge-platform/infra-managers/maintenance/pkg/api/maintmgr/v1"
	"github.com/sirupsen/logrus"
)

type RealDownloadExecutor struct {
	log           *logrus.Entry
	commandRunner CommandRunner
}

type CommandRunner interface {
	RunCommand(ctx context.Context, name string, args ...string) ([]byte, error)
}

type RealCommandRunner struct{}

func (r *RealCommandRunner) RunCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

func NewDownloadExecutor(log *logrus.Entry) *RealDownloadExecutor {
	return &RealDownloadExecutor{
		log:           log,
		commandRunner: &RealCommandRunner{},
	}
}

func (r *Downloader) setStatus(status pb.UpdateStatus_StatusType) error {
	if err := r.metadataController.SetMetaUpdateStatus(status); err != nil {
		return fmt.Errorf("cannot set metadata update status to %v: %w", status, err)
	}
	return nil
}

// Download starts the real download process with inbc
// prependToImageURL will be prepended to the image URL on download
func (r *RealDownloadExecutor) Download(ctx context.Context, prependToImageURL string, source *pb.OSProfileUpdateSource) error {
	actualUrl := prependToImageURL + source.OsImageUrl

	r.log.Info("DOWNLOAD: started")

	if err := r.runInbcCommand(ctx, "sota", "-m", "download-only", "-s", source.OsImageSha, "-u", actualUrl); err != nil {
		return fmt.Errorf("cannot download: %w", err)
	}

	r.log.Info("DOWNLOAD: finished")
	return nil
}

func (r *RealDownloadExecutor) runInbcCommand(ctx context.Context, args ...string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	cmdArgs := append([]string{"inbc"}, args...)
	output, err := r.commandRunner.RunCommand(ctx, "sudo", cmdArgs...)
	if err != nil {
		return fmt.Errorf("command failed. output: \n%s\nerror: %w", string(output), err)
	}
	return nil
}
