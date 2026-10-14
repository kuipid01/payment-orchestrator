package merchants

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

var (
	ErrTeamMemberExists   = errors.New("merchants: team member already exists")
	ErrTeamOwnerProtected = errors.New("merchants: owner membership cannot be changed")
)

// Repository owns persistence for merchants, users, access grants and refresh
// tokens. It performs no business decisions and calls no external services.
type Repository struct{ DB *gorm.DB }

// NewRepository builds the merchant repository.
func NewRepository(db *gorm.DB) *Repository { return &Repository{DB: db} }

/* --------------------------------- merchants -------------------------------- */

// CreateMerchant inserts a merchant.
func (r *Repository) CreateMerchant(m *Merchant) error { return r.DB.Create(m).Error }

// CreateMerchantAndUser inserts an identity, merchant, and owner membership in one transaction.
func (r *Repository) CreateMerchantAndUser(m *Merchant, u *User) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		m.OwnerUserID = u.ID
		if err := tx.Create(m).Error; err != nil {
			return err
		}
		if err := tx.Create(u).Error; err != nil {
			return err
		}
		return tx.Create(&UserMerchant{
			ID:         idForMembership(),
			UserID:     u.ID,
			MerchantID: m.ID,
			Role:       "Owner",
			Status:     "active",
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
		}).Error
	})
}

// MerchantByID loads a merchant scoped by id.
func (r *Repository) MerchantByID(id string) (*Merchant, error) {
	var m Merchant
	if err := r.DB.First(&m, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// UpdateMerchant persists profile changes.
func (r *Repository) UpdateMerchant(m *Merchant) error {
	m.UpdatedAt = time.Now().UTC()
	return r.DB.Save(m).Error
}

/* ----------------------------------- users ---------------------------------- */

// CreateUser inserts a user.
func (r *Repository) CreateUser(u *User) error { return r.DB.Create(u).Error }

// CreateUserMerchant creates an explicit merchant membership.
func (r *Repository) CreateUserMerchant(m *UserMerchant) error {
	return r.DB.Create(m).Error
}

// UserByEmail loads a user by lower-cased email.
func (r *Repository) UserByEmail(email string) (*User, error) {
	var u User
	if err := r.DB.First(&u, "email = ?", email).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// UpdateUserName persists a user's display name.
func (r *Repository) UpdateUserName(userID, name string) error {
	return r.DB.Model(&User{}).Where("id = ?", userID).Update("name", name).Error
}

// ChangeUserPassword persists a password update and revokes every refresh
// session atomically.
func (r *Repository) ChangeUserPassword(userID, passwordHash string, at time.Time) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&User{}).Where("id = ?", userID).Update("password_hash", passwordHash).Error; err != nil {
			return err
		}
		return tx.Model(&RefreshToken{}).
			Where("user_id = ? AND revoked_at IS NULL", userID).
			Update("revoked_at", at).Error
	})
}

// UserByID loads a user by id.
func (r *Repository) UserByID(id string) (*User, error) {
	var u User
	if err := r.DB.First(&u, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// UserByGoogleSub loads a user by Google subject.
func (r *Repository) UserByGoogleSub(subject string) (*User, error) {
	var u User
	if err := r.DB.First(&u, "google_sub = ?", subject).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// LinkGoogleSub attaches a Google subject and marks the email verified.
func (r *Repository) LinkGoogleSub(userID, subject string) error {
	return r.DB.Model(&User{}).
		Where("id = ? AND (google_sub IS NULL OR google_sub = ?)", userID, subject).
		Updates(map[string]any{
			"google_sub":                    subject,
			"email_verified":                true,
			"email_verification_token_hash": nil,
			"email_verification_expires_at": nil,
		}).Error
}

// UserMerchants lists the merchant memberships for a user.
func (r *Repository) UserMerchants(userID string) ([]UserMerchant, error) {
	var memberships []UserMerchant
	if err := r.DB.Where("user_id = ?", userID).Order("created_at").Find(&memberships).Error; err != nil {
		return nil, err
	}
	return memberships, nil
}

// MembershipsByUser lists every organization a user belongs to, with its name.
//
// It is intentionally not scoped to the caller's current merchant: this is what
// lets a developer who accepted invitations from several merchants see all of
// them, even though a session only ever acts in one at a time.
func (r *Repository) MembershipsByUser(userID string) ([]MembershipView, error) {
	var rows []MembershipView
	err := r.DB.Table("user_merchants AS um").
		Select("um.merchant_id, m.business_name AS merchant, um.role, um.status, (m.owner_user_id = um.user_id) AS owner, um.created_at").
		Joins("JOIN merchants AS m ON m.id = um.merchant_id").
		Where("um.user_id = ?", userID).
		Order("um.created_at").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// DefaultUserMerchant returns the first active membership for a user.
func (r *Repository) DefaultUserMerchant(userID string) (*UserMerchant, error) {
	var membership UserMerchant
	if err := r.DB.Where("user_id = ? AND status = ?", userID, "active").Order("created_at").First(&membership).Error; err != nil {
		return nil, err
	}
	return &membership, nil
}

// UserMerchantByID loads one membership by its unique membership ID.
func (r *Repository) UserMerchantByID(id string) (*UserMerchant, error) {
	var membership UserMerchant
	if err := r.DB.First(&membership, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &membership, nil
}

// UserMerchantByUserAndMerchant resolves an active membership by its two entity IDs.
func (r *Repository) UserMerchantByUserAndMerchant(userID, merchantID string) (*UserMerchant, error) {
	var membership UserMerchant
	if err := r.DB.Where("user_id = ? AND merchant_id = ? AND status = ?", userID, merchantID, "active").First(&membership).Error; err != nil {
		return nil, err
	}
	return &membership, nil
}

// UsersByMerchant lists active members of a merchant.
func (r *Repository) UsersByMerchant(merchantID string, developersOnly bool) ([]User, error) {
	var memberships []UserMerchant
	q := r.DB.Where("merchant_id = ? AND status = ?", merchantID, "active")
	if err := q.Find(&memberships).Error; err != nil {
		return nil, err
	}
	if !developersOnly {
		var users []User
		for _, membership := range memberships {
			user, err := r.UserByID(membership.UserID)
			if err != nil {
				continue
			}
			users = append(users, *user)
		}
		return users, nil
	}

	var userIDs []string
	for _, membership := range memberships {
		userIDs = append(userIDs, membership.UserID)
	}
	var users []User
	if len(userIDs) == 0 {
		return users, nil
	}
	if err := r.DB.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// UpdateUserStatus changes a member's status.
func (r *Repository) UpdateUserStatus(id, status string) error {
	return r.DB.Model(&UserMerchant{}).Where("id = ?", id).Update("status", status).Error
}

// TeamMemberRecord is one membership joined with its login identity.
type TeamMemberRecord struct {
	MembershipID string     `json:"id"`
	UserID       string     `json:"-"`
	Name         string     `json:"name"`
	Email        string     `json:"email"`
	Role         string     `json:"role"`
	Status       string     `json:"status"`
	LastActiveAt *time.Time `json:"last_active_at"`
}

// TeamByMerchant lists active account memberships for a merchant.
func (r *Repository) TeamByMerchant(merchantID string) ([]TeamMemberRecord, error) {
	var rows []TeamMemberRecord
	err := r.DB.Table("user_merchants AS um").
		Select("um.id AS membership_id, um.user_id, users.name, users.email, um.role, um.status, users.last_active_at").
		Joins("JOIN users ON users.id = um.user_id").
		Where("um.merchant_id = ? AND um.status = ?", merchantID, "active").
		Order("um.created_at ASC").
		Scan(&rows).Error
	return rows, err
}

// AddTeamMember creates or reactivates a membership without changing an existing
// owner or administrator role.
func (r *Repository) AddTeamMember(merchantID, userID, role string, at time.Time) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		var membership UserMerchant
		err := tx.Where("merchant_id = ? AND user_id = ?", merchantID, userID).First(&membership).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return tx.Create(&UserMerchant{
				ID:         idForMembership(),
				UserID:     userID,
				MerchantID: merchantID,
				Role:       role,
				Status:     "active",
				CreatedAt:  at,
				UpdatedAt:  at,
			}).Error
		case err != nil:
			return err
		case membership.Status == "active":
			return ErrTeamMemberExists
		case membership.Role == "Owner":
			return ErrTeamOwnerProtected
		default:
			return tx.Model(&membership).Updates(map[string]any{
				"role": role, "status": "active", "updated_at": at,
			}).Error
		}
	})
}

// RevokeTeamMember revokes one non-owner membership scoped to its merchant.
func (r *Repository) RevokeTeamMember(merchantID, membershipID, actorUserID string, at time.Time) error {
	result := r.DB.Model(&UserMerchant{}).
		Where(
			"merchant_id = ? AND id = ? AND role <> ? AND status = ? AND user_id <> ?",
			merchantID, membershipID, "Owner", "active", actorUserID,
		).
		Updates(map[string]any{"status": "revoked", "updated_at": at})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// TouchLastActive records activity for the dashboard roster.
func (r *Repository) TouchLastActive(id string, at time.Time) error {
	return r.DB.Model(&User{}).Where("id = ?", id).Update("last_active_at", at).Error
}

/* ------------------------------- email tokens -------------------------------- */

// SetEmailVerification stores a fresh verification token hash.
func (r *Repository) SetEmailVerification(userID, tokenHash string, expiresAt time.Time) error {
	res := r.DB.Model(&User{}).Where("id = ? AND email_verified = FALSE", userID).Updates(map[string]any{
		"email_verification_token_hash": tokenHash,
		"email_verification_expires_at": expiresAt,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// VerifyEmail consumes a verification token atomically.
func (r *Repository) VerifyEmail(userID, tokenHash string, now time.Time) error {
	res := r.DB.Model(&User{}).Where(
		"id = ? AND email_verified = FALSE AND email_verification_token_hash = ? AND email_verification_expires_at > ?",
		userID, tokenHash, now,
	).Updates(map[string]any{
		"email_verified":                true,
		"email_verification_token_hash": nil,
		"email_verification_expires_at": nil,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UserByVerificationToken resolves a pending verification token.
func (r *Repository) UserByVerificationToken(tokenHash string, now time.Time) (*User, error) {
	var u User
	err := r.DB.Where(
		"email_verification_token_hash = ? AND email_verified = FALSE AND email_verification_expires_at > ?",
		tokenHash, now,
	).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// SetPasswordReset stores a fresh password-reset token hash.
func (r *Repository) SetPasswordReset(userID, tokenHash string, expiresAt time.Time) error {
	res := r.DB.Model(&User{}).Where("id = ?", userID).Updates(map[string]any{
		"password_reset_token_hash": tokenHash,
		"password_reset_expires_at": expiresAt,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UserByPasswordResetToken resolves a pending reset token.
func (r *Repository) UserByPasswordResetToken(tokenHash string, now time.Time) (*User, error) {
	var u User
	err := r.DB.Where(
		"password_reset_token_hash = ? AND password_reset_expires_at > ?", tokenHash, now,
	).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// ResetPassword sets a new password and consumes the reset token.
func (r *Repository) ResetPassword(userID, tokenHash, passwordHash string, now time.Time) (*User, error) {
	var user User
	err := r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&user, "id = ?", userID).Error; err != nil {
			return err
		}
		res := tx.Model(&User{}).Where(
			"id = ? AND password_reset_token_hash = ? AND password_reset_expires_at > ?",
			userID, tokenHash, now,
		).Updates(map[string]any{
			"password_hash":             passwordHash,
			"password_reset_token_hash": nil,
			"password_reset_expires_at": nil,
			"last_active_at":            now,
		})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &user, nil
}

/* ----------------------------- developer access ------------------------------ */

// UpsertAccess grants (or updates) a developer's permissions on an application.
func (r *Repository) UpsertAccess(a *DeveloperAccess) error {
	return r.DB.Exec(
		`INSERT INTO developer_access (id, merchant_id, user_id, application_id, permissions, created_at)
		 VALUES (?, ?, ?, ?, ?::jsonb, ?)
		 ON CONFLICT (user_id, application_id) DO UPDATE SET permissions = EXCLUDED.permissions`,
		a.ID, a.MerchantID, a.UserID, a.ApplicationID, rawPermissions(a.Permissions.Data), a.CreatedAt,
	).Error
}

// AccessByUser lists a user's grants.
func (r *Repository) AccessByUser(userID string) ([]DeveloperAccess, error) {
	var rows []DeveloperAccess
	if err := r.DB.Where("user_id = ?", userID).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// RevokeAccess deletes every grant a user holds.
func (r *Repository) RevokeAccess(userID string) error {
	return r.DB.Where("user_id = ?", userID).Delete(&DeveloperAccess{}).Error
}

// CreateDeveloper creates a developer account for a login identity.
func (r *Repository) CreateDeveloper(d *Developer) error {
	return r.DB.Create(d).Error
}

// DeveloperByUser loads a developer account for a user.
func (r *Repository) DeveloperByUser(userID string) (*Developer, error) {
	var developer Developer
	if err := r.DB.Where("user_id = ?", userID).First(&developer).Error; err != nil {
		return nil, err
	}
	return &developer, nil
}

// DeveloperMerchantAccessesByMerchant lists invitations for a merchant.
func (r *Repository) DeveloperMerchantAccessesByMerchant(merchantID string) ([]DeveloperMerchantAccess, error) {
	var accesses []DeveloperMerchantAccess
	if err := r.DB.Where("merchant_id = ?", merchantID).Order("created_at DESC").Find(&accesses).Error; err != nil {
		return nil, err
	}
	return accesses, nil
}

// DeveloperMerchantAccessesByMerchantAndStatus lists invitations with a status filter.
func (r *Repository) DeveloperMerchantAccessesByMerchantAndStatus(merchantID, status string) ([]DeveloperMerchantAccess, error) {
	var accesses []DeveloperMerchantAccess
	if err := r.DB.Where("merchant_id = ? AND status = ?", merchantID, status).Order("created_at DESC").Find(&accesses).Error; err != nil {
		return nil, err
	}
	return accesses, nil
}

// CreateDeveloperMerchantAccess creates one independent merchant invitation.
func (r *Repository) CreateDeveloperMerchantAccess(a *DeveloperMerchantAccess) error {
	return r.DB.Create(a).Error
}

// DeveloperMerchantAccessByID loads one invitation.
func (r *Repository) DeveloperMerchantAccessByID(id string) (*DeveloperMerchantAccess, error) {
	var access DeveloperMerchantAccess
	if err := r.DB.First(&access, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &access, nil
}

// DeveloperMerchantAccessByIDOrDeveloper finds a merchant access by its
// invitation ID (pending invites) or developer ID (accepted access).
func (r *Repository) DeveloperMerchantAccessByIDOrDeveloper(merchantID, id string) (*DeveloperMerchantAccess, error) {
	var access DeveloperMerchantAccess
	if err := r.DB.Where("merchant_id = ? AND (id = ? OR developer_id = ?)", merchantID, id, id).
		First(&access).Error; err != nil {
		return nil, err
	}
	return &access, nil
}

// DeveloperMerchantAccessByDeveloperAndMerchant loads an invitation by pair.
func (r *Repository) DeveloperMerchantAccessByDeveloperAndMerchant(developerID, merchantID string) (*DeveloperMerchantAccess, error) {
	var access DeveloperMerchantAccess
	if err := r.DB.Where("developer_id = ? AND merchant_id = ?", developerID, merchantID).First(&access).Error; err != nil {
		return nil, err
	}
	return &access, nil
}

// DeveloperMerchantAccessesByUser lists every merchant grant held by a signed-in
// developer. It is the read behind the developer platform's "your access" view,
// so unlike most reads here it deliberately crosses merchants and is keyed by
// the login identity rather than one organization.
func (r *Repository) DeveloperMerchantAccessesByUser(userID string) ([]DeveloperMerchantAccess, error) {
	var accesses []DeveloperMerchantAccess
	err := r.DB.Model(&DeveloperMerchantAccess{}).
		Joins("JOIN developers ON developers.id = developer_merchant_access.developer_id").
		Where("developers.user_id = ?", userID).
		Order("developer_merchant_access.created_at DESC").
		Find(&accesses).Error
	return accesses, err
}

// DeveloperMerchantAccessesByEmail lists every grant addressed to an email
// across all merchants. It lets a signed-in invitee see an invitation they have
// not accepted yet, before any developer record exists.
func (r *Repository) DeveloperMerchantAccessesByEmail(email string) ([]DeveloperMerchantAccess, error) {
	var accesses []DeveloperMerchantAccess
	err := r.DB.Where("lower(invitee_email) = ?", strings.ToLower(strings.TrimSpace(email))).
		Order("created_at DESC").
		Find(&accesses).Error
	return accesses, err
}

// DeveloperMerchantAccessByEmail finds an invitation for a merchant and email.
func (r *Repository) DeveloperMerchantAccessByEmail(merchantID, email string) (*DeveloperMerchantAccess, error) {
	var access DeveloperMerchantAccess
	if err := r.DB.Where("merchant_id = ? AND lower(invitee_email) = ?", merchantID, strings.ToLower(email)).First(&access).Error; err != nil {
		return nil, err
	}
	return &access, nil
}

// DeveloperMerchantAccessByTokenHash loads a pending invitation by its opaque token hash.
func (r *Repository) DeveloperMerchantAccessByTokenHash(tokenHash string) (*DeveloperMerchantAccess, error) {
	var access DeveloperMerchantAccess
	if err := r.DB.Where("invitation_token_hash = ?", tokenHash).First(&access).Error; err != nil {
		return nil, err
	}
	return &access, nil
}

// ErrDeveloperRevoked reports that the identity is a developer whose access was
// revoked. A revoked developer may not silently rejoin by accepting a fresh
// invitation.
var ErrDeveloperRevoked = errors.New("merchants: developer access is revoked")

// ErrInvitationUnavailable reports that an invitation is no longer pending and
// does not already belong to the developer presenting the token.
var ErrInvitationUnavailable = errors.New("merchants: invitation is not available")

// AcceptDeveloperInvitationInput is the complete unit of work for a developer
// joining a merchant through an invitation.
type AcceptDeveloperInvitationInput struct {
	// User is created inside the transaction when IsNewUser is set.
	User      *User
	IsNewUser bool
	// Developer is created when the identity has no developer profile yet.
	Developer    *Developer
	MerchantID   string
	InvitationID string
	Now          time.Time
}

// AcceptDeveloperInvitation atomically creates the login identity (when new) and
// the developer profile (on first join), activates the membership, and consumes
// the invitation. Like CreateMerchantAndUser the repository owns the transaction,
// because the whole unit of work is persistence: a partial write would leave an
// account without access, or a grant without an owner.
//
// Contract:
//   - an existing membership keeps an Owner or Admin role; everyone else settles
//     on Developer, so accepting an invitation never demotes an organization's owner.
//   - the invitation update is predicated on `status = 'pending'`, making a
//     replay idempotent: it succeeds only when the invitation is already active
//     for this same developer, and otherwise reports ErrInvitationUnavailable.
//   - a revoked developer profile reports ErrDeveloperRevoked.
func (r *Repository) AcceptDeveloperInvitation(in AcceptDeveloperInvitationInput) (*Developer, error) {
	developer := in.Developer
	err := r.DB.Transaction(func(tx *gorm.DB) error {
		if in.IsNewUser {
			if err := tx.Create(in.User).Error; err != nil {
				return err
			}
		}

		var existing Developer
		switch lookupErr := tx.Where("user_id = ?", in.User.ID).First(&existing).Error; {
		case errors.Is(lookupErr, gorm.ErrRecordNotFound):
			if err := tx.Create(developer).Error; err != nil {
				return err
			}
		case lookupErr != nil:
			return lookupErr
		default:
			// One developer profile per identity, shared across merchants.
			developer = &existing
		}
		if developer.Status == "revoked" {
			return ErrDeveloperRevoked
		}

		if err := upsertMembership(tx, in.User.ID, in.MerchantID, in.Now); err != nil {
			return err
		}

		result := tx.Model(&DeveloperMerchantAccess{}).
			Where("id = ? AND status = ?", in.InvitationID, "pending").
			Updates(map[string]any{
				"developer_id": developer.ID,
				"status":       "active",
				"accepted_at":  in.Now,
				"updated_at":   in.Now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			var current DeveloperMerchantAccess
			if err := tx.First(&current, "id = ?", in.InvitationID).Error; err != nil {
				return err
			}
			if current.Status != "active" || current.DeveloperID == nil || *current.DeveloperID != developer.ID {
				return ErrInvitationUnavailable
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return developer, nil
}

// upsertMembership grants an active membership, creating it when absent. An
// existing Owner or Admin keeps their role, so an invitation can never demote
// the person who owns the organization.
func upsertMembership(tx *gorm.DB, userID, merchantID string, at time.Time) error {
	var membership UserMerchant
	err := tx.Where("user_id = ? AND merchant_id = ?", userID, merchantID).First(&membership).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return tx.Create(&UserMerchant{
			ID:         idForMembership(),
			UserID:     userID,
			MerchantID: merchantID,
			Role:       "Developer",
			Status:     "active",
			CreatedAt:  at,
			UpdatedAt:  at,
		}).Error
	}
	if err != nil {
		return err
	}
	updates := map[string]any{"status": "active", "updated_at": at}
	if membership.Role != "Owner" && membership.Role != "Admin" {
		updates["role"] = "Developer"
	}
	return tx.Model(&UserMerchant{}).Where("id = ?", membership.ID).Updates(updates).Error
}

// RevokeDeveloperMerchantAccess revokes one invitation.
func (r *Repository) RevokeDeveloperMerchantAccess(id string) error {
	return r.DB.Model(&DeveloperMerchantAccess{}).
		Where("id = ? AND status != ?", id, "revoked").
		Update("status", "revoked").Error
}

// RemoveDeveloperMerchantAccess revokes one merchant's invitation and login
// membership, without affecting the developer's access to other merchants.
func (r *Repository) RemoveDeveloperMerchantAccess(merchantID, accessID string, now time.Time) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		var access DeveloperMerchantAccess
		if err := tx.Where("merchant_id = ? AND id = ?", merchantID, accessID).First(&access).Error; err != nil {
			return err
		}
		if err := tx.Model(&DeveloperMerchantAccess{}).
			Where("merchant_id = ? AND id = ?", merchantID, accessID).
			Updates(map[string]any{"status": "revoked", "updated_at": now}).Error; err != nil {
			return err
		}
		if access.DeveloperID == nil {
			return nil
		}

		var developer Developer
		if err := tx.First(&developer, "id = ?", *access.DeveloperID).Error; err != nil {
			return err
		}
		if err := tx.Model(&UserMerchant{}).
			Where("merchant_id = ? AND user_id = ? AND role = ?", merchantID, developer.UserID, "Developer").
			Updates(map[string]any{"status": "revoked", "updated_at": now}).Error; err != nil {
			return err
		}
		return tx.Where("merchant_id = ? AND user_id = ?", merchantID, developer.UserID).
			Delete(&DeveloperAccess{}).Error
	})
}

/* ------------------------------ refresh tokens ------------------------------- */

// CreateRefreshToken stores a refresh token hash.
func (r *Repository) CreateRefreshToken(merchantID, userID, tokenHash string, expiresAt, createdAt time.Time) error {
	return r.DB.Create(&RefreshToken{
		ID:         idForRefresh(),
		MerchantID: merchantID,
		UserID:     userID,
		TokenHash:  tokenHash,
		ExpiresAt:  expiresAt,
		CreatedAt:  createdAt,
	}).Error
}

// RefreshTokenByHash loads a refresh token by its hash.
func (r *Repository) RefreshTokenByHash(tokenHash string) (*RefreshToken, error) {
	var token RefreshToken
	if err := r.DB.First(&token, "token_hash = ?", tokenHash).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

// ActiveRefreshTokens lists unexpired refresh tokens for one user in one
// merchant context. Raw token hashes are never returned from this repository
// method's callers.
func (r *Repository) ActiveRefreshTokens(userID, merchantID string, at time.Time) ([]RefreshToken, error) {
	var tokens []RefreshToken
	err := r.DB.Where(
		"user_id = ? AND merchant_id = ? AND revoked_at IS NULL AND expires_at > ?",
		userID, merchantID, at,
	).Order("created_at DESC").Find(&tokens).Error
	return tokens, err
}

// RevokeRefreshToken revokes one token.
func (r *Repository) RevokeRefreshToken(tokenHash string, at time.Time) error {
	return r.DB.Model(&RefreshToken{}).
		Where("token_hash = ? AND revoked_at IS NULL", tokenHash).
		Update("revoked_at", at).Error
}

// RevokeRefreshTokenByID revokes one session by its opaque database ID, scoped
// to both the user and merchant that own it.
func (r *Repository) RevokeRefreshTokenByID(userID, merchantID, id string, at time.Time) error {
	result := r.DB.Model(&RefreshToken{}).
		Where("id = ? AND user_id = ? AND merchant_id = ? AND revoked_at IS NULL", id, userID, merchantID).
		Update("revoked_at", at)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// RevokeUserRefreshTokens revokes every live token for a user (logout-all,
// password reset, or reuse detection).
func (r *Repository) RevokeUserRefreshTokens(userID string, at time.Time) error {
	return r.DB.Model(&RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", at).Error
}

func rawPermissions(perms []string) string {
	return gormxJSONStrings(perms)
}
