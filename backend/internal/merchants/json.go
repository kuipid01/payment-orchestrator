package merchants

import (
	"encoding/json"

	"github.com/kuipid01/payconnect/internal/platform/gormx"
)

func gormxJSONStrings(values []string) string {
	encoded, err := json.Marshal(values)
	if err != nil {
		return "[]"
	}
	return string(encoded)
}

func idForMembership() string { return gormx.NewID("um") }
func idForRefresh() string    { return gormx.NewID("rt") }
