package payments

import (
	"encoding/json"

	"github.com/kuipid01/payconnect/internal/platform/gormx"
)

func gormxJSON(value map[string]any) string {
	if value == nil {
		return "{}"
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func newID(prefix string) string { return gormx.NewID(prefix) }
