package developer

import (
	"time"

	"github.com/gofiber/fiber/v2"

	apihttp "github.com/kuipid01/payconnect/internal/http"
	"github.com/kuipid01/payconnect/internal/merchants"
	"github.com/kuipid01/payconnect/internal/platform/identity"
)

type Handler struct {
	svc       *Service
	responder *apihttp.Responder
}

func NewHandler(svc *Service, responder *apihttp.Responder) *Handler {
	return &Handler{svc: svc, responder: responder}
}

func (h *Handler) List(c *fiber.Ctx) error {
	principal := identity.From(c)
	developers, err := h.svc.List(principal.MerchantID)
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, developers)
}

// MyAccess handles GET /v1/developers/me. It answers "what was I invited to?"
// for the signed-in identity: each merchant grant, its permissions, and the
// applications and developer-scoped keys it reaches.
func (h *Handler) MyAccess(c *fiber.Ctx) error {
	principal := identity.From(c)
	email := ""
	if user, ok := principal.User.(*merchants.User); ok {
		email = user.Email
	}
	rows, err := h.svc.MyAccess(principal.UserID, email)
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, rows)
}

// TestWebhook handles POST /v1/developers/me/applications/{id}/test-webhook. It
// delivers a sample event to the application's endpoint so a developer can
// verify their receiver end to end.
func (h *Handler) TestWebhook(c *fiber.Ctx) error {
	principal := identity.From(c)
	email := ""
	if user, ok := principal.User.(*merchants.User); ok {
		email = user.Email
	}
	delivery, err := h.svc.TestWebhook(c.UserContext(), principal.UserID, email, c.Params("id"))
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, delivery)
}

func (h *Handler) Invite(c *fiber.Ctx) error {
	var req struct {
		MerchantID  string   `json:"merchant_id"`
		Email       string   `json:"email"`
		Name        string   `json:"name"`
		Permissions []string `json:"permissions"`
	}
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	principal := identity.From(c)
	if req.MerchantID == "" {
		req.MerchantID = principal.MerchantID
	}
	result, err := h.svc.Invite(c.UserContext(), InviteInput{
		MerchantID:  req.MerchantID,
		Email:       req.Email,
		Name:        req.Name,
		Permissions: req.Permissions,
		InvitedBy:   principal.UserID,
	})
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.CreatedCompat(c, result, fiber.Map{"ok": true})
}

// Accept handles POST /v1/developers/accept.
//
// Deprecated: superseded by the token-based AcceptInvitation flow, which works
// for invitees who have no session yet. It remains only so existing clients keep
// working, and it cannot join a second merchant for an existing developer.
func (h *Handler) Accept(c *fiber.Ctx) error {
	var req struct {
		Token string `json:"token"`
	}
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	principal := identity.From(c)
	if err := h.svc.Accept(c.UserContext(), req.Token, principal.UserID); err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.SuccessCompat(c, fiber.Map{"accepted": true}, fiber.Map{"accepted": true})
}

// PreviewInvitation handles GET /v1/developers/invitations/:token. It is public:
// the token is the whole authorization, and the response is scoped to exactly
// what the accept screen must render.
func (h *Handler) PreviewInvitation(c *fiber.Ctx) error {
	preview, err := h.svc.Preview(c.Params("token"))
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, preview)
}

// AcceptInvitation handles POST /v1/developers/invitations/:token/accept. It is
// public by design: the invitee has no session yet, so the single-use token plus
// the password they present *are* the authentication. The session is issued last
// and written to the same cookies login uses.
func (h *Handler) AcceptInvitation(c *fiber.Ctx) error {
	var req struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	result, err := h.svc.AcceptInvitation(c.UserContext(), AcceptInput{
		Token:    c.Params("token"),
		Name:     req.Name,
		Password: req.Password,
	})
	if err != nil {
		return h.responder.Error(c, err)
	}
	h.setSessionCookies(c, result.Session)
	return h.responder.Success(c, fiber.Map{
		"merchant_id":   result.MerchantID,
		"merchant_name": result.MerchantName,
		"role":          result.Role,
	})
}

// setSessionCookies mirrors the auth handler's cookie policy so an accepted
// invitee is signed in exactly as if they had logged in.
func (h *Handler) setSessionCookies(c *fiber.Ctx, session *Session) {
	if session == nil {
		return
	}
	cookie := func(name, value string, maxAge int) {
		c.Cookie(&fiber.Cookie{
			Name:     name,
			Value:    value,
			Path:     "/",
			MaxAge:   maxAge,
			HTTPOnly: true,
			Secure:   c.Protocol() == "https",
			SameSite: fiber.CookieSameSiteLaxMode,
		})
	}
	cookie(identity.CookieAccessToken, session.AccessToken, int(15*time.Minute/time.Second))
	cookie(identity.CookieRefreshToken, session.RefreshToken, int(30*24*time.Hour/time.Second))
}

// Revoke handles DELETE /v1/developers/invitations.
func (h *Handler) Revoke(c *fiber.Ctx) error {
	var req struct {
		InvitationID string `json:"invitation_id"`
	}
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	principal := identity.From(c)
	if err := h.svc.Revoke(principal.MerchantID, req.InvitationID, principal.UserID); err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.SuccessCompat(c, fiber.Map{"revoked": true}, fiber.Map{"revoked": true})
}
