package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// JSON custom type for handling JSONB fields
type JSON map[string]string

// Scan implements the sql.Scanner interface
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = make(map[string]string)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	// Create a temporary map to store the unmarshaled data
	temp := make(map[string]string)
	if err := json.Unmarshal(bytes, &temp); err != nil {
		return err
	}

	*j = temp
	return nil
}

// Value implements the driver.Valuer interface
func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return json.Marshal(j)
}
