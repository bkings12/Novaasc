package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/novaacs/go-acs/internal/tenant"
	"go.uber.org/zap"
)

// FromSlug resolves tenant from URL param :tenant
// Route: POST /cwmp/:tenant
func FromSlug(repo tenant.Repository, log *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		slug := c.Params("tenant")
		if slug == "" {
			return c.Status(400).SendString("missing tenant")
		}

		t, err := repo.GetBySlug(c.Context(), slug)
		if err != nil {
			log.Warn("tenant not found by slug",
				zap.String("slug", slug),
				zap.String("ip", c.IP()),
			)
			return c.Status(404).SendString("tenant not found")
		}

		c.Locals("tenant", t)
		c.SetUserContext(tenant.WithTenant(c.UserContext(), t))
		return c.Next()
	}
}

// FromAPIKey resolves tenant from X-ACS-Key header
// Route: POST /cwmp or POST / (fallback)
func FromAPIKey(repo tenant.Repository, log *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := c.Get("X-ACS-Key")
		if key == "" {
			key = "dev-api-key-change-in-prod"
		}

		t, err := repo.GetByAPIKey(c.Context(), key)
		if err != nil {
			log.Warn("tenant not found by api key",
				zap.String("ip", c.IP()),
			)
			return c.Status(403).SendString("invalid ACS key")
		}

		c.Locals("tenant", t)
		c.SetUserContext(tenant.WithTenant(c.UserContext(), t))
		return c.Next()
	}
}

// TenantFromLocals returns the tenant stored by middleware (nil if missing).
func TenantFromLocals(c *fiber.Ctx) *tenant.Tenant {
	t, _ := c.Locals("tenant").(*tenant.Tenant)
	return t
}
