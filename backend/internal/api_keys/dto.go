package api_keys

import "time"

// Response is the public key representation. The secret is never included.
type Response struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Prefix        string `json:"prefix"`
	Environment   string `json:"environment"`
	Status        string `json:"status"`
	Masked        string `json:"masked"`
	SecretMasked  string `json:"secret_masked"`
	ApplicationID string `json:"application_id,omitempty"`
	// Scope is "general" (team-shared) or "developer" (one grant).
	Scope string `json:"scope"`
	// DeveloperAccessID is set only for developer-scoped keys.
	DeveloperAccessID string `json:"developer_access_id,omitempty"`
	// Revealable reports whether the secret can be read again through
	// POST /api-keys/{id}/reveal. True for test keys, false for live keys (shown
	// once at creation) and for anything minted before test-key recovery existed.
	Revealable bool    `json:"revealable"`
	CreatedBy  string  `json:"created_by"`
	LastUsedAt *string `json:"last_used_at"`
	ExpiresAt  *string `json:"expires_at"`
	RevokedAt  *string `json:"revoked_at"`
	CreatedAt  string  `json:"created_at"`
}

// ToResponse maps a key onto the public DTO.
func ToResponse(k *Key, now time.Time) Response {
	return Response{
		ID:                k.ID,
		Name:              k.Name,
		Type:              k.Type,
		Prefix:            k.Prefix,
		Environment:       k.Environment,
		Status:            k.Status(now),
		Masked:            k.SecretMasked,
		SecretMasked:      k.SecretMasked,
		ApplicationID:     k.ApplicationIDValue(),
		Scope:             k.Scope(),
		DeveloperAccessID: k.DeveloperAccessIDValue(),
		Revealable:        k.Recoverable(),
		CreatedBy:         k.CreatedBy,
		LastUsedAt:        formatTime(k.LastUsedAt),
		ExpiresAt:         formatTime(k.ExpiresAt),
		RevokedAt:         formatTime(k.RevokedAt),
		CreatedAt:         k.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// CreatedResponse carries the one-time secret plus the key metadata.
type CreatedResponse struct {
	Key    Response `json:"key"`
	Secret string   `json:"secret"`
}

func formatTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	formatted := t.UTC().Format(time.RFC3339)
	return &formatted
}
