//Copyright Robert C. Taylor 2018
//Distributed under the terms of the LICENSE file

package jsonhal

import (
	"encoding/json"
)

//CollectionValue represents a JSON entry in Collection.
type CollectionValue struct {
	Properties map[string]interface{}
}

//MarshalJSON is the custom json serializer of CollectionValue
func (cv *CollectionValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(cv.Properties)
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

//CreateLink is a shortcut function used to create a HAL+JSON link object to be placed inside
//of a collection.
func CreateLink(href string) *CollectionValue {
	link := NewValue()
	link.Properties["href"] = href
	return link
}
