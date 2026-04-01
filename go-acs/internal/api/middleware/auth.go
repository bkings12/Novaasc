package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/novaacs/go-acs/internal/auth"
	"github.com/novaacs/go-acs/internal/tenant"
)

func Auth(svc *auth.Service, tenantRepo tenant.Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Path() == "/api/v1/ws" {
			return c.Next()
		}
		header := c.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			return fiber.NewError(fiber.StatusUnauthorized, "authorization required")
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")

		claims, err := svc.ValidateAccessToken(tokenStr)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired token")
		}

		t, err := tenantRepo.GetByID(c.Context(), claims.TenantID)
		if err != nil || !t.Active {
			return fiber.NewError(fiber.StatusForbidden, "tenant inactive or not found")
		}

		c.Locals("claims", claims)
		c.Locals("tenant", t)
		c.SetUserContext(tenant.WithTenant(c.UserContext(), t))
		return c.Next()
	}
}

func RequireRole(minimum auth.Role) fiber.Handler {
	rank := map[auth.Role]int{
		auth.RoleReadOnly:   1,
		auth.RoleUser:       2,
		auth.RoleAdmin:      3,
		auth.RoleSuperAdmin: 4,
	}
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals("claims").(*auth.Claims)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
		if rank[claims.Role] < rank[minimum] {
			return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
		}
		return c.Next()
	}
}

func ClaimsFromCtx(c *fiber.Ctx) *auth.Claims {
	claims, _ := c.Locals("claims").(*auth.Claims)
	return claims
}
