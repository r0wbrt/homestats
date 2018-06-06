//Copyright Robert C. Taylor 2018
//Distributed under the terms of the LICENSE file

package stream

import "time"

//StorageType is the name of the underlying method used to store the value.
//The units are in JavaScript units.
type StorageType string

const (
	//StorageNumber stores a value as a number.
	StorageNumber StorageType = "number"
	//StorageString stores the value as a string.
	StorageString StorageType = "string"
	//StorageBoolean stores the value as a boolean.
	StorageBoolean StorageType = "boolean"
	//StorageTime stores the value as a date.
	StorageTime StorageType = "date"
)

//TypeSchema represents the schema of a single value in a measurment.
type TypeSchema struct {
	//Name of the value
	Name string
	//How the unit will be stored.
	StorageUnit StorageType
	//The real world unit of the measurment.
	MeasurmentUnit string
}

//Stream describes a single data set.
type Stream struct {
	//Name of the dataset.
	Name string
	//Human readable description of this stream.
	Description string
	//A global unique GUID describing this stream. Should be a 64 bit values encoded in base 16 (hex).
	GUID string
	//Schema of the measurment
	Schema []TypeSchema
	//The duration a single measurment is kept before being deleted.
	RetentionPolicy time.Duration
}

//DataSetValue is a single value in a measurment
type DataSetValue struct {
	Name  string
	Value string
}

//DataSetMeasurment is a single measurment in a data set.
type DataSetMeasurment struct {
	Time   time.Time
	Values []DataSetValue
}
