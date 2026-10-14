package logs

import (
	"bytes"
	"encoding/csv"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	apperrors "github.com/kuipid01/payconnect/internal/errors"
	apihttp "github.com/kuipid01/payconnect/internal/http"
	"github.com/kuipid01/payconnect/internal/platform/identity"
)

// Handler maps HTTP requests onto the log projections.
type Handler struct {
	repo      *Repository
	responder *apihttp.Responder
}

// NewHandler builds the logs handler.
func NewHandler(repo *Repository, responder *apihttp.Responder) *Handler {
	return &Handler{repo: repo, responder: responder}
}

// Response is the public request-log representation.
type Response struct {
	ID            string `json:"id"`
	ApplicationID string `json:"application_id,omitempty"`
	Environment   string `json:"environment,omitempty"`
	Method        string `json:"method"`
	Path          string `json:"path"`
	StatusCode    int    `json:"status_code"`
	DurationMs    int    `json:"duration_ms"`
	SourceIP      string `json:"source_ip,omitempty"`
	RequestID     string `json:"request_id,omitempty"`
	CreatedAt     string `json:"created_at"`
}

// List handles GET /v1/logs.
func (h *Handler) List(c *fiber.Ctx) error {
	principal := identity.From(c)
	page := apihttp.ParsePagination(c)
	environment := c.Query("environment")
	if principal.IsAPIKey() {
		environment = principal.Environment
	}
	filter := Filter{
		MerchantID:    principal.MerchantID,
		ApplicationID: c.Query("application"),
		Environment:   environment,
		Method:        c.Query("method"),
		StatusClass:   c.Query("status_class"),
		RequestID:     c.Query("request_id"),
		Offset:        page.Offset(),
		Limit:         page.Limit,
	}
	rows, err := h.repo.APILogs(filter)
	if err != nil {
		return h.responder.Error(c, err)
	}
	total, err := h.repo.CountAPILogs(filter)
	if err != nil {
		return h.responder.Error(c, err)
	}
	out := make([]Response, 0, len(rows))
	for i := range rows {
		out = append(out, toResponse(&rows[i]))
	}
	return h.responder.Paginated(c, out, apihttp.Page{
		Page:    page.Page,
		Limit:   page.Limit,
		Total:   &total,
		HasMore: int64(page.Offset()+len(rows)) < total,
	})
}

// Get handles GET /v1/logs/{id}.
func (h *Handler) Get(c *fiber.Ctx) error {
	principal := identity.From(c)
	row, err := h.repo.APILogByID(principal.MerchantID, c.Params("id"))
	if err != nil {
		return h.responder.Error(c, apperrors.NotFound("log_not_found", "Request log not found."))
	}
	return h.responder.Success(c, toResponse(row))
}

// Audit handles GET /v1/audit-logs.
func (h *Handler) Audit(c *fiber.Ctx) error {
	principal := identity.From(c)
	page := apihttp.ParsePagination(c)
	severity := c.Query("severity")
	rows, err := h.repo.AuditLogs(principal.MerchantID, severity, page.Offset(), page.Limit)
	if err != nil {
		return h.responder.Error(c, err)
	}
	total, err := h.repo.CountAuditLogs(principal.MerchantID, severity)
	if err != nil {
		return h.responder.Error(c, err)
	}
	return h.responder.Paginated(c, rows, apihttp.Page{
		Page: page.Page, Limit: page.Limit, Total: &total,
		HasMore: int64(page.Offset()+len(rows)) < total,
	})
}

// ExportAudit handles GET /v1/audit-logs/export and returns the filtered trail
// as a CSV attachment.
func (h *Handler) ExportAudit(c *fiber.Ctx) error {
	principal := identity.From(c)
	rows, err := h.repo.AllAuditLogs(principal.MerchantID, c.Query("severity"))
	if err != nil {
		return h.responder.Error(c, err)
	}
	var body bytes.Buffer
	writer := csv.NewWriter(&body)
	if err := writer.Write([]string{"id", "at", "actor_name", "actor_role", "action", "detail", "severity"}); err != nil {
		return h.responder.Error(c, err)
	}
	for _, row := range rows {
		if err := writer.Write([]string{
			row.ID, row.CreatedAt.UTC().Format(time.RFC3339), row.ActorName, row.ActorRole,
			row.Action, row.Detail, row.Severity,
		}); err != nil {
			return h.responder.Error(c, err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return h.responder.Error(c, err)
	}
	c.Set(fiber.HeaderContentType, "text/csv; charset=utf-8")
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="audit-logs.csv"`)
	c.Set("Content-Length", strconv.Itoa(body.Len()))
	return c.Send(body.Bytes())
}

func toResponse(row *APILog) Response {
	return Response{
		ID:            row.ID,
		ApplicationID: row.ApplicationID,
		Environment:   row.Environment,
		Method:        row.Method,
		Path:          row.Path,
		StatusCode:    row.StatusCode,
		DurationMs:    row.DurationMs,
		SourceIP:      row.SourceIP,
		RequestID:     row.RequestID,
		CreatedAt:     row.CreatedAt.UTC().Format(time.RFC3339),
	}
}
