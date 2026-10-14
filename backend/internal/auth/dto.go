package auth

import (
	"github.com/kuipid01/payconnect/internal/merchants"
)

// UserResponse is the public user representation.
type UserResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	Role          string `json:"role"`
	Status        string `json:"status"`
	MerchantID    string `json:"merchant_id"`
	EmailVerified bool   `json:"email_verified"`
}

// ToUserResponse maps a user and selected merchant membership onto the public DTO.
func ToUserResponse(u *merchants.User, membership ...string) UserResponse {
	selectedMerchantID := ""
	selectedRole := u.Role
	if len(membership) > 0 {
		selectedMerchantID = membership[0]
	}
	if len(membership) > 1 {
		selectedRole = membership[1]
	}
	return UserResponse{
		ID:            u.ID,
		Name:          u.Name,
		Email:         u.Email,
		Role:          selectedRole,
		Status:        u.Status,
		MerchantID:    selectedMerchantID,
		EmailVerified: u.EmailVerified,
	}
}

// LegacyUser mirrors the shape the existing dashboard reads at the root of the
// sign-in response.
func LegacyUser(u *merchants.User) map[string]any {
	return map[string]any{
		"id":            u.ID,
		"name":          u.Name,
		"email":         u.Email,
		"role":          u.Role,
		"status":        u.Status,
		"emailVerified": u.EmailVerified,
	}
}

// SessionResponse is the sign-in/refresh payload.
type SessionResponse struct {
	AccessToken  string       `json:"access_token"`
	TokenType    string       `json:"token_type"`
	ExpiresAt    string       `json:"expires_at"`
	RefreshToken string       `json:"refresh_token"`
	User         UserResponse `json:"user"`
}

// ToSessionResponse maps tokens and a user onto the public DTO.
func ToSessionResponse(t *Tokens, u *merchants.User) SessionResponse {
	return SessionResponse{
		AccessToken:  t.AccessToken,
		TokenType:    t.TokenType,
		ExpiresAt:    t.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		RefreshToken: t.RefreshToken,
		User:         ToUserResponse(u, t.MerchantID, t.Role),
	}
}
