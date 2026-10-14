// Package merchants owns the merchant (account) and user domain. A merchant is
// the paying customer of PayConnect; users are its dashboard members.
package merchants

import (
	"time"

	"github.com/kuipid01/payconnect/internal/platform/gormx"
)

// Merchant is an organization that owns providers, applications, and API keys.
type Merchant struct {
	ID              string    `gorm:"primaryKey;size:64" json:"id"`
	OwnerUserID     string    `gorm:"column:owner_user_id;index" json:"owner_user_id"`
	BusinessName    string    `gorm:"column:business_name" json:"business_name"`
	Email           string    `json:"email"`
	Country         string    `json:"country"`
	Status          string    `json:"status"` // active | suspended | pending | closed
	DefaultCurrency string    `gorm:"column:default_currency" json:"default_currency"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName binds the model to the renamed table.
func (Merchant) TableName() string { return "merchants" }

// IsActive reports whether the merchant may transact.
func (m *Merchant) IsActive() bool { return m != nil && m.Status == "active" }

// User is the globally unique login identity for a person.
type User struct {
	ID                         string     `gorm:"primaryKey;size:64" json:"id"`
	Name                       string     `json:"name"`
	Email                      string     `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash               string     `json:"-"`
	GoogleSub                  *string    `gorm:"column:google_sub;uniqueIndex" json:"-"`
	EmailVerified              bool       `gorm:"column:email_verified;not null;default:false" json:"email_verified"`
	EmailVerificationTokenHash *string    `gorm:"column:email_verification_token_hash;uniqueIndex" json:"-"`
	EmailVerificationExpiresAt *time.Time `gorm:"column:email_verification_expires_at" json:"-"`
	PasswordResetTokenHash     *string    `gorm:"column:password_reset_token_hash;uniqueIndex" json:"-"`
	PasswordResetExpiresAt     *time.Time `gorm:"column:password_reset_expires_at" json:"-"`
	Role                       string     `gorm:"-" json:"-"` // selected merchant membership role
	Status                     string     `json:"status"`     // active | revoked
	LastActiveAt               *time.Time `gorm:"column:last_active_at" json:"-"`
	CreatedAt                  time.Time  `json:"created_at"`
}

// TableName binds the model to the users table.
func (User) TableName() string { return "users" }

// IsActive reports whether the user may authenticate.
func (u *User) IsActive() bool { return u != nil && u.Status != "revoked" }

// UserMerchant is a user's membership in one merchant organization.
type UserMerchant struct {
	ID         string    `gorm:"primaryKey;size:64" json:"id"`
	UserID     string    `gorm:"column:user_id;index" json:"user_id"`
	MerchantID string    `gorm:"column:merchant_id;index" json:"merchant_id"`
	Role       string    `json:"role"`   // Owner | Admin | Developer | Viewer
	Status     string    `json:"status"` // active | invited | revoked
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName binds the membership model to the user_merchants table.
func (UserMerchant) TableName() string { return "user_merchants" }

// MembershipView is a membership joined with its organization's name. It is the
// read model behind the organization switcher, so unlike most repository reads
// it is deliberately not scoped to one merchant.
type MembershipView struct {
	MerchantID string    `json:"merchant_id"`
	Merchant   string    `json:"merchant"`
	Role       string    `json:"role"`
	Status     string    `json:"status"`
	Owner      bool      `json:"owner"`
	CreatedAt  time.Time `json:"created_at"`
}

// Developer is an independent developer account owned by one login identity.
type Developer struct {
	ID        string    `gorm:"primaryKey;size:64" json:"id"`
	UserID    string    `gorm:"column:user_id;uniqueIndex" json:"user_id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"` // active | suspended | revoked
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName binds the model to the developers table.
func (Developer) TableName() string { return "developers" }

// DeveloperMerchantAccess is one independently accepted invitation to one merchant.
type DeveloperMerchantAccess struct {
	ID                  string                `gorm:"primaryKey;size:64" json:"id"`
	DeveloperID         *string               `gorm:"column:developer_id;index" json:"developer_id,omitempty"`
	MerchantID          string                `gorm:"column:merchant_id;index" json:"merchant_id"`
	InviteeEmail        string                `gorm:"column:invitee_email" json:"invitee_email"`
	Status              string                `json:"status"` // pending | active | revoked
	Permissions         gormx.JSONB[[]string] `json:"permissions"`
	InvitedBy           string                `gorm:"column:invited_by" json:"invited_by"`
	InvitationTokenHash string                `gorm:"column:invitation_token_hash;uniqueIndex" json:"-"`
	ExpiresAt           *time.Time            `gorm:"column:expires_at" json:"expires_at,omitempty"`
	AcceptedAt          *time.Time            `gorm:"column:accepted_at" json:"accepted_at,omitempty"`
	CreatedAt           time.Time             `json:"created_at"`
	UpdatedAt           time.Time             `gorm:"column:updated_at" json:"updated_at"`
}

// TableName binds the model to the developer_merchant_access table.
func (DeveloperMerchantAccess) TableName() string { return "developer_merchant_access" }

// DeveloperAccess is the legacy application-scoped grant retained for existing
// developer-facing API-key workflows.
type DeveloperAccess struct {
	ID            string                `gorm:"primaryKey;size:64" json:"id"`
	MerchantID    string                `gorm:"column:merchant_id" json:"-"`
	UserID        string                `gorm:"column:user_id" json:"user_id"`
	ApplicationID string                `gorm:"column:application_id" json:"application_id"`
	Permissions   gormx.JSONB[[]string] `json:"permissions"`
	CreatedAt     time.Time             `json:"created_at"`
}

// TableName binds the model to the developer_access table.
func (DeveloperAccess) TableName() string { return "developer_access" }

// RefreshToken is a rotating, revocable dashboard session credential. Only the
// SHA-256 hash is stored.
type RefreshToken struct {
	ID         string     `gorm:"primaryKey;size:64" json:"-"`
	MerchantID string     `gorm:"column:merchant_id" json:"-"`
	UserID     string     `gorm:"column:user_id" json:"-"`
	TokenHash  string     `gorm:"column:token_hash;uniqueIndex;size:88" json:"-"`
	ExpiresAt  time.Time  `gorm:"column:expires_at" json:"-"`
	RevokedAt  *time.Time `gorm:"column:revoked_at" json:"-"`
	CreatedAt  time.Time  `gorm:"column:created_at" json:"-"`
}

// TableName binds the model to the refresh_tokens table.
func (RefreshToken) TableName() string { return "refresh_tokens" }
