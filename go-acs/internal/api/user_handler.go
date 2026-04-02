package api

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/novaacs/go-acs/internal/api/middleware"
	"github.com/novaacs/go-acs/internal/auth"
	"go.uber.org/zap"
)

func (h *Handler) ListUsers(c *fiber.Ctx) error {
	t := tenantFromCtx(c)
	users, err := h.authSvc.ListUsers(c.Context(), t.ID)
	if err != nil {
		h.log.Error("list users", zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list users")
	}
	return c.JSON(fiber.Map{"data": users, "count": len(users)})
}

func (h *Handler) CreateUser(c *fiber.Ctx) error {
	t := tenantFromCtx(c)
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	role := auth.Role(body.Role)
	if body.Role == "viewer" {
		role = auth.RoleReadOnly
	}
	u, err := h.authSvc.CreateUser(c.Context(), t.ID, body.Email, body.Password, role)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidRole) {
			return fiber.NewError(fiber.StatusBadRequest, "role must be admin, user, or readonly (viewer)")
		}
		if errors.Is(err, auth.ErrDuplicate) {
			return fiber.NewError(fiber.StatusConflict, "email already exists")
		}
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(u)
}

func (h *Handler) DeactivateUser(c *fiber.Ctx) error {
	t := tenantFromCtx(c)
	claims := middleware.ClaimsFromCtx(c)
	id := c.Params("id")
	if id == "" {
		return fiber.NewError(fiber.StatusBadRequest, "id required")
	}
	err := h.authSvc.DeactivateUser(c.Context(), t.ID, claims.UserID, id)
	if err != nil {
		if errors.Is(err, auth.ErrCannotDeactivateSelf) {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to deactivate user")
	}
	return c.SendStatus(fiber.StatusNoContent)
}
