package configuration

import (
	"path/filepath"
)

// Constants and other config variables used throughout the configuration module

const (
	AGENT            = "configuration"
	STATE_CHANNEL    = "+/state"
	COMMAND_CHANNEL  = "configuration/command/+"
	UPDATE_CHANNEL   = "configuration/update/"
	RESPONSE_CHANNEL = "configuration/response/"

	ORCHESTRATOR = "orchestrator"
	ATTRIB_NAME  = "name"
)

var (
	INTEL_MANAGEABILITY_ETC_PATH_PREFIX   = "/path/to/intel_manageability_etc"   // Update this path as needed
	INTEL_MANAGEABILITY_RAW_ETC           = "/path/to/intel_manageability_raw"   // Update this path as needed
	INTEL_MANAGEABILITY_SHARE_PATH_PREFIX = "/path/to/intel_manageability_share" // Update this path as needed
	BROKER_ETC_PATH                       = "/path/to/broker_etc"                // Update this path as needed

	DEFAULT_LOGGING_PATH = filepath.Join(INTEL_MANAGEABILITY_ETC_PATH_PREFIX, "public", "configuration-agent", "logging.ini")

	// SCHEMA_LOCATION        = filepath.Join(INTEL_MANAGEABILITY_SHARE_PATH_PREFIX, AGENT+"-agent", "iotg_inb_schema.xsd")
	SCHEMA_LOCATION        = "/home/intel/rishi/inbm/intel-inb-mgm/temp/iotg_inb_schema.xsd"
	CONFIG_SCHEMA_LOCATION = filepath.Join(INTEL_MANAGEABILITY_SHARE_PATH_PREFIX, AGENT+"-agent", "inb_config_schema.xsd")
	XML_LOCATION           = filepath.Join(INTEL_MANAGEABILITY_RAW_ETC, "intel_manageability.conf")
	CONFIG_LOCATION        = filepath.Join(INTEL_MANAGEABILITY_RAW_ETC, "tc_config.conf")

	CLIENT_CERTS = filepath.Join(BROKER_ETC_PATH, "public", "configuration-agent", "configuration-agent.crt")
	CLIENT_KEYS  = filepath.Join(BROKER_ETC_PATH, "secret", "configuration-agent", "configuration-agent.key")

	AGENTS = []string{"diagnostic", "telemetry", "dispatcher", "sota", "all"}
)
