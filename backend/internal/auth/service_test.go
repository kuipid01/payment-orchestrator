package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"
)

func TestOpaqueTokenIsRandomAndHashable(t *testing.T) {
	value, hash, err := newOpaqueToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(value) == 0 || len(hash) != sha256.Size*2 {
		t.Fatalf("unexpected token/hash lengths: token=%d hash=%d", len(value), len(hash))
	}
	if value == hash {
		t.Fatal("session tokens and email tokens must never be stored in plaintext")
	}
	if hash != hashOpaque(value) {
		t.Fatal("hash is not deterministic")
	}
	sum := sha256.Sum256([]byte(value))
	if hex.EncodeToString(sum[:]) != hash {
		t.Fatal("hash is not SHA-256 of the token")
	}
}

func TestOpaqueTokensAreUnique(t *testing.T) {
	seen := make(map[string]bool, 64)
	for i := 0; i < 64; i++ {
		value, _, err := newOpaqueToken()
		if err != nil {
			t.Fatal(err)
		}
		if seen[value] {
			t.Fatal("duplicate token generated")
		}
		seen[value] = true
	}
}

func TestCredentialTTLs(t *testing.T) {
	if emailVerificationTTL != time.Hour {
		t.Fatalf("email verification TTL = %s, want 1h", emailVerificationTTL)
	}
	if passwordResetTTL != 30*time.Minute {
		t.Fatalf("password reset TTL = %s, want 30m", passwordResetTTL)
	}
}
