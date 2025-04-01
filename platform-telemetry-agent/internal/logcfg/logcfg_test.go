// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
package logcfg_test

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
	"github.com/open-edge-platform/edge-node-agents/platform-telemetry-agent/internal/logcfg"
	pb "github.com/open-edge-platform/infra-managers/telemetry/pkg/api/telemetrymgr/v1"
	"github.com/stretchr/testify/assert"
)

func TestUpdateHostLogConfigWithTmpFile(t *testing.T) {
	testUpdateHostLogConfigWithTmpFile(t, "/tmp")
	testUpdateHostLogConfigWithTmpFile(t, "./")
}

func testUpdateHostLogConfigWithTmpFile(t *testing.T, tmpDir string) {
	// Common setup
	logcfg.TmpFileDir = tmpDir
	fluentbitFilePath := "fluentbit_unittest.conf"
	defer os.Remove(fluentbitFilePath)

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

	file, err := os.Create(fluentbitFilePath)
	if err != nil {
		t.Error("Error create fluentbit_unittest.conf")
	}
	defer file.Close()

	fluentbitHostPath := "../../configs/fluentbit-host-gold.yaml"

	result, _ := logcfg.UpdateHostLogConfig(context.Background(), resp, fluentbitFilePath, fluentbitHostPath, false)
	assert.NotNil(t, result)
	os.Remove(fluentbitFilePath)
}

func TestErrorUpdateHostLogConfig(t *testing.T) {
	logcfg.TmpFileDir = "./"
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
	// fail template
	_, err := logcfg.UpdateHostLogConfig(context.Background(), resp, "", "", false)
	assert.NotNil(t, err)

	// fail copy
	fluentbitHostPath := "../../configs/fluentbit-host-gold.yaml"
	_, err = logcfg.UpdateHostLogConfig(context.Background(), resp, "", fluentbitHostPath, false)
	assert.NotNil(t, err)

	os.Remove("fluentbit-tmp.conf")
}

func TestUpdateClusterLogConfigWithTmpFile(t *testing.T) {
	testUpdateClusterLogConfigWithTmpFile(t, "/tmp")
	testUpdateClusterLogConfigWithTmpFile(t, "./")
}

func testUpdateClusterLogConfigWithTmpFile(t *testing.T, tmpDir string) {

	data := `[SERVICE]
    Flush     5
    Log_Level info

[INPUT] ## Input for Rancher agent log (kubelet log)
    Name           tail
    Tag            EdgeNode_KubeletLog
    Path           /var/lib/rancher/rke2/agent/logs/kubelet.log
    Mem_Buf_Limit 5MB
    Read_from_Head true
    Log_Level info

[INPUT] ## Input for syslog
    Name           tail
    Tag            EdgeNode_Syslog
    Path           /var/log/syslog
    Parser         syslog-rfc3164
    Read_from_Head true
    Mem_Buf_Limit 5MB
    Skip_Long_Lines On

[INPUT] ## Input for auth.log
    Name           tail
    Tag            EdgeNode_AuthLog
    Path           /var/log/auth.log
    Parser         syslog-rfc3164
    Read_from_Head true
    Mem_Buf_Limit 5MB
    Skip_Long_Lines On

[INPUT] ## Input for logs/traces received on port
    Name         opentelemetry
    Tag          EdgeNode_OtelLogs
    Listen       0.0.0.0
    Port         8888
    tls          on
    tls.verify   on
    tls.ca_file  test/ca.crt
    tls.crt_file test/tls.crt
    tls.key_file test/tls.key

[FILTER] ## Filter for adding UUID of Edge Node to all logs
    Name   record_modifier
    Match  *
    Record UUID {{ .Values.nodeUUID }}


[FILTER] ## Filter for Rancher agent log (kubelet log) tagging
    Name   record_modifier
    Match  EdgeNode_KubeletLog
    Record FileType KubeletLog


[FILTER] ## Filter for adding host name of Edge Node to all logs
	Name   record_modifier
	Match  *
	Record Hostname {{ .Values.hostname }}

[FILTER] ## Filter for syslog tagging
    Name   record_modifier
    Match  EdgeNode_Syslog
    Record FileType SystemLog

[FILTER] ## Filter for auth log tagging
    Name   record_modifier
    Match  EdgeNode_AuthLog
    Record FileType AuthLog



[OUTPUT] ## Output Edge Node Container logs.
    Name      forward
    Match     Container.*
    Unix_Path /run/platform-observability-agent/fluent-bit/container-logs.sock

[OUTPUT] ## Output Edge Node System logs.
	Name      forward
	Match     EdgeNode_*
	Unix_Path /run/platform-observability-agent/fluent-bit/host-logs.sock

[OUTPUT] ## Output container logs for edgenode applications. Logs not originating from an edgenode system namespace will be considered application logs
    Name      forward
    Match     Application_EdgeNode.*
    Unix_Path /run/platform-observability-agent/fluent-bit/application-logs.sock	
	`

	// Open file for writing (create if not exists, truncate if exists)
	file, _ := os.Create("./fb-configmap.conf")
	defer file.Close()
	// Write data to the file
	_, _ = file.WriteString(data)

	logcfg.TmpFileDir = tmpDir
	logcfg.ConfigMapCommand = "cat ./fb-configmap.conf"
	fluentbitClusterPath := "../../configs/fluentbit-cluster-gold.yaml"
	resp := &pb.GetTelemetryConfigResponse{
		HostGuid:  "mock-host-guid",
		Timestamp: "2023-01-15T12:34:56+00:00",
		Cfg: []*pb.GetTelemetryConfigResponse_TelemetryCfg{
			{
				Input:    "opentelemetry",
				Type:     pb.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_LOGS,
				Kind:     pb.CollectorKind_COLLECTOR_KIND_CLUSTER,
				Level:    pb.SeverityLevel_SEVERITY_LEVEL_INFO,
				Interval: 60,
			},
		},
	}

	result, _ := logcfg.UpdateClusterLogConfig(context.Background(), resp, fluentbitClusterPath, false)
	assert.NotNil(t, result)

	resp.Cfg[0].Level = -1
	result, _ = logcfg.UpdateClusterLogConfig(context.Background(), resp, fluentbitClusterPath, false)
	assert.NotNil(t, result)

	resp.Cfg[0].Level = pb.SeverityLevel_SEVERITY_LEVEL_UNSPECIFIED
	result, _ = logcfg.UpdateClusterLogConfig(context.Background(), resp, fluentbitClusterPath, false)
	assert.NotNil(t, result)

	resp.Cfg[0].Level = pb.SeverityLevel_SEVERITY_LEVEL_DEBUG
	result, _ = logcfg.UpdateClusterLogConfig(context.Background(), resp, fluentbitClusterPath, false)
	assert.NotNil(t, result)

	resp.Cfg[0].Level = pb.SeverityLevel_SEVERITY_LEVEL_CRITICAL
	result, _ = logcfg.UpdateClusterLogConfig(context.Background(), resp, fluentbitClusterPath, false)
	assert.NotNil(t, result)

	resp.Cfg[0].Level = pb.SeverityLevel_SEVERITY_LEVEL_ERROR
	result, _ = logcfg.UpdateClusterLogConfig(context.Background(), resp, fluentbitClusterPath, false)
	assert.NotNil(t, result)

	resp.Cfg[0].Level = pb.SeverityLevel_SEVERITY_LEVEL_WARN
	result, _ = logcfg.UpdateClusterLogConfig(context.Background(), resp, fluentbitClusterPath, false)
	assert.NotNil(t, result)

	resp.Cfg[0].Input = "NOT_EXIST_INPUT"
	result, _ = logcfg.UpdateClusterLogConfig(context.Background(), resp, fluentbitClusterPath, false)
	assert.NotNil(t, result)

	// create fail template
	sourceFile, _ := os.Open(fluentbitClusterPath)
	defer sourceFile.Close()
	destinationFile, _ := os.Create("./fb-cluster.yaml")
	defer destinationFile.Close()
	_, _ = io.Copy(destinationFile, sourceFile)

	templateFailContent, _ := utils.ReadFileNoLinks("./fb-cluster.yaml")
	text := string(templateFailContent)
	cfgupdatedtext := strings.ReplaceAll(text, "Record UUID {{ .Values.nodeUUID }}", "RecordUUID{{ .Values.nodeUUID }}")
	cfgupdatedtext = strings.ReplaceAll(cfgupdatedtext, "default", "fd")
	cfgupdatedtext = strings.ReplaceAll(cfgupdatedtext, "input", "fd")
	cfgupdatedtext = strings.ReplaceAll(cfgupdatedtext, "output", "fd")
	cfgupdatedtext = strings.ReplaceAll(cfgupdatedtext, "filter", "fd")
	udestinationFile, _ := os.Create("./fb-cluster.yaml")
	defer udestinationFile.Close()
	_, _ = udestinationFile.WriteString(cfgupdatedtext)

	resp.Cfg[0].Input = "opentelemetry"
	result, _ = logcfg.UpdateClusterLogConfig(context.Background(), resp, "./fb-cluster.yaml", false)
	assert.NotNil(t, result)

	// create no header
	text = string(templateFailContent)
	hfgupdatedtext := strings.ReplaceAll(text, "header", "fh")
	hdestinationFile, _ := os.Create("./fb-cluster-noheader.yaml")
	defer hdestinationFile.Close()
	_, _ = udestinationFile.WriteString(hfgupdatedtext)

	result, _ = logcfg.UpdateClusterLogConfig(context.Background(), resp, "./fb-cluster-noheader.yaml", false)
	assert.NotNil(t, result)
	os.Remove("fluentbit-cluster-tmp.conf")
	os.Remove("./fb-configmap.conf")
	os.Remove("./fb-cluster.yaml")
	os.Remove("./fb-cluster-noheader.yaml")
}

func TestErrorUpdateClusterLogConfig(t *testing.T) {
	logcfg.TmpFileDir = "./"
	logcfg.ConfigMapCommand = "kubectl get configmap fluent-bit-config -n observability -o jsonpath='{.data.fluent-bit\\.conf}'"
	resp := &pb.GetTelemetryConfigResponse{
		HostGuid:  "mock-host-guid",
		Timestamp: "2023-01-15T12:34:56+00:00",
		Cfg: []*pb.GetTelemetryConfigResponse_TelemetryCfg{
			{

				Input:    "opentelemetry",
				Type:     pb.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS,
				Kind:     pb.CollectorKind_COLLECTOR_KIND_HOST,
				Level:    pb.SeverityLevel_SEVERITY_LEVEL_INFO,
				Interval: 60,
			},
		},
	}

	result, _ := logcfg.UpdateClusterLogConfig(context.Background(), resp, "", false)
	assert.NotNil(t, result)

	os.Remove("fluentbit-cluster-tmp.conf")
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

	file, err := os.Create("fluenbit_unittest.conf")
	if err != nil {
		t.Error("Error create fluentbit_unittest.conf")
	}
	defer file.Close()
	fluentbitFilePath := "fluenbit_unittest.conf"
	err = createFileWithWrongContent("unittest_config.conf")
	if err != nil {
		t.Error("Error creating file:", err)
	}
	fluentbitHostPath := "unittest_config.conf"
	fluentbitClusterPath := "unittest_config.conf"

	result, _ := logcfg.UpdateHostLogConfig(context.Background(), resp, fluentbitFilePath, fluentbitHostPath, false)
	result1, _ := logcfg.UpdateClusterLogConfig(context.Background(), resp, fluentbitClusterPath, false)
	assert.NotNil(t, result)
	assert.NotNil(t, result1)
	os.Remove(fluentbitFilePath)
	os.Remove(fluentbitHostPath)
	os.Remove("fluentbit-cluster-tmp.conf")
}

func createFileWithWrongContent(filename string) error {
	content := `- key: "header"
	type: "service"
	tag: "service"
	multiline_data: |
	  [SERVICE]
		  flush        5
		  daemon       Off
		  log_level    info
		  storage.path              /var/log/edge-node/poa
		  storage.sync              normal
		  storage.checksum          off
		  storage.max_chunks_up     128 # 128 is a default value
		  storage.backlog.mem_limit 10M

- key: "systemd"
  type: "input"
  tag: "systemd"
  multiline_data: |
	[INPUT]
		Name           systemd
		Tag            Host_Systemd
		Mem_Buf_Limit  5MB
		Read_from_Head true
		Mem_Buf_Limit  5MB
		Skip_Long_Lines On
		Host         {{ .Values.Logging.URL }}


[OUTPUT]
    name https
    match *
    host  test.kind.internal
    port  123
    tls on
    tls.verify on
    Metrics_uri /test_host_logs
    Logs_uri /test_host_logs
    Traces_uri /test_host_logs
	`

	err := os.WriteFile(filename, []byte(content), 0600)
	if err != nil {
		return err
	}

	return nil
}

func TestErrorOnEmptyTemplate(t *testing.T) {
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

	errGoldTmpFile := "fluenbit_unittest.conf"
	file, err := os.Create(errGoldTmpFile)
	if err != nil {
		t.Error("Error create fluentbit_unittest.conf")
	}
	defer file.Close()

	fluentbitFilePath := "fluenbit_unittest.conf"
	result, _ := logcfg.UpdateHostLogConfig(context.Background(), resp, fluentbitFilePath, errGoldTmpFile, false)
	result1, _ := logcfg.UpdateClusterLogConfig(context.Background(), resp, errGoldTmpFile, false)
	assert.NotNil(t, result)
	assert.NotNil(t, result1)
	os.Remove(errGoldTmpFile)

}
