package configuration

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

var (
	UNKNOWN = map[string]interface{}{"rc": 1, "message": "Unknown command invoked"}
	RESP_OK = map[string]interface{}{"rc": 0, "message": "Command Success"}
)

type Broker struct {
	keyValueStore IKeyValueStore
	mqttClient    *MQTTClient
}

func NewBroker(keyValueStore IKeyValueStore, useTLS bool) *Broker {
	mqttClient := NewMQTTClient(
		"configuration-agent",
		"localhost",
		1883,
		60,
		true,
		useTLS,
		"/path/to/ca.crt",
		"/path/to/client.crt",
		"/path/to/client.key",
	)

	return &Broker{
		keyValueStore: keyValueStore,
		mqttClient:    mqttClient,
	}
}

func (b *Broker) onMessage(client MQTT.Client, msg MQTT.Message) {
	log.Printf("Message received: %s on topic: %s", msg.Payload(), msg.Topic())
}

func (b *Broker) onCommand(client MQTT.Client, msg MQTT.Message) {
	var request map[string]interface{}
	if err := json.Unmarshal(msg.Payload(), &request); err != nil {
		log.Printf("Unable to parse command/request ID. Verify request is in the correct format. %v", err)
		return
	}

	log.Printf("Received command request: %v on topic: %s", request, msg.Topic())
	b.execute(request)
}

func (b *Broker) execute(request map[string]interface{}) {
	requestID := request["id"]
	resp := map[string]interface{}{"rc": 0, "message": RESP_OK}

	defer func() {
		response, _ := json.Marshal(resp)
		b.mqttClient.Publish(fmt.Sprintf("%s%s", RESPONSE_CHANNEL, requestID), response, 0, false)
	}()

	command, headers, path, value, valueString, err := b.parseRequest(request)
	if err != nil {
		log.Printf("Error parsing request: %v", err)
		resp = map[string]interface{}{"rc": 1, "message": err.Error()}
		return
	}

	if command == "get_element" || command == "set_element" || command == "append" || command == "remove" {
		if valueString != "" {
			headers = b.getParent(valueString)
		} else if path != "" {
			headers = b.getParent(path)
		}
	}

	log.Printf("command: %s, headers: %s, path: %s, value: %s, valueString: %s", command, headers, path, value, valueString)

	switch command {
	case "get_element":
		var err error
		resp["data"], err = b.getElementName(&headers, &path, &valueString)
		if err != nil {
			log.Printf("Error getting element name: %v", err)
			resp = map[string]interface{}{"rc": 1, "message": err.Error()}
		}
	case "set_element":
		b.setElementName(headers, path, value, valueString)
	case "load":
		b.load(path)
	case "append":
		b.append(headers, path, value, valueString)
	case "remove":
		b.remove(headers, path, value, valueString)
	default:
		log.Printf("Unknown command: %s invoked", command)
		resp = map[string]interface{}{"rc": 1, "message": UNKNOWN}
	}
}

func (b *Broker) parseRequest(request map[string]interface{}) (string, string, string, string, string, error) {
	command := request["cmd"].(string)
	path := ""
	if p, ok := request["path"]; ok {
		path = p.(string)
	}
	value := ""
	if v, ok := request["value"]; ok {
		value = v.(string)
	}
	headers := ""
	if h, ok := request["headers"]; ok {
		headers = h.(string)
	}
	valueString := ""
	if vs, ok := request["valueString"]; ok {
		valueString = vs.(string)
	}
	return command, headers, path, value, valueString, nil
}

func (b *Broker) getParent(valueString string) string {
	valueString = strings.Split(valueString, ":")[0]
	parent, err := b.keyValueStore.GetParent(valueString)
	if err != nil {
		log.Printf("Error getting parent: %v", err)
		return ""
	}
	return parent
}

func (b *Broker) getElementName(headers, path, valueString *string) (string, error) {
	if headers != nil && path != nil {
		if *path == ORCHESTRATOR {
			element, err := b.keyValueStore.GetElement(fmt.Sprintf("%s/%s", *headers, *path), nil, true)
			if err != nil {
				return "", err
			}
			return element, nil
		}
		element, err := b.keyValueStore.GetElement(fmt.Sprintf("%s/%s", *headers, *path), nil, false)
		if err != nil {
			return "", err
		}
		return element, nil
	}
	if headers != nil && valueString != nil {
		if strings.Split(*valueString, ":")[0] == ORCHESTRATOR {
			return b.keyValueStore.GetElement(*headers, valueString, true)
		}
		return b.keyValueStore.GetElement(*headers, valueString, false)
	}
	return "", fmt.Errorf("Invalid request: no path or header")
}

func (b *Broker) setElementName(headers, path, value, valueString string) {
	if value == "" && valueString == "" {
		log.Println("Value was not set")
		return
	}

	if path != "" && value != "" {
		b.keyValueStore.SetElement(path, value, nil, false)
		b.mqttClient.Publish(UPDATE_CHANNEL+path, value, 0, false)
	} else if headers != "" && valueString != "" {
		var paths string
		if strings.Split(valueString, ":")[0] == ORCHESTRATOR {
			paths, _ = b.keyValueStore.SetElement(headers, "", &valueString, true)
		} else {
			paths, _ = b.keyValueStore.SetElement(headers, "", &valueString, false)
		}
		b.publishNewValues(paths)
	} else {
		log.Println("Invalid parameters sent")
	}
}

func (b *Broker) load(path string) {
	b.keyValueStore.Load(path)
	b.PublishInitialValues()
}

func (b *Broker) append(headers, path, value, valueString string) {
	if value == "" && valueString == "" {
		log.Println("Value and value string are not set")
		return
	}

	if path != "" && value != "" {
		if _, err := b.keyValueStore.Append(path, value); err == nil {
			b.mqttClient.Publish(UPDATE_CHANNEL+path, value, 0, false)
		}
	} else if headers != "" && valueString != "" {
		paths, _ := b.keyValueStore.Append(headers, valueString)
		b.publishNewValues(paths)
	} else {
		log.Println("Invalid parameters sent to append")
	}
}

func (b *Broker) remove(headers, path, value, valueString string) {
	if value == "" && valueString == "" {
		log.Println("Please specify the element value")
		return
	}

	if path != "" && value != "" {
		if _, err := b.keyValueStore.Remove(path, &value, nil); err == nil {
			b.mqttClient.Publish(UPDATE_CHANNEL+path, value, 0, false)
		}
	} else if headers != "" && valueString != "" {
		paths, err := b.keyValueStore.Remove(headers, &valueString, nil)
		if err == nil {
			b.publishNewValues(paths)
		}
	} else {
		log.Println("Invalid parameters sent to remove")
	}
}

func (b *Broker) publishNewValues(paths string) {
	pathList := strings.Split(paths, ";")
	for _, path := range pathList {
		if path == "" {
			continue
		}
		listObj := strings.SplitN(path, ":", 2)
		value := listObj[1]
		path := listObj[0]
		log.Printf("Publishing new value on: %s%s:%s", UPDATE_CHANNEL, path, value)
		b.mqttClient.Publish(UPDATE_CHANNEL+path, value, 0, false)
	}
}

func (b *Broker) PublishInitialValues() {
	agents := []string{"agent1", "agent2", "agent3"} // Replace with actual agent list
	for _, agent := range agents {
		b.publishAgentValues(agent)
	}
}

func (b *Broker) publishAgentValues(agent string) {
	children, _ := b.keyValueStore.GetChildren(agent)
	for child, value := range children {
		path := agent + "/" + child
		log.Printf("Publishing initial agent value on: %s%s:%s", UPDATE_CHANNEL, path, value)
		b.mqttClient.Publish(UPDATE_CHANNEL+path, value, 0, true)
	}
}

func (b *Broker) brokerStop() {
	b.mqttClient.Stop()
}
