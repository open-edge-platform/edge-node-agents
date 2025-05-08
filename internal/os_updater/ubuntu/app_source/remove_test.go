package appsource

import (
	"fmt"
	"testing"

	pb "github.com/intel/intel-inb-manageability/pkg/api/inbd/v1"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestRemove_Success(t *testing.T) {
    fs := afero.NewMemMapFs() // Use an in-memory filesystem for testing

    // Create mock GPG key and source file
    gpgKeyPath := "/usr/share/keyrings/example-key.gpg"
    sourceFilePath := "/etc/apt/sources.list.d/example.list"

    remover := &Remover{
        fs: fs,
        removeGpgKeyFunc: func(fs afero.Fs, gpgKeyName string) error {
            return nil
        },
		removeSourceFileFunc: func(fs afero.Fs, sourceFilePath string) error {
            return nil
        },
		isExistGpgKeyFileFunc: func(fs afero.Fs, gpgKeyName string) bool {
			return true
		},
		isExistSourceFileFunc: func(fs afero.Fs, sourceFilePath string) bool {
			return true
		},		
    }

    req := &pb.RemoveApplicationSourceRequest{
        GpgKeyName: gpgKeyPath,
        Filename:   sourceFilePath,
    }

    // Call the Remove function
    err := remover.Remove(req)
    assert.NoError(t, err)
}

func TestRemove_GpgKeyFileDNE(t *testing.T) {
    fs := afero.NewMemMapFs() // Use an in-memory filesystem for testing

    remover := &Remover{
        fs: fs,
        removeSourceFileFunc: func(fs afero.Fs, sourceFilePath string) error {
            return nil
        },
		isExistGpgKeyFileFunc: func(fs afero.Fs, gpgKeyName string) bool {
			return false
		},
		isExistSourceFileFunc: func(fs afero.Fs, sourceFilePath string) bool {
			return true
		},	
    }

    req := &pb.RemoveApplicationSourceRequest{
        GpgKeyName: "nonexistent-key.gpg",
        Filename:   "/etc/apt/sources.list.d/example.list",
    }

    // Call the Remove function
    err := remover.Remove(req)
	assert.NoError(t, err)
}

func TestRemove_GpgKeyRemovalFailure(t *testing.T) {
    fs := afero.NewMemMapFs() // Use an in-memory filesystem for testing

    // Create a mock GPG key
    gpgKeyPath := "/usr/share/keyrings/example-key.gpg"
    err := afero.WriteFile(fs, gpgKeyPath, []byte("mock GPG key"), 0644)
    assert.NoError(t, err)

    remover := &Remover{
        fs: fs,
        removeGpgKeyFunc: func(fs afero.Fs, gpgKeyName string) error {
            return fmt.Errorf("mock GPG key removal error")
        },
        removeSourceFileFunc: func(fs afero.Fs, sourceFilePath string) error {
            return nil
        },
		isExistGpgKeyFileFunc: func(fs afero.Fs, gpgKeyName string) bool {
			return true
		},
		isExistSourceFileFunc: func(fs afero.Fs, sourceFilePath string) bool {
			return true
		},	
    }

    req := &pb.RemoveApplicationSourceRequest{
        GpgKeyName: gpgKeyPath,
        Filename:   "/etc/apt/sources.list.d/example.list",
    }

    // Call the Remove function
    err = remover.Remove(req)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "error removing GPG key")
}

func TestRemove_SourceFileDNE(t *testing.T) {
    fs := afero.NewMemMapFs() // Use an in-memory filesystem for testing

    remover := &Remover{
        fs: fs,
		isExistSourceFileFunc: func(fs afero.Fs, sourceFilePath string) bool {
			return false
		},	
    }

    req := &pb.RemoveApplicationSourceRequest{
        Filename:   "nonexistent-file.list",
    }

    // Call the Remove function
    err := remover.Remove(req)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "source file does not exist")
}

func TestRemove_SourceFileRemovalFailure(t *testing.T) {
    fs := afero.NewMemMapFs() // Use an in-memory filesystem for testing

    // Create a mock source file
    sourceFilePath := "/etc/apt/sources.list.d/example.list"
    err := afero.WriteFile(fs, sourceFilePath, []byte("mock source file"), 0644)
    assert.NoError(t, err)

    remover := &Remover{
        fs: fs,
        removeGpgKeyFunc: func(fs afero.Fs, gpgKeyName string) error {
            return nil
        },
        removeSourceFileFunc: func(fs afero.Fs, sourceFilePath string) error {
            return fmt.Errorf("mock source file removal error")
        },
		isExistGpgKeyFileFunc: func(fs afero.Fs, gpgKeyName string) bool {
			return true
		},
		isExistSourceFileFunc: func(fs afero.Fs, sourceFilePath string) bool {
			return true
		},	
    }

    req := &pb.RemoveApplicationSourceRequest{
        GpgKeyName: "/usr/share/keyrings/example-key.gpg",
        Filename:   sourceFilePath,
    }

    // Call the Remove function
    err = remover.Remove(req)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "error removing application source file")
}
