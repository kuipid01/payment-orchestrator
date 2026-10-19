package providers

import (
	"testing"

	"github.com/kuipid01/payconnect/internal/constants"
)

func TestValidateMonnifyCredentialsRequiresAPIKeyAndContractCode(t *testing.T) {
	if err := validateProviderCredentialFields(constants.ProviderMonnify, "", "contract"); err == nil {
		t.Fatal("expected missing API key to be rejected")
	}
	if err := validateProviderCredentialFields(constants.ProviderMonnify, "api-key", ""); err == nil {
		t.Fatal("expected missing contract code to be rejected")
	}
	if err := validateProviderCredentialFields(constants.ProviderMonnify, "api-key", "contract"); err != nil {
		t.Fatalf("valid Monnify credentials rejected: %v", err)
	}
}
