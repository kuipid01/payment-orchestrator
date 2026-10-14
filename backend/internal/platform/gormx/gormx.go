// Package gormx holds shared GORM plumbing: ID generation and a generic JSONB
// column type used by domain models across modules.
package gormx

import (
	"crypto/rand"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
)

// NewID returns a prefixed random identifier, e.g. pay_9f3c1b….
func NewID(prefix string) string {
	raw := make([]byte, 6)
	_, _ = rand.Read(raw)
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(raw))
}

// JSONB is a generic JSONB column that serializes to its underlying Go value.
type JSONB[T any] struct {
	Data T `json:"Value"`
}

func (j JSONB[T]) GormDataType() string { return "jsonb" }

func (j *JSONB[T]) Scan(v any) error {
	bytes, ok := v.([]byte)
	if !ok {
		return errors.New("gormx: failed to unmarshal JSONB: not bytes")
	}
	return json.Unmarshal(bytes, &j.Data)
}

func (j JSONB[T]) Value() (driver.Value, error) {
	return json.Marshal(j.Data)
}
