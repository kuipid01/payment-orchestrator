package merchants

import (
	"github.com/gofiber/fiber/v2"

	apperrors "github.com/kuipid01/payconnect/internal/errors"
	apihttp "github.com/kuipid01/payconnect/internal/http"
	"github.com/kuipid01/payconnect/internal/platform/identity"
)

// Handler maps HTTP requests onto the merchant service.
type Handler struct {
	svc       *Service
	responder *apihttp.Responder
}

// NewHandler builds the merchant handler.
func NewHandler(svc *Service, responder *apihttp.Responder) *Handler {
	return &Handler{svc: svc, responder: responder}
}

// Get handles GET /v1/merchant.
func (h *Handler) Get(c *fiber.Ctx) error {
	principal := identity.From(c)
	m, err := h.svc.Get(principal.MerchantID)
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, ToResponse(m))
}

// Update handles PATCH /v1/merchant.
func (h *Handler) Update(c *fiber.Ctx) error {
	principal := identity.From(c)
	if err := requireTeamAdmin(principal); err != nil {
		return h.responder.Error(c, err)
	}
	var req struct {
		BusinessName    *string `json:"business_name"`
		Country         *string `json:"country"`
		DefaultCurrency *string `json:"default_currency"`
	}
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	actor, _ := principal.User.(*User)
	m, err := h.svc.Update(principal.MerchantID, UpdateInput{
		BusinessName:    req.BusinessName,
		Country:         req.Country,
		DefaultCurrency: req.DefaultCurrency,
		ActorName:       actor.Name,
		ActorRole:       actor.Role,
	})
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, ToResponse(m))
}

// Team handles GET /v1/team.
func (h *Handler) Team(c *fiber.Ctx) error {
	principal := identity.From(c)
	members, err := h.svc.Team(principal.MerchantID)
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, members)
}

// AddTeamMember handles POST /v1/team.
func (h *Handler) AddTeamMember(c *fiber.Ctx) error {
	principal := identity.From(c)
	if err := requireTeamAdmin(principal); err != nil {
		return h.responder.Error(c, err)
	}
	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := apihttp.Body(c, &req); err != nil {
		return h.responder.Error(c, err)
	}
	actor, _ := principal.User.(*User)
	member, err := h.svc.AddTeamMember(
		principal.MerchantID, req.Email, req.Role, actor.Name, actor.Role,
	)
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Created(c, member)
}

// RemoveTeamMember handles DELETE /v1/team/:id.
func (h *Handler) RemoveTeamMember(c *fiber.Ctx) error {
	principal := identity.From(c)
	if err := requireTeamAdmin(principal); err != nil {
		return h.responder.Error(c, err)
	}
	actor, _ := principal.User.(*User)
	if err := h.svc.RemoveTeamMember(
		principal.MerchantID, c.Params("id"), principal.UserID, actor.Name, actor.Role,
	); err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Success(c, fiber.Map{"removed": true})
}

func requireTeamAdmin(principal *identity.Principal) error {
	if principal == nil || !principal.IsSession() {
		return apperrors.AuthenticationRequired()
	}
	user, ok := principal.User.(*User)
	if !ok || (user.Role != "Owner" && user.Role != "Admin") {
		return apperrors.PermissionDenied("Only an organization owner or admin can manage these settings.")
	}
	return nil
}
