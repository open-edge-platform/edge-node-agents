// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-edge-platform/edge-node-agents/reporting-agent/config"
	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/model"
	"github.com/open-edge-platform/edge-node-agents/reporting-agent/internal/testutil"
)

func TestGetKubernetesDataSuccess(t *testing.T) {
	testutil.ClearMockOutputs()
	// Prepare fake files
	tmpK3Kubectl, err := os.CreateTemp(t.TempDir(), "kubectl")
	require.NoError(t, err, "os.CreateTemp should not error for kubectl")
	defer os.Remove(tmpK3Kubectl.Name())
	tmpK3KubeConfig, err := os.CreateTemp(t.TempDir(), "kubeconfig")
	require.NoError(t, err, "os.CreateTemp should not error for kubeconfig")
	defer os.Remove(tmpK3KubeConfig.Name())

	// Mock kubectl version
	versionJSON := []byte(`{"serverVersion":{"gitVersion":"v1.29.3"}}`)
	testutil.SetMockOutput(tmpK3Kubectl.Name(), []string{"--kubeconfig", tmpK3KubeConfig.Name(), "version", "-o", "json"}, versionJSON, nil)

	// Mock kubectl get
	appsJSON := []byte(`{"items":[{"metadata":{"namespace":"foo","labels":{"app.kubernetes.io/name":"app1"}}}]}`)
	testutil.SetMockOutput(tmpK3Kubectl.Name(),
		[]string{"--kubeconfig", tmpK3KubeConfig.Name(), "get", "deployments,statefulsets,daemonsets", "-A", "-o", "json"}, appsJSON, nil)

	cfg := createK8sConfig(tmpK3Kubectl.Name(), tmpK3KubeConfig.Name(), "/rke2/kubectl/not/exist", "/rke2/kubeconfig/not/exist")
	out, err := GetKubernetesData(testutil.TestCmdExecutor, cfg)
	require.NoError(t, err, "GetKubernetesData should not return error")
	require.Equal(t, "k3s", out.Provider, "Provider should be k3s")
	require.Equal(t, "v1.29.3", out.ServerVersion, "ServerVersion should be v1.29.3")
	require.Len(t, out.Applications, 1, "Applications should have length 1")

	app := out.Applications[0]
	require.Equal(t, "app1", app.AppName, "AppName should be app1")
	require.Empty(t, app.Name, "Name should be empty")
	require.Empty(t, app.Version, "Version should be empty")
	require.Empty(t, app.AppVersion, "AppVersion should be empty")
	require.Empty(t, app.AppPartOf, "AppPartOf should be empty")
}

func TestGetKubernetesDataGetKubeError(t *testing.T) {
	cfg := createK8sConfig("/k3s/kubectl/not/exist", "/k3s/kubeconfig/not/exist", "/rke2/kubectl/not/exist", "/rke2/kubeconfig/not/exist")
	out, err := GetKubernetesData(testutil.TestCmdExecutor, cfg)
	require.ErrorContains(t, err, "failed to get Kubernetes config", "Should error on invalid kube config")
	require.Empty(t, out.Applications, "Applications should be empty on error")
}

func TestGetKubernetesDataVersionCommandError(t *testing.T) {
	testutil.ClearMockOutputs()
	tmpK3Kubectl, err := os.CreateTemp(t.TempDir(), "kubectl")
	require.NoError(t, err, "os.CreateTemp should not error for kubectl")
	defer os.Remove(tmpK3Kubectl.Name())
	tmpK3KubeConfig, err := os.CreateTemp(t.TempDir(), "kubeconfig")
	require.NoError(t, err, "os.CreateTemp should not error for kubeconfig")
	defer os.Remove(tmpK3KubeConfig.Name())

	testutil.SetMockOutput(tmpK3Kubectl.Name(), []string{"--kubeconfig", tmpK3KubeConfig.Name(), "version", "-o", "json"}, nil, errors.New("fail"))
	cfg := createK8sConfig(tmpK3Kubectl.Name(), tmpK3KubeConfig.Name(), "/rke2/kubectl/not/exist", "/rke2/kubeconfig/not/exist")
	out, err := GetKubernetesData(testutil.TestCmdExecutor, cfg)
	require.ErrorContains(t, err, "failed to get Kubernetes server version", "Should error on kubectl version failure")
	require.Empty(t, out.Applications, "Applications should be empty on error")
}

func TestGetKubernetesDataGetCommandError(t *testing.T) {
	testutil.ClearMockOutputs()
	tmpK3Kubectl, err := os.CreateTemp(t.TempDir(), "kubectl")
	require.NoError(t, err, "os.CreateTemp should not error for kubectl")
	defer os.Remove(tmpK3Kubectl.Name())
	tmpK3KubeConfig, err := os.CreateTemp(t.TempDir(), "kubeconfig")
	require.NoError(t, err, "os.CreateTemp should not error for kubeconfig")
	defer os.Remove(tmpK3KubeConfig.Name())

	versionJSON := []byte(`{"serverVersion":{"gitVersion":"v1.29.3"}}`)
	testutil.SetMockOutput(tmpK3Kubectl.Name(), []string{"--kubeconfig", tmpK3KubeConfig.Name(), "version", "-o", "json"}, versionJSON, nil)
	testutil.SetMockOutput(tmpK3Kubectl.Name(),
		[]string{"--kubeconfig", tmpK3KubeConfig.Name(), "get", "deployments,statefulsets,daemonsets", "-A", "-o", "json"}, nil, errors.New("fail"))
	cfg := createK8sConfig(tmpK3Kubectl.Name(), tmpK3KubeConfig.Name(), "/rke2/kubectl/not/exist", "/rke2/kubeconfig/not/exist")
	out, err := GetKubernetesData(testutil.TestCmdExecutor, cfg)
	require.ErrorContains(t, err, "failed to get Kubernetes resources", "Should error on kubectl get failure")
	require.Empty(t, out.Applications, "Applications should be empty on error")
}

func TestGetKubernetesDataParseApplicationsError(t *testing.T) {
	testutil.ClearMockOutputs()
	tmpK3Kubectl, err := os.CreateTemp(t.TempDir(), "kubectl")
	require.NoError(t, err, "os.CreateTemp should not error for kubectl")
	defer os.Remove(tmpK3Kubectl.Name())
	tmpK3KubeConfig, err := os.CreateTemp(t.TempDir(), "kubeconfig")
	require.NoError(t, err, "os.CreateTemp should not error for kubeconfig")
	defer os.Remove(tmpK3KubeConfig.Name())

	versionJSON := []byte(`{"serverVersion":{"gitVersion":"v1.29.3"}}`)
	testutil.SetMockOutput(tmpK3Kubectl.Name(), []string{"--kubeconfig", tmpK3KubeConfig.Name(), "version", "-o", "json"}, versionJSON, nil)
	testutil.SetMockOutput(tmpK3Kubectl.Name(),
		[]string{"--kubeconfig", tmpK3KubeConfig.Name(), "get", "deployments,statefulsets,daemonsets", "-A", "-o", "json"}, []byte("not a json"), nil)
	cfg := createK8sConfig(tmpK3Kubectl.Name(), tmpK3KubeConfig.Name(), "/rke2/kubectl/not/exist", "/rke2/kubeconfig/not/exist")
	out, err := GetKubernetesData(testutil.TestCmdExecutor, cfg)
	require.ErrorContains(t, err, "failed to parse Kubernetes applications", "Should error on invalid JSON")
	require.Empty(t, out.Applications, "Applications should be empty on error")
}

func TestGetKubernetesDataRke2Success(t *testing.T) {
	testutil.ClearMockOutputs()
	// Prepare fake files for rke2
	tmpRke2Kubectl, err := os.CreateTemp(t.TempDir(), "rke2_kubectl")
	require.NoError(t, err, "os.CreateTemp should not error for rke2 kubectl")
	defer os.Remove(tmpRke2Kubectl.Name())
	tmpRke2KubeConfig, err := os.CreateTemp(t.TempDir(), "rke2_kubeconfig")
	require.NoError(t, err, "os.CreateTemp should not error for rke2 kubeconfig")
	defer os.Remove(tmpRke2KubeConfig.Name())

	// Mock kubectl version
	versionJSON := []byte(`{"serverVersion":{"gitVersion":"v1.30.0"}}`)
	testutil.SetMockOutput(tmpRke2Kubectl.Name(), []string{"--kubeconfig", tmpRke2KubeConfig.Name(), "version", "-o", "json"}, versionJSON, nil)

	// Mock kubectl get
	appsJSON := []byte(`{"items":[{"metadata":{"namespace":"foo","labels":{"app.kubernetes.io/name":"rke2-app"}}}]}`)
	testutil.SetMockOutput(tmpRke2Kubectl.Name(),
		[]string{"--kubeconfig", tmpRke2KubeConfig.Name(), "get", "deployments,statefulsets,daemonsets", "-A", "-o", "json"}, appsJSON, nil)

	// k3s paths are invalid, rke2 paths are valid
	cfg := createK8sConfig("/k3s/kubectl/not/exist", "/k3s/kubeconfig/not/exist", tmpRke2Kubectl.Name(), tmpRke2KubeConfig.Name())
	out, err := GetKubernetesData(testutil.TestCmdExecutor, cfg)
	require.NoError(t, err, "GetKubernetesData should not return error for rke2 paths")
	require.Equal(t, "rke2", out.Provider, "Provider should be rke2")
	require.Equal(t, "v1.30.0", out.ServerVersion, "ServerVersion should be v1.30.0")
	require.Len(t, out.Applications, 1, "Applications should have length 1")
	app := out.Applications[0]
	require.Equal(t, "rke2-app", app.AppName, "AppName should be rke2-app")
}

func TestGetKubeSuccess(t *testing.T) {
	tmpK3Kubectl, err := os.CreateTemp(t.TempDir(), "kubectl")
	require.NoError(t, err, "os.CreateTemp should not error for kubectl")
	defer os.Remove(tmpK3Kubectl.Name())
	tmpK3KubeConfig, err := os.CreateTemp(t.TempDir(), "kubeconfig")
	require.NoError(t, err, "os.CreateTemp should not error for kubeconfig")
	defer os.Remove(tmpK3KubeConfig.Name())

	cfg := createK8sConfig(tmpK3Kubectl.Name(), tmpK3KubeConfig.Name(), "/rke2/kubectl/not/exist", "/rke2/kubeconfig/not/exist")
	k, err := getKube(cfg)
	require.NoError(t, err, "getKube should not return error")
	require.Equal(t, tmpK3Kubectl.Name(), k.kubectlPath, "kubectlPath should match tmpK3Kubectl.Name()")
	require.Equal(t, tmpK3KubeConfig.Name(), k.kubeconfigPath, "kubeconfigPath should match tmpK3KubeConfig.Name()")
	require.Equal(t, "k3s", k.provider, "provider should be k3s")
}

func TestGetKubeNoValidFiles(t *testing.T) {
	cfg := createK8sConfig("/k3s/kubectl/not/exist", "/k3s/kubeconfig/not/exist", "/rke2/kubectl/not/exist", "/rke2/kubeconfig/not/exist")
	_, err := getKube(cfg)
	require.ErrorContains(t, err, "no valid kubeconfig and kubectl files found", "Should error on missing files")
}

func TestSplitKubectlPath(t *testing.T) {
	ctl, args := splitKubectlPath("/usr/bin/kubectl --token foo")
	require.Equal(t, "/usr/bin/kubectl", ctl, "ctl should be /usr/bin/kubectl")
	require.Equal(t, []string{"--token", "foo"}, args, "args should be [--token foo]")

	ctl, args = splitKubectlPath("")
	require.Equal(t, "", ctl, "ctl should be empty string")
	require.Nil(t, args, "args should be nil for empty input")

	ctl, args = splitKubectlPath("kubectl")
	require.Equal(t, "kubectl", ctl, "ctl should be kubectl")
	require.Empty(t, args, "args should be empty for single word")
}

func TestParseServerVersionSuccess(t *testing.T) {
	input := []byte(`{"serverVersion":{"gitVersion":"v1.29.3"}}`)
	require.Equal(t, "v1.29.3", parseServerVersion(input), "Should parse version v1.29.3")
}

func TestParseServerVersionMalformed(t *testing.T) {
	input := []byte(`not a json`)
	require.Equal(t, "", parseServerVersion(input), "Should return empty string for malformed JSON")
}

func TestParseApplicationsSuccess(t *testing.T) {
	raw, err := os.ReadFile("./testdata/kubectl_duplicates.json")
	require.NoError(t, err, "os.ReadFile should not error for kubectl_duplicates.json")
	apps, err := parseApplications(raw)
	require.NoError(t, err, "parseApplications should not return error")
	require.Len(t, apps, 3, "Should parse 3 applications")
	require.Contains(t, apps, model.KubernetesApplication{Name: "intel-app", Version: "1.2.3", AppName: "k8s-app", AppVersion: "2.3.4", AppPartOf: "intel"},
		"Should contain intel-app")
	require.Contains(t, apps, model.KubernetesApplication{AppName: "only-k8s", AppVersion: "v2"}, "Should contain only-k8s")
	require.Contains(t, apps, model.KubernetesApplication{Name: "only-intel", Version: "v1"}, "Should contain only-intel")
}

func TestParseApplicationsDuplicate(t *testing.T) {
	// Two identical apps, only one should be returned
	data := map[string]any{
		"items": []any{
			map[string]any{
				"metadata": map[string]any{
					"namespace": "foo",
					"labels": map[string]any{
						labelK8sAppName:    "dup",
						labelK8sAppVersion: "2.3.4",
					},
				},
			},
			map[string]any{
				"metadata": map[string]any{
					"namespace": "foo",
					"labels": map[string]any{
						labelK8sAppName:    "dup",
						labelK8sAppVersion: "2.3.4",
					},
				},
			},
		},
	}
	b, err := json.Marshal(data)
	require.NoError(t, err, "json.Marshal should not error")
	apps, err := parseApplications(b)
	require.NoError(t, err, "parseApplications should not return error")
	require.Len(t, apps, 1, "Should deduplicate identical applications")
}

func TestParseApplicationsExcludedNamespace(t *testing.T) {
	// Should skip excluded namespace
	data := map[string]any{
		"items": []any{
			map[string]any{
				"metadata": map[string]any{
					"namespace": "kube-system",
					"labels": map[string]any{
						labelK8sAppName: "should-not-be-included",
					},
				},
			},
		},
	}
	b, err := json.Marshal(data)
	require.NoError(t, err, "json.Marshal should not error")
	apps, err := parseApplications(b)
	require.NoError(t, err, "parseApplications should not return error")
	require.Empty(t, apps, "Should skip excluded namespaces")
}

func TestParseApplicationsNoLabels(t *testing.T) {
	// Should skip if labels is nil
	data := map[string]any{
		"items": []any{
			map[string]any{
				"metadata": map[string]any{
					"namespace": "foo",
				},
			},
		},
	}
	b, err := json.Marshal(data)
	require.NoError(t, err, "json.Marshal should not error")
	apps, err := parseApplications(b)
	require.NoError(t, err, "parseApplications should not return error")
	require.Empty(t, apps, "Should skip items with no labels")
}

func TestParseApplicationsMalformedJSON(t *testing.T) {
	_, err := parseApplications([]byte("not a json"))
	require.ErrorContains(t, err, "failed to unmarshal Kubernetes resources data", "Should error on malformed JSON")
}

func TestParseApplicationsWithRealKubectlOutput(t *testing.T) {
	raw, err := os.ReadFile("./testdata/kubectl_real.json")
	require.NoError(t, err, "os.ReadFile should not error for kubectl_real.json")
	apps, err := parseApplications(raw)
	require.NoError(t, err, "parseApplications should not return error")

	expected := map[string]model.KubernetesApplication{
		"wordpress": {
			Name:       "wordpress",
			Version:    "6.4.3",
			AppName:    "wordpress",
			AppVersion: "6.4.3",
			AppPartOf:  "something-bigger",
			HelmChart:  "wordpress-19.4.3",
		},
		"nginx": {
			Name:       "",
			Version:    "",
			AppName:    "nginx",
			AppVersion: "1.25.3",
			AppPartOf:  "",
			HelmChart:  "nginx-15.9.3",
		},
		"mariadb": {
			Name:       "",
			Version:    "",
			AppName:    "mariadb",
			AppVersion: "11.2.2",
			AppPartOf:  "",
			HelmChart:  "mariadb-15.2.3",
		},
	}

	found := map[string]bool{}
	for _, app := range apps {
		exp, ok := expected[app.AppName]
		if ok {
			require.Equal(t, exp, app, "Application struct should match expected for %s", app.AppName)
			found[app.AppName] = true
		}
	}
	for name := range expected {
		require.True(t, found[name], "expected app %s not found", name)
	}
}

func TestParseApplicationsEmptyItems(t *testing.T) {
	b := []byte(`{"items":[]}`)
	apps, err := parseApplications(b)
	require.NoError(t, err, "parseApplications should not return error for empty items")
	require.Empty(t, apps, "Should return empty slice for empty items")
}

func TestParseApplicationsMissingItemsField(t *testing.T) {
	b := []byte(`{}`)
	apps, err := parseApplications(b)
	require.NoError(t, err, "parseApplications should not return error for missing items field")
	require.Empty(t, apps, "Should return empty slice for missing items field")
}

func TestFileExists(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "k8s_test")
	require.NoError(t, err, "os.CreateTemp should not error")
	defer os.Remove(tmp.Name())

	require.True(t, fileExists(tmp.Name()), "fileExists should return true for existing file")
	require.False(t, fileExists("/not/existing/file/xyz"), "fileExists should return false for non-existing file")
}

func createK8sConfig(k3sKubectlPath, k3sKubeConfigPath, rke2KubectlPath, rke2KubeConfigPath string) config.K8sConfig {
	return config.K8sConfig{
		K3sKubectlPath:     k3sKubectlPath,
		K3sKubeConfigPath:  k3sKubeConfigPath,
		Rke2KubectlPath:    rke2KubectlPath,
		Rke2KubeConfigPath: rke2KubeConfigPath,
	}
}
