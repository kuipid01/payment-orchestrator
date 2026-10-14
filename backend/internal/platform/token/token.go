// Package token issues and verifies the signed access tokens used by the
// dashboard session. Tokens are Ed25519-signed (EdDSA) JWT-style payloads whose
// key pair is derived deterministically from the configured JWT_SECRET, so
// every instance in a fleet verifies the same tokens without shared state.
package token

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const algEdDSA = "EdDSA"

var (
	// ErrInvalidToken means the token is malformed or the signature failed.
	ErrInvalidToken = errors.New("token: invalid token")
	// ErrExpiredToken means the token is well-formed but past its expiry.
	ErrExpiredToken = errors.New("token: token has expired")
)

// Claims is the token payload. Field names follow the JWT conventions.
type Claims struct {
	Sub string `json:"sub"` // user id
	Org string `json:"org"` // merchant id
	Typ string `json:"typ"` // access | refresh
	Iat int64  `json:"iat"`
	Exp int64  `json:"exp"`
	Iss string `json:"iss"`
	Aud string `json:"aud"`
}

// Signer issues and verifies access tokens with a derived Ed25519 key pair.
type Signer struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	issuer     string
	audience   string
	ttl        time.Duration
}

// New derives a key pair from the secret and returns a signer.
// The secret must be at least 16 bytes; production requires 32 (see config).
func New(secret, issuer, audience string, ttl time.Duration) (*Signer, error) {
	if len(secret) < 16 {
		return nil, fmt.Errorf("token: secret must be at least 16 bytes")
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	seed := sha256.Sum256([]byte("payconnect/token/v1:" + secret))
	privateKey := ed25519.NewKeyFromSeed(seed[:])
	publicKey, ok := privateKey.Public().(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("token: derived key is not Ed25519")
	}
	return &Signer{
		privateKey: privateKey,
		publicKey:  publicKey,
		issuer:     issuer,
		audience:   audience,
		ttl:        ttl,
	}, nil
}

// Issue signs an access token for a user inside a merchant.
func (s *Signer) Issue(userID, merchantID string) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(s.ttl)
	claims := Claims{
		Sub: userID,
		Org: merchantID,
		Typ: "access",
		Iat: now.Unix(),
		Exp: expiresAt.Unix(),
		Iss: s.issuer,
		Aud: s.audience,
	}
	token, err := s.sign(claims)
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}

// TTL returns the configured access-token lifetime.
func (s *Signer) TTL() time.Duration { return s.ttl }

func (s *Signer) sign(claims Claims) (string, error) {
	header := map[string]string{"alg": algEdDSA, "typ": "JWT"}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("token: encode header: %w", err)
	}
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("token: encode claims: %w", err)
	}
	signingInput := b64(headerJSON) + "." + b64(payloadJSON)
	signature := ed25519.Sign(s.privateKey, []byte(signingInput))
	return signingInput + "." + b64(signature), nil
}

// Verify checks the signature, issuer, audience and expiry.
func (s *Signer) Verify(raw string) (*Claims, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}
	signingInput := parts[0] + "." + parts[1]
	signature, err := unb64(parts[2])
	if err != nil {
		return nil, ErrInvalidToken
	}
	if !ed25519.Verify(s.publicKey, []byte(signingInput), signature) {
		return nil, ErrInvalidToken
	}
	payload, err := unb64(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, ErrInvalidToken
	}
	if claims.Iss != s.issuer || claims.Aud != s.audience {
		return nil, ErrInvalidToken
	}
	if time.Now().UTC().Unix() >= claims.Exp {
		return nil, ErrExpiredToken
	}
	if claims.Sub == "" || claims.Org == "" {
		return nil, ErrInvalidToken
	}
	return &claims, nil
}

func b64(data []byte) string { return base64.RawURLEncoding.EncodeToString(data) }

func unb64(value string) ([]byte, error) { return base64.RawURLEncoding.DecodeString(value) }
