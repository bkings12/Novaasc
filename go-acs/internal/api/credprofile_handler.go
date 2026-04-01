package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/novaacs/go-acs/internal/credprofile"
)

func (h *Handler) ListCredProfiles(c *fiber.Ctx) error {
	t := tenantFromCtx(c)
	profiles, err := h.credProfileRepo.List(c.Context(), t.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list profiles")
	}
	return c.JSON(fiber.Map{"data": profiles, "count": len(profiles)})
}

func (h *Handler) CreateCredProfile(c *fiber.Ctx) error {
	t := tenantFromCtx(c)
	var p credprofile.Profile
	if err := c.BodyParser(&p); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	p.TenantID = t.ID
	if err := h.credProfileRepo.Create(c.Context(), &p); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create profile")
	}
	return c.Status(fiber.StatusCreated).JSON(p)
}

func (h *Handler) UpdateCredProfile(c *fiber.Ctx) error {
	t := tenantFromCtx(c)
	id := c.Params("id")
	var p credprofile.Profile
	if err := c.BodyParser(&p); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	p.ID = id
	p.TenantID = t.ID
	if err := h.credProfileRepo.Update(c.Context(), &p); err != nil {
		if err == credprofile.ErrNotFound {
			return fiber.NewError(fiber.StatusNotFound, "profile not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update profile")
	}
	return c.JSON(p)
}

func (h *Handler) DeleteCredProfile(c *fiber.Ctx) error {
	t := tenantFromCtx(c)
	id := c.Params("id")
	if err := h.credProfileRepo.Delete(c.Context(), t.ID, id); err != nil {
		if err == credprofile.ErrNotFound {
			return fiber.NewError(fiber.StatusNotFound, "profile not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete profile")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) UpdateTenantDefaults(c *fiber.Ctx) error {
	t := tenantFromCtx(c)
	var body struct {
		DefaultCRUsername string `json:"default_cr_username"`
		DefaultCRPassword string `json:"default_cr_password"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	t.DefaultCRUsername = body.DefaultCRUsername
	t.DefaultCRPassword = body.DefaultCRPassword
	if err := h.tenantRepo.Update(c.Context(), t); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update tenant")
	}
	return c.JSON(fiber.Map{
		"message": "tenant default credentials updated",
	})
}
