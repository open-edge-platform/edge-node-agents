package configuration

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

const (
	MQTT_HOST_ENV         = "MQTT_HOST"
	MQTT_HOST_PORT_ENV    = "MQTT_PORT"
	MQTT_CA_CERTS_ENV     = "MQTT_CA_CERTS"
	MQTT_CLIENT_CERTS_ENV = "MQTT_CLIENT_CERTS"
	MQTT_CLIENT_KEYS_ENV  = "MQTT_CLIENT_KEYS"
)

type MQTTClient struct {
	ClientID    string
	Broker      string
	Port        int
	KeepAlive   int
	EnvConfig   bool
	TLS         bool
	CACerts     string
	ClientCerts string
	ClientKeys  string
	Client      MQTT.Client
	midToTopic  map[int]string
	lock        sync.Mutex
	topics      map[string]MQTT.MessageHandler
}

func NewMQTTClient(clientID, broker string, port, keepAlive int, envConfig, tls bool, caCerts, clientCerts, clientKeys string) *MQTTClient {
	var mqttHost, mqttCACerts, mqttClientCerts, mqttClientKeys string
	var mqttPort int

	if envConfig {
		mqttHost = getEnvOrDefault(MQTT_HOST_ENV, broker)
		mqttPort = getEnvOrDefaultInt(MQTT_HOST_PORT_ENV, port)
		mqttCACerts = getEnvOrDefault(MQTT_CA_CERTS_ENV, caCerts)
		mqttClientCerts = getEnvOrDefault(MQTT_CLIENT_CERTS_ENV, clientCerts)
		mqttClientKeys = getEnvOrDefault(MQTT_CLIENT_KEYS_ENV, clientKeys)
	} else {
		mqttHost = broker
		mqttPort = port
		mqttCACerts = caCerts
		mqttClientCerts = clientCerts
		mqttClientKeys = clientKeys
	}

	mqttClient := &MQTTClient{
		ClientID:    clientID,
		Broker:      mqttHost,
		Port:        mqttPort,
		KeepAlive:   keepAlive,
		EnvConfig:   envConfig,
		TLS:         tls,
		CACerts:     mqttCACerts,
		ClientCerts: mqttClientCerts,
		ClientKeys:  mqttClientKeys,
		midToTopic:  make(map[int]string),
		topics:      make(map[string]MQTT.MessageHandler),
	}

	opts := MQTT.NewClientOptions().AddBroker(fmt.Sprintf("tcp://%s:%d", mqttHost, mqttPort)).SetClientID(clientID)
	opts.SetKeepAlive(time.Duration(keepAlive) * time.Second)

	if tls {
		tlsConfig, err := newTLSConfig(mqttCACerts, mqttClientCerts, mqttClientKeys)
		if err != nil {
			log.Fatalf("Failed to create TLS config: %v", err)
		}
		opts.SetTLSConfig(tlsConfig)
	}

	mqttClient.Client = MQTT.NewClient(opts)

	mqttClient.Client.AddRoute("#", mqttClient.onSubscribe)

	if token := mqttClient.Client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", token.Error())
	}

	log.Printf("Connected to MQTT broker: %s on port: %d", mqttHost, mqttPort)

	return mqttClient
}

func (m *MQTTClient) onSubscribe(client MQTT.Client, message MQTT.Message) {
	log.Printf("Subscribed to topic: %s", message.Topic())
	if qos := message.Qos(); qos == 128 {
		log.Printf("Failed to subscribe to topic: %s", message.Topic())
	}

	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.midToTopic, int(message.MessageID()))
}

func (m *MQTTClient) LoopOnce(timeout time.Duration, maxPackets int) {
	// Process network events for the specified timeout duration
	time.Sleep(timeout)
	if !m.Client.IsConnected() {
		log.Println("Client is not connected")
	}
}

func (m *MQTTClient) Start() {
	// m.Client.Start() is not needed as the client starts automatically upon connection
}

func (m *MQTTClient) Stop() {
	m.Client.Disconnect(250)
	log.Println("Disconnected from MQTT broker")
}

func (m *MQTTClient) Publish(topic string, payload interface{}, qos byte, retain bool) {
	token := m.Client.Publish(topic, qos, retain, payload)
	token.Wait()
	log.Printf("Published message: %s on topic: %s with retain: %t", payload, topic, retain)
}

func (m *MQTTClient) PublishAndWaitResponse(topic, responseTopic, payload string, timeoutSeconds int) (string, error) {
	response := ""
	responseReceived := make(chan bool)

	m.Client.Subscribe(responseTopic, 0, func(client MQTT.Client, message MQTT.Message) {
		response = string(message.Payload())
		responseReceived <- true
	})

	m.Publish(topic, payload, 0, false)

	select {
	case <-responseReceived:
		log.Printf("Received response: %s on topic: %s", response, responseTopic)
	case <-time.After(time.Duration(timeoutSeconds) * time.Second):
		return "", fmt.Errorf("no response received within %d seconds", timeoutSeconds)
	}

	m.Client.Unsubscribe(responseTopic)
	return response, nil
}

func (m *MQTTClient) Subscribe(topic string, callback MQTT.MessageHandler, qos byte) {
	if _, ok := m.topics[topic]; ok {
		log.Printf("Topic: %s has already been subscribed to", topic)
		return
	}

	token := m.Client.Subscribe(topic, qos, callback)
	token.Wait()
	m.topics[topic] = callback
	log.Printf("Subscribed to topic: %s", topic)
}

func (m *MQTTClient) Unsubscribe(topic string) {
	token := m.Client.Unsubscribe(topic)
	token.Wait()
	delete(m.topics, topic)
	log.Printf("Unsubscribed from topic: %s", topic)
}

func getEnvOrDefault(envKey, defaultValue string) string {
	if value, exists := os.LookupEnv(envKey); exists {
		return value
	}
	return defaultValue
}

func getEnvOrDefaultInt(envKey string, defaultValue int) int {
	if value, exists := os.LookupEnv(envKey); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func newTLSConfig(caCerts, clientCerts, clientKeys string) (*tls.Config, error) {
	certpool := x509.NewCertPool()
	pemCerts, err := ioutil.ReadFile(caCerts)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certs: %w", err)
	}
	certpool.AppendCertsFromPEM(pemCerts)

	cert, err := tls.LoadX509KeyPair(clientCerts, clientKeys)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certs: %w", err)
	}

	return &tls.Config{
		RootCAs:            certpool,
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
		},
	}, nil
}
