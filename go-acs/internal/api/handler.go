package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/novaacs/go-acs/internal/auth"
	"github.com/novaacs/go-acs/internal/backup"
	"github.com/novaacs/go-acs/internal/connreq"
	"github.com/novaacs/go-acs/internal/credprofile"
	"github.com/novaacs/go-acs/internal/device"
	"github.com/novaacs/go-acs/internal/provisioning"
	"github.com/novaacs/go-acs/internal/task"
	"github.com/novaacs/go-acs/internal/tenant"
	"go.uber.org/zap"
)

type Handler struct {
	log             *zap.Logger
	deviceRepo      device.Repository
	taskRepo        task.Repository
	provRepo        provisioning.Repository
	tenantRepo      tenant.Repository
	authSvc         *auth.Service
	connreqClient   *connreq.Client
	backupRepo      backup.Repository
	backupSvc       *backup.Service
	credProfileRepo credprofile.Repository
}

func tenantFromCtx(c *fiber.Ctx) *tenant.Tenant {
	t, _ := c.Locals("tenant").(*tenant.Tenant)
	return t
}

func (h *Handler) Health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *Handler) Ready(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ready"})
}

func (h *Handler) Stats(c *fiber.Ctx) error {
	t := tenantFromCtx(c)

	onlineTrue := true
	onlineFalse := false

	_, totalOnline, err := h.deviceRepo.List(c.Context(), t.ID,
		device.DeviceFilter{Online: &onlineTrue, Limit: 1})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to count devices")
	}
	_, totalOffline, err := h.deviceRepo.List(c.Context(), t.ID,
		device.DeviceFilter{Online: &onlineFalse, Limit: 1})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to count devices")
	}

	pendingFilter := task.Filter{Status: task.StatusPending, Limit: 1}
	failedFilter := task.Filter{Status: task.StatusFailed, Limit: 1}
	_, totalPending, err := h.taskRepo.ListForTenant(c.Context(), t.ID, pendingFilter)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to count tasks")
	}
	_, totalFailed, err := h.taskRepo.ListForTenant(c.Context(), t.ID, failedFilter)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to count tasks")
	}

	return c.JSON(fiber.Map{
		"devices": fiber.Map{
			"online":  totalOnline,
			"offline": totalOffline,
			"total":   totalOnline + totalOffline,
		},
		"tasks": fiber.Map{
			"pending": totalPending,
			"failed":  totalFailed,
		},
	})
}
