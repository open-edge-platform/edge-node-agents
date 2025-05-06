package utils

import (
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestCopyFile_ValidFile(t *testing.T) {
	fs := afero.NewOsFs()
	srcPath := "/tmp/source.txt"
	destPath := "/tmp/destination.txt"

	// Create a source file
	err := afero.WriteFile(fs, srcPath, []byte("test content"), 0644)
	assert.NoError(t, err)

	// Call the CopyFile function
	err = CopyFile(fs, srcPath, destPath)
	assert.NoError(t, err)

	// Verify the destination file exists and has the correct content
	content, err := afero.ReadFile(fs, destPath)
	assert.NoError(t, err)
	assert.Equal(t, "test content", string(content))

	// Clean up
	_ = fs.Remove(srcPath)
	_ = fs.Remove(destPath)
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

	// Call the CopyFile function
	err = CopyFile(fs, srcPath, destPath)
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

	// Create a symlink pointing to the real file
	err = os.Symlink(targetPath, symlinkPath)
	assert.NoError(t, err)
	defer os.Remove(symlinkPath)

	// Call the CopyFile function with the symlink as the source
	err = CopyFile(fs, symlinkPath, destPath)
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

	// Call the CopyFile function
	err = CopyFile(fs, srcPath, destPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "outside the allowed directories")
}

func TestOpenFile_ValidFile(t *testing.T) {
	fs := afero.NewOsFs() // Use an in-memory filesystem
	filePath := "/tmp/testfile.txt"

	// Create a valid file in the allowed directory
	err := afero.WriteFile(fs, filePath, []byte("test content"), 0644)
	assert.NoError(t, err)

	// Call the Open function
	file, err := OpenFile(fs, filePath, os.O_RDWR|os.O_CREATE, 0644)
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
	assert.NoError(t, err)

	// Call the Open function
	file, err := OpenFile(fs, filePath, os.O_RDWR|os.O_CREATE, 0644)
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
	assert.NoError(t, err)
	err = os.Symlink(targetPath, symlinkPath) // Use os.Symlink for creating symlinks
	assert.NoError(t, err)

	// Call the Open function with the symlink
	file, err := OpenFile(fs, symlinkPath, os.O_RDWR|os.O_CREATE, 0644)
	assert.Error(t, err)
	assert.Nil(t, file)
	assert.Contains(t, err.Error(), "file is a symlink")

	// Clean up
	_ = fs.Remove(targetPath)
	_ = fs.Remove(symlinkPath)
}

func TestOpen_ValidFile(t *testing.T) {
	fs := afero.NewOsFs() // Use an in-memory filesystem
	filePath := "/tmp/testfile.txt"

	// Create a valid file in the allowed directory
	err := afero.WriteFile(fs, filePath, []byte("test content"), 0644)
	assert.NoError(t, err)

	// Call the Open function
	file, err := Open(fs, filePath)
	assert.NoError(t, err)
	assert.NotNil(t, file)

	// Clean up
	_ = fs.Remove(filePath)
}

func TestOpen_FileOutsideAllowedDirs(t *testing.T) {
	fs := afero.NewMemMapFs() // Use an in-memory filesystem
	filePath := "/unauthorized/testfile.txt"

	// Create a file outside the allowed directories
	err := afero.WriteFile(fs, filePath, []byte("test content"), 0644)
	assert.NoError(t, err)

	// Call the Open function
	file, err := Open(fs, filePath)
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
	assert.NoError(t, err)
	err = os.Symlink(targetPath, symlinkPath) // Use os.Symlink for creating symlinks
	assert.NoError(t, err)

	// Call the Open function with the symlink
	file, err := Open(fs, symlinkPath)
	assert.Error(t, err)
	assert.Nil(t, file)
	assert.Contains(t, err.Error(), "file is a symlink")

	// Clean up
	_ = fs.Remove(targetPath)
	_ = fs.Remove(symlinkPath)
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
	assert.NoError(t, err)

	// Call the isFilePathSymLink function
	err = isFilePathSymLink(filePath)
	assert.NoError(t, err)

	// Clean up
	_ = fs.Remove(filePath)
}

func TestIsFilePathSymLink_IsSymlinkFile(t *testing.T) {
	fs := afero.NewOsFs() // Use the real filesystem
	targetPath := "/tmp/realfile.txt"
	symlinkPath := "/tmp/symlink.txt"

	// Create a real file
	err := afero.WriteFile(fs, targetPath, []byte("test content"), 0644)
	assert.NoError(t, err)

	// Create a symlink pointing to the real file
	err = os.Symlink(targetPath, symlinkPath) // Use os.Symlink for creating symlinks
	assert.NoError(t, err)

	// Verify the symlink
	fileInfo, err := os.Lstat(symlinkPath)
	assert.NoError(t, err)
	assert.True(t, fileInfo.Mode()&os.ModeSymlink != 0, "Expected symlink, but got mode: %v", fileInfo.Mode())

	// Call the isFilePathSymLink function with the symlink
	err = isFilePathSymLink(symlinkPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file is a symlink")

	// Clean up
	_ = fs.Remove(targetPath)
	_ = fs.Remove(symlinkPath)
}
