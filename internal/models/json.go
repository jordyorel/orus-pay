package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// JSON custom type for handling JSON data
type JSON struct {
	data interface{}
}

// NewJSON creates a new JSON instance
func NewJSON(data interface{}) JSON {
	return JSON{data: data}
}

// Value implements the driver.Valuer interface
func (j JSON) Value() (driver.Value, error) {
	if j.data == nil {
		return nil, nil
	}
	return json.Marshal(j.data)
}

// Scan implements the sql.Scanner interface
func (j *JSON) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, &j.data)
}

// MarshalJSON returns the JSON encoding
func (j JSON) MarshalJSON() ([]byte, error) {
	if b, ok := j.data.([]byte); ok {
		return b, nil
	}
	if j.data == nil {
		return []byte("null"), nil
	}
	return json.Marshal(j.data)
}

// UnmarshalJSON sets the JSON encoding
func (j *JSON) UnmarshalJSON(data []byte) error {
	if j == nil {
		return errors.New("nil pointer")
	}
	return json.Unmarshal(data, &j.data)
}
