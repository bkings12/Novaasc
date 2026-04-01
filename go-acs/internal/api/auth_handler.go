package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/novaacs/go-acs/internal/api/middleware"
)

func (h *Handler) Login(c *fiber.Ctx) error {
	var body struct {
		Tenant   string `json:"tenant"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	if body.Email == "" || body.Password == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email and password required")
	}
	if body.Tenant == "" {
		body.Tenant = "default"
	}

	pair, err := h.authSvc.Login(c.Context(), body.Tenant, body.Email, body.Password)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
	}

	return c.JSON(pair)
}

func (h *Handler) RefreshToken(c *fiber.Ctx) error {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.BodyParser(&body); err != nil || body.RefreshToken == "" {
		return fiber.NewError(fiber.StatusBadRequest, "refresh_token required")
	}

	pair, err := h.authSvc.Refresh(c.Context(), body.RefreshToken)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired refresh token")
	}

	return c.JSON(pair)
}

func (h *Handler) GetMe(c *fiber.Ctx) error {
	claims := middleware.ClaimsFromCtx(c)
	t := tenantFromCtx(c)
	return c.JSON(fiber.Map{
		"user_id":   claims.UserID,
		"email":     claims.Email,
		"role":      claims.Role,
		"tenant_id": t.ID,
		"tenant":    t.Slug,
	})
}
