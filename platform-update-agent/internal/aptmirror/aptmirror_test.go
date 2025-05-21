// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package aptmirror

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"reflect"
	"testing"

	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestNewController(t *testing.T) {
	controller := NewController()
	assert.Equal(t, reflect.ValueOf(UpdatePackages).Pointer(), reflect.ValueOf(controller.UpdatePackages).Pointer())
	assert.Equal(t, reflect.ValueOf(ConfigureOsAptRepo).Pointer(), reflect.ValueOf(controller.ConfigureOsAptRepo).Pointer())
	assert.Equal(t, reflect.ValueOf(ConfigureCustomAptRepos).Pointer(), reflect.ValueOf(controller.ConfigureCustomAptRepos).Pointer())
}

func TestAptMirror_CleanupCustomRepos_happyPath(t *testing.T) {
	aptSourceDirectory = "testdata/"
	aptController := AptController{
		CleanupCustomRepos: CleanupCustomRepos,
	}
	commandExecutor = utils.NewExecutor[exec.Cmd](testCmdCompletedSuccessfully, utils.ExecuteAndReadOutput)

	err := aptController.CleanupCustomRepos()

	assert.NoError(t, err)
}

func TestAptMirror_CleanupCustomRepos_shouldFailAfterInbmFailed(t *testing.T) {
	aptSourceDirectory = "testdata/"
	aptController := AptController{
		CleanupCustomRepos: CleanupCustomRepos,
	}
	commandExecutor = utils.NewExecutor[exec.Cmd](testCmdFailed, utils.ExecuteAndReadOutput)

	err := aptController.CleanupCustomRepos()

	assert.ErrorContains(t, err, "failed to remove the existing custom repo")
}

func TestAptMirror_ConfigureCustomAptRepo_shouldFailAsCustomRepoIsNotValid(t *testing.T) {
	customRepoWithMissingUrisAndSuites := []string{"Types: deb\nComponents: release\nSigned-By:"}
	aptController := AptController{
		ConfigureOsAptRepo:      nil,
		ConfigureCustomAptRepos: ConfigureCustomAptRepos,
		CleanupCustomRepos:      nil,
		UpdatePackages:          nil,
	}

	err := aptController.ConfigureCustomAptRepos(customRepoWithMissingUrisAndSuites)

	assert.ErrorContains(t, err, fmt.Sprintf("incomplete custom repo data - %v", customRepoWithMissingUrisAndSuites[0]))
}

func TestAptMirror_ConfigureCustomAptRepo_happyPath(t *testing.T) {
	customRepos := []string{"Types: deb\nURIs: https://test.com\nSuites: example\nComponents: release\nSigned-By:\npublic GPG key"}
	aptController := AptController{
		ConfigureOsAptRepo:      nil,
		ConfigureCustomAptRepos: ConfigureCustomAptRepos,
		CleanupCustomRepos:      nil,
		UpdatePackages:          nil,
	}

	commandExecutor = utils.NewExecutor[exec.Cmd](testCmdCompletedSuccessfully, utils.ExecuteAndReadOutput)
	err := aptController.ConfigureCustomAptRepos(customRepos)

	assert.NoError(t, err)
}

func TestAptMirror_ConfigureCustomAptRepo_shouldFailAfterInbmFailed(t *testing.T) {
	customRepos := []string{"Types: deb\nURIs: https://test.com\nSuites: example\nComponents: release\nSigned-By:\npublic GPG key"}
	aptController := AptController{
		ConfigureOsAptRepo:      nil,
		ConfigureCustomAptRepos: ConfigureCustomAptRepos,
		CleanupCustomRepos:      nil,
		UpdatePackages:          nil,
	}

	commandExecutor = utils.NewExecutor[exec.Cmd](testCmdFailed, utils.ExecuteAndReadOutput)
	err := aptController.ConfigureCustomAptRepos(customRepos)

	assert.ErrorContains(t, err, "failed to configure custom repo -")
}

func TestAptMirror_ConfigureDeprecatedRepos_shouldFailWhenNoFileAccess(t *testing.T) {
	socketPath := "/tmp/mysocket.sock"
	_ = os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	assert.NoError(t, err)
	defer listener.Close()
	customRepos := []string{"deb [signed-by=/tmp/key.gpg] http://new.apt.repo.com/ jammy-backports main universe multiverse restricted"}
	aptController := AptController{
		AptRepoFile: socketPath,
	}

	sut := aptController.ConfigureDeprecatedCustomAptRepos

	assert.ErrorContains(t, sut(customRepos), fmt.Sprintf("failed to write custom repos to %v file", socketPath))
}

func TestAptMirror_ConfigureDeprecatedRepos_shouldFailWhenNotSignedRepo(t *testing.T) {
	customRepos := []string{"deb http://new.apt.repo.com/ jammy-backports main universe multiverse restricted"}
	aptController := AptController{
		AptRepoFile: "/etc/default/grub.d/12345.list",
	}

	sut := aptController.ConfigureDeprecatedCustomAptRepos

	assert.ErrorContains(t, sut(customRepos), "invalid custom repo - missing signed key")
}

func TestAptMirror_ConfigureDeprecatedRepos_shouldFailAfterSymlinkIsInputted(t *testing.T) {
	customRepos := []string{"deb [signed-by=/tmp/key.gpg] http://new.apt.repo.com/ jammy-backports main"}
	symLinkPath := "/tmp/symlink_temp.txt"
	aptController := AptController{
		AptRepoFile: symLinkPath,
	}
	file, _ := os.CreateTemp("", "apt_repo_temp.txt")
	defer file.Close()
	err := os.Symlink(file.Name(), symLinkPath)
	assert.Nil(t, err)
	defer os.Remove(symLinkPath)
	defer os.Remove(file.Name())

	sut := aptController.ConfigureDeprecatedCustomAptRepos

	assert.ErrorContains(t, sut(customRepos), fmt.Sprintf("%v is a symlink", symLinkPath))
}

func TestAptMirror_ConfigureDeprecatedRepos_happyPath(t *testing.T) {
	customRepos := []string{"deb [signed-by=/tmp/key.gpg] http://new.apt.repo.com/ jammy-backports main"}
	file, _ := os.CreateTemp("", "apt_repo_temp.txt")
	defer file.Close()
	defer os.Remove(file.Name())
	aptController := AptController{
		AptRepoFile: file.Name(),
	}

	sut := aptController.ConfigureDeprecatedCustomAptRepos

	assert.NoError(t, sut(customRepos))

	fileContent, err := os.ReadFile(file.Name())
	assert.NoError(t, err)
	assert.Equal(t, customRepos[0], string(fileContent))
}

func TestAptMirror_IsDeprecatedFormat_shouldBeTrueWhenSingleLineFormat(t *testing.T) {
	customRepos := []string{"deb [signed-by=/tmp/key.gpg] http://new.apt.repo.com/ jammy-backports main"}
	aptController := AptController{}

	sut := aptController.IsDeprecatedFormat

	assert.True(t, sut(customRepos))
}

func TestAptMirror_IsDeprecatedFormat_shouldBeFalseWhenMultiLineFormat(t *testing.T) {
	customRepos := []string{"Types: deb\nURIs: https://test.com\nSuites: example"}
	aptController := AptController{}

	sut := aptController.IsDeprecatedFormat

	assert.False(t, sut(customRepos))
}

func TestAptMirror_ConfigureForwardProxy_happyPath(t *testing.T) {
	repoUri := "https://target-file-server.com"
	forwardProxyUri := "http://localhost"
	customRepos := []string{fmt.Sprintf("#ReleaseService\nTypes: deb\nURIs: %v\nSuites: example\nComponents: release\nSigned-By:\npublic GPG key", repoUri)}
	aptController := AptController{}
	rpConfigFile, err := os.CreateTemp("/tmp", "forward-proxy.conf")
	assert.NoError(t, err)
	forwardProxyConfPath = rpConfigFile.Name()
	defer rpConfigFile.Close()
	rpContent := "localhost:8080 {\nbind 127.0.0.1\ntls cert.pem key.pem\nreverse_proxy https://to-be-replaced.com {\n    header_up Authorization \"Bearer eyJ0eXA1w\"\n    header_up Host {upstream_hostport}\n }}"
	err = os.WriteFile(forwardProxyConfPath, []byte(rpContent), 0600)
	assert.NoError(t, err)

	commandExecutor = utils.NewExecutor[exec.Cmd](testCmdCompletedSuccessfully, utils.ExecuteAndReadOutput)

	sut := aptController.ConfigureForwardProxy
	assert.NoError(t, sut(customRepos))

	updatedRpConfig, err := os.ReadFile(forwardProxyConfPath)
	assert.NoError(t, err)
	assert.Contains(t, string(updatedRpConfig), repoUri)
	assert.Contains(t, customRepos[0], forwardProxyUri)
}

func TestAptMirror_ConfigureForwardProxy_shouldFailWhenUriIsMissing(t *testing.T) {
	repoUri := ""
	customRepos := []string{fmt.Sprintf("#ReleaseService\nTypes: deb\nURIs: %v\nSuites: example\nComponents: release\nSigned-By:\npublic GPG key", repoUri)}
	aptController := AptController{}

	sut := aptController.ConfigureForwardProxy
	assert.ErrorContains(t, sut(customRepos), "failed to validate source URL")
}

func TestAptMirror_ConfigureForwardProxy_shouldFailWhenUriIsInvalid(t *testing.T) {
	aptController := AptController{}

	uriMissingProtocol := "target-file-server.com"
	customRepos := []string{fmt.Sprintf("#ReleaseService\nTypes: deb\nURIs: %v\nSuites: example\nComponents: release\nSigned-By:\npublic GPG key", uriMissingProtocol)}
	sut := aptController.ConfigureForwardProxy
	assert.ErrorContains(t, sut(customRepos), fmt.Sprintf("failed to validate source URL ['%v']", uriMissingProtocol))

	uriBadProtocolSeparator := "https//target-file-server.com"
	customRepos = []string{fmt.Sprintf("#ReleaseService\nTypes: deb\nURIs: %v\nSuites: example\nComponents: release\nSigned-By:\npublic GPG key", uriBadProtocolSeparator)}
	sut = aptController.ConfigureForwardProxy
	assert.ErrorContains(t, sut(customRepos), fmt.Sprintf("failed to validate source URL ['%v']", uriBadProtocolSeparator))

	multipleUris := "https://rs-1.com http://rs-2.com"
	customRepos = []string{fmt.Sprintf("#ReleaseService\nTypes: deb\nURIs: %v\nSuites: example\nComponents: release\nSigned-By:\npublic GPG key", multipleUris)}
	sut = aptController.ConfigureForwardProxy
	assert.ErrorContains(t, sut(customRepos), fmt.Sprintf("release service doesn't support multiple releaseServiceUrls - %v", multipleUris))
}

func TestAptMirror_ConfigureForwardProxy_shouldFailWhenRpConfigIsEmpty(t *testing.T) {
	repoUri := "https://target-file-server.com"
	customRepos := []string{fmt.Sprintf("#ReleaseService\nTypes: deb\nURIs: %v\nSuites: example\nComponents: release\nSigned-By:\npublic GPG key", repoUri)}
	aptController := AptController{}
	rpConfigFile, err := os.CreateTemp("/tmp", "forward-proxy.conf")
	assert.NoError(t, err)
	forwardProxyConfPath = rpConfigFile.Name()
	defer rpConfigFile.Close()

	sut := aptController.ConfigureForwardProxy
	assert.ErrorContains(t, sut(customRepos), "failed to update forward proxy config")
}

func TestAptMirror_readAndReplaceToRPUrl_shouldFailAsUriKeyIsMissing(t *testing.T) {
	repoMissingUriKey := "#ReleaseService\nTypes: deb\nSuites: example\nComponents: release\nSigned-By:\npublic GPG key"
	forwardProxyUrl = "http://localhost"

	updatedRepo, releaseServiceUrl, err := readAndReplaceToRPUrl(repoMissingUriKey)

	assert.ErrorContains(t, err, "URIs key not found")
	assert.Emptyf(t, updatedRepo, "updatedRepo is not empty")
	assert.Emptyf(t, releaseServiceUrl, "releaseServiceUrl is not empty")
}

func TestAptMirror_readAndReplaceToRPUrl_shouldFailAsNewlineIsMissing(t *testing.T) {
	repoMissingUriKey := "Types: deb\nURIs: https://test.com"
	forwardProxyUrl = "http://localhost"

	updatedRepo, releaseServiceUrl, err := readAndReplaceToRPUrl(repoMissingUriKey)

	assert.ErrorContains(t, err, "URIs end line not found")
	assert.Emptyf(t, updatedRepo, "updatedRepo is not empty")
	assert.Emptyf(t, releaseServiceUrl, "releaseServiceUrl is not empty")
}

func TestAptMirror_updateForwardProxyConfig_shouldFailAfterSymlinkIsInputted(t *testing.T) {
	symLinkPath := "/tmp/symlink_temp.cfg"
	releaseServiceUrl := ""
	file, _ := os.CreateTemp("", "fp_config_temp.cfg")
	defer file.Close()
	err := os.Symlink(file.Name(), symLinkPath)
	assert.Nil(t, err)
	defer os.Remove(symLinkPath)
	defer os.Remove(file.Name())

	err = updateForwardProxyConfig(symLinkPath, releaseServiceUrl)

	assert.ErrorContains(t, err, fmt.Sprintf("%v is a symlink", symLinkPath))
}

func TestAptMirror_updateForwardProxyConfig_shouldFailWhenNoFileAccess(t *testing.T) {
	socketPath := "/tmp/mysocket.sock"
	_ = os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	assert.NoError(t, err)
	defer listener.Close()
	releaseServiceUrl := ""

	err = updateForwardProxyConfig(socketPath, releaseServiceUrl)

	assert.ErrorContains(t, err, "failed to read forward proxy config")
}

func TestAptMirror_updateForwardProxyConfig_shouldFailAsProxyPassIsMissing(t *testing.T) {
	forwardProxyConfPath = "/tmp/incorrect-file.cfg"
	releaseServiceUrl := ""
	forwardProxyConfigFile, err := os.CreateTemp("/tmp", "forward-proxy.conf")
	assert.NoError(t, err)
	forwardProxyConfPath = forwardProxyConfigFile.Name()
	defer forwardProxyConfigFile.Close()
	forwardProxyFileContent := "localhost:8080 {\nbind 127.0.0.1\ntls cert.pem key.pem\n {\n    header_up Authorization \"Bearer eyJ0eXA1w\"\n    header_up Host {upstream_hostport}\n }}"
	err = os.WriteFile(forwardProxyConfPath, []byte(forwardProxyFileContent), 0600)
	assert.NoError(t, err)

	err = updateForwardProxyConfig(forwardProxyConfPath, releaseServiceUrl)

	assert.ErrorContains(t, err, "not found reverse_proxy key")
}

func TestAptMirror_updateForwardProxyConfig_shouldFailAsProxyPassEndLineIsMissing(t *testing.T) {
	forwardProxyConfPath = "/tmp/incorrect-file.cfg"
	releaseServiceUrl := ""
	forwardProxyConfigFile, err := os.CreateTemp("/tmp", "forward-proxy.conf")
	assert.NoError(t, err)
	forwardProxyConfPath = forwardProxyConfigFile.Name()
	defer forwardProxyConfigFile.Close()
	forwardProxyFileContent := "localhost:8080 {\nbind 127.0.0.1\ntls cert.pem key.pem\nreverse_proxy https://to-be-replaced.com\n    header_up Authorization \"Bearer eyJ0eXA1w\"\n    header_up Host {upstream_hostport}\n }}"
	err = os.WriteFile(forwardProxyConfPath, []byte(forwardProxyFileContent), 0600)
	assert.NoError(t, err)

	err = updateForwardProxyConfig(forwardProxyConfPath, releaseServiceUrl)

	assert.ErrorContains(t, err, "not found reverse_proxy endline key")
}

func TestAptMirror_splitURL_happyPath(t *testing.T) {
	rootRsURL := "https://test-url.com:60443"
	remainingRsURL := "/remaning/URL/path"
	fullRsURL := rootRsURL + remainingRsURL

	rootURL, remainingURL, err := splitURL(fullRsURL)

	assert.NoError(t, err)
	assert.Equal(t, rootURL, rootRsURL)
	assert.Equal(t, remainingURL, remainingRsURL)
}

func TestAptMirror_splitURL_happyPathOnlyRootUrl(t *testing.T) {
	rootRsURL := "https://test-url.com:60443"
	remainingRsURL := ""
	fullRsURL := rootRsURL + remainingRsURL

	rootURL, remainingURL, err := splitURL(fullRsURL)

	assert.NoError(t, err)
	assert.Equal(t, rootURL, rootRsURL)
	assert.Empty(t, remainingURL)
}

func TestAptMirror_splitURL_shouldFailWhenInvalidUrlSchema(t *testing.T) {
	rootRsURL := "ht!!tps://test-url.com:60443"
	remainingRsURL := "/remaning/URL/path"
	fullRsURL := rootRsURL + remainingRsURL

	rootURL, remainingURL, err := splitURL(fullRsURL)

	assert.ErrorContains(t, err, "first path segment in URL cannot contain colon")
	assert.Empty(t, rootURL)
	assert.Empty(t, remainingURL)
}

func TestAptMirror_ConfigureOsAptRepo_shouldPassAlthoughOsRepoUrlIsEmpty(t *testing.T) {
	osRepoURL := ""

	commandExecutor = utils.NewExecutor[any](nil, nil)
	err := ConfigureOsAptRepo(osRepoURL)

	assert.NoError(t, err)
}

func TestAptMirror_ConfigureOsAptRepo_shouldFailAfterInbmFailed(t *testing.T) {
	osRepoURL := "http://linux-ftp.fi.intel.com/pub/mirrors/ubuntu"

	commandExecutor = utils.NewExecutor[exec.Cmd](testCmdFailed, utils.ExecuteAndReadOutput)
	err := ConfigureOsAptRepo(osRepoURL)

	assert.Error(t, err)
}

func TestAptMirror_ConfigureOsAptRepo_shouldSuccessfullyConfigureOsAptRepo(t *testing.T) {
	osRepoURL := "http://linux-ftp.fi.intel.com/pub/mirrors/ubuntu"
	sourcesListTmplContent := "deb <repoURL> jammy multiverse restricted main universe"
	file, err := os.CreateTemp("", "sources.list")
	assert.NoError(t, err)
	aptSourcesListTemplatePath = file.Name()
	defer file.Close()
	defer os.Remove(file.Name())
	err = os.WriteFile(file.Name(), []byte(sourcesListTmplContent), 0600)
	assert.NoError(t, err)

	commandExecutor = utils.NewExecutor[exec.Cmd](testCmdCompletedSuccessfully, utils.ExecuteAndReadOutput)
	err = ConfigureOsAptRepo(osRepoURL)

	assert.NoError(t, err)
}

func TestAptMirror_ConfigureOsAptRepo_shouldFailAfterSymlinkIsInputted(t *testing.T) {
	osRepoURL := "http://linux-ftp.fi.intel.com/pub/mirrors/ubuntu"
	symLinkPath := "/tmp/symlink_temp.cfg"
	file, err := os.CreateTemp("", "sources.list")
	assert.NoError(t, err)
	aptSourcesListTemplatePath = symLinkPath
	defer file.Close()
	err = os.Symlink(file.Name(), symLinkPath)
	assert.NoError(t, err)
	defer os.Remove(file.Name())
	defer os.Remove(symLinkPath)

	err = ConfigureOsAptRepo(osRepoURL)

	assert.ErrorContains(t, err, fmt.Sprintf("%v is a symlink", symLinkPath))
}

func TestAptMirror_ConfigureOsAptRepo_shouldFailWhenNoFileAccess(t *testing.T) {
	osRepoURL := "http://linux-ftp.fi.intel.com/pub/mirrors/ubuntu"
	socketPath := "/tmp/mysocket.sock"
	_ = os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	assert.NoError(t, err)
	defer listener.Close()
	aptSourcesListTemplatePath = socketPath

	err = ConfigureOsAptRepo(osRepoURL)

	assert.ErrorContains(t, err, "failed to read apt sources list template")
}

func TestAptMirror_UpdatePackages_shouldUpdatePackages(t *testing.T) {
	commandExecutor = utils.NewExecutor[exec.Cmd](testCmdCompletedSuccessfully, utils.ExecuteAndReadOutput)
	err := UpdatePackages()

	assert.NoError(t, err)
}

func TestAptMirror_UpdatePackages_updatePackagesFailed(t *testing.T) {
	commandExecutor = utils.NewExecutor[exec.Cmd](testCmdFailed, utils.ExecuteAndReadOutput)
	err := UpdatePackages()

	assert.Error(t, err)
}

func testCmdCompletedSuccessfully(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestCmdCompletedSuccessfully", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func testCmdFailed(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestCmdFailed", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_TEST_PROCESS=1"}
	return cmd
}

func TestCmdCompletedSuccessfully(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}

	fmt.Fprintf(os.Stdout, "successfully completed cmd")
	os.Exit(0)
}

func TestCmdFailed(t *testing.T) {
	if os.Getenv("GO_TEST_PROCESS") != "1" {
		return
	}

	fmt.Fprintf(os.Stderr, "process failed")
	os.Exit(1)
}
