package middleware

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"

	apperrors "github.com/kuipid01/payconnect/internal/errors"
	"github.com/kuipid01/payconnect/internal/merchants"
	"github.com/kuipid01/payconnect/internal/platform/identity"
)

func TestRequireRolesAllowsOnlyApplicationManagers(t *testing.T) {
	for _, test := range []struct {
		role       string
		wantStatus int
	}{
		{role: "Owner", wantStatus: http.StatusNoContent},
		{role: "Admin", wantStatus: http.StatusNoContent},
		{role: "Developer", wantStatus: http.StatusForbidden},
		{role: "Viewer", wantStatus: http.StatusForbidden},
	} {
		t.Run(test.role, func(t *testing.T) {
			app := fiber.New(fiber.Config{
				ErrorHandler: func(c *fiber.Ctx, err error) error {
					var appErr *apperrors.AppError
					if errors.As(err, &appErr) {
						return c.SendStatus(appErr.HTTPStatus)
					}
					return c.SendStatus(http.StatusInternalServerError)
				},
			})
			app.Use(func(c *fiber.Ctx) error {
				identity.WithPrincipal(c, &identity.Principal{
					Mode: identity.ModeSession,
					User: merchants.User{Role: test.role},
				})
				return c.Next()
			})
			app.Delete("/applications/:id",
				RequireRoles("Owner", "Admin"),
				func(c *fiber.Ctx) error { return c.SendStatus(http.StatusNoContent) },
			)

			req, err := http.NewRequest(http.MethodDelete, "/applications/app_test", nil)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != test.wantStatus {
				t.Fatalf("DELETE status = %d, want %d", resp.StatusCode, test.wantStatus)
			}
		})
	}
}
