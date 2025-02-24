package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// JSON type for flexible storage
type JSON map[string]interface{}

// Value implements the driver.Valuer interface
func (j JSON) Value() (driver.Value, error) {
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface
func (j *JSON) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, &j)
}

// MarshalJSON returns the JSON encoding
func (j JSON) MarshalJSON() ([]byte, error) {
	if j == nil {
		return []byte("null"), nil
	}
	return json.Marshal(j)
}

// UnmarshalJSON sets the JSON encoding
func (j *JSON) UnmarshalJSON(data []byte) error {
	if j == nil {
		return errors.New("nil pointer")
	}
	return json.Unmarshal(data, &j)
}
