//Copyright Robert C. Taylor 2018
//Distributed under the terms of the LICENSE file

package jsonhal

import (
	"bytes"
	"encoding/json"
	"fmt"
)

//CollectionValue represents a JSON entry in Collection.
type CollectionValue struct {
	Properties map[string]interface{}
}

//MarshalJSON is the custom json serializer of CollectionValue
func (cv *CollectionValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(cv.Properties)
}

//UnmarshalJSON is the custom JSON deserializer of CollectionValue
func (cv *CollectionValue) UnmarshalJSON(data []byte) error {

	var m *map[string]interface{}

	err := json.Unmarshal(data, &m)
	if err != nil {
		return err
	}

	cv.Properties = *m

	return nil
}

//NewCollection returns a new CollectionValue with all public fields initialized and ready for use.
func NewCollection() *Collection {
	c := &Collection{}
	c.Values = make(map[string][]*CollectionValue)
	return c
}

//NewValue returns a new CollectionValue with all public fields initialized and ready for use.
func NewValue() *CollectionValue {
	c := &CollectionValue{}
	c.Properties = make(map[string]interface{})
	return c
}

//Collection is a HAL+JSON collection. This object gets encoded according to the rules of a _link of _embeded object.
type Collection struct {
	Values map[string][]*CollectionValue
}

//MarshalJSON is the custom json serializer of Collection
func (c *Collection) MarshalJSON() ([]byte, error) {

	jsonObj := make(map[string]interface{})

	for k, v := range c.Values {
		if len(v) <= 0 {
			continue //Dont include empty keys
		} else if len(v) == 1 {
			jsonObj[k] = v[0] //When only one is present, json+HAL expects the naked object
		} else {
			jsonObj[k] = v //When more then one is present, the objects are in an array
		}
	}

	return json.Marshal(jsonObj)
}

func (c *Collection) UnmarshalJSON(data []byte) error {

	var jsonCollection *map[string]*_JSONHALValue
	err := json.Unmarshal(data, &jsonCollection)
	if err != nil {
		return err
	}

	c.Values = make(map[string][]*CollectionValue)

	for k, v := range *jsonCollection {
		c.Values[k] = v.Values
	}

	return nil
}

type _JSONHALValue struct {
	Values []*CollectionValue
}

func (jhv *_JSONHALValue) UnmarshalJSON(data []byte) error {

	//First read the next token to determine if this is a
	//naked object, or an array.
	buf := bytes.NewBuffer(data)
	decoder := json.NewDecoder(buf)

	tok, err := decoder.Token()
	if err != nil {
		return err
	}

	//Naked object set to null
	if tok == nil {
		return nil
	}

	delim, ok := tok.(json.Delim)
	if !ok {
		return fmt.Errorf("jsonhal : Unexpected token, expected { or [")
	}

	switch delim.String() {
	case "{":

		var val *CollectionValue
		err = json.Unmarshal(data, &val)
		if err == nil {
			jhv.Values = append(jhv.Values, val)
		}
	case "[":
		err = json.Unmarshal(data, &jhv.Values)
	default:
		err = fmt.Errorf("jsonhal : Unexpected token, expected { or [")
	}

	return err
}

//CreateLink is a shortcut function used to create a HAL+JSON link object to be placed inside
//of a collection.
func CreateLink(href string) *CollectionValue {
	link := NewValue()
	link.Properties["href"] = href
	return link
}
