package configuration

// Commands represents the supported commands in Configuration Agent
type Commands int

const (
	GetElement Commands = iota
	SetElement
	Load
	Append
	Remove
)

var commandStrings = map[Commands]string{
	GetElement: "get_element",
	SetElement: "set_element",
	Load:       "load",
	Append:     "append",
	Remove:     "remove",
}

// String returns the string representation of the Commands
func (c Commands) String() string {
	return commandStrings[c]
}
