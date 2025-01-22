package configuration

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	// SCHEMA_LOCATION = "/home/intel/rishi/inbm/intel-inb-mgm/temp/iotg_inb_schema.xsd"
	IOTG_INB_CONF                = "/home/intel/rishi/inbm/intel-inb-mgm/temp/inb_config_schema.xsd"
	INVALID_SCHEMA_FILE_LOCATION = "/etc/intel-manageability/intel_manageability.conf"
)

var (
	GOOD_XML = `<?xml version="1.0" encoding="UTF-8"?>
    <configurations><all><dbs>ON</dbs></all>
    <telemetry><collectionIntervalSeconds>60</collectionIntervalSeconds>
    <publishIntervalSeconds>300</publishIntervalSeconds><maxCacheSize>100</maxCacheSize>
    <containerHealthIntervalSeconds>600</containerHealthIntervalSeconds>
    </telemetry><diagnostic><minStorageMB>100</minStorageMB><minMemoryMB>200</minMemoryMB>
    <minPowerPercent>20</minPowerPercent><sotaSW>docker
    </sotaSW></diagnostic><dispatcher><dbsRemoveImageOnFailedContainer>true</dbsRemoveImageOnFailedContainer>
    <trustedRepositories>https://sample</trustedRepositories>
    </dispatcher><orchestrator name="csl-agent"><orchestratorResponse>true</orchestratorResponse>
    <ip>/etc/ip</ip><token>/etc/token</token><certFile>/etc/pem</certFile></orchestrator><sota>
    <ubuntuAptSource>https://sample2</ubuntuAptSource>
    <proceedWithoutRollback>false</proceedWithoutRollback></sota></configurations>`

	BAD_XML = `<?xml version="1.0" encoding="UTF-8"?>
    <configurations><all><dbs>ON</dbs></all>
    <telemetry><collectionIntervalSeconds>60</collectionIntervalSeconds>
    <publishIntervalSeconds>300</publishIntervalSeconds><maxCacheSize>100</maxCacheSize>
    <containerHealthIntervalSeconds>600</containerHealthIntervalSeconds>
    </telemetry><diagnostic><minStorageMB>100</minStorageMB><minMemoryMB>200</minMemoryMB>
    <minPowerPercent>20<sotaSW>docker
    </sotaSW></diagnostic><dispatcher><dbsRemoveImageOnFailedContainer>
    true</dbsRemoveImageOnFailedContainer>
    <trustedRepositories>https://sample</trustedRepositories></dispatcher>
    <sota><ubuntuAptSource>https://sample2</ubuntuAptSource>
    <proceedWithoutRollback>false</proceedWithoutRollback></sota></configurations>`

	INVALID_XML = `<?xml version="1.0" encoding="UTF-8"?>
    <configurations><all><dbs>ON</dbs></all>
    <telemetry><collectionIntervalSeconds>60</collectionIntervalSeconds>
    <publishIntervalSeconds>300</publishIntervalSeconds><maxCacheSize>100</maxCacheSize>
    <containerHealthIntervalSeconds>600</containerHealthIntervalSeconds>
    </telemetry><diagnostic><minStorageMB>100</minStorageMB><minMemoryMB>200</minMemoryMB>
    <minPowerPercent>20</minPowerPercent><sotaSW>docker
    </sotaSW></diagnostic><dispatcher><trustedRepositories>
    https://sample</trustedRepositories></dispatcher>
    <sota><ubuntuAptSource>https://sample2</ubuntuAptSource>
    <proceedWithoutRollback>false</proceedWithoutRollbacks></sota></configurations>`
)

func TestXmlParser(t *testing.T) {
	t.Run("test_parser_creation_success", func(t *testing.T) {
		xmlStore, err := NewXmlKeyValueStore(GOOD_XML, false, SCHEMA_LOCATION)
		assert.NoError(t, err)
		assert.NotNil(t, xmlStore)
	})

	t.Run("test_parser_creation_failure", func(t *testing.T) {
		_, err := NewXmlKeyValueStore(BAD_XML, false, SCHEMA_LOCATION)
		assert.Error(t, err)
	})

	t.Run("test_xsd_validation_failure", func(t *testing.T) {
		_, err := NewXmlKeyValueStore(INVALID_XML, false, SCHEMA_LOCATION)
		assert.Error(t, err)
	})

	t.Run("test_invalid_schema_file_path_failure", func(t *testing.T) {
		_, err := NewXmlKeyValueStore(INVALID_SCHEMA_FILE_LOCATION, true, SCHEMA_LOCATION)
		assert.Error(t, err)
	})

	t.Run("test_validate_intel_manageability_conf", func(t *testing.T) {
		_, err := NewXmlKeyValueStore(IOTG_INB_CONF, true, SCHEMA_LOCATION)
		assert.NoError(t, err)
	})

	t.Run("test_get_element", func(t *testing.T) {
		xmlStore, err := NewXmlKeyValueStore(GOOD_XML, false, SCHEMA_LOCATION)
		assert.NoError(t, err)
		element, err := xmlStore.GetElement("telemetry/maxCacheSize", nil, false)
		assert.NoError(t, err)
		assert.Equal(t, "100", element)
	})

	t.Run("test_set_element", func(t *testing.T) {
		xmlStore, err := NewXmlKeyValueStore(GOOD_XML, false, SCHEMA_LOCATION)
		assert.NoError(t, err)
		_, err = xmlStore.SetElement("telemetry/maxCacheSize", "200", nil, false)
		assert.NoError(t, err)
		element, err := xmlStore.GetElement("telemetry/maxCacheSize", nil, false)
		assert.NoError(t, err)
		assert.Equal(t, "200", element)
	})

	t.Run("test_get_children", func(t *testing.T) {
		xmlStore, err := NewXmlKeyValueStore(GOOD_XML, false, SCHEMA_LOCATION)
		assert.NoError(t, err)
		children, err := xmlStore.GetChildren("diagnostic")
		assert.NoError(t, err)
		expected := map[string]string{
			"minMemoryMB":     "200",
			"minPowerPercent": "20",
			"minStorageMB":    "100",
			"sotaSW":          "docker",
		}
		assert.Equal(t, expected, children)
	})

	t.Run("test_get_parent_success", func(t *testing.T) {
		xmlStore, err := NewXmlKeyValueStore(GOOD_XML, false, SCHEMA_LOCATION)
		assert.NoError(t, err)
		parent, err := xmlStore.GetParent("maxCacheSize")
		assert.NoError(t, err)
		assert.Equal(t, "telemetry", parent)
	})
}
