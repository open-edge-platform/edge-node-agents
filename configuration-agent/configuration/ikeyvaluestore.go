package configuration

// IKeyValueStore is an interface for managing key/value pairs
type IKeyValueStore interface {
	// GetElement gets element value from the specified path
	// @param path: path to the key in the structure
	// @param elementString: String of ; separated elements whose values need to be returned
	// @param isAttribute: determines if the element attribute value needs to be returned
	GetElement(path string, elementString *string, isAttribute bool) (string, error)

	// SetElement sets the element value at the specified path
	// @param path: path to the key
	// @param value: value to set at the key
	// @param valueString: multiple variable value string separated by ;
	// @param isAttribute: determines if the element attribute value needs to be set
	SetElement(path string, value string, valueString *string, isAttribute bool) (string, error)

	// Load loads a new key/value pair file
	// @param path: path to file
	Load(path string) error

	// Append appends the element value at the specified path
	// @param path: path to the key or multiple paths separated by ;
	// @param valueString: value to append at the key or multiple values separated by ;
	Append(path string, valueString string) (string, error)

	// Remove removes the element value at the specified path
	// @param path: path to the key
	// @param value: value to remove from the key
	// @param valueString: multiple variable value string separated by ;
	Remove(path string, value *string, valueString *string) (string, error)

	// GetChildren gets all children under the specified path
	// @param path: path to use
	GetChildren(path string) (map[string]string, error)

	// GetParent finds the parent of the child element from XML file
	// @param childElement: child element tag
	GetParent(childElement string) (string, error)
}
