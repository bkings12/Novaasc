package api

import (
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/novaacs/go-acs/internal/provisioning"
)

func (h *Handler) ListRules(c *fiber.Ctx) error {
	t := tenantFromCtx(c)
	rules, err := h.provRepo.List(c.Context(), t.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list rules")
	}
	return c.JSON(fiber.Map{"data": rules, "count": len(rules)})
}

func (h *Handler) GetRule(c *fiber.Ctx) error {
	t := tenantFromCtx(c)
	id := c.Params("id")
	rule, err := h.provRepo.GetByID(c.Context(), t.ID, id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "rule not found")
	}
	return c.JSON(rule)
}

func (h *Handler) CreateRule(c *fiber.Ctx) error {
	t := tenantFromCtx(c)

	var body struct {
		Name              string          `json:"name"`
		Description       string          `json:"description"`
		Priority          int             `json:"priority"`
		Trigger           string          `json:"trigger"`
		MatchManufacturer string          `json:"match_manufacturer"`
		MatchOUI          string          `json:"match_oui"`
		MatchProductClass string          `json:"match_product_class"`
		MatchModelName    string          `json:"match_model_name"`
		MatchSWVersion    string          `json:"match_sw_version"`
		Actions           json.RawMessage `json:"actions"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	if body.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name required")
	}
	if body.Trigger == "" {
		body.Trigger = "ANY"
	}
	if body.Actions == nil {
		body.Actions = json.RawMessage("[]")
	}

	rule := &provisioning.Rule{
		TenantID:          t.ID,
		Name:              body.Name,
		Description:       body.Description,
		Priority:          body.Priority,
		Active:            true,
		Trigger:           body.Trigger,
		MatchManufacturer: body.MatchManufacturer,
		MatchOUI:          body.MatchOUI,
		MatchProductClass: body.MatchProductClass,
		MatchModelName:    body.MatchModelName,
		MatchSWVersion:    body.MatchSWVersion,
		ActionsRaw:        body.Actions,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := h.provRepo.Create(c.Context(), rule); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create rule")
	}
	return c.Status(fiber.StatusCreated).JSON(rule)
}

func (h *Handler) UpdateRule(c *fiber.Ctx) error {
	t := tenantFromCtx(c)
	id := c.Params("id")

	rule, err := h.provRepo.GetByID(c.Context(), t.ID, id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "rule not found")
	}

	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	if v, ok := body["name"].(string); ok {
		rule.Name = v
	}
	if v, ok := body["description"].(string); ok {
		rule.Description = v
	}
	if v, ok := body["trigger"].(string); ok {
		rule.Trigger = v
	}
	if v, ok := body["active"].(bool); ok {
		rule.Active = v
	}
	if v, ok := body["match_manufacturer"].(string); ok {
		rule.MatchManufacturer = v
	}
	if v, ok := body["match_oui"].(string); ok {
		rule.MatchOUI = v
	}
	if v, ok := body["match_product_class"].(string); ok {
		rule.MatchProductClass = v
	}
	if v, ok := body["match_model_name"].(string); ok {
		rule.MatchModelName = v
	}
	if v, ok := body["match_sw_version"].(string); ok {
		rule.MatchSWVersion = v
	}
	if v, ok := body["priority"].(float64); ok {
		rule.Priority = int(v)
	}
	if v, ok := body["actions"]; ok {
		raw, _ := json.Marshal(v)
		rule.ActionsRaw = raw
	}
	rule.UpdatedAt = time.Now()

	if err := h.provRepo.Update(c.Context(), rule); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to update rule")
	}
	return c.JSON(rule)
}

func (h *Handler) DeleteRule(c *fiber.Ctx) error {
	t := tenantFromCtx(c)
	id := c.Params("id")
	if err := h.provRepo.Delete(c.Context(), t.ID, id); err != nil {
		if err == provisioning.ErrNotFound {
			return fiber.NewError(fiber.StatusNotFound, "rule not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete rule")
	}
	return c.Status(fiber.StatusNoContent).Send(nil)
}
