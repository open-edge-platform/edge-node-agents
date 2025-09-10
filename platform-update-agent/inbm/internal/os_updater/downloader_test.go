package osupdater

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOSDownloader_Download_Success(t *testing.T) {
	// Create an instance of OSDownloader
	downloader := &OSDownloader{}

	// Call the Download method
	err := downloader.Download()

	// Assert that no error is returned
	assert.NoError(t, err, "Download should not return an error")
}

func TestOSDownloader_InterfaceImplementation(t *testing.T) {
	// Ensure OSDownloader implements the Downloader interface
	var downloader Downloader = &OSDownloader{}

	// Call the Download method through the interface
	err := downloader.Download()

	// Assert that no error is returned
	assert.NoError(t, err, "Download should not return an error")
}
