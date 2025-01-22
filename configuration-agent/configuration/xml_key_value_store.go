package configuration

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lestrrat-go/libxml2"
	"github.com/lestrrat-go/libxml2/xsd"
)

const PARSE_TIME_SECS = 10 * time.Second

type XmlException struct {
	Message string
}

func (e *XmlException) Error() string {
	return e.Message
}

type XMLElement struct {
	XMLName xml.Name
	Attr    []xml.Attr   `xml:",any,attr"`
	Content []byte       `xml:",innerxml"`
	Nodes   []XMLElement `xml:",any"`
}

type XmlKeyValueStore struct {
	isFile         bool
	schemaLocation string
	xmlContent     string
	root           *XMLElement
}

// Load implements IKeyValueStore.

func (x *XmlKeyValueStore) Load(xmlFilePath string) error {
	filePath, err := GetCanonicalRepresentationOfPath(xmlFilePath)
	if err != nil {
		return err
	}

	if x.isFile {
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			log.Printf("New XML file to be loaded not found at '%s'", filePath)
			return &XmlException{Message: "New XML file to be loaded not found"}
		}
	}

	if err := x.parseXmlInTimeLimit(filePath); err != nil {
		return err
	}
	log.Println("Loaded file was successfully validated")

	backupFile := x.xmlContent + "_bak"
	if err := os.WriteFile(backupFile, []byte(x.xmlContent), 0644); err != nil {
		return err
	}
	log.Printf("Backup file: %s", backupFile)

	if err := os.WriteFile(filePath, []byte(x.xmlContent), 0644); err != nil {
		return err
	}

	return nil
}

func NewXmlKeyValueStore(xmlPath string, isFile bool, schemaLocation string) (*XmlKeyValueStore, error) {
	canonicalSchemaLocation, err := GetCanonicalRepresentationOfPath(schemaLocation)
	if err != nil {
		return nil, err
	}

	var xmlContent string
	if isFile {
		canonicalXmlPath, err := GetCanonicalRepresentationOfPath(xmlPath)
		if err != nil {
			return nil, err
		}
		content, err := os.ReadFile(canonicalXmlPath)
		if err != nil {
			return nil, err
		}
		xmlContent = string(content)
	} else {
		xmlContent = xmlPath
	}

	store := &XmlKeyValueStore{
		isFile:         isFile,
		schemaLocation: canonicalSchemaLocation,
		xmlContent:     xmlContent,
	}

	if err := store.parseXmlInTimeLimit(xmlContent); err != nil {
		return nil, err
	}

	return store, nil
}

func (x *XmlKeyValueStore) parseXmlInTimeLimit(xmlContent string) error {
	done := make(chan error, 1)

	go func() {
		done <- x.getRoot(xmlContent)
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(PARSE_TIME_SECS):
		return &XmlException{Message: "XML Parser timed out"}
	}
}

func (x *XmlKeyValueStore) getRoot(xmlContent string) error {
	log.Printf("XML : %s", MaskSecurityInfo(xmlContent))
	root, err := x.validate(xmlContent)
	if err != nil {
		return &XmlException{Message: "Error with xml: " + err.Error()}
	}
	x.root = root
	return nil
}

func (x *XmlKeyValueStore) validate(xmlContent string) (*XMLElement, error) {
	log.Printf("validating XML file: %s", MaskSecurityInfo(xmlContent))
	if _, err := os.Stat(x.schemaLocation); os.IsNotExist(err) {
		return nil, &XmlException{Message: "Schema file not found"}
	}

	if isSymlink(x.schemaLocation) {
		return nil, &XmlException{Message: "Schema file location is a symlink"}
	}

	var parsedDoc XMLElement
	decoder := xml.NewDecoder(strings.NewReader(xmlContent))
	if err := decoder.Decode(&parsedDoc); err != nil {
		return nil, err
	}

	schemaContent, err := ioutil.ReadFile(x.schemaLocation)
	if err != nil {
		return nil, &XmlException{Message: "Error reading schema file: " + err.Error()}
	}

	// Implement XML schema validation logic here
	// Assuming a function validateXMLSchema exists
	if err := validateXMLSchema(schemaContent, xmlContent); err != nil {
		return nil, err
	}

	return &parsedDoc, nil
}

func (x *XmlKeyValueStore) GetElement(path string, elementString *string, isAttribute bool) (string, error) {
	if elementString == nil {
		var val string
		var err error
		if isAttribute {
			val, err = x.getAttributeValue(path)
		} else {
			val, err = x.getValue(path)
		}
		if err != nil {
			return "", err
		}
		return path + ":" + val, nil
	} else {
		elements := strings.Split(*elementString, ",")
		var results strings.Builder
		for _, ele := range elements {
			varList := strings.SplitN(ele, ":", 2)
			varName := strings.Trim(varList[0], "'")
			var val string
			var err error
			if isAttribute {
				val, err = x.getAttributeValue(path + "/" + varName)
			} else {
				val, err = x.getValue(path + "/" + varName)
			}
			if err != nil {
				return "", err
			}
			results.WriteString("{" + path + "/" + varName + ":" + val + "},")
		}
		return results.String(), nil
	}
}

func (x *XmlKeyValueStore) SetElement(xpath string, value string, valueString *string, isAttribute bool) (string, error) {
	if valueString == nil {
		if err := x.setElementValue(xpath, value); err != nil {
			return "", err
		}
		return value, nil
	} else {
		values := strings.Split(*valueString, ",")
		var paths strings.Builder
		for _, val := range values {
			valList := strings.SplitN(val, ":", 2)
			ele := valList[0]
			value := valList[1]
			if isAttribute {
				if err := x.setElementAttributeValue(ele, value); err != nil {
					return "", err
				}
			} else {
				if err := x.setElementValue(xpath+"/"+ele, value); err != nil {
					return "", err
				}
			}
			paths.WriteString(xpath + "/" + ele + ":" + value + ";")
		}
		return paths.String(), nil
	}
}

func (x *XmlKeyValueStore) GetChildren(xpath string) (map[string]string, error) {
	children := make(map[string]string)
	elements := x.findElements(xpath)

	if elements == nil {
		return nil, &XmlException{Message: fmt.Sprintf("Cannot find children at specified path: %s", xpath)}
	}
	for _, each := range elements {
		children[each.XMLName.Local] = string(each.Content)
	}
	return children, nil
}

func (x *XmlKeyValueStore) Append(path string, valueString string) (string, error) {
	log.Println("")
	pathValues, err := x.GetElement(path, &valueString, false)
	if err != nil {
		return "", err
	}
	elementHeader := strings.Split(strings.Trim(pathValues, "{}"), ":")[0]
	elementValue := strings.Split(strings.Trim(pathValues, "{}"), ":")[1] + "\n\t    " + strings.Split(valueString, ":")[1] + "\n\t"
	paths, err := x.SetElement(elementHeader, "", &elementValue, false)
	if err != nil {
		return "", err
	}
	return paths, nil
}

func (x *XmlKeyValueStore) Remove(path string, value *string, valueString *string) (string, error) {
	log.Println("")
	if valueString == nil {
		valueString = new(string)
	}
	pathValues, err := x.GetElement(path, valueString, false)
	if err != nil {
		return "", err
	}
	elementHeaderPath := strings.Split(strings.Trim(pathValues, "{}"), ":")[0]
	elementName := strings.Split(elementHeaderPath, "/")[1]
	elementHeader := strings.Split(elementHeaderPath, "/")[0]
	elementPathValues := strings.Split(strings.Trim(pathValues, "{}"), ":")[1]
	valueToRemove := strings.Split(*valueString, ":")[1]

	if elementPathValues == "" {
		errorMsg := fmt.Sprintf("The element path has no values listed in the conf file: %s", elementHeaderPath)
		log.Println(errorMsg)
		return "", &ConfigurationException{Message: errorMsg}
	} else {
		elementValues := strings.Split(strings.TrimSpace(elementPathValues), "\n")
		for i, v := range elementValues {
			elementValues[i] = strings.TrimSpace(v)
		}
		if contains(elementValues, valueToRemove) {
			log.Println("string exists in the element's value list")
			elementValues = remove(elementValues, valueToRemove)
			updatedElementValues := "\n\t    " + strings.Join(elementValues, "\n\t    ") + "\n\t"
			newValueString := elementName + ":" + updatedElementValues
			paths, err := x.SetElement(elementHeader, "", &newValueString, false)
			if err != nil {
				return "", err
			}
			return paths, nil
		} else {
			errorMsg := fmt.Sprintf("The following element path doesn't contain the value to remove: %s", elementHeaderPath)
			log.Println(errorMsg)
			return "", &ConfigurationException{Message: errorMsg}
		}
	}
}

func (x *XmlKeyValueStore) getAttributeValue(path string) (string, error) {
	xmlTag := path[strings.LastIndex(path, "/")+1:]
	elements := x.findElements(xmlTag)
	if len(elements) == 0 {
		return "", &XmlException{Message: fmt.Sprintf("Cannot find element at specified path: %s", path)}
	}
	for _, attr := range elements[0].Attr {
		if attr.Name.Local == ATTRIB_NAME {
			return attr.Value, nil
		}
	}
	return "", nil
}

func (x *XmlKeyValueStore) getValue(path string) (string, error) {
	elem := x.findElementText(path)
	if elem == "" {
		return "", &XmlException{Message: fmt.Sprintf("Cannot find element at specified path: %s", path)}
	}
	return elem, nil
}

func (x *XmlKeyValueStore) writeToFile(filePath string) error {
	xmlStr, err := xml.MarshalIndent(x.root, "", "  ")
	if err != nil {
		return &XmlException{Message: fmt.Sprintf("Unable to marshal XML: %s", err)}
	}
	if err := ioutil.WriteFile(filePath, xmlStr, 0644); err != nil {
		log.Printf("Unable to write at specified path: %s", filePath)
		return &XmlException{Message: "Unable to write configuration changes"}
	}
	return nil
}

func (x *XmlKeyValueStore) validateFile() error {
	if _, err := x.validate(x.xmlContent); err != nil {
		return &ConfigurationException{Message: fmt.Sprintf("Configuration Set Element Failed: %s Keeping old value", err)}
	}
	return nil
}

func (x *XmlKeyValueStore) updateFile(elements []*XMLElement, value string, isAttribute bool) error {
	if !x.isFile {
		return &ConfigurationException{Message: "cannot write non-file XML key value store to file"}
	}

	var oldValue string
	if isAttribute {
		for _, attr := range elements[0].Attr {
			if attr.Name.Local == ATTRIB_NAME {
				oldValue = attr.Value
				attr.Value = value
			}
		}
	} else {
		oldValue = string(elements[0].Content)
		elements[0].Content = []byte(value)
	}
	if err := x.writeToFile(x.xmlContent); err != nil {
		return err
	}
	if err := x.validateFile(); err != nil {
		elements[0].Content = []byte(oldValue)
		x.writeToFile(x.xmlContent)
		return err
	}
	return nil
}

func (x *XmlKeyValueStore) setElementValue(xpath string, value string) error {
	elements := x.findElements(xpath)
	if len(elements) > 0 {
		if x.isFile {
			return x.updateFile(elements, value, false)
		} else {
			elements[0].Content = []byte(value)
		}
	} else {
		return &XmlException{Message: fmt.Sprintf("Cannot find element at specified path: %s", xpath)}
	}
	return nil
}

func (x *XmlKeyValueStore) setElementAttributeValue(elementTag string, attributeValue string) error {
	elements := x.findElements(elementTag)
	if len(elements) > 0 {
		if x.isFile {
			return x.updateFile(elements, attributeValue, true)
		} else {
			for _, attr := range elements[0].Attr {
				if attr.Name.Local == ATTRIB_NAME {
					attr.Value = attributeValue
				}
			}
		}
	} else {
		return &XmlException{Message: fmt.Sprintf("Cannot find element at specified path: %s", elementTag)}
	}
	return nil
}

func (x *XmlKeyValueStore) GetParent(childElement string) (string, error) {
	tree := x.root
	var parent *XMLElement

	parentMap := make(map[*XMLElement]*XMLElement)
	for _, p := range tree.Nodes {
		for _, c := range p.Nodes {
			parentMap[&c] = &p
		}
	}

	for _, e := range tree.Nodes {
		if e.XMLName.Local == childElement {
			parent = parentMap[&e]
		}
	}

	if parent == nil {
		return "", &XmlException{Message: fmt.Sprintf("Cannot find parent with specified child tag: %s", childElement)}
	}

	return parent.XMLName.Local, nil
}

func (x *XmlKeyValueStore) findElements(xpath string) []*XMLElement {
	var elements []*XMLElement
	x.findElementsRecursive(x.root, strings.Split(xpath, "/"), &elements)
	return elements
}

func (x *XmlKeyValueStore) findElementsRecursive(element *XMLElement, path []string, elements *[]*XMLElement) {
	if len(path) == 0 {
		*elements = append(*elements, element)
		return
	}
	for _, node := range element.Nodes {
		if node.XMLName.Local == path[0] {
			x.findElementsRecursive(&node, path[1:], elements)
		}
	}
}

func (x *XmlKeyValueStore) findElementText(xpath string) string {
	elements := x.findElements(xpath)
	if len(elements) > 0 {
		return string(elements[0].Content)
	}
	return ""
}

func isSymlink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}

func GetCanonicalRepresentationOfPath(path string) (string, error) {
	expandedPath := os.ExpandEnv(path)

	absPath, err := filepath.Abs(expandedPath)
	if err != nil {
		return "", err
	}

	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", err
	}

	normPath := filepath.Clean(realPath)
	return filepath.FromSlash(normPath), nil
}

func validateXMLSchema(schemaContent []byte, xmlContent string) error {
	// Parse the XML schema
	schema, err := xsd.Parse(schemaContent)
	if err != nil {
		return fmt.Errorf("failed to parse XML schema: %v", err)
	}
	defer schema.Free()

	// Parse the XML content
	doc, err := libxml2.ParseString(xmlContent)
	if err != nil {
		return fmt.Errorf("failed to parse XML content: %v", err)
	}
	defer doc.Free()

	// Validate the XML content against the schema
	if err := schema.Validate(doc); err != nil {
		return fmt.Errorf("XML validation error: %v", err)
	}

	return nil
}

func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func remove(slice []string, item string) []string {
	for i, v := range slice {
		if v == item {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}
