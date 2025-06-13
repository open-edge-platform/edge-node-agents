package ubuntu

import (
    "testing"

    "github.com/stretchr/testify/assert"
    pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.inbm/pkg/api/inbd/v1"
)

func TestDownloader_Download(t *testing.T) {
    // Create a Downloader instance
    downloader := Downloader{
        Request: &pb.UpdateSystemSoftwareRequest{},
    }

    // Call the Download method
    err := downloader.Download()

    // Assertions
    assert.NoError(t, err, "Download should not return an error")
}