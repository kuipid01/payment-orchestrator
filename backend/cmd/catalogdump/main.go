// Command catalogdump writes the plan catalog as JSON.
//
// The frontend keeps a copy of the catalog so the marketing page still renders
// when the API is unreachable. That copy is *generated* by this command rather
// than typed by hand, because a hand-kept second copy of a price is exactly how
// two surfaces end up disagreeing:
//
//	go run ./cmd/catalogdump > ../frontend/src/lib/plan-catalog.json
//
// The API serves the same Go values, so the live and offline catalogs are the
// same numbers by construction.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kuipid01/payconnect/internal/billing"
)

func main() {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(billing.ToCatalogResponse(billing.Catalog())); err != nil {
		fmt.Fprintln(os.Stderr, "catalogdump:", err)
		os.Exit(1)
	}
}
