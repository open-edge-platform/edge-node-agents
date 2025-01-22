/*
   MQTT Configuration variables

   @copyright: Copyright 2017-2024 Intel Corporation All Rights Reserved.
   @license: Intel, see licenses/LICENSE for more details.
*/

package configuration

import (
	"path/filepath"
)

// MQTT connection variables
const (
	DEFAULT_MQTT_HOST       = "localhost"
	DEFAULT_MQTT_PORT       = 8883
	MQTT_KEEPALIVE_INTERVAL = 60
)

var (
	//BROKER_ETC_PATH    = "/path/to/broker/etc" // Update this path as needed
	DEFAULT_MQTT_CERTS = filepath.Join(BROKER_ETC_PATH, "public", "mqtt-ca", "mqtt-ca.crt")
)
