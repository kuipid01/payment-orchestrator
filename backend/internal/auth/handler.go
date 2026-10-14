package auth

import (
	"time"

	"github.com/gofiber/fiber/v2"

	apperrors "github.com/kuipid01/payconnect/internal/errors"
	apihttp "github.com/kuipid01/payconnect/internal/http"
	"github.com/kuipid01/payconnect/internal/merchants"
	"github.com/kuipid01/payconnect/internal/platform/identity"
)

// Handler maps HTTP requests onto the auth service.
type Handler struct {
	svc       *Service
	responder *apihttp.Responder
}

// NewHandler builds the auth handler.
func NewHandler(svc *Service, responder *apihttp.Responder) *Handler {
	return &Handler{svc: svc, responder: responder}
}

// Register handles POST /v1/auth/register.
func (h *Handler) Register(c *fiber.Ctx) error {
	var req struct {
		Name             string `json:"name"`
		Email            string `json:"email"`
		Password         string `json:"password"`
		OrganizationName string `json:"organizationName"`
		BusinessName     string `json:"business_name"`
	}
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	businessName := req.BusinessName
	if businessName == "" {
		businessName = req.OrganizationName
	}
	user, err := h.svc.Register(c.UserContext(), RegisterInput{
		Name:         req.Name,
		Email:        req.Email,
		Password:     req.Password,
		BusinessName: businessName,
	})
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.CreatedCompat(c,
		fiber.Map{"verificationRequired": true, "user": ToUserResponse(user)},
		fiber.Map{"verificationRequired": true, "user": LegacyUser(user)})
}

// Login handles POST /v1/auth/login.
func (h *Handler) Login(c *fiber.Ctx) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	tokens, user, err := h.svc.Login(req.Email, req.Password)
	if err != nil {
		return h.responder.Error(c, err)
	}
	h.setSessionCookies(c, tokens)
	return h.sessionResponse(c, tokens, user)
}

// Refresh handles POST /v1/auth/refresh.
func (h *Handler) Refresh(c *fiber.Ctx) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
		Refresh      string `json:"refreshToken"`
	}
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	value := req.RefreshToken
	if value == "" {
		value = req.Refresh
	}
	if value == "" {
		value = c.Cookies(identity.CookieRefreshToken)
	}
	tokens, user, err := h.svc.Refresh(value)
	if err != nil {
		return h.responder.Error(c, err)
	}
	h.setSessionCookies(c, tokens)
	return h.sessionResponse(c, tokens, user)
}

// Logout handles POST /v1/auth/logout by revoking the supplied refresh token.
func (h *Handler) Logout(c *fiber.Ctx) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
		Refresh      string `json:"refreshToken"`
	}
	_ = apihttp.Body(c, &req)
	value := req.RefreshToken
	if value == "" {
		value = req.Refresh
	}
	if value == "" {
		value = c.Cookies(identity.CookieRefreshToken)
	}
	h.svc.Logout(value)
	c.Cookie(&fiber.Cookie{Name: identity.CookieRefreshToken, Value: "", Path: "/", MaxAge: -1, HTTPOnly: true, Secure: c.Protocol() == "https", SameSite: fiber.CookieSameSiteLaxMode})
	c.Cookie(&fiber.Cookie{Name: identity.CookieAccessToken, Value: "", Path: "/", MaxAge: -1, HTTPOnly: true, Secure: c.Protocol() == "https", SameSite: fiber.CookieSameSiteLaxMode})
	return h.responder.SuccessCompat(c, fiber.Map{"revoked": true}, fiber.Map{"ok": true})
}

// VerifyEmail handles POST /v1/auth/verify-email.
func (h *Handler) VerifyEmail(c *fiber.Ctx) error {
	var req struct {
		Token string `json:"token"`
	}
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	if err := h.svc.VerifyEmail(req.Token); err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.SuccessCompat(c, fiber.Map{"verified": true}, fiber.Map{"verified": true})
}

// ResendVerification handles POST /v1/auth/resend-verification.
func (h *Handler) ResendVerification(c *fiber.Ctx) error {
	var req struct {
		Email string `json:"email"`
	}
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	if err := h.svc.ResendVerification(c.UserContext(), req.Email); err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.SuccessCompat(c,
		fiber.Map{"message": "If the account needs verification, an email will be sent."},
		fiber.Map{"ok": true, "message": "If the account needs verification, an email will be sent."})
}

// ForgotPassword handles POST /v1/auth/forgot-password.
func (h *Handler) ForgotPassword(c *fiber.Ctx) error {
	var req struct {
		Email string `json:"email"`
	}
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	if err := h.svc.ForgotPassword(c.UserContext(), req.Email); err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.SuccessCompat(c,
		fiber.Map{"message": "If the account exists, a password reset email has been sent."},
		fiber.Map{"ok": true, "message": "If the account exists, a password reset email has been sent."})
}

// ResetPassword handles POST /v1/auth/reset-password.
func (h *Handler) ResetPassword(c *fiber.Ctx) error {
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	user, err := h.svc.ResetPassword(req.Token, req.Password)
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.SuccessCompat(c,
		fiber.Map{"reset": true, "user": ToUserResponse(user)},
		fiber.Map{"ok": true})
}

// Google handles POST /v1/auth/google.
func (h *Handler) Google(c *fiber.Ctx) error {
	var req struct {
		Credential string `json:"credential"`
	}
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	tokens, user, err := h.svc.LoginWithGoogle(c.UserContext(), req.Credential)
	if err != nil {
		return h.responder.Error(c, err)
	}
	h.setSessionCookies(c, tokens)
	return h.sessionResponse(c, tokens, user)
}

// Me handles GET /v1/auth/me. It tells the dashboard which principal is
// signed in and, for sessions, returns the full user profile so the client can
// hydrate without a second call.
func (h *Handler) Me(c *fiber.Ctx) error {
	principal := identity.From(c)
	if principal == nil {
		return h.responder.Error(c, apperrors.AuthenticationRequired())
	}
	body := fiber.Map{
		"mode":        principal.Mode,
		"merchant_id": principal.MerchantID,
		"user_id":     principal.UserID,
		"environment": principal.Environment,
	}
	if user, ok := principal.User.(*merchants.User); ok {
		body["user"] = ToUserResponse(user, principal.MerchantID, user.Role)
		if merchant, err := h.svc.Merchant(principal.MerchantID); err == nil {
			body["organization"] = merchant.BusinessName
		}
	}
	return h.responder.Success(c, body)
}

// Memberships handles GET /v1/auth/memberships: every organization the signed-in
// user can act in, with the current one marked. This is what lets one dashboard
// serve an owner who also develops for other merchants.
func (h *Handler) Memberships(c *fiber.Ctx) error {
	principal := identity.From(c)
	if principal == nil || principal.UserID == "" {
		return h.responder.Error(c, apperrors.AuthenticationRequired())
	}
	views, err := h.svc.Memberships(principal.UserID, principal.MerchantID)
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, views)
}

// Switch handles POST /v1/auth/switch: it re-issues the session bound to another
// organization the caller is already an active member of. IssueFor re-derives
// the role from that membership and refuses anything else, so a caller cannot
// switch into an organization they have no access to.
func (h *Handler) Switch(c *fiber.Ctx) error {
	var req struct {
		MerchantID string `json:"merchant_id"`
		Merchant   string `json:"merchantId"`
	}
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	merchantID := req.MerchantID
	if merchantID == "" {
		merchantID = req.Merchant
	}
	if merchantID == "" {
		return h.responder.Error(c, apperrors.Validation(apperrors.CodeValidationError, "merchant_id is required."))
	}
	principal := identity.From(c)
	if principal == nil || principal.UserID == "" {
		return h.responder.Error(c, apperrors.AuthenticationRequired())
	}
	tokens, user, err := h.svc.IssueFor(principal.UserID, merchantID)
	if err != nil {
		return h.responder.Error(c, err)
	}
	h.setSessionCookies(c, tokens)
	return h.sessionResponse(c, tokens, user)
}

func (h *Handler) setSessionCookies(c *fiber.Ctx, tokens *Tokens) {
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
	cookie(identity.CookieAccessToken, tokens.AccessToken, int(15*time.Minute/time.Second))
	cookie(identity.CookieRefreshToken, tokens.RefreshToken, int(30*24*time.Hour/time.Second))
}

func (h *Handler) sessionResponse(c *fiber.Ctx, tokens *Tokens, user any) error {
	u, _ := user.(*merchantsUser)
	response := ToSessionResponse(tokens, u)
	return h.responder.SuccessCompat(c, response, fiber.Map{
		"token":       tokens.AccessToken,
		"accessToken": tokens.AccessToken,
		"user":        LegacyUser(u),
	})
}
