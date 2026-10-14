package vault

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestOperationLogsDoNotContainSecrets(t *testing.T) {
	var output bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(previousLogger)

	masterKey := "vault-master-key-123456789012345"
	plaintext := "provider-secret-value"
	instance, err := New(masterKey)
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, err := instance.Encrypt(plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := instance.Decrypt(ciphertext); err != nil {
		t.Fatal(err)
	}

	apiKey, _, err := GenerateAPIKey("test")
	if err != nil {
		t.Fatal(err)
	}
	keyHash := HashKey(apiKey)
	webhookSecret, _, err := GenerateWebhookSecret()
	if err != nil {
		t.Fatal(err)
	}

	logs := output.String()
	for _, secret := range []string{masterKey, plaintext, apiKey, keyHash, webhookSecret} {
		if strings.Contains(logs, secret) {
			t.Fatalf("logs contain secret value %q", secret)
		}
	}
	// Per-operation events ("credential encrypted"/"credential decrypted")
	// are intentionally absent: they fire on every read and write and only
	// repeated byte counts, so the vault logs the outcomes worth keeping
	// instead. This list is those outcomes.
	for _, event := range []string{"credential vault initialized", "developer API key hashed", "webhook signing secret generated"} {
		if !strings.Contains(logs, event) {
			t.Errorf("logs do not contain operation event %q", event)
		}
	}
}
