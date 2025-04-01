// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
package metriccfg_test

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
	"github.com/open-edge-platform/edge-node-agents/platform-telemetry-agent/internal/metriccfg"
	pb "github.com/open-edge-platform/infra-managers/telemetry/pkg/api/telemetrymgr/v1"
	"github.com/stretchr/testify/assert"
)

func TestUpdateHostMetricConfigWithTmpFile(t *testing.T) {
	testUpdateHostMetricConfigWithTmpFile(t, "/tmp")
	testUpdateHostMetricConfigWithTmpFile(t, "./")
}

func testUpdateHostMetricConfigWithTmpFile(t *testing.T, tmpDir string) {
	// Common setup
	metriccfg.TmpFileDir = tmpDir
	telegrafFilePath := "telegraf_unittest.conf"
	defer os.Remove(telegrafFilePath)

	resp := &pb.GetTelemetryConfigResponse{
		HostGuid:  "mock-host-guid",
		Timestamp: "2023-01-15T12:34:56+00:00",
		Cfg: []*pb.GetTelemetryConfigResponse_TelemetryCfg{
			{
				Input:    "mock-input",
				Type:     pb.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS,
				Kind:     pb.CollectorKind_COLLECTOR_KIND_HOST,
				Level:    pb.SeverityLevel_SEVERITY_LEVEL_INFO,
				Interval: 60,
			},
		},
	}

	// Test with telegrafHostGoldPath as telegraf-host-gold.conf
	file, err := os.Create(telegrafFilePath)
	if err != nil {
		t.Error("Error creating telegraf_unittest.conf")
	}
	defer file.Close()

	telegrafHostGoldPath := "../../configs/telegraf-host-gold.conf"
	result, _ := metriccfg.UpdateHostMetricConfig(context.Background(), resp, telegrafFilePath, telegrafHostGoldPath, false)
	assert.NotNil(t, result)

	// Test with telegrafHostGoldPath as telegraf-host-gold.yaml
	telegrafHostGoldPath = "../../configs/telegraf-host-gold.yaml"
	result, _ = metriccfg.UpdateHostMetricConfig(context.Background(), resp, telegrafFilePath, telegrafHostGoldPath, false)
	assert.NotNil(t, result)

	// create fail template
	sourceFile, _ := os.Open(telegrafHostGoldPath)
	defer sourceFile.Close()
	destinationFile, _ := os.Create("./tg-host.yaml")
	defer destinationFile.Close()
	_, _ = io.Copy(destinationFile, sourceFile)

	templateFailContent, _ := utils.ReadFileNoLinks("./tg-host.yaml")
	text := string(templateFailContent)
	cfgupdatedtext := strings.ReplaceAll(text, "agent", "fa")
	udestinationFile, _ := os.Create("./tg-host.yaml")
	defer udestinationFile.Close()
	_, _ = udestinationFile.WriteString(cfgupdatedtext)

	_, err = metriccfg.UpdateHostMetricConfig(context.Background(), resp, telegrafFilePath, "./tg-host.yaml", false)
	assert.NotNil(t, err)

	os.Remove("./tg-host.yaml")
}

func TestErrorReadConfigFilePath(t *testing.T) {
	metriccfg.TmpFileDir = "./"
	telegrafHostGoldPath := "../../configs/telegraf-host-gold.yaml"

	resp := &pb.GetTelemetryConfigResponse{
		HostGuid:  "mock-host-guid",
		Timestamp: "2023-01-15T12:34:56+00:00",
		Cfg: []*pb.GetTelemetryConfigResponse_TelemetryCfg{
			{
				Input:    "mock-input",
				Type:     pb.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS,
				Kind:     pb.CollectorKind_COLLECTOR_KIND_HOST,
				Level:    pb.SeverityLevel_SEVERITY_LEVEL_INFO,
				Interval: 60,
			},
		},
	}
	result, _ := metriccfg.UpdateHostMetricConfig(context.Background(), resp, "", telegrafHostGoldPath, false)
	assert.NotNil(t, result)
	os.Remove("telegraf-tmp.conf")
}

func TestErrorUpdateHostMetricConfig(t *testing.T) {
	metriccfg.TmpFileDir = "./"
	resp := &pb.GetTelemetryConfigResponse{
		HostGuid:  "mock-host-guid",
		Timestamp: "2023-01-15T12:34:56+00:00",
		Cfg: []*pb.GetTelemetryConfigResponse_TelemetryCfg{
			{
				Input:    "mock-input",
				Type:     pb.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS,
				Kind:     pb.CollectorKind_COLLECTOR_KIND_HOST,
				Level:    pb.SeverityLevel_SEVERITY_LEVEL_INFO,
				Interval: 60,
			},
		},
	}

	_, err := metriccfg.UpdateHostMetricConfig(context.Background(), resp, "", "", false)
	assert.NotNil(t, err)

	metriccfg.TmpFileDir = "./"
	resp = &pb.GetTelemetryConfigResponse{
		HostGuid:  "mock-host-guid",
		Timestamp: "2023-01-15T12:34:56+00:00",
		Cfg: []*pb.GetTelemetryConfigResponse_TelemetryCfg{
			{
				Input:    "mock-input-err",
				Type:     pb.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS,
				Kind:     pb.CollectorKind_COLLECTOR_KIND_HOST,
				Level:    pb.SeverityLevel_SEVERITY_LEVEL_INFO,
				Interval: 334,
			},
		},
	}

	_, err = metriccfg.UpdateHostMetricConfig(context.Background(), resp, "", "", false)
	assert.NotNil(t, err)

	os.Remove("telegraf-tmp.conf")
}

func TestUpdateClusterMetricConfigWithTmpFile(t *testing.T) {
	testUpdateClusterMetricConfigWithTmpFile(t, "/tmp")
	testUpdateClusterMetricConfigWithTmpFile(t, "./")
}

func testUpdateClusterMetricConfigWithTmpFile(t *testing.T, tmpDir string) {
	// Data to write to the file
	data := `[agent]
    interval = "30s"
    round_interval = true
    metric_batch_size = 1000
    metric_buffer_limit = 10000
    collection_jitter = "0s"
    flush_interval = "5s"
    flush_jitter = "0s"
    precision = ""
    debug = false
    quiet = false
    logfile = ""
    hostname = "$HOSTNAME"
    omit_hostname = false
  
  # Read metrics from kubernetes resources
  [[inputs.kube_inventory]]
    ## URL for the kubernetes API
    url = https://10.10.10.10:9999
    ## Namespace to check, use "" to check all namespaces
    namespace = ""
    ## Use TLS buyt skip verification
    insecure_skip_verify = true

    # Receive OpenTelemetry traces, metrics, and logs over gRPC
   [[inputs.opentelemetry]]
    ## Override the default (0.0.0.0:4317) destination OpenTelemetry gRPC service
    ## address:port
    service_address = "0.0.0.0:{{ .Values.telegraf.otelport }}"

    ## Override the default (5s) new connection timeout
    # timeout = "5s"

    ## Override the default (prometheus-v1) metrics schema.
    ## Supports: "prometheus-v1", "prometheus-v2"
    ## For more information about the alternatives, read the Prometheus input
    ## plugin notes.
    # metrics_schema = "prometheus-v1"

    ## Optional TLS Config.
    ## For advanced options: https://github.com/influxdata/telegraf/blob/v1.18.3/docs/TLS.md
    ##
    ## Set one or more allowed client CA certificate file names to
    ## enable mutually authenticated TLS connections.
    tls_allowed_cacerts = ["{{ .Values.certs.certsDest }}/ca.crt"]
    ## Add service certificate and key.
    tls_cert = "{{ .Values.certs.certsDest }}/tls.crt"
    tls_key = "{{ .Values.certs.certsDest }}/tls.key"	
	tls_key {{test}}
		
  [[outputs.prometheus_client]]
    ## Address to listen on.
    listen = ":9105"
  
    ## Metric version controls the mapping from Telegraf metrics into
    ## Prometheus format.  When using the prometheus input, use the same value in
    ## both plugins to ensure metrics are round-tripped without modification.
    ##
    ##   example: metric_version = 1;
    ##            metric_version = 2; recommended version
    metric_version = 2
  
    ## Use HTTP Basic Authentication.
    # basic_username = "Foo"
    # basic_password = ""
  
    ## If set, the IP Ranges which are allowed to access metrics.
    ##   ex: ip_range = ["192.168.0.0/24", "192.168.1.0/30"]
    # ip_range = ["192.168.0.0/24", "192.168.1.0/30"]
  
    ## Path to publish the metrics on.
    path = "/metrics"
  
    ## Expiration interval for each metric. 0 == no expiration
    # expiration_interval = "60s"
  
    ## Collectors to enable, valid entries are "gocollector" and "process".
    ## If unset, both are enabled.
    # collectors_exclude = ["gocollector", "process"]
  
    ## Send string metrics as Prometheus labels.
    ## Unless set to false all string metrics will be sent as labels.
    # string_as_label = true
  
    ## If set, enable TLS with the given certificate.
    tls_cert = "/opt/telegraf/certs/tls.crt"
    tls_key = "/opt/telegraf/certs/tls.key"
  
    ## Set one or more allowed client CA certificate file names to
    ## enable mutually authenticated TLS connections
    tls_allowed_cacerts = ["/opt/telegraf/certs/ca.crt"]
  
    ## Export metric collection time.
    export_timestamp = true`

	// Open file for writing (create if not exists, truncate if exists)
	file, _ := os.Create("./configmap.conf")
	defer file.Close()
	// Write data to the file
	_, _ = file.WriteString(data)

	metriccfg.TmpFileDir = tmpDir
	metriccfg.ConfigMapCommand = "cat ./configmap.conf"
	telegrafClusterGoldPath := "../../configs/telegraf-cluster-gold.yaml"

	resp := &pb.GetTelemetryConfigResponse{
		HostGuid:  "mock-host-guid",
		Timestamp: "2023-01-15T12:34:56+00:00",
		Cfg: []*pb.GetTelemetryConfigResponse_TelemetryCfg{
			{
				Input:    "kube_inventory",
				Type:     pb.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS,
				Kind:     pb.CollectorKind_COLLECTOR_KIND_CLUSTER,
				Level:    pb.SeverityLevel_SEVERITY_LEVEL_INFO,
				Interval: 60,
			},
		},
	}

	result, _ := metriccfg.UpdateClusterMetricConfig(context.Background(), resp, telegrafClusterGoldPath, false)
	assert.NotNil(t, result)

	// create fail template
	sourceFile, _ := os.Open(telegrafClusterGoldPath)
	defer sourceFile.Close()
	destinationFile, _ := os.Create("./tg-cluster.yaml")
	defer destinationFile.Close()
	_, _ = io.Copy(destinationFile, sourceFile)

	// failed agent
	templateFailContent, _ := utils.ReadFileNoLinks("./tg-cluster.yaml")
	text := string(templateFailContent)
	cfgupdatedtext := strings.ReplaceAll(text, "agent", "ft")
	udestinationFile, _ := os.Create("./tg-cluster.yaml")
	defer udestinationFile.Close()
	_, _ = udestinationFile.WriteString(cfgupdatedtext)

	result, _ = metriccfg.UpdateClusterMetricConfig(context.Background(), resp, "./tg-cluster.yaml", false)
	assert.NotNil(t, result)

	// failt output
	ndestinationFile, _ := os.Create("./tg-cluster-2.yaml")
	defer ndestinationFile.Close()
	nfgupdatedtext := strings.ReplaceAll(text, "output", "ft")
	_, _ = ndestinationFile.WriteString(nfgupdatedtext)

	result, _ = metriccfg.UpdateClusterMetricConfig(context.Background(), resp, "./tg-cluster-2.yaml", false)
	assert.NotNil(t, result)

	os.Remove("telegraf-tmp-cluster.conf")
	os.Remove("./configmap.conf")
	os.Remove("./tg-cluster.yaml")
	os.Remove("./tg-cluster-2.yaml")
}

func TestErrorUpdateClusterMetricConfig(t *testing.T) {
	metriccfg.TmpFileDir = "./"
	resp := &pb.GetTelemetryConfigResponse{
		HostGuid:  "mock-host-guid",
		Timestamp: "2023-01-15T12:34:56+00:00",
		Cfg: []*pb.GetTelemetryConfigResponse_TelemetryCfg{
			{
				Input:    "mock-input",
				Type:     pb.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS,
				Kind:     pb.CollectorKind_COLLECTOR_KIND_CLUSTER,
				Level:    pb.SeverityLevel_SEVERITY_LEVEL_INFO,
				Interval: 60,
			},
		},
	}

	_, err := metriccfg.UpdateClusterMetricConfig(context.Background(), resp, "", false)
	assert.NotNil(t, err)

	os.Remove("telegraf-tmp-cluster.conf")
}

func TestErrorOnMarshall(t *testing.T) {
	resp := &pb.GetTelemetryConfigResponse{
		HostGuid:  "mock-host-guid",
		Timestamp: "2023-01-15T12:34:56+00:00",
		Cfg: []*pb.GetTelemetryConfigResponse_TelemetryCfg{
			{
				Input:    "mock-input",
				Type:     pb.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS,
				Kind:     pb.CollectorKind_COLLECTOR_KIND_HOST,
				Level:    pb.SeverityLevel_SEVERITY_LEVEL_INFO,
				Interval: 60,
			},
		},
	}

	file, err := os.Create("telegraf_unittest.conf")
	if err != nil {
		t.Error("Error create telegraf_unittest.conf")
	}
	defer file.Close()
	telegrafFilePath := "telegraf_unittest.conf"
	err = createFileWithWrongContent("unittest_config.conf")
	if err != nil {
		t.Error("Error creating file:", err)
	}
	telegrafHostPath := "unittest_config.conf"
	telegrafClusterPath := "unittest_config.conf"

	result, _ := metriccfg.UpdateHostMetricConfig(context.Background(), resp, telegrafFilePath, telegrafHostPath, false)
	result1, _ := metriccfg.UpdateClusterMetricConfig(context.Background(), resp, telegrafClusterPath, false)
	assert.NotNil(t, result)
	assert.NotNil(t, result1)
	os.Remove(telegrafFilePath)
	os.Remove(telegrafHostPath)
	os.Remove("telegraf-cluster-tmp.conf")
}

func createFileWithWrongContent(filename string) error {
	content := `[agent]
	interval = "66s"
	round_interval = true
	metric_batch_size = 1000
	metric_buffer_limit = 10000
	collection_jitter = "0s"
	flush_interval = "5s"
	flush_jitter = "0s"
	precision = ""
	debug = false
	quiet = false
	logfile = ""
	hostname = "$HOSTNAME"
	omit_hostname = false
  
	`

	err := os.WriteFile(filename, []byte(content), 0600)
	if err != nil {
		return err
	}

	return nil
}

func TestUpdateClusterMetricConfigErrorConfigMap(t *testing.T) {
	metriccfg.TmpFileDir = "./"
	metriccfg.ConfigMapCommand = ""
	telegrafClusterGoldPath := "../../configs/telegraf-cluster-gold.yaml"

	resp := &pb.GetTelemetryConfigResponse{
		HostGuid:  "mock-host-guid",
		Timestamp: "2023-01-15T12:34:56+00:00",
		Cfg: []*pb.GetTelemetryConfigResponse_TelemetryCfg{
			{
				Input:    "kube_inventory",
				Type:     pb.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS,
				Kind:     pb.CollectorKind_COLLECTOR_KIND_CLUSTER,
				Level:    pb.SeverityLevel_SEVERITY_LEVEL_INFO,
				Interval: 60,
			},
		},
	}

	_, err := metriccfg.UpdateClusterMetricConfig(context.Background(), resp, telegrafClusterGoldPath, false)
	assert.NotNil(t, err)

	os.Remove("telegraf-tmp-cluster.conf")
}

func TestUpdateClusterMetricConfigWithErr(t *testing.T) {
	// Data to write to the file
	data := `[agent]
    interval = "30s"
    round_interval = true
    metric_batch_size = 1000
    metric_buffer_limit = 10000
    collection_jitter = "0s"
    flush_interval = "5s"
    flush_jitter = "0s"
    precision = ""
    debug = false
    quiet = false
    logfile = ""
    hostname = "$HOSTNAME"
    omit_hostname = false
  
  # Read metrics from kubernetes resources
  [[inputs.kube_inventory]]
    ## URL for the kubernetes API
    url = https://10.10.10.10:9999
    ## Namespace to check, use "" to check all namespaces
    namespace = ""
    ## Use TLS buyt skip verification
    insecure_skip_verify = true

    # Receive OpenTelemetry traces, metrics, and logs over gRPC
   [[inputs.opentelemetry]]
    ## Override the default (0.0.0.0:4317) destination OpenTelemetry gRPC service
    ## address:port
    
    ## Override the default (5s) new connection timeout
    # timeout = "5s"

    ## Override the default (prometheus-v1) metrics schema.
    ## Supports: "prometheus-v1", "prometheus-v2"
    ## For more information about the alternatives, read the Prometheus input
    ## plugin notes.
    # metrics_schema = "prometheus-v1"

    ## Optional TLS Config.
    ## For advanced options: https://github.com/influxdata/telegraf/blob/v1.18.3/docs/TLS.md
    ##
    ## Set one or more allowed client CA certificate file names to
    ## enable mutually authenticated TLS connections.
    tls_allowed_cacerts = ["{{ .Values.certs.certsDest }}/ca.crt"]
    ## Add service certificate and key.
    tls_cert = "{{ .Values.certs.certsDest }}/tls.crt"
    tls_key = "{{ .Values.certs.certsDest }}/tls.key"	
	tls_key {{test}}
		
  [[outputs.prometheus_client]]
    ## Address to listen on.
    listen = ":9105"
  
    ## Metric version controls the mapping from Telegraf metrics into
    ## Prometheus format.  When using the prometheus input, use the same value in
    ## both plugins to ensure metrics are round-tripped without modification.
    ##
    ##   example: metric_version = 1;
    ##            metric_version = 2; recommended version
    metric_version = 2
  
    ## Use HTTP Basic Authentication.
    # basic_username = "Foo"
    # basic_password = ""
  
    ## If set, the IP Ranges which are allowed to access metrics.
    ##   ex: ip_range = ["192.168.0.0/24", "192.168.1.0/30"]
    # ip_range = ["192.168.0.0/24", "192.168.1.0/30"]
  
    ## Path to publish the metrics on.
    path = "/metrics"
  
    ## Expiration interval for each metric. 0 == no expiration
    # expiration_interval = "60s"
  
    ## Collectors to enable, valid entries are "gocollector" and "process".
    ## If unset, both are enabled.
    # collectors_exclude = ["gocollector", "process"]
  
    ## Send string metrics as Prometheus labels.
    ## Unless set to false all string metrics will be sent as labels.
    # string_as_label = true
  
    ## If set, enable TLS with the given certificate.
    tls_cert = "/opt/telegraf/certs/tls.crt"
    tls_key = "/opt/telegraf/certs/tls.key"
  
    ## Set one or more allowed client CA certificate file names to
    ## enable mutually authenticated TLS connections
    tls_allowed_cacerts = ["/opt/telegraf/certs/ca.crt"]
  
    ## Export metric collection time.
    export_timestamp = true`

	// Open file for writing (create if not exists, truncate if exists)
	file, _ := os.Create("./configmap.conf")
	defer file.Close()
	// Write data to the file
	_, _ = file.WriteString(data)

	metriccfg.TmpFileDir = "/tmp/"
	metriccfg.ConfigMapCommand = "cat ./configmap.conf"
	telegrafClusterGoldPath := "../../configs/telegraf-cluster-gold.yaml"
	metriccfg.TmpClusterFile = "~test"

	resp := &pb.GetTelemetryConfigResponse{
		HostGuid:  "mock-host-guid",
		Timestamp: "2023-01-15T12:34:56+00:00",
		Cfg: []*pb.GetTelemetryConfigResponse_TelemetryCfg{
			{
				Input:    "kube_inventory",
				Type:     pb.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS,
				Kind:     pb.CollectorKind_COLLECTOR_KIND_CLUSTER,
				Level:    pb.SeverityLevel_SEVERITY_LEVEL_INFO,
				Interval: 60,
			},
		},
	}

	_, err := metriccfg.UpdateClusterMetricConfig(context.Background(), resp, telegrafClusterGoldPath, false)
	if err == nil {
		t.Error(err)
	}

	os.Remove("telegraf-tmp-cluster.conf")
	os.Remove("./configmap.conf")
}
