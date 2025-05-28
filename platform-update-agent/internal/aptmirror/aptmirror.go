// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package aptmirror

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/logger"
	"github.com/open-edge-platform/edge-node-agents/platform-update-agent/internal/utils"
)

// these need to be variables so that they can be overridden in tests
var (
	aptSourceDirectory         = "/etc/apt/sources.list.d/"
	forwardProxyConfPath       = "/etc/caddy/pua.caddy"
	forwardProxyUrl            = "http://localhost:60444"
	aptSourcesListTemplatePath = "/etc/edge-node/node/confs/apt.sources.list.template"
)

const (
	AptRepoPath                           string = "/etc/apt/sources.list.d/pua.list"
	ERR_INVALID_SIGNATURE                 string = "The following signatures were invalid"
	AptUpdateCommandStr                          = "sudo apt update"
	InbcSourceOsUpdateCommandStr                 = "sudo inbc source os update --sources"
	InbcSourceApplicationAddCommandStr           = "sudo inbc source application add"
	InbcSourceApplicationRemoveCommandStr        = "sudo inbc source application remove --filename"
	CaddyReloadCommandStr                        = "sudo systemctl reload caddy"
)

var (
	log             = logger.Logger()
	commandExecutor = utils.NewExecutor(exec.Command, utils.ExecuteAndReadOutput)
)

// toCommandSlice splits the various space-separated arguments in a command string into a slice of strings
func toCommandSlice(commandStr string) []string {
	return strings.Split(commandStr, " ")
}

func CleanupCustomRepos() error {
	dir, err := os.ReadDir(aptSourceDirectory)
	if err != nil {
		return fmt.Errorf("failed to read the existing custom repos - %v", err)
	}

	for _, file := range dir {
		if file.IsDir() {
			continue
		}
		_, err = commandExecutor.Execute(append(toCommandSlice(InbcSourceApplicationRemoveCommandStr), file.Name()))
		if err != nil {
			return fmt.Errorf("failed to remove the existing custom repo - %v", err)
		}
	}
	return nil
}

func ConfigureCustomAptRepos(customRepos []string) error {

	for i, repo := range customRepos {
		if !isCustomRepoValid(repo) {
			return fmt.Errorf("incomplete custom repo data - %v", repo)
		}

		inbcSourceApplicationAddCommand := append(toCommandSlice(InbcSourceApplicationAddCommandStr), "--filename", fmt.Sprintf("pua-%v.sources", i), "--sources")
		inbcSourceApplicationAddCommand = append(inbcSourceApplicationAddCommand, strings.Split(repo, "\n")...)

		_, err := commandExecutor.Execute(inbcSourceApplicationAddCommand)
		if err != nil {
			return fmt.Errorf("failed to configure custom repo - %v", err)
		}
	}
	log.Infof("Added new external apt repositories")
	return nil
}

func isCustomRepoValid(repo string) bool {
	if strings.Contains(repo, "Types:") && strings.Contains(repo, "URIs:") && strings.Contains(repo, "Suites:") &&
		strings.Contains(repo, "Components:") && strings.Contains(repo, "Signed-By:") {
		return true
	}
	return false
}

func ConfigureOsAptRepo(osRepoURL string) error {
	if len(osRepoURL) == 0 {
		log.Infof("OS repo URL is empty. Skip source apt repo configuration.")
		return nil
	}

	err := utils.IsSymlink(aptSourcesListTemplatePath)
	if err != nil {
		return err
	}

	aptSourcesListTmpl, err := os.ReadFile(aptSourcesListTemplatePath)
	if err != nil {
		return fmt.Errorf("failed to read apt sources list template - %v", err)
	}
	aptSourcesList := strings.ReplaceAll(string(aptSourcesListTmpl), "<repoURL>", osRepoURL)

	_, err = commandExecutor.Execute(append(toCommandSlice(InbcSourceOsUpdateCommandStr), strings.Split(aptSourcesList, "\n")...))
	if err != nil {
		return err
	}
	return nil
}

func UpdatePackages() error {
	out, err := commandExecutor.Execute(toCommandSlice(AptUpdateCommandStr))
	if err != nil {
		return fmt.Errorf("failed to execute shell command - %v", err)
	}

	if strings.Contains(string(out), ERR_INVALID_SIGNATURE) {
		return fmt.Errorf("failed to verify signature - %v", string(out))
	}

	return nil
}

// APT supports two formats of source list files:
// One-Line-Style Format - deprecated due to the inability to add signed key contents as a string
// Multi-Line DEB822 Source Format
func (k *AptController) IsDeprecatedFormat(customRepos []string) bool {
	newlineChar := "\n"

	for _, repo := range customRepos {
		if !strings.Contains(repo, newlineChar) {
			return true
		}
	}
	return false
}

func (k *AptController) ConfigureDeprecatedCustomAptRepos(customRepos []string) error {

	for _, repo := range customRepos {
		if !strings.Contains(repo, "signed-by=") {
			return fmt.Errorf("invalid custom repo - missing signed key")
		}
	}

	aptSourceList := []byte(strings.Join(customRepos[:], "\n"))

	err := utils.IsSymlink(k.AptRepoFile)
	if err != nil {
		return err
	}

	err = os.WriteFile(k.AptRepoFile, aptSourceList, 0600)
	if err != nil {
		return fmt.Errorf("failed to write custom repos to %v file - %v", k.AptRepoFile, err)
	}
	log.Infof("Added new external APT repositories %v", customRepos)
	return nil
}

func (k *AptController) ConfigureForwardProxy(customRepos []string) error {
	releaseServiceTag := "#ReleaseService"

	for i, repo := range customRepos {
		if !strings.Contains(repo, releaseServiceTag) {
			continue
		}

		updatedRepo, releaseServiceUrls, err := readAndReplaceToRPUrl(repo)
		if err != nil {
			return err
		}

		customRepos[i] = updatedRepo

		err = updateForwardProxyConfig(forwardProxyConfPath, releaseServiceUrls)
		if err != nil {
			return fmt.Errorf("failed to update forward proxy config - %v", err)
		}

		_, err = commandExecutor.Execute(toCommandSlice(CaddyReloadCommandStr))
		if err != nil {
			return fmt.Errorf("failed to reload forward proxy config - %v", err)
		}
		break
	}
	return nil
}

func readAndReplaceToRPUrl(repo string) (string, string, error) {
	urisIndex := strings.Index(repo, "URIs:")
	if urisIndex == -1 {
		return "", "", fmt.Errorf("URIs key not found")
	}

	urisEndLineIndex := strings.Index(repo[urisIndex:], "\n") + urisIndex
	if urisEndLineIndex-urisIndex == -1 {
		return "", "", fmt.Errorf("URIs end line not found")
	}

	urisValue := repo[urisIndex:urisEndLineIndex]
	releaseServiceUrls, _ := strings.CutPrefix(urisValue, "URIs:")
	releaseServiceUrls = strings.TrimSpace(releaseServiceUrls)

	if len(strings.Split(releaseServiceUrls, " ")) > 1 {
		return "", "", fmt.Errorf("release service doesn't support multiple releaseServiceUrls - %v", releaseServiceUrls)
	}

	rootReleaseServiceURL, remainingURL, err := splitURL(releaseServiceUrls)
	if err != nil {
		return "", "", fmt.Errorf("error parsing URL: %v", err)
	}

	updatedRepo := repo[:urisIndex] + fmt.Sprintf("URIs: %v%v", forwardProxyUrl, remainingURL) + repo[urisEndLineIndex:]

	return updatedRepo, rootReleaseServiceURL, nil
}

func splitURL(fullURL string) (string, string, error) {
	parsedURL, err := url.Parse(fullURL)
	if err != nil {
		return "", "", err
	}

	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return "", "", fmt.Errorf("failed to validate source URL ['%v']", fullURL)
	}

	rootURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)
	remainingURL := strings.TrimPrefix(fullURL, rootURL)

	return rootURL, remainingURL, nil
}

func updateForwardProxyConfig(forwardProxyConfPath, releaseServiceUrl string) error {
	err := utils.IsSymlink(forwardProxyConfPath)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(forwardProxyConfPath)
	if err != nil {
		return fmt.Errorf("failed to read forward proxy config - %v", err)
	}

	if len(content) == 0 {
		return fmt.Errorf("no content found in forward proxy config file")
	}

	rpConfig := string(content)

	proxyPassIndex := strings.Index(rpConfig, "reverse_proxy")
	if proxyPassIndex == -1 {
		return fmt.Errorf("not found reverse_proxy key")
	}

	proxyPassEndLineIndex := strings.Index(rpConfig[proxyPassIndex:], "{\n") + proxyPassIndex
	if proxyPassEndLineIndex-proxyPassIndex == -1 {
		return fmt.Errorf("not found reverse_proxy endline key")
	}

	rpConfig = rpConfig[:proxyPassIndex] + fmt.Sprintf("reverse_proxy %v ", releaseServiceUrl) + rpConfig[proxyPassEndLineIndex:]

	err = os.WriteFile(forwardProxyConfPath, []byte(rpConfig), 0600)
	if err != nil {
		return fmt.Errorf("failed to write forward proxy config to %v file - %v", forwardProxyConfPath, err)
	}

	return nil
}

type AptController struct {
	ConfigureOsAptRepo      func(osRepoURL string) error
	ConfigureCustomAptRepos func(customRepos []string) error
	CleanupCustomRepos      func() error
	UpdatePackages          func() error
	AptRepoFile             string
}

func NewController() *AptController {
	return &AptController{
		ConfigureOsAptRepo:      ConfigureOsAptRepo,
		ConfigureCustomAptRepos: ConfigureCustomAptRepos,
		CleanupCustomRepos:      CleanupCustomRepos,
		UpdatePackages:          UpdatePackages,
		AptRepoFile:             AptRepoPath,
	}
}
