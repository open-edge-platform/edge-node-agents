package utils

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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
	assert.Contains(t, err.Error(), "access to the path is outside the allowed directories")
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
	assert.Contains(t, err.Error(), "path contains symlinks")

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
	assert.Contains(t, err.Error(), "access to the path is outside the allowed directories")
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
	assert.Contains(t, err.Error(), "access to the path is outside the allowed directories")
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
	assert.Contains(t, err.Error(), "path contains symlinks")
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
	assert.Contains(t, err.Error(), "access to the path is outside the allowed directories")
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
	assert.Contains(t, err.Error(), "access to the path is outside the allowed directories")
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
	assert.Contains(t, err.Error(), "path contains symlinks")
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
	assert.Contains(t, err.Error(), "access to the path is outside the allowed directories")
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
	assert.Contains(t, err.Error(), "path contains symlinks")
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
	assert.Contains(t, err.Error(), "access to the path is outside the allowed directories")
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
	assert.Contains(t, err.Error(), "path contains symlinks")
}

// ReadFile Test Cases
func TestReadFile_Success(t *testing.T) {
	fs := afero.NewOsFs()
	testContent := "Hello, World!\nThis is test content."

	// Create a test file in /tmp
	tmpFile, err := os.CreateTemp("/tmp", "read_test_*.txt")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(testContent)
	assert.NoError(t, err)
	tmpFile.Close()

	// Test reading
	data, err := ReadFile(fs, tmpFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, testContent, string(data))
}

func TestReadFile_FileOutsideAllowedDirectories(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Try to read from /home (not in allowedBaseDirs)
	data, err := ReadFile(fs, "/home/test.txt")
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "access to the path is outside the allowed directories")
}

func TestReadFile_FileDoesNotExist(t *testing.T) {
	fs := afero.NewOsFs()

	data, err := ReadFile(fs, "/tmp/nonexistent_file.txt")
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "no such file or directory")
}

func TestReadFile_SymlinkFile(t *testing.T) {
	fs := afero.NewOsFs()

	// Create a regular file
	tmpFile, err := os.CreateTemp("/tmp", "symlink_target_*.txt")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.WriteString("target content")
	assert.NoError(t, err)
	tmpFile.Close()

	// Create a symlink to it
	symlinkPath := "/tmp/test_symlink_read.txt"
	err = os.Symlink(tmpFile.Name(), symlinkPath)
	assert.NoError(t, err)
	defer os.Remove(symlinkPath)

	// Try to read the symlink
	data, err := ReadFile(fs, symlinkPath)
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "file path was changed via symlink")
}

func TestReadFile_EmptyFile(t *testing.T) {
	fs := afero.NewOsFs()

	// Create an empty file
	tmpFile, err := os.CreateTemp("/tmp", "empty_test_*.txt")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Test reading empty file
	data, err := ReadFile(fs, tmpFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, []byte{}, data)
}

func TestReadFile_LargeFile(t *testing.T) {
	fs := afero.NewOsFs()

	// Create a large content file
	largeContent := strings.Repeat("Large content line\n", 1000)
	tmpFile, err := os.CreateTemp("/tmp", "large_test_*.txt")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(largeContent)
	assert.NoError(t, err)
	tmpFile.Close()

	// Test reading large file
	data, err := ReadFile(fs, tmpFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, largeContent, string(data))
}

// WriteFile Test Cases
func TestWriteFile_Success(t *testing.T) {
	fs := afero.NewOsFs()
	testContent := []byte("Hello, World!\nThis is test content.")
	filePath := "/tmp/write_test.txt"
	defer os.Remove(filePath)

	err := WriteFile(fs, filePath, testContent, 0644)
	assert.NoError(t, err)

	// Verify file was written correctly
	data, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, testContent, data)

	// Check file permissions
	info, err := os.Stat(filePath)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0644), info.Mode().Perm())
}

func TestWriteFile_FileOutsideAllowedDirectories(t *testing.T) {
	fs := afero.NewMemMapFs()
	testContent := []byte("test content")

	err := WriteFile(fs, "/home/test.txt", testContent, 0644)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "access to the path is outside the allowed directories")
}

func TestWriteFile_SymlinkFile(t *testing.T) {
	fs := afero.NewOsFs()
	testContent := []byte("test content")

	// Create a regular file
	tmpFile, err := os.CreateTemp("/tmp", "symlink_target_*.txt")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Create a symlink to it
	symlinkPath := "/tmp/write_test_symlink.txt"
	err = os.Symlink(tmpFile.Name(), symlinkPath)
	assert.NoError(t, err)
	defer os.Remove(symlinkPath)

	// Try to write to the symlink
	err = WriteFile(fs, symlinkPath, testContent, 0644)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file path was changed via symlink")
}

func TestWriteFile_OverwriteExistingFile(t *testing.T) {
	fs := afero.NewOsFs()
	filePath := "/tmp/overwrite_test.txt"
	defer os.Remove(filePath)

	// Write initial content
	initialContent := []byte("initial content")
	err := WriteFile(fs, filePath, initialContent, 0644)
	assert.NoError(t, err)

	// Overwrite with new content
	newContent := []byte("new content")
	err = WriteFile(fs, filePath, newContent, 0644)
	assert.NoError(t, err)

	// Verify new content
	data, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, newContent, data)
}

func TestWriteFile_CreateDirectoriesIfNeeded(t *testing.T) {
	fs := afero.NewOsFs()
	filePath := "/tmp/new_dir/write_test.txt"
	testContent := []byte("test content")

	// Create the directory first (since our function doesn't create directories)
	err := os.MkdirAll(filepath.Dir(filePath), 0755)
	assert.NoError(t, err)
	defer os.RemoveAll("/tmp/new_dir")

	err = WriteFile(fs, filePath, testContent, 0644)
	assert.NoError(t, err)

	// Verify file was written
	data, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, testContent, data)
}

func TestWriteFile_EmptyContent(t *testing.T) {
	fs := afero.NewOsFs()
	filePath := "/tmp/empty_write_test.txt"
	defer os.Remove(filePath)

	err := WriteFile(fs, filePath, []byte{}, 0644)
	assert.NoError(t, err)

	// Verify empty file was created
	data, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, []byte{}, data)
}

func TestWriteFile_DifferentPermissions(t *testing.T) {
	fs := afero.NewOsFs()
	testContent := []byte("test content")
	filePath := "/tmp/perm_test.txt"
	defer os.Remove(filePath)

	err := WriteFile(fs, filePath, testContent, 0600)
	assert.NoError(t, err)

	// Check file permissions
	info, err := os.Stat(filePath)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

// CreateTempFile Test Cases
func TestCreateTempFile_Success(t *testing.T) {
	fs := afero.NewOsFs()

	tmpFile, err := CreateTempFile(fs, "/tmp", "test_*.json")
	assert.NoError(t, err)
	assert.NotNil(t, tmpFile)
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Verify file exists and is in /tmp
	assert.True(t, strings.HasPrefix(tmpFile.Name(), "/tmp/test_"))
	assert.True(t, strings.HasSuffix(tmpFile.Name(), ".json"))

	// Verify file is readable/writable
	_, err = tmpFile.WriteString("test content")
	assert.NoError(t, err)
}

func TestCreateTempFile_DefaultTempDir(t *testing.T) {
	fs := afero.NewOsFs()

	tmpFile, err := CreateTempFile(fs, "", "test_*.txt")
	assert.NoError(t, err)
	assert.NotNil(t, tmpFile)
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Should be in system temp directory (/tmp on most systems)
	assert.Contains(t, tmpFile.Name(), "test_")
	assert.True(t, strings.HasSuffix(tmpFile.Name(), ".txt"))
}

func TestCreateTempFile_DirectoryOutsideAllowedPaths(t *testing.T) {
	fs := afero.NewOsFs()

	// Try to create temp file in /home (not in allowedBaseDirs)
	tmpFile, err := CreateTempFile(fs, "/home", "test_*.json")
	assert.Error(t, err)
	assert.Nil(t, tmpFile)
	assert.Contains(t, err.Error(), "path not allowed")
}

func TestCreateTempFile_UniqueNames(t *testing.T) {
	fs := afero.NewOsFs()

	// Create multiple temp files to ensure they have unique names
	var tmpFiles []*os.File
	var fileNames []string

	for i := 0; i < 5; i++ {
		tmpFile, err := CreateTempFile(fs, "/tmp", "unique_test_*.txt")
		assert.NoError(t, err)
		assert.NotNil(t, tmpFile)
		tmpFiles = append(tmpFiles, tmpFile)
		fileNames = append(fileNames, tmpFile.Name())
	}

	// Clean up
	for _, tmpFile := range tmpFiles {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}

	// Verify all names are unique
	nameSet := make(map[string]bool)
	for _, name := range fileNames {
		assert.False(t, nameSet[name], "File name should be unique: %s", name)
		nameSet[name] = true
	}
}

func TestCreateTempFile_PatternWithoutWildcard(t *testing.T) {
	fs := afero.NewOsFs()

	tmpFile, err := CreateTempFile(fs, "/tmp", "no_wildcard")
	assert.NoError(t, err)
	assert.NotNil(t, tmpFile)
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Should still create a unique file even without * in pattern
	assert.Contains(t, tmpFile.Name(), "no_wildcard")
}

func TestCreateTempFile_CleanupOnError(t *testing.T) {
	fs := afero.NewOsFs()

	// This should fail due to directory outside allowed paths
	tmpFile, err := CreateTempFile(fs, "/home", "cleanup_test_*.txt")
	assert.Error(t, err)
	assert.Nil(t, tmpFile)

	// The temp file should have been cleaned up automatically
	// We can't easily test this directly, but the error indicates the validation failed
	assert.Contains(t, err.Error(), "path not allowed")
}

func TestCreateTempFile_WriteAndRead(t *testing.T) {
	fs := afero.NewOsFs()

	tmpFile, err := CreateTempFile(fs, "/tmp", "write_read_test_*.json")
	assert.NoError(t, err)
	assert.NotNil(t, tmpFile)

	filePath := tmpFile.Name()

	// Write to temp file
	testData := []byte(`{"test": "data", "number": 42}`)
	_, err = tmpFile.Write(testData)
	assert.NoError(t, err)
	tmpFile.Close()

	// Read from temp file using ReadFile
	readData, err := ReadFile(fs, filePath)
	assert.NoError(t, err)
	assert.Equal(t, testData, readData)

	// Cleanup
	err = os.Remove(filePath)
	assert.NoError(t, err)
}

// Integration Test Cases
func TestReadWriteFile_Integration(t *testing.T) {
	fs := afero.NewOsFs()
	testFilePath := "/tmp/read_write_integration_test.txt"
	testContent := []byte("Integration test content\nLine 2\nLine 3")

	defer os.Remove(testFilePath)

	t.Run("Write then read file", func(t *testing.T) {
		// Write file
		err := WriteFile(fs, testFilePath, testContent, 0644)
		assert.NoError(t, err)

		// Read file
		readContent, err := ReadFile(fs, testFilePath)
		assert.NoError(t, err)

		assert.Equal(t, testContent, readContent)
	})
}

func TestCreateTempFile_Integration(t *testing.T) {
	fs := afero.NewOsFs()

	t.Run("Create, write, read, and cleanup temp file", func(t *testing.T) {
		// Create temp file
		tmpFile, err := CreateTempFile(fs, "/tmp", "integration_test_*.json")
		assert.NoError(t, err)
		assert.NotNil(t, tmpFile)

		filePath := tmpFile.Name()

		// Write to temp file
		testData := []byte(`{"test": "data", "number": 42}`)
		_, err = tmpFile.Write(testData)
		assert.NoError(t, err)
		tmpFile.Close()

		// Read from temp file using ReadFile
		readData, err := ReadFile(fs, filePath)
		assert.NoError(t, err)
		assert.Equal(t, testData, readData)

		// Cleanup
		err = os.Remove(filePath)
		assert.NoError(t, err)
	})
}

func TestAllFunctions_WithMemoryFS(t *testing.T) {
	// Test with in-memory filesystem for allowed directories
	fs := afero.NewMemMapFs()

	t.Run("Memory FS operations in allowed directories", func(t *testing.T) {
		testContent := []byte("memory fs test content")
		filePath := "/tmp/memory_test.txt"

		// Write file
		err := WriteFile(fs, filePath, testContent, 0644)
		assert.NoError(t, err)

		// Read file
		readContent, err := ReadFile(fs, filePath)
		assert.NoError(t, err)
		assert.Equal(t, testContent, readContent)

		// Verify file exists
		exists, err := afero.Exists(fs, filePath)
		assert.NoError(t, err)
		assert.True(t, exists)
	})
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
	assert.Contains(t, err.Error(), "access to the directory is outside the allowed directories")
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
	assert.Contains(t, err.Error(), "path contains symlinks")
}

func TestMkdirAll_BaseDirectoryAllowed(t *testing.T) {
	fs := afero.NewMemMapFs()
	// Test that we can create the base directory itself
	err := MkdirAll(fs, "/tmp", 0755)
	assert.NoError(t, err)
	exists, err := afero.DirExists(fs, "/tmp")
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestIsFilePathAbsolute_DirValid(t *testing.T) {
	err := isFilePathAbsolute("/tmp/testdir")
	assert.NoError(t, err)
}

func TestIsFilePathAbsolute_DirInvalid(t *testing.T) {
	err := isFilePathAbsolute("/unauthorized/testdir")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "access to the path is outside the allowed directories")
}

func TestIsFilePathSymLink_DirNonExistent(t *testing.T) {
	// Should not error if dir does not exist
	err := isFilePathSymLink("/tmp/doesnotexist")
	assert.NoError(t, err)
}

func TestIsFilePathSymLink_DirIsSymlink(t *testing.T) {
	fs := afero.NewOsFs()
	targetDir := "/tmp/realtestdir2"
	symlinkDir := "/tmp/symlinkdir2"
	_ = fs.MkdirAll(targetDir, 0755)
	defer func() { _ = fs.RemoveAll(targetDir) }()
	_ = os.Symlink(targetDir, symlinkDir)
	defer func() { _ = fs.Remove(symlinkDir) }()
	err := isFilePathSymLink(symlinkDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path contains symlinks")
}

func TestIsDirPathAbsolute_ValidDir(t *testing.T) {
	err := isDirPathAbsolute("/tmp/testdir")
	assert.NoError(t, err)
}

func TestIsDirPathAbsolute_ValidBaseDir(t *testing.T) {
	// Test that base directories themselves are allowed (unlike isFilePathAbsolute)
	err := isDirPathAbsolute("/tmp")
	assert.NoError(t, err)
}

func TestIsDirPathAbsolute_InvalidDir(t *testing.T) {
	err := isDirPathAbsolute("/unauthorized/testdir")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "access to the directory is outside the allowed directories")
}

func TestIsDirPathAbsolute_RelativePath(t *testing.T) {
	err := isDirPathAbsolute("relative/path")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "access to the directory is outside the allowed directories")
}

func TestDifferenceBetweenFileAndDirPathValidation(t *testing.T) {
	// isFilePathAbsolute should reject base directories themselves
	err := isFilePathAbsolute("/tmp")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "access to the path is outside the allowed directories")

	// isDirPathAbsolute should allow base directories themselves
	err = isDirPathAbsolute("/tmp")
	assert.NoError(t, err)

	// Both should allow subdirectories
	err = isFilePathAbsolute("/tmp/testfile.txt")
	assert.NoError(t, err)

	err = isDirPathAbsolute("/tmp/testdir")
	assert.NoError(t, err)
}

// Additional tests for IsFileExist function coverage
func TestIsFileExist_WithMemMapFs(t *testing.T) {
	fs := afero.NewMemMapFs()
	testFile := "/tmp/test_exists.txt"

	// Test non-existent file
	exists := IsFileExist(fs, "/tmp/nonexistent.txt")
	assert.False(t, exists)

	// Create a test file and test it exists
	err := afero.WriteFile(fs, testFile, []byte("test content"), 0644)
	assert.NoError(t, err)
	exists = IsFileExist(fs, testFile)
	assert.True(t, exists)
}

// Additional tests for validateOpenedFile function coverage
func TestValidateOpenedFile_TOCTOU_Protection(t *testing.T) {
	fs := afero.NewOsFs()
	testFile := "/tmp/validate_toctou_test.txt"
	testContent := []byte("TOCTOU protection test")

	// Create a test file
	err := afero.WriteFile(fs, testFile, testContent, 0644)
	assert.NoError(t, err)
	defer os.Remove(testFile)

	// Open the file and validate - should pass
	file, err := fs.Open(testFile)
	assert.NoError(t, err)
	defer file.Close()

	err = validateOpenedFile(file, testFile)
	assert.NoError(t, err)
}

// Test for additional file permission scenarios
func TestFileOperations_AdditionalPermissions(t *testing.T) {
	fs := afero.NewOsFs()
	testFile := "/tmp/perm_test.txt"
	testContent := []byte("permission test content")

	// Test with read-only permission
	t.Run("ReadOnlyPermission", func(t *testing.T) {
		os.Remove(testFile) // Clean up first

		err := WriteFile(fs, testFile, testContent, 0400)
		assert.NoError(t, err)

		// Verify we can still read it
		readContent, err := ReadFile(fs, testFile)
		assert.NoError(t, err)
		assert.Equal(t, testContent, readContent)

		os.Remove(testFile)
	})
}

// Test for enhanced symlink detection in various scenarios
func TestSymlinkDetection_EnhancedScenarios(t *testing.T) {
	// Test DMI path handling - should not error even with symlinks
	dmiPath := "/sys/class/dmi/id/product_name"
	err := isFilePathSymLink(dmiPath)
	assert.NoError(t, err, "DMI paths should be allowed even with symlinks")

	// Test non-existent parent directory
	nonExistentPath := "/tmp/definitely_does_not_exist_12345/file.txt"
	err = isFilePathSymLink(nonExistentPath)
	assert.NoError(t, err, "Non-existent paths should not cause symlink errors")
}

// Test for error scenarios in TOCTOU-protected file operations
func TestTOCTOU_ErrorScenarios(t *testing.T) {
	fs := afero.NewOsFs()

	t.Run("SymlinkInReadFile", func(t *testing.T) {
		testFile := "/tmp/toctou_read_test.txt"
		symlinkPath := "/tmp/toctou_read_symlink.txt"

		// Create test file and symlink
		err := afero.WriteFile(fs, testFile, []byte("test"), 0644)
		assert.NoError(t, err)
		defer os.Remove(testFile)

		err = os.Symlink(testFile, symlinkPath)
		assert.NoError(t, err)
		defer os.Remove(symlinkPath)

		// Reading through symlink should fail
		_, err = ReadFile(fs, symlinkPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file path was changed via symlink")
	})

	t.Run("SymlinkInWriteFile", func(t *testing.T) {
		testFile := "/tmp/toctou_write_test.txt"
		symlinkPath := "/tmp/toctou_write_symlink.txt"
		testContent := []byte("write test")

		// Create test file and symlink
		err := afero.WriteFile(fs, testFile, []byte("original"), 0644)
		assert.NoError(t, err)
		defer os.Remove(testFile)

		err = os.Symlink(testFile, symlinkPath)
		assert.NoError(t, err)
		defer os.Remove(symlinkPath)

		// Writing through symlink should fail
		err = WriteFile(fs, symlinkPath, testContent, 0644)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file path was changed via symlink")
	})
}

// Test CreateTempFile with additional validation scenarios
func TestCreateTempFile_ValidationScenarios(t *testing.T) {
	fs := afero.NewOsFs()

	// Test successful creation in allowed directory
	tmpFile, err := CreateTempFile(fs, "/tmp", "enhanced_test_*.txt")
	assert.NoError(t, err)
	assert.NotNil(t, tmpFile)

	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Verify file was created with expected pattern
	assert.True(t, strings.HasPrefix(tmpFile.Name(), "/tmp/enhanced_test_"))
	assert.True(t, strings.HasSuffix(tmpFile.Name(), ".txt"))

	// Verify file can be written to and read from
	testData := []byte("temp file test data")
	_, err = tmpFile.Write(testData)
	assert.NoError(t, err)

	// Reset file position for reading
	_, err = tmpFile.Seek(0, 0)
	assert.NoError(t, err)

	readData := make([]byte, len(testData))
	_, err = tmpFile.Read(readData)
	assert.NoError(t, err)
	assert.Equal(t, testData, readData)
}
