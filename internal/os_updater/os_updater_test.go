package osupdater

import (
	"fmt"
	"testing"

	"github.com/intel/intel-inb-manageability/internal/inbd/utils"
	"github.com/spf13/afero"

	pb "github.com/intel/intel-inb-manageability/pkg/api/inbd/v1"
	"github.com/stretchr/testify/assert"
)

func TestUpdateOS_Success(t *testing.T) {
	mockFactory := &MockUpdaterFactory{
		CreateDownloaderFunc: func(*pb.UpdateSystemSoftwareRequest) Downloader {
			return &MockDownloader{
				DownloadFunc: func() error { return nil },
			}
		},
		CreateUpdaterFunc: func(executor utils.Executor, req *pb.UpdateSystemSoftwareRequest) Updater {
			return &MockUpdater{
				UpdateFunc: func() (bool, error) { return true, nil },
			}
		},
		CreateRebooterFunc: func(executor utils.Executor, req *pb.UpdateSystemSoftwareRequest) Rebooter {
			return &MockRebooter{
				RebootFunc: func() error { return nil },
			}
		},
		CreateCleanerFunc: func(executor utils.Executor, path string) Cleaner {
			return &MockCleaner{
				CleanFunc: func() error { return nil },
			}
		},
		CreateSnapshotterFunc: func(executor utils.Executor, req *pb.UpdateSystemSoftwareRequest) Snapshotter {
			return &MockSnapshotter{
				SnapshotFunc: func() error { return nil },
			}
		},
	}

	req := &pb.UpdateSystemSoftwareRequest{Mode: *pb.UpdateSystemSoftwareRequest_DOWNLOAD_MODE_NO_DOWNLOAD.Enum()}

	var updater *OSUpdater = &OSUpdater{
		req: req,
		isProceedWithoutRollbackFunc: func(*utils.Configurations) bool {
			return true
		},
		loadConfigFunc: func(afero.Fs, string) (*utils.Configurations, error) {
			return &utils.Configurations{}, nil
		},
	}

	resp, err := updater.UpdateOS(mockFactory)

	assert.NoError(t, err)
	assert.Equal(t, int32(200), resp.StatusCode)
	assert.Equal(t, "Success", resp.Error)
}

func TestUpdateOS_DownloadError(t *testing.T) {
	mockFactory := &MockUpdaterFactory{
		CreateDownloaderFunc: func(*pb.UpdateSystemSoftwareRequest) Downloader {
			return &MockDownloader{
				DownloadFunc: func() error { return fmt.Errorf("download error") },
			}
		},
	}

	req := &pb.UpdateSystemSoftwareRequest{Mode: pb.UpdateSystemSoftwareRequest_DOWNLOAD_MODE_FULL}

	var updater *OSUpdater = &OSUpdater{
		req: req,
		isProceedWithoutRollbackFunc: func(*utils.Configurations) bool {
			return true
		},
		loadConfigFunc: func(afero.Fs, string) (*utils.Configurations, error) {
			return &utils.Configurations{}, nil
		},
	}
	resp, err := updater.UpdateOS(mockFactory)

	assert.NoError(t, err)
	assert.Equal(t, int32(500), resp.StatusCode)
	assert.Equal(t, "download error", resp.Error)
}

func TestUpdateOS_SnapshotErrorProceedWithoutRollback(t *testing.T) {
	mockFactory := &MockUpdaterFactory{
		CreateDownloaderFunc: func(*pb.UpdateSystemSoftwareRequest) Downloader {
			return &MockDownloader{
				DownloadFunc: func() error { return nil },
			}
		},
		CreateSnapshotterFunc: func(utils.Executor, *pb.UpdateSystemSoftwareRequest) Snapshotter {
			return &MockSnapshotter{
				SnapshotFunc: func() error { return fmt.Errorf("snapshot error") },
			}
		},
		CreateUpdaterFunc: func(executor utils.Executor, req *pb.UpdateSystemSoftwareRequest) Updater {
			return &MockUpdater{
				UpdateFunc: func() (bool, error) { return true, nil },
			}
		},
		CreateCleanerFunc: func(utils.Executor, string) Cleaner {
			return &MockCleaner{
				CleanFunc: func() error { return nil },
			}
		},
		CreateRebooterFunc: func(executor utils.Executor, req *pb.UpdateSystemSoftwareRequest) Rebooter {
			return &MockRebooter{
				RebootFunc: func() error { return nil },
			}
		},
	}

	req := &pb.UpdateSystemSoftwareRequest{Mode: pb.UpdateSystemSoftwareRequest_DOWNLOAD_MODE_FULL}

	var updater *OSUpdater = &OSUpdater{
		req: req,
		isProceedWithoutRollbackFunc: func(*utils.Configurations) bool {
			return true
		},
		loadConfigFunc: func(afero.Fs, string) (*utils.Configurations, error) {
			return &utils.Configurations{}, nil
		},
	}

	resp, err := updater.UpdateOS(mockFactory)

	assert.NoError(t, err)
	assert.Equal(t, int32(200), resp.StatusCode)
	assert.Equal(t, "Success", resp.Error)
}

func TestUpdateOS_SnapshotErrorDoNotProceedWithoutRollback(t *testing.T) {
	mockFactory := &MockUpdaterFactory{
		CreateDownloaderFunc: func(*pb.UpdateSystemSoftwareRequest) Downloader {
			return &MockDownloader{
				DownloadFunc: func() error { return nil },
			}
		},
		CreateSnapshotterFunc: func(utils.Executor, *pb.UpdateSystemSoftwareRequest) Snapshotter {
			return &MockSnapshotter{
				SnapshotFunc: func() error { return fmt.Errorf("snapshot error") },
			}
		},
		CreateCleanerFunc: func(utils.Executor, string) Cleaner {
			return &MockCleaner{
				CleanFunc: func() error { return nil },
			}
		},
	}

	req := &pb.UpdateSystemSoftwareRequest{Mode: pb.UpdateSystemSoftwareRequest_DOWNLOAD_MODE_FULL}

	var updater *OSUpdater = &OSUpdater{
		req: req,
		isProceedWithoutRollbackFunc: func(*utils.Configurations) bool {
			return false
		},
		loadConfigFunc: func(afero.Fs, string) (*utils.Configurations, error) {
			return &utils.Configurations{}, nil
		},
	}
	resp, err := updater.UpdateOS(mockFactory)

	assert.NoError(t, err)
	assert.Equal(t, int32(500), resp.StatusCode)
	assert.Contains(t, resp.Error, "snapshot error")
}

func TestUpdateOS_UpdateError(t *testing.T) {
	mockFactory := &MockUpdaterFactory{
		CreateDownloaderFunc: func(*pb.UpdateSystemSoftwareRequest) Downloader {
			return &MockDownloader{
				DownloadFunc: func() error { return nil },
			}
		},
		CreateSnapshotterFunc: func(utils.Executor, *pb.UpdateSystemSoftwareRequest) Snapshotter {
			return &MockSnapshotter{
				SnapshotFunc: func() error { return nil },
			}
		},
		CreateUpdaterFunc: func(utils.Executor, *pb.UpdateSystemSoftwareRequest) Updater {
			return &MockUpdater{
				UpdateFunc: func() (bool, error) { return false, fmt.Errorf("update error") },
			}
		},
		CreateCleanerFunc: func(utils.Executor, string) Cleaner {
			return &MockCleaner{
				CleanFunc: func() error { return nil },
			}
		},
	}

	req := &pb.UpdateSystemSoftwareRequest{Mode: pb.UpdateSystemSoftwareRequest_DOWNLOAD_MODE_FULL}

	var updater *OSUpdater = &OSUpdater{
		req: req,
		isProceedWithoutRollbackFunc: func(*utils.Configurations) bool {
			return true
		},
		loadConfigFunc: func(afero.Fs, string) (*utils.Configurations, error) {
			return &utils.Configurations{}, nil
		},
	}

	resp, err := updater.UpdateOS(mockFactory)

	assert.NoError(t, err)
	assert.Equal(t, int32(500), resp.StatusCode)
	assert.Equal(t, "update error", resp.Error)
}

func TestUpdateOS_RebootError(t *testing.T) {
	mockFactory := &MockUpdaterFactory{
		CreateDownloaderFunc: func(*pb.UpdateSystemSoftwareRequest) Downloader {
			return &MockDownloader{
				DownloadFunc: func() error { return nil },
			}
		},
		CreateUpdaterFunc: func(utils.Executor, *pb.UpdateSystemSoftwareRequest) Updater {
			return &MockUpdater{
				UpdateFunc: func() (bool, error) { return true, nil },
			}
		},
		CreateSnapshotterFunc: func(utils.Executor, *pb.UpdateSystemSoftwareRequest) Snapshotter {
			return &MockSnapshotter{
				SnapshotFunc: func() error { return nil },
			}
		},
		CreateCleanerFunc: func(utils.Executor, string) Cleaner {
			return &MockCleaner{
				CleanFunc: func() error { return nil },
			}
		},
		CreateRebooterFunc: func(executor utils.Executor, req *pb.UpdateSystemSoftwareRequest) Rebooter {
			return &MockRebooter{
				RebootFunc: func() error { return fmt.Errorf("reboot error") },
			}
		},
	}

	req := &pb.UpdateSystemSoftwareRequest{Mode: pb.UpdateSystemSoftwareRequest_DOWNLOAD_MODE_FULL}

	var updater *OSUpdater = &OSUpdater{
		req: req,
		isProceedWithoutRollbackFunc: func(*utils.Configurations) bool {
			return true
		},
		loadConfigFunc: func(afero.Fs, string) (*utils.Configurations, error) {
			return &utils.Configurations{}, nil
		},
	}

	resp, err := updater.UpdateOS(mockFactory)

	assert.NoError(t, err)
	assert.Equal(t, int32(500), resp.StatusCode)
	assert.Equal(t, "reboot error", resp.Error)
}
