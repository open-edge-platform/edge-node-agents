package configuration

// ConfigurationException is a custom error type for configuration errors
type ConfigurationException struct {
	Message string
}

func (e *ConfigurationException) Error() string {
	return e.Message
}

// NewConfigurationException creates a new ConfigurationException
func NewConfigurationException(message string) error {
	return &ConfigurationException{
		Message: message,
	}
}
