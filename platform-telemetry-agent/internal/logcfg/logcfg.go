// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
package logcfg

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/open-edge-platform/edge-node-agents/platform-telemetry-agent/internal/helper"
	"github.com/open-edge-platform/edge-node-agents/platform-telemetry-agent/internal/logger"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
	pb "github.com/open-edge-platform/infra-managers/telemetry/pkg/api/telemetrymgr/v1"
)

var log = logger.Logger
var TmpFileDir = "/tmp"
var FileOwner = "platform-telemetry-agent"

type FluentBitTemplate struct {
	Key           string     `yaml:"key"`
	Type          string     `yaml:"type"`
	Tag           string     `yaml:"tag"`
	IsInit        bool       `yaml:"isInit"`
	DefaultValues LogDefault `yaml:"chart_values"`
	MultilineData string     `yaml:"multiline_data"`
}

type LogDefault struct {
	Keys map[string]string `yaml:",inline"`
}

func UpdateHostLogConfig(ctx context.Context, cfg *pb.GetTelemetryConfigResponse, cfgFilePath string, cfgTemplatePath string, isInit bool) (bool, error) {

	log.Printf("Update Native Log Config: %s", "Started")

	templateContent, err := utils.ReadFileNoLinks(cfgTemplatePath)
	if err != nil {
		log.Errorf("Error on reading fluent-bit config template: %s Err: %s", cfgTemplatePath, err)
		return false, err
	}

	var fbTemplate []FluentBitTemplate
	if err := yaml.Unmarshal(templateContent, &fbTemplate); err != nil {
		log.Errorf("Error on marshaling fluent-bit template: %s Err: %s", cfgTemplatePath, err)
		return false, err
	}

	cfgUpdatedStr, err := updateCfg(cfg, fbTemplate, pb.CollectorKind_COLLECTOR_KIND_HOST, isInit)
	if err != nil {
		log.Errorf("Error on updating fluent-bit config on latest changes: %s Err: %s", cfgTemplatePath, err)
		return false, err
	}

	tempFilePath := TmpFileDir + "fluentbit-tmp.conf"
	err = saveTmpFile(cfgUpdatedStr, tempFilePath)
	if err != nil {
		log.Errorf("Error on writing fluent-bit temp file: %s Err: %s", cfgTemplatePath, err)
		return false, err
	}

	_, _, err = helper.RunExec(ctx, false, "sudo", "mv", tempFilePath, cfgFilePath)
	if err != nil {
		log.Errorf("Failed to copy latest config file to destination: %s", err)
		return false, err
	}

	_, _, err = helper.RunExec(ctx, false, "sudo", "chown", FileOwner, cfgFilePath)
	if err != nil {
		log.Errorf("Failed to change ownership of config file to %s: %s", FileOwner, err)
		return false, err
	}
	_, _, err = helper.RunExec(ctx, false, "sudo", "systemctl", "restart", "platform-observability-logging")
	if err != nil {
		log.Errorf("Failed to restart platform-observability-logging service: %s", err)
		return false, err
	}

	log.Printf("Update Native Log Config: %s", "Done")

	return true, nil
}

func UpdateClusterLogConfig(ctx context.Context, cfg *pb.GetTelemetryConfigResponse, cfgTemplatePath string, isInit bool) (bool, error) {

	log.Printf("Update Cluster Log Config: %s", "Started")

	templateContent, err := utils.ReadFileNoLinks(cfgTemplatePath)
	if err != nil {
		log.Errorf("Error on reading fluent-bit config template: %s Err: %s", cfgTemplatePath, err)
		return false, err
	}

	var fbTemplate []FluentBitTemplate
	if err := yaml.Unmarshal(templateContent, &fbTemplate); err != nil {
		log.Errorf("Error on marshaling fluent-bit template: %s Err: %s", cfgTemplatePath, err)
		return false, err
	}

	// Execute kubectl command directly to avoid issues with string field parsing
	// Build the full command args: sudo + kubectl command parts + kubectl args
	kubectlCmd := append([]string{"sudo"}, helper.KubectlArgs...)
	kubectlCmd = append(kubectlCmd, "get", "configmap", "fluent-bit", "-n", "observability", "-o", `jsonpath={.data.fluent-bit\.conf}`)
	_, currConfigMap, err := helper.RunExec(ctx, false, kubectlCmd...)
	if err != nil {
		log.Errorf("Error on get fluent-bit configmap Err: %s", err)
		return false, err
	}

	updatedFBTemplate := updateTemplate(fbTemplate, currConfigMap)

	cfgUpdatedStr, err := updateCfg(cfg, updatedFBTemplate, pb.CollectorKind_COLLECTOR_KIND_CLUSTER, isInit)
	if err != nil {
		log.Errorf("Error on updating fluent-bit config on latest changes: %s Err: %s", cfgTemplatePath, err)
		return false, err
	}

	tempFilePath := TmpFileDir + "fluentbit-cluster-tmp.conf"
	err = saveTmpFile(cfgUpdatedStr, tempFilePath)
	if err != nil {
		log.Errorf("Error on writing fluent-bit temp file: %s Err: %s", cfgTemplatePath, err)
		return false, err
	}

	tempFileContent, err := utils.ReadFileNoLinks(tempFilePath)
	if err != nil {
		log.Errorf("Error reading Fluentbit temp config file: %s Err: %s", tempFilePath, err)
		return false, err
	}

	// Step 1: Construct the JSON payload for kubectl patch command
	jsonCfgChanged := map[string]interface{}{
		"data": map[string]interface{}{
			"fluent-bit.conf": string(tempFileContent),
		},
	}

	// Step 3: Convert the JSON payload to a string
	jsonStr, err := json.Marshal(jsonCfgChanged)
	if err != nil {
		log.Errorf("Error marshaling JSON: %s", err)
		return false, err
	}

	// Build kubectl patch command
	patchCmd := append([]string{"sudo"}, helper.KubectlArgs...)
	patchCmd = append(patchCmd, "patch", "configmap", "fluent-bit-config", "-p", string(jsonStr), "-n", "observability")
	_, _, err = helper.RunExec(ctx, false, patchCmd...)
	if err != nil {
		log.Errorf("Failed to update fluent-bit-config configmap: %s", err)
		return false, err
	}

	// Build kubectl delete command
	deleteCmd := append([]string{"sudo"}, helper.KubectlArgs...)
	deleteCmd = append(deleteCmd, "delete", "pods", "-l", "app.kubernetes.io/name=fluent-bit", "-n", "observability")
	_, _, err = helper.RunExec(ctx, false, deleteCmd...)
	if err != nil {
		log.Errorf("Failed to restart Telegraf: %s", err)
		return false, err
	}

	log.Printf("Update Cluster Log Config: %s", "Done")

	return true, nil
}

func updateTemplate(dataset []FluentBitTemplate, configmap string) []FluentBitTemplate {
	// loop data sets
	for dataIndex, data := range dataset {
		lines := strings.Split(data.MultilineData, "\n")
		for lineIndex, line := range lines {
			if strings.Contains(line, "{{") {
				editLine := replaceValue(line, configmap, data.DefaultValues)
				lines[lineIndex] = editLine
			}
		}
		dataset[dataIndex].MultilineData = strings.Join(lines, "\n")
	}
	return dataset
}

func replaceValue(line, configmap string, defaultVal LogDefault) string {
	//get identifier
	result := ""
	re := regexp.MustCompile(`{{.*?}}`)
	indentifier, index, err := extractIdentifier(line)
	if err != nil {
		return re.ReplaceAllString(line, getDefaultValue(line, defaultVal))
	}
	confLines := strings.Split(configmap, "\n")
	for _, confLine := range confLines {

		if strings.Contains(confLine, indentifier) {
			configMatching := extractIdentifierByNumber(confLine, index)
			if configMatching == confLine {
				result := re.ReplaceAllString(line, getDefaultValue(line, defaultVal))
				return result
			}
			if configMatching == indentifier {
				result = confLine
				return result
			}
		}
	}
	result = re.ReplaceAllString(line, getDefaultValue(line, defaultVal))
	return result
}

func extractIdentifier(fullLine string) (string, int, error) {
	identifier := ""
	lineSplit := strings.Split(fullLine, " ")
	for index, line := range lineSplit {
		if line == "{{" {
			//return identifier
			return identifier, index, nil
		}
		if line != "" {
			identifier += line + " "
		}
	}
	return fullLine, 0, fmt.Errorf("identifier not found in line: %s", fullLine)
}

func extractIdentifierByNumber(fullLine string, idLocation int) string {
	identifier := ""
	lineSplit := strings.Split(fullLine, " ")
	for index, line := range lineSplit {
		if index == idLocation {
			return identifier
		}
		if line != "" {
			identifier += line + " "
		}
	}

	return fullLine
}

func getDefaultValue(fullLine string, defaultVal LogDefault) string {
	result := "error"
	for key, value := range defaultVal.Keys {
		if strings.Contains(fullLine, key) {
			return value
		}
	}

	return result
}

func getInputTemplateFromKey(key string, typ string, dataset []FluentBitTemplate) (string, string, error) {

	var tag []string
	result := ""
	filter := ""
	for _, entry := range dataset {
		if entry.Key == key && entry.Type == typ {
			result += entry.MultilineData + "\n"
			tag = append(tag, entry.Tag)
		}
	}

	for _, entry := range dataset {
		for _, val := range tag {
			if entry.Tag == val && entry.Type == "filter" {
				filter += entry.MultilineData + "\n"
			}
		}
	}

	if result == "" && typ != "filter" {
		return result, filter, fmt.Errorf("key=%s not found in template", key)
	}

	return result, filter, nil
}

func getDefaultInputTemplate(typ string, dataset []FluentBitTemplate, isInit bool) (string, string, error) {

	var tag []string
	result := ""
	filter := ""
	for _, entry := range dataset {
		if (!isInit && entry.Type == typ) || (isInit && entry.IsInit) {
			result += entry.MultilineData + "\n"
			tag = append(tag, entry.Tag)
		}
	}

	for _, entry := range dataset {
		for _, val := range tag {
			if entry.Tag == val && entry.Type == "filter" {
				filter += entry.MultilineData + "\n"
			}
		}
	}

	if result == "" && typ != "filter" {
		return result, filter, fmt.Errorf("key=%s not found in template", typ)
	}

	return result, filter, nil
}

func saveTmpFile(cfgUpdatedStr string, tempFilePath string) error {

	file, err := os.Create(tempFilePath)
	if err != nil {
		log.Errorf("Error on creating fluent-bit temp file: %s Err: %s", tempFilePath, err)
		return err
	}
	defer file.Close()

	// Remove leading whitespaces to preserve original indentation
	cfgUpdatedStr = strings.TrimSpace(cfgUpdatedStr)
	_, err = file.WriteString(cfgUpdatedStr)
	if err != nil {
		log.Errorf("Error on writing fluent-bit temp file: %s Err: %s", tempFilePath, err)
		return err
	}

	return nil
}

func updateCfg(cfg *pb.GetTelemetryConfigResponse, fbTemplate []FluentBitTemplate, kind pb.CollectorKind, isInit bool) (string, error) {

	header := ""
	itxTemplate, _, err := getInputTemplateFromKey("header", "service", fbTemplate)
	if err != nil {
		log.Errorf("Error on getting header from fluent-bit template: %s", err)
		return "", err
	} else {
		header += itxTemplate + "\n"
	}

	inputCfg := ""
	filterCfg := ""
	for _, itx := range cfg.Cfg {

		if itx.Type == pb.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_LOGS && itx.Kind == kind {
			itxTemplate, itxFilter, err := getInputTemplateFromKey(itx.Input, "input", fbTemplate)
			if err != nil {
				log.Infof("Input [%s] not found in fluent-bit template: %s", itx.Input, err)
			} else {
				itxTemplateEdit := addLogLevel(itxTemplate, itx.Level)
				inputCfg += itxTemplateEdit + "\n"
				filterCfg += itxFilter + "\n"
			}

		}
	}

	//Adding defaut input
	itxTemplate, itxFilter, err := getDefaultInputTemplate("default", fbTemplate, isInit)
	if err != nil {
		log.Infof("Value [%s] not found in fluent-bit template: %s", "DEFAULT", err)
	} else {
		inputCfg += itxTemplate + "\n"
		filterCfg += itxFilter + "\n"
	}

	filterAll := ""
	itxTemplate, _, err = getInputTemplateFromKey("filter_all", "filter", fbTemplate)
	if err != nil {
		log.Infof("No filter found in fluent-bit template: %s", err)
	} else if itxTemplate != "" {
		filterAll += itxTemplate + "\n"
	}

	output := ""
	itxTemplate, _, err = getInputTemplateFromKey("output", "output", fbTemplate)
	if err != nil {
		log.Infof("Output not found in fluent-bit template: %s", err)
	} else {
		output += itxTemplate + "\n"
	}

	finalContent := header + inputCfg + filterCfg + filterAll + output

	if finalContent == "" {
		return finalContent, fmt.Errorf("invalid update, empty content")
	}

	return finalContent, nil
}

func addLogLevel(cfgCurrentRaw string, loglevel pb.SeverityLevel) string {
	var level string
	switch int32(loglevel) {
	case int32(pb.SeverityLevel_SEVERITY_LEVEL_UNSPECIFIED):
		level = "info"
	case int32(pb.SeverityLevel_SEVERITY_LEVEL_CRITICAL):
		level = "info"
	case int32(pb.SeverityLevel_SEVERITY_LEVEL_ERROR):
		level = "error"
	case int32(pb.SeverityLevel_SEVERITY_LEVEL_WARN):
		level = "warn"
	case int32(pb.SeverityLevel_SEVERITY_LEVEL_INFO):
		level = "info"
	case int32(pb.SeverityLevel_SEVERITY_LEVEL_DEBUG):
		level = "debug"
	default:
		level = "info"
	}
	// Construct the loglevel configuration with the provided value
	logLevelConfig := fmt.Sprintf("\n    Log_Level %s\n", level)

	// Insert the loglevel configuration after the [inputs] section
	return strings.TrimSpace(cfgCurrentRaw) + logLevelConfig
}
