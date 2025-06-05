// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/open-edge-platform/edge-node-agents/reporting-agent/config"
	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/model"
	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/utils"
)

const (
	labelIntelAppName    = "com.intel.edgeplatform.application.name"
	labelIntelAppVersion = "com.intel.edgeplatform.application.version"
	labelK8sAppName      = "app.kubernetes.io/name"
	labelK8sAppVersion   = "app.kubernetes.io/version"
	labelK8sAppPartOf    = "app.kubernetes.io/part-of"
	labelHelmChart       = "helm.sh/chart"
)

type kube struct {
	kubectlPath    string
	kubectlArgs    []string
	kubeconfigPath string
	provider       string
}

// GetKubernetesData retrieves Kubernetes data using the provided executor and configuration.
func GetKubernetesData(executor utils.CmdExecutor, k8sCfg config.K8sConfig) (model.Kubernetes, error) {
	k8sData := model.Kubernetes{Applications: []model.KubernetesApplication{}}

	kube, err := getKube(k8sCfg)
	if err != nil {
		return k8sData, fmt.Errorf("failed to get Kubernetes config: %w", err)
	}

	k8sData.Provider = kube.provider

	versionArgs := append(kube.kubectlArgs, "--kubeconfig", kube.kubeconfigPath, "version", "-o", "json")
	serverVersionBytes, err := utils.ReadFromCommand(executor, kube.kubectlPath, versionArgs...)
	if err != nil {
		return k8sData, fmt.Errorf("failed to get Kubernetes server version: %w", err)
	}
	k8sData.ServerVersion = parseServerVersion(serverVersionBytes)

	appArgs := append(kube.kubectlArgs, "--kubeconfig", kube.kubeconfigPath, "get", "deployments,statefulsets,daemonsets", "-A", "-o", "json")
	applicationsBytes, err := utils.ReadFromCommand(executor, kube.kubectlPath, appArgs...)
	if err != nil {
		return k8sData, fmt.Errorf("failed to get Kubernetes resources: %w", err)
	}

	k8sData.Applications, err = parseApplications(applicationsBytes)
	if err != nil {
		return k8sData, fmt.Errorf("failed to parse Kubernetes applications: %w", err)
	}

	return k8sData, nil
}

func getKube(cfg config.K8sConfig) (*kube, error) {
	kubeCandidates := []struct {
		kubectlPath    string
		kubeConfigPath string
		provider       string
	}{
		{cfg.K3sKubectlPath, cfg.K3sKubeConfigPath, "k3s"},
		{cfg.Rke2KubectlPath, cfg.Rke2KubeConfigPath, "rke2"},
	}

	for _, c := range kubeCandidates {
		kubectlPath, kubectlArgs := splitKubectlPath(c.kubectlPath)
		if kubectlPath != "" && fileExists(kubectlPath) && fileExists(c.kubeConfigPath) {
			return &kube{
				kubectlPath:    kubectlPath,
				kubectlArgs:    kubectlArgs,
				kubeconfigPath: c.kubeConfigPath,
				provider:       c.provider,
			}, nil
		}
	}

	return nil, errors.New("no valid kubeconfig and kubectl files found")
}

// Split kubectl path and optional argument.
func splitKubectlPath(path string) (string, []string) {
	parts := strings.Fields(path)
	if len(parts) == 0 {
		return "", nil
	}
	return parts[0], parts[1:]
}

func parseServerVersion(jsonData []byte) string {
	var versionInfo struct {
		ServerVersion struct {
			GitVersion string `json:"gitVersion"`
		} `json:"serverVersion"`
	}
	if err := json.Unmarshal(jsonData, &versionInfo); err != nil {
		return ""
	}
	return versionInfo.ServerVersion.GitVersion
}

// parseApplications parses the JSON output of `kubectl get` and filters applications with specific labels and namespaces.
func parseApplications(jsonData []byte) ([]model.KubernetesApplication, error) {
	applications := []model.KubernetesApplication{}
	var resources struct {
		Items []struct {
			Metadata struct {
				Namespace string            `json:"namespace"`
				Labels    map[string]string `json:"labels"`
			} `json:"metadata"`
		} `json:"items"`
	}

	err := json.Unmarshal(jsonData, &resources)
	if err != nil {
		return applications, fmt.Errorf("failed to unmarshal Kubernetes resources data: %w", err)
	}

	uniqueApplications := map[string]struct{}{}
	for _, item := range resources.Items {
		namespace := item.Metadata.Namespace
		if _, excluded := excludedNamespaces[namespace]; excluded {
			continue
		}

		labels := item.Metadata.Labels
		if labels == nil {
			continue
		}

		name := labels[labelIntelAppName]
		version := labels[labelIntelAppVersion]
		appName := labels[labelK8sAppName]
		appVersion := labels[labelK8sAppVersion]
		appPartOf := labels[labelK8sAppPartOf]
		helmChart := labels[labelHelmChart]

		hasMandatoryLabel := false
		application := model.KubernetesApplication{}
		if name != "" {
			application.Name = name
			application.Version = version
			hasMandatoryLabel = true
		}
		if appName != "" {
			application.AppName = appName
			application.AppVersion = appVersion
			application.AppPartOf = appPartOf
			application.HelmChart = helmChart
			hasMandatoryLabel = true
		}

		// Include the application only if it has Name or AppName labels and is unique
		if hasMandatoryLabel {
			key := application.GetKey()
			if _, exists := uniqueApplications[key]; !exists {
				uniqueApplications[key] = struct{}{}
				applications = append(applications, application)
			}
		}
	}

	return applications, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

var excludedNamespaces = map[string]struct{}{
	"akri":                        {},
	"calico-system":               {},
	"cattle-fleet-system":         {},
	"cattle-impersonation-system": {},
	"cattle-system":               {},
	"cattle-ui-plugin-system":     {},
	"cdi":                         {},
	"cert-manager":                {},
	"default":                     {},
	"east":                        {},
	"edge-system":                 {},
	"gatekeeper-system":           {},
	"ingress-nginx":               {},
	"intel-gpu-extension":         {},
	"interconnect":                {},
	"istio-operator":              {},
	"istio-system":                {},
	"kube-node-lease":             {},
	"kube-public":                 {},
	"kube-system":                 {},
	"kubevirt":                    {},
	"local":                       {},
	"metallb-system":              {},
	"nfd":                         {},
	"observability":               {},
	"openebs":                     {},
	"orchestrator-system":         {},
	"sriov-network-operator":      {},
	"tigera-operator":             {},
	"west":                        {},
}
