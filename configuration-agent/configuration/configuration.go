package configuration

import (
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

type Configuration struct {
	WindowsService
	argv    []string
	running bool
}

func NewConfiguration(args []string) *Configuration {
	if args == nil {
		args = []string{}
	}
	return &Configuration{
		argv:           args,
		running:        false,
		WindowsService: *NewWindowsService(),
	}
}

func (c *Configuration) setUpSignalHandlers(handler func(os.Signal)) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		for {
			sig := <-sigs
			handler(sig)
		}
	}()
}

func (c *Configuration) setUpLogging() *log.Logger {
	logConfigPath := getLogConfigPath()
	log.Printf("Looking for logging configuration file at %s", logConfigPath)
	// Assuming fileConfig is a custom function to set up logging from a config file
	fileConfig(logConfigPath)
	return log.New(os.Stdout, "Configuration: ", log.LstdFlags)
}

func (c *Configuration) quitIfWrongGoVersion(logger *log.Logger) {
	if runtime.Version() < "go1.12" {
		logger.Fatalf("Go version must be 1.12 or higher. Go interpreter version: %s", runtime.Version())
		os.Exit(1)
	}
}

func (c *Configuration) svcStop() {
	c.running = false
}

func (c *Configuration) svcMain() {
	c.start()
}

func (c *Configuration) start() {
	c.running = true

	sigHandler := func(sig os.Signal) {
		if sig == syscall.SIGINT || sig == syscall.SIGTERM {
			c.running = false
		}
	}

	if runtime.GOOS != "windows" {
		c.setUpSignalHandlers(sigHandler)
	}

	logger := c.setUpLogging()
	c.quitIfWrongGoVersion(logger)

	logger.Println("Configuration Agent is running.")

	// Hardcoded to use XML until we have a request to support other tools.
	config, err := NewXmlKeyValueStore(XML_LOCATION, true, SCHEMA_LOCATION)
	if err != nil {
		log.Fatalf("Failed to create XML Key Value Store: %v", err)
	}
	useTLS := os.Getenv("USE_TLS") == "true" || os.Getenv("USE_TLS") == "1" || os.Getenv("USE_TLS") == "t"
	broker := NewBroker(config, useTLS)

	broker.PublishInitialValues()

	for c.running {
		time.Sleep(1 * time.Second)
	}

	broker.brokerStop()
}

func getLogConfigPath() string {
	if path, ok := os.LookupEnv("LOGGERCONFIG"); ok {
		return path
	}
	return DEFAULT_LOGGING_PATH
}

func fileConfig(logConfigPath string) {
	// Implement the logging configuration setup here
}

func main() {
	if runtime.GOOS == "windows" {
		RunService("inbm-configuration", "Configuration Agent", "Intel Manageability agent handling framework configuration", NewConfiguration(nil))
	} else {
		configuration := NewConfiguration(os.Args[1:])
		configuration.start()
	}
}
