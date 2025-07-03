package utils

import (
	"errors"
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"
)

func TestIsBTRFSFileSystem_Success(t *testing.T) {
	// Define a statfsFunc that simulates a BTRFS filesystem
	statfsFunc := func(path string, stat *unix.Statfs_t) error {
		stat.Type = 0x9123683E // BTRFS magic number
		return nil
	}

	// Call IsBTRFS with the injected statfsFunc
	isBtrfs, err := IsBTRFSFileSystem("/valid/path", statfsFunc)
	assert.NoError(t, err)
	assert.True(t, isBtrfs, "The filesystem should be identified as BTRFS")
}

func TestIsBTRFSFileSystem_NotBTRFS(t *testing.T) {
	// Define a statfsFunc that simulates a non-BTRFS filesystem
	statfsFunc := func(path string, stat *unix.Statfs_t) error {
		stat.Type = 0xEF53 // EXT4 magic number
		return nil
	}

	// Call IsBTRFS with the injected statfsFunc
	isBtrfs, err := IsBTRFSFileSystem("/valid/path", statfsFunc)
	assert.NoError(t, err)
	assert.False(t, isBtrfs, "The filesystem should not be identified as BTRFS")
}

func TestIsBTRFSFileSystem_StatfsError(t *testing.T) {
	// Define a statfsFunc that simulates an error
	statfsFunc := func(path string, stat *unix.Statfs_t) error {
		return errors.New("mock error")
	}

	// Call IsBTRFS with the injected statfsFunc
	isBtrfs, err := IsBTRFSFileSystem("/invalid/path", statfsFunc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock error")
	assert.False(t, isBtrfs, "The filesystem should not be identified as BTRFS due to an error")
}

func TestRemoveFile_Success(t *testing.T) {
	fs := afero.NewOsFs()

	// Create a mock file
	filePath := "/tmp/testfile.txt"
	err := afero.WriteFile(fs, filePath, []byte("test content"), 0644)

	assert.NoError(t, err)

	// Call RemoveFile
	err = RemoveFile(fs, filePath)
	assert.NoError(t, err)

	// Verify the file was removed
	exists, err := afero.Exists(fs, filePath)
	assert.NoError(t, err)
	assert.False(t, exists, "File should have been removed")
}

func TestRemoveFile_NotAbsolutePath(t *testing.T) {
	fs := afero.NewMemMapFs() // Use an in-memory filesystem for testing

	// Call RemoveFile with a relative path
	err := RemoveFile(fs, "relative/path/to/file.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "access to the file is outside the allowed directories")
}

func TestRemoveFile_IsSymlink(t *testing.T) {
	fs := afero.NewOsFs() // Use the real filesystem for symlink testing
	targetPath := "/tmp/realfile.txt"
	symlinkPath := "/tmp/symlink.txt"

	// Create a real file
	err := afero.WriteFile(fs, targetPath, []byte("test content"), 0644)
	assert.NoError(t, err)
	defer func() {
		if err := fs.Remove(targetPath); err != nil {
			t.Errorf("failed to remove file: %v", err)
		}
	}()

	// Create a symlink pointing to the real file
	err = os.Symlink(targetPath, symlinkPath)
	assert.NoError(t, err)
	defer os.Remove(symlinkPath)

	// Call RemoveFile with the symlink
	err = RemoveFile(fs, symlinkPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file is a symlink")

	// Verify the symlink still exists
	exists, err := afero.Exists(fs, symlinkPath)
	assert.NoError(t, err)
	assert.True(t, exists, "Symlink should still exist")
}

func TestRemoveFile_FileDoesNotExist(t *testing.T) {
	fs := afero.NewOsFs() // Use an in-memory filesystem for testing

	// Call RemoveFile with a non-existent file
	err := RemoveFile(fs, "/tmp/nonexistent.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}

func TestRemoveFile_OutsideAllowedDirectories(t *testing.T) {
	fs := afero.NewMemMapFs() // Use an in-memory filesystem for testing

	// Create a file outside the allowed directories
	filePath := "/unauthorized/testfile.txt"
	err := afero.WriteFile(fs, filePath, []byte("test content"), 0644)
	assert.NoError(t, err)
	defer func() {
		if err := fs.Remove(filePath); err != nil {
			t.Errorf("failed to remove file: %v", err)
		}
	}()

	// Call RemoveFile
	err = RemoveFile(fs, filePath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "access to the file is outside the allowed directories")
}

func TestCopyFile_ValidFile(t *testing.T) {
	fs := afero.NewOsFs()
	srcPath := "/tmp/source.txt"
	destPath := "/tmp/destination.txt"

	// Create a source file
	err := afero.WriteFile(fs, srcPath, []byte("test content"), 0644)
	assert.NoError(t, err)
	defer func() {
		if err := fs.Remove(srcPath); err != nil {
			t.Errorf("failed to remove file: %v", err)
		}
	}()

	// Call the CopyFile function
	err = CopyFile(fs, srcPath, destPath)
	assert.NoError(t, err)
	defer func() {
		if err := fs.Remove(destPath); err != nil {
			t.Errorf("failed to remove file: %v", err)
		}
	}()

	// Verify the destination file exists and has the correct content
	content, err := afero.ReadFile(fs, destPath)
	assert.NoError(t, err)
	assert.Equal(t, "test content", string(content))
}

func TestCopyFile_SourceFileDoesNotExist(t *testing.T) {
	fs := afero.NewOsFs() // Use an in-memory filesystem
	srcPath := "/tmp/nonexistent.txt"
	destPath := "/tmp/destination.txt"

	// Call the CopyFile function
	err := CopyFile(fs, srcPath, destPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}

func TestCopyFile_DestinationPathInvalid(t *testing.T) {
	fs := afero.NewOsFs() // Use an in-memory filesystem
	srcPath := "/tmp/source.txt"
	destPath := "/invalid/destination.txt"

	// Create a source file
	err := afero.WriteFile(fs, srcPath, []byte("test content"), 0644)
	assert.NoError(t, err)
	defer func() {
		if err := fs.Remove(srcPath); err != nil {
			t.Errorf("failed to remove file: %v", err)
		}
	}()

	// Call the CopyFile function
	err = CopyFile(fs, srcPath, destPath)
	// Can't remove destination file, since it doesn't get created due to the simulated error

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "access to the file is outside the allowed directories")
}

func TestCopyFile_SourceFileIsSymlink(t *testing.T) {
	fs := afero.NewOsFs() // Use the real filesystem for symlink testing
	targetPath := "/tmp/realfile.txt"
	symlinkPath := "/tmp/symlink.txt"
	destPath := "/tmp/destination.txt"

	// Create a real file
	err := afero.WriteFile(fs, targetPath, []byte("test content"), 0644)
	assert.NoError(t, err)
	defer func() {
		if err := fs.Remove(targetPath); err != nil {
			t.Errorf("failed to remove file: %v", err)
		}
	}()

	// Create a symlink pointing to the real file
	err = os.Symlink(targetPath, symlinkPath)
	assert.NoError(t, err)
	defer func() {
		if err := fs.Remove(symlinkPath); err != nil {
			t.Errorf("failed to remove file: %v", err)
		}
	}()

	// Call the CopyFile function with the symlink as the source
	err = CopyFile(fs, symlinkPath, destPath)
	// Can't remove destination file, since it doesn't get created due to the simulated error

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file is a symlink")
}

func TestCopyFile_SourceFileOutsideAllowedDirs(t *testing.T) {
	fs := afero.NewMemMapFs() // Use an in-memory filesystem
	srcPath := "/unauthorized/source.txt"
	destPath := "/tmp/destination.txt"

	// Create a source file outside the allowed directories
	err := afero.WriteFile(fs, srcPath, []byte("test content"), 0644)
	assert.NoError(t, err)
	defer func() {
		if err := fs.Remove(srcPath); err != nil {
			t.Errorf("failed to remove file: %v", err)
		}
	}()

	// Call the CopyFile function
	err = CopyFile(fs, srcPath, destPath)
	// Can't remove destination file, since it doesn't get created due to the simulated error

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "outside the allowed directories")
}

func TestOpenFile_ValidFile(t *testing.T) {
	fs := afero.NewOsFs() // Use an in-memory filesystem
	filePath := "/tmp/testfile.txt"

	// Create a valid file in the allowed directory
	err := afero.WriteFile(fs, filePath, []byte("test content"), 0644)
	// Can't remove file, since it doesn't get created due to the simulated error

	assert.NoError(t, err)

	// Call the Open function
	file, err := OpenFile(fs, filePath, os.O_RDWR|os.O_CREATE, 0644)
	defer func() {
		if err := file.Close(); err != nil {
			t.Errorf("failed to remove file: %v", err)
		}
	}()

	assert.NoError(t, err)
	assert.NotNil(t, file)

	// Clean up
	_ = fs.Remove(filePath)
}

func TestOpenFile_FileOutsideAllowedDirs(t *testing.T) {
	fs := afero.NewMemMapFs() // Use an in-memory filesystem
	filePath := "/unauthorized/testfile.txt"

	// Create a file outside the allowed directories
	err := afero.WriteFile(fs, filePath, []byte("test content"), 0644)
	defer func() {
		if err := fs.Remove(filePath); err != nil {
			t.Errorf("failed to remove file: %v", err)
		}
	}()
	assert.NoError(t, err)

	// Call the Open function
	file, err := OpenFile(fs, filePath, os.O_RDWR|os.O_CREATE, 0644)
	// Can't close file, since it doesn't get opened due to the simulated error

	assert.Error(t, err)
	assert.Nil(t, file)
	assert.Contains(t, err.Error(), "outside the allowed directories")
}

func TestOpenFile_FileIsSymlink(t *testing.T) {
	fs := afero.NewOsFs() // Use the real filesystem
	targetPath := "/tmp/realfile.txt"
	symlinkPath := "/tmp/symlink.txt"

	// Create a real file and a symlink pointing to it
	err := afero.WriteFile(fs, targetPath, []byte("test content"), 0644)
	defer func() {
		if err := fs.Remove(targetPath); err != nil {
			t.Errorf("failed to remove file: %v", err)
		}
	}()
	assert.NoError(t, err)
	err = os.Symlink(targetPath, symlinkPath) // Use os.Symlink for creating symlinks
	defer func() {
		if err := fs.Remove(symlinkPath); err != nil {
			t.Errorf("failed to remove file: %v", err)
		}
	}()
	assert.NoError(t, err)

	// Call the Open function with the symlink
	file, err := OpenFile(fs, symlinkPath, os.O_RDWR|os.O_CREATE, 0644)
	// Can't close file, since it doesn't get opened due to the simulated error

	assert.Error(t, err)
	assert.Nil(t, file)
	assert.Contains(t, err.Error(), "file is a symlink")
}

func TestOpen_ValidFile(t *testing.T) {
	fs := afero.NewOsFs() // Use an in-memory filesystem
	filePath := "/tmp/testfile.txt"

	// Create a valid file in the allowed directory
	err := afero.WriteFile(fs, filePath, []byte("test content"), 0644)
	defer func() {
		if err := fs.Remove(filePath); err != nil {
			t.Errorf("failed to remove file: %v", err)
		}
	}()
	assert.NoError(t, err)

	// Call the Open function
	file, err := Open(fs, filePath)
	defer func() {
		if err := file.Close(); err != nil {
			t.Errorf("failed to remove file: %v", err)
		}
	}()
	assert.NoError(t, err)
	assert.NotNil(t, file)
}

func TestOpen_FileOutsideAllowedDirs(t *testing.T) {
	fs := afero.NewMemMapFs() // Use an in-memory filesystem
	filePath := "/unauthorized/testfile.txt"

	// Create a file outside the allowed directories
	err := afero.WriteFile(fs, filePath, []byte("test content"), 0644)
	defer func() {
		if err := fs.Remove(filePath); err != nil {
			t.Errorf("failed to remove file: %v", err)
		}
	}()
	assert.NoError(t, err)

	// Call the Open function
	file, err := Open(fs, filePath)
	// Can't close file, since it doesn't get opened due to the simulated error

	assert.Error(t, err)
	assert.Nil(t, file)
	assert.Contains(t, err.Error(), "outside the allowed directories")
}

func TestOpen_FileIsSymlink(t *testing.T) {
	fs := afero.NewOsFs() // Use the real filesystem
	targetPath := "/tmp/realfile.txt"
	symlinkPath := "/tmp/symlink.txt"

	// Create a real file and a symlink pointing to it
	err := afero.WriteFile(fs, targetPath, []byte("test content"), 0644)
	defer func() {
		if err := fs.Remove(targetPath); err != nil {
			t.Errorf("failed to remove file: %v", err)
		}
	}()
	assert.NoError(t, err)
	err = os.Symlink(targetPath, symlinkPath) // Use os.Symlink for creating symlinks
	defer func() {
		if err := fs.Remove(symlinkPath); err != nil {
			t.Errorf("failed to remove file: %v", err)
		}
	}()
	assert.NoError(t, err)

	// Call the Open function with the symlink
	file, err := Open(fs, symlinkPath)
	// Can't close file, since it doesn't get opened due to the simulated error

	assert.Error(t, err)
	assert.Nil(t, file)
	assert.Contains(t, err.Error(), "file is a symlink")
}

func TestIsFilePathAbsolute_ValidPath(t *testing.T) {
	filePath := "/etc/testfile.txt"

	// Call the isFilePathAbsolute function
	err := isFilePathAbsolute(filePath)
	assert.NoError(t, err)
}

func TestIsFilePathAbsolute_InvalidPath(t *testing.T) {
	filePath := "/unauthorized/testfile.txt"

	// Call the isFilePathAbsolute function
	err := isFilePathAbsolute(filePath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "outside the allowed directories")
}

func TestIsFilePathSymLink_ValidFile(t *testing.T) {
	fs := afero.NewOsFs() // Use an in-memory filesystem
	filePath := "/tmp/testfile.txt"

	// Create a valid file
	err := afero.WriteFile(fs, filePath, []byte("test content"), 0644)
	defer func() {
		if err := fs.Remove(filePath); err != nil {
			t.Errorf("failed to remove file: %v", err)
		}
	}()
	assert.NoError(t, err)

	// Call the isFilePathSymLink function
	err = isFilePathSymLink(filePath)
	assert.NoError(t, err)
}

func TestIsFilePathSymLink_IsSymlinkFile(t *testing.T) {
	fs := afero.NewOsFs() // Use the real filesystem
	targetPath := "/tmp/realfile.txt"
	symlinkPath := "/tmp/symlink.txt"

	// Create a real file
	err := afero.WriteFile(fs, targetPath, []byte("test content"), 0644)
	defer func() {
		if err := fs.Remove(targetPath); err != nil {
			t.Errorf("failed to remove file: %v", err)
		}
	}()
	assert.NoError(t, err)

	// Create a symlink pointing to the real file
	err = os.Symlink(targetPath, symlinkPath) // Use os.Symlink for creating symlinks
	defer func() {
		if err := fs.Remove(symlinkPath); err != nil {
			t.Errorf("failed to remove file: %v", err)
		}
	}()
	assert.NoError(t, err)

	// Verify the symlink
	fileInfo, err := os.Lstat(symlinkPath)
	assert.NoError(t, err)
	assert.True(t, fileInfo.Mode()&os.ModeSymlink != 0, "Expected symlink, but got mode: %v", fileInfo.Mode())

	// Call the isFilePathSymLink function with the symlink
	err = isFilePathSymLink(symlinkPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file is a symlink")
}

func TestMkdirAll_ValidDir(t *testing.T) {
	fs := afero.NewMemMapFs()
	dirPath := "/tmp/testdir"
	err := MkdirAll(fs, dirPath, 0755)
	assert.NoError(t, err)
	exists, err := afero.DirExists(fs, dirPath)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestMkdirAll_InvalidDir(t *testing.T) {
	fs := afero.NewMemMapFs()
	dirPath := "/unauthorized/testdir"
	err := MkdirAll(fs, dirPath, 0755)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "outside the allowed directories")
}

func TestMkdirAll_SymlinkDir(t *testing.T) {
	fs := afero.NewOsFs()
	targetDir := "/tmp/realtestdir"
	symlinkDir := "/tmp/symlinkdir"
	_ = fs.MkdirAll(targetDir, 0755)
	defer func() { _ = fs.RemoveAll(targetDir) }()
	_ = os.Symlink(targetDir, symlinkDir)
	defer func() { _ = fs.Remove(symlinkDir) }()
	err := MkdirAll(fs, symlinkDir, 0755)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "directory is a symlink")
}

func TestIsDirPathAbsolute_Valid(t *testing.T) {
	err := isDirPathAbsolute("/tmp/testdir")
	assert.NoError(t, err)
}

func TestIsDirPathAbsolute_Invalid(t *testing.T) {
	err := isDirPathAbsolute("/unauthorized/testdir")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "outside the allowed directories")
}

func TestIsDirPathSymLink_NonExistent(t *testing.T) {
	// Should not error if dir does not exist
	err := isDirPathSymLink("/tmp/doesnotexist")
	assert.NoError(t, err)
}

func TestIsDirPathSymLink_IsSymlink(t *testing.T) {
	fs := afero.NewOsFs()
	targetDir := "/tmp/realtestdir2"
	symlinkDir := "/tmp/symlinkdir2"
	_ = fs.MkdirAll(targetDir, 0755)
	defer func() { _ = fs.RemoveAll(targetDir) }()
	_ = os.Symlink(targetDir, symlinkDir)
	defer func() { _ = fs.Remove(symlinkDir) }()
	err := isDirPathSymLink(symlinkDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "directory is a symlink")
}
