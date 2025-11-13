// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
package metriccfg

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/open-edge-platform/edge-node-agents/platform-telemetry-agent/internal/helper"
	"github.com/open-edge-platform/edge-node-agents/platform-telemetry-agent/internal/logger"

	"github.com/open-edge-platform/edge-node-agents/common/pkg/utils"
	pb "github.com/open-edge-platform/infra-managers/telemetry/pkg/api/telemetrymgr/v1"
)

var log = logger.Logger
var TmpFileDir = "/tmp/"
var TmpHostFile = "telegraf-tmp.conf"
var TmpClusterFile = "telegraf-tmp-cluster.conf"

type TelegrafTemplate struct {
	Key           string        `yaml:"key"`
	Type          string        `yaml:"type"`
	IsInit        bool          `yaml:"isInit"`
	DefaultValues MetricDefault `yaml:"chart_values"`
	MultilineData string        `yaml:"multiline_data"`
}

type MetricDefault struct {
	Keys map[string]string `yaml:",inline"`
}

func UpdateHostMetricConfig(ctx context.Context, cfg *pb.GetTelemetryConfigResponse, cfgFilePath string, cfgTemplatePath string, isInit bool) (bool, error) {

	log.Printf("Update Native Telegraf: %s", "Started")

	templateContent, err := utils.ReadFileNoLinks(cfgTemplatePath)
	if err != nil {
		log.Errorf("Error on reading telegraf config template: %s Err: %s", cfgTemplatePath, err)
		return false, err
	}

	var tlTemplate []TelegrafTemplate
	if err := yaml.Unmarshal(templateContent, &tlTemplate); err != nil {
		log.Errorf("Error on marshaling telegraf template: %s Err: %s", cfgTemplatePath, err)
		return false, err
	}

	cfgUpdatedStr, err := updateCfg(cfg, tlTemplate, pb.CollectorKind_COLLECTOR_KIND_HOST, isInit)
	if err != nil {
		log.Errorf("Error on updating telegraf config on latest changes: %s Err: %s", cfgTemplatePath, err)
		return false, err
	}

	//write latest update to temp file
	//tempFilePath := TmpFileDir + "telegraf-tmp.conf"
	tempFilePath := TmpFileDir + TmpHostFile
	err = saveTmpFile(cfgUpdatedStr, tempFilePath)
	if err != nil {
		log.Errorf("Error on writing telegraf temp file: %s Err: %s", cfgTemplatePath, err)
		return false, err
	}

	//this is to ensure could use sudo to replace the file in /etc/telegraf to avoid potential permission issue
	//sudo mv telegraf-tmp.conf  /etc/telegraf/telegraf.conf
	_, _, err = helper.RunExec(ctx, false, "sudo", "mv", tempFilePath, cfgFilePath)
	if err != nil {
		log.Errorf("Failed to move latest config file to destination: %s", err)
		return false, err
	}

	_, _, err = helper.RunExec(ctx, false, "sudo", "systemctl", "restart", "platform-observability-metrics")
	if err != nil {
		log.Errorf("Failed to update Telegraf config file: %s", err)
		return false, err
	}

	log.Printf("Update Native Telegraf %s", "Done")

	return true, nil
}

func UpdateClusterMetricConfig(ctx context.Context, cfg *pb.GetTelemetryConfigResponse, cfgTemplatePath string, isInit bool) (bool, error) {

	log.Printf("Cluster Telegraf Update: %s", "Started")

	templateContent, err := utils.ReadFileNoLinks(cfgTemplatePath)
	if err != nil {
		log.Errorf("Error on reading telegraf config template: %s Err: %s", cfgTemplatePath, err)
		return false, err
	}

	var tlTemplate []TelegrafTemplate
	if err := yaml.Unmarshal(templateContent, &tlTemplate); err != nil {
		log.Errorf("Error on marshaling telegraf template: %s Err: %s", cfgTemplatePath, err)
		return false, err
	}

	// Execute kubectl command directly to avoid issues with string field parsing
	// Build the full command args: sudo + kubectl command parts + kubectl args
	kubectlCmd := append([]string{"sudo"}, helper.KubectlArgs...)
	kubectlCmd = append(kubectlCmd, "get", "configmap", "telegraf-config", "-n", "observability", "-o", `jsonpath={.data.base-ext-telegraf\.conf}`)
	_, currConfigMap, err := helper.RunExec(ctx, false, kubectlCmd...)
	if err != nil {
		log.Errorf("Error on get telegraf configmap Err: %s", err)
		return false, err
	}

	updatedTLTemplate := updateTemplate(tlTemplate, currConfigMap)

	cfgUpdatedStr, err := updateCfg(cfg, updatedTLTemplate, pb.CollectorKind_COLLECTOR_KIND_CLUSTER, isInit)
	if err != nil {
		log.Errorf("Error on updating telegraf config on latest changes: %s Err: %s", cfgTemplatePath, err)
		return false, err
	}

	//write latest update to temp file
	//tempFilePath := TmpFileDir + "telegraf-tmp-cluster.conf"
	tempFilePath := TmpFileDir + TmpClusterFile
	err = saveTmpFile(cfgUpdatedStr, tempFilePath)
	if err != nil {
		log.Errorf("Error on writing telegraf temp file: %s Err: %s", cfgTemplatePath, err)
		return false, err
	}

	tempFileContent, err := utils.ReadFileNoLinks(tempFilePath)
	if err != nil {
		log.Errorf("Error reading telegraf temp config file: %s Err: %s", tempFilePath, err)
		return false, err
	}

	// Step 1: Construct the JSON payload for kubectl patch command
	jsonCfgChanged := map[string]interface{}{
		"data": map[string]interface{}{
			"telegraf.conf": string(tempFileContent),
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
	patchCmd = append(patchCmd, "patch", "configmap", "telegraf-config", "-p", string(jsonStr), "-n", "observability")
	_, _, err = helper.RunExec(ctx, false, patchCmd...)
	if err != nil {
		log.Errorf("Failed to update Telegraf configmap: %s", err)
		return false, err
	}

	// Build kubectl delete command
	deleteCmd := append([]string{"sudo"}, helper.KubectlArgs...)
	deleteCmd = append(deleteCmd, "delete", "pods", "-l", "app.kubernetes.io/name=telegraf", "-n", "observability")
	_, _, err = helper.RunExec(ctx, false, deleteCmd...)
	if err != nil {
		log.Errorf("Failed to restart Telegraf: %s", err)
		return false, err
	}

	log.Printf("Cluster Telegraf Update: %s", "Done")

	return true, nil
}

func updateTemplate(dataset []TelegrafTemplate, configmap string) []TelegrafTemplate {
	//split configmap to array
	configmapArray := helper.SplitStringAsSections("[", configmap)
	// loop data sets
	for dataIndex, data := range dataset {
		lines := strings.Split(data.MultilineData, "\n")
		for lineIndex, line := range lines {
			if strings.Contains(line, "{{") {
				editLine := replaceValue(line, helper.GetSectionBasedonKey(configmapArray, data.Key), data.DefaultValues)
				lines[lineIndex] = editLine
			}
		}
		dataset[dataIndex].MultilineData = strings.Join(lines, "\n")
	}
	return dataset
}

func replaceValue(line, configmap string, defaultVal MetricDefault) string {

	//get identifier
	result := ""
	re := regexp.MustCompile(`{{.*?}}`)
	parts := strings.Split(line, "=")
	indentifier := strings.TrimSpace(parts[0])
	confLines := strings.Split(configmap, "\n")
	for _, confLine := range confLines {
		if strings.Contains(confLine, indentifier) {
			configParts := strings.Split(confLine, "=")
			configMatching := strings.TrimSpace(configParts[0])
			if configMatching == strings.TrimSpace(confLine) {
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

func getDefaultValue(fullLine string, defaultVal MetricDefault) string {
	result := "error"
	for key, value := range defaultVal.Keys {
		if strings.Contains(fullLine, key) {
			return value
		}
	}

	return result
}

func getInputTemplateFromKey(key string, typ string, dataset []TelegrafTemplate) (string, error) {

	result := ""
	for _, entry := range dataset {
		if entry.Key == key && entry.Type == typ {
			result += entry.MultilineData + "\n"
		}
	}

	if result == "" {
		return result, fmt.Errorf("key=%s not found in template", key)
	}

	return result, nil
}

func getDefaultInputTemplate(typ string, dataset []TelegrafTemplate, isInit bool) (string, error) {
	result := ""
	for _, entry := range dataset {
		// get init value or default value
		if (!isInit && entry.Type == typ) || (isInit && entry.IsInit) {
			result += entry.MultilineData + "\n"
		}
	}

	if result == "" {
		return result, fmt.Errorf("type=%s not found in template", typ)
	}

	return result, nil
}

func saveTmpFile(cfgUpdatedStr string, tempFilePath string) error {

	file, err := os.Create(tempFilePath)
	if err != nil {
		log.Errorf("Error on creating telegraf temp file: %s Err: %s", tempFilePath, err)
		return err
	}
	defer file.Close()

	// Remove leading whitespaces to preserve original indentation
	cfgUpdatedStr = strings.TrimSpace(cfgUpdatedStr)
	_, err = file.WriteString(cfgUpdatedStr)
	if err != nil {
		log.Errorf("Error on writing telegraf temp file: %s Err: %s", tempFilePath, err)
		return err
	}

	return nil
}

func updateCfg(cfg *pb.GetTelemetryConfigResponse, tlTemplate []TelegrafTemplate, kind pb.CollectorKind, isInit bool) (string, error) {

	header := ""
	itxTemplate, err := getInputTemplateFromKey("agent", "global", tlTemplate)
	if err != nil {
		log.Errorf("Error on getting global from telegraf template: %s", err)
		return "", err
	} else {
		header += itxTemplate + "\n"
	}

	inputCfg := ""
	for _, itx := range cfg.Cfg {

		if itx.Type == pb.TelemetryResourceKind_TELEMETRY_RESOURCE_KIND_METRICS && itx.Kind == kind {
			itxTemplate, err := getInputTemplateFromKey(itx.Input, "input", tlTemplate)
			if err != nil {
				log.Infof("Input [%s] not found in telegraf template: %s", itx.Input, err)
			} else {
				itxTemplateEdit := addInterval(itxTemplate, strconv.FormatInt(itx.Interval, 10))
				inputCfg += itxTemplateEdit + "\n"
			}
		}
	}

	//Adding defaut input
	// add isInit checking
	itxTemplate, err = getDefaultInputTemplate("default", tlTemplate, isInit)
	if err != nil {
		log.Infof("Value [%s] not found in telegraf template: %s", "DEFAULT", err)
	} else {
		inputCfg += itxTemplate + "\n"
	}

	output := ""
	itxTemplate, err = getInputTemplateFromKey("output", "output", tlTemplate)
	if err != nil {
		log.Infof("Output not found in telegraf template: %s", err)
	} else {
		output += itxTemplate + "\n"
	}

	finalContent := header + inputCfg + output

	if finalContent == "" {
		return finalContent, fmt.Errorf("invalid update, empty content")
	}

	return finalContent, nil
}

func addInterval(cfgCurrentRaw, interval string) string {

	// Construct the interval configuration with the provided value
	intervalConfig := fmt.Sprintf("\n  interval = \"%sm\"\n", interval)

	// Insert the interval configuration after the [inputs] section
	return strings.TrimSpace(cfgCurrentRaw) + intervalConfig
}
