package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// JSON is a custom type for handling JSON data in GORM
type JSON json.RawMessage

// Value implements the driver.Valuer interface
func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return string(j), nil
}

// Scan implements the sql.Scanner interface
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	s, ok := value.([]byte)
	if !ok {
		return errors.New("invalid scan source")
	}

	*j = append((*j)[0:0], s...)
	return nil
}

// MarshalJSON returns the JSON encoding
func (j JSON) MarshalJSON() ([]byte, error) {
	if j == nil {
		return []byte("null"), nil
	}
	return j, nil
}

// UnmarshalJSON sets the JSON encoding
func (j *JSON) UnmarshalJSON(data []byte) error {
	if j == nil {
		return errors.New("nil pointer")
	}
	*j = append((*j)[0:0], data...)
	return nil
}
