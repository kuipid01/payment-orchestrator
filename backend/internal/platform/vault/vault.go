// Package vault implements the credential vault: AES-256-GCM encryption for
// provider secrets and webhook signing secrets, plus API key generation/hashing.
// Secrets encrypted here are never rendered, exported, or returned by any API.
package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

var ErrInvalidKey = errors.New("vault: encryption key must be 16, 24, or 32 bytes")

type Vault struct {
	gcm cipher.AEAD
}

// New builds a vault from a raw or hex/base64-encoded key.
func New(key string) (*Vault, error) {
	raw, err := decodeKey(key)
	if err != nil {
		slog.Error("vault key decoding failed", "error", err)
		return nil, err
	}
	block, err := aes.NewCipher(raw)
	if err != nil {
		slog.Error("vault cipher initialization failed", "key_bytes", len(raw), "error", err)
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		slog.Error("vault AEAD initialization failed", "error", err)
		return nil, err
	}
	slog.Info("credential vault initialized", "algorithm", "AES-GCM", "key_bytes", len(raw))
	return &Vault{gcm: gcm}, nil
}

func decodeKey(key string) ([]byte, error) {
	if decoded, err := hex.DecodeString(key); err == nil && len(key) >= 32 {
		slog.Debug("decoded vault key", "encoding", "hex", "key_bytes", len(decoded))
		return decoded, nil
	}
	if decoded, err := base64.StdEncoding.DecodeString(key); err == nil && len(decoded) >= 16 {
		slog.Debug("decoded vault key", "encoding", "base64", "key_bytes", len(decoded))
		return decoded, nil
	}
	raw := []byte(key)
	slog.Debug("using raw vault key", "key_bytes", len(raw))
	return raw, nil
}

// Encrypt seals plaintext into nonce||ciphertext.
//
// The per-operation trace that used to live here is deliberately gone: it fired
// on every credential write and read, carried only byte counts, and duplicated
// the outcome the caller already logs. Failures below are still loud.
func (v *Vault) Encrypt(plaintext string) ([]byte, error) {
	nonce := make([]byte, v.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		slog.Error("credential encryption failed", "stage", "nonce_generation", "error", err)
		return nil, err
	}
	ciphertext := v.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return ciphertext, nil
}

// Decrypt opens nonce||ciphertext produced by Encrypt.
func (v *Vault) Decrypt(data []byte) (string, error) {
	if len(data) < v.gcm.NonceSize() {
		err := errors.New("vault: ciphertext too short")
		slog.Warn("credential decryption rejected", "reason", "ciphertext_too_short", "ciphertext_bytes", len(data))
		return "", err
	}
	nonce, ciphertext := data[:v.gcm.NonceSize()], data[v.gcm.NonceSize():]
	plaintext, err := v.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		slog.Warn("credential decryption failed", "error", err)
		return "", err
	}
	return string(plaintext), nil
}

/* -------------------------------- API keys ---------------------------------- */

const keyAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

// GenerateAPIKey returns a full developer key: vpay_<env>_<32 random chars>.
func GenerateAPIKey(env string) (full string, masked string, err error) {
	slog.Info("generating developer API key", "environment", env)
	randBytes := make([]byte, 32)
	if _, err := rand.Read(randBytes); err != nil {
		slog.Error("developer API key generation failed", "error", err)
		return "", "", err
	}
	var sb strings.Builder
	sb.WriteString("vpay_")
	sb.WriteString(env)
	sb.WriteString("_")
	for _, b := range randBytes {
		sb.WriteByte(keyAlphabet[int(b)%len(keyAlphabet)])
	}
	full = sb.String()
	masked = fmt.Sprintf("vpay_%s_%s%s", env, "••••••••••••", full[len(full)-4:])
	slog.Info("developer API key generated", "environment", env, "key_bytes", len(full))
	return full, masked, nil
}

// HashKey produces the deterministic lookup hash stored for each API key.
func HashKey(full string) string {
	slog.Info("hashing developer API key", "key_bytes", len(full))
	sum := sha256.Sum256([]byte(full))
	hash := hex.EncodeToString(sum[:])
	slog.Info("developer API key hashed", "hash_bytes", len(hash))
	return hash
}

// GenerateWebhookSecret returns whsec_<base64url 24 bytes>.
func GenerateWebhookSecret() (full string, masked string, err error) {
	slog.Info("generating webhook signing secret")
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		slog.Error("webhook signing secret generation failed", "error", err)
		return "", "", err
	}
	full = "whsec_" + base64.RawURLEncoding.EncodeToString(raw)
	masked = "whsec_••••••••••••" + full[len(full)-4:]
	slog.Info("webhook signing secret generated", "secret_bytes", len(full))
	return full, masked, nil
}

/* ------------------------------ key material -------------------------------- */

const keyEntropyBytes = 32

// GenerateKey mints a full API key of the form <prefix>_<env>_<marker>_<random>
// and its masked display form. The full value is returned exactly once, to the
// caller that created it; only HashKey(full) is ever persisted.
func GenerateKey(prefix, env, marker string) (full string, masked string, err error) {
	if prefix == "" {
		prefix = "pc"
	}
	raw := make([]byte, keyEntropyBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	var sb strings.Builder
	sb.WriteString(prefix)
	sb.WriteString("_")
	sb.WriteString(env)
	sb.WriteString("_")
	sb.WriteString(marker)
	sb.WriteString("_")
	for _, b := range raw {
		sb.WriteByte(keyAlphabet[int(b)%len(keyAlphabet)])
	}
	full = sb.String()
	masked = prefix + "_" + env + "_" + marker + "_" + strings.Repeat("•", 12) + full[len(full)-4:]
	return full, masked, nil
}

// SignWebhook produces the hex HMAC-SHA256 signature sent with merchant
// webhook deliveries.
func SignWebhook(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyWebhookSignature compares a received signature with the expected one in
// constant time.
func VerifyWebhookSignature(secret string, body []byte, signature string) bool {
	expected := SignWebhook(secret, body)
	return hmac.Equal([]byte(expected), []byte(strings.TrimSpace(signature)))
}
