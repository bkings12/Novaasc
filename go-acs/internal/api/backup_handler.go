package api

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/novaacs/go-acs/internal/api/middleware"
	"github.com/novaacs/go-acs/internal/backup"
	"github.com/novaacs/go-acs/internal/task"
)

func (h *Handler) ListBackups(c *fiber.Ctx) error {
	t := tenantFromCtx(c)
	serial := c.Params("serial")
	limit := c.QueryInt("limit", 20)
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	if _, err := h.deviceRepo.GetBySerial(c.Context(), t.ID, serial); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "device not found")
	}
	backups, err := h.backupRepo.ListForDevice(c.Context(), t.ID, serial, int64(limit))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list backups")
	}
	return c.JSON(fiber.Map{
		"data":  backups,
		"count": len(backups),
	})
}

func (h *Handler) CreateBackup(c *fiber.Ctx) error {
	t := tenantFromCtx(c)
	serial := c.Params("serial")
	claims := middleware.ClaimsFromCtx(c)
	createdBy := "api"
	if claims != nil {
		createdBy = claims.UserID
	}

	var body struct {
		Label   string `json:"label"`
		Refresh bool   `json:"refresh"`
	}
	_ = c.BodyParser(&body)
	if body.Label == "" {
		body.Label = "manual"
	}

	if body.Refresh {
		tk := &task.Task{
			TenantID:       t.ID,
			DeviceSerial:   serial,
			Type:           task.TypeGetParameterValues,
			Status:         task.StatusPending,
			Priority:       50,
			ParameterNames: []string{"Device.", "InternetGatewayDevice."},
			CreatedAt:      time.Now(),
			Timeout:        int64(5 * time.Minute),
			CreatedBy:      "backup:refresh",
		}
		if err := h.taskRepo.Enqueue(c.Context(), tk); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to enqueue refresh task")
		}
		return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
			"message": "GetParameterValues queued — call backup again after task completes, or use wake to speed it up",
			"task_id": tk.ID,
		})
	}

	b, err := h.backupSvc.TakeBackup(c.Context(), t.ID, serial, body.Label, createdBy)
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(b)
}

func (h *Handler) GetBackup(c *fiber.Ctx) error {
	t := tenantFromCtx(c)
	serial := c.Params("serial")
	id := c.Params("id")

	b, err := h.backupRepo.GetByID(c.Context(), t.ID, id)
	if err != nil {
		if err == backup.ErrNotFound {
			return fiber.NewError(fiber.StatusNotFound, "backup not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get backup")
	}
	if b.DeviceSerial != serial {
		return fiber.NewError(fiber.StatusNotFound, "backup not found")
	}
	return c.JSON(b)
}

func (h *Handler) DeleteBackup(c *fiber.Ctx) error {
	t := tenantFromCtx(c)
	id := c.Params("id")

	if err := h.backupRepo.Delete(c.Context(), t.ID, id); err != nil {
		if err == backup.ErrNotFound {
			return fiber.NewError(fiber.StatusNotFound, "backup not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete backup")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) RestoreBackup(c *fiber.Ctx) error {
	t := tenantFromCtx(c)
	id := c.Params("id")
	claims := middleware.ClaimsFromCtx(c)
	createdBy := "api"
	if claims != nil {
		createdBy = claims.UserID
	}

	job, err := h.backupSvc.StartRestore(c.Context(), t.ID, id, createdBy)
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, err.Error())
	}
	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"job_id":       job.ID,
		"total_chunks": job.TotalChunks,
		"task_count":   len(job.TaskIDs),
		"message": fmt.Sprintf(
			"restore started — %d SetParameterValues tasks queued across %d chunks",
			len(job.TaskIDs), job.TotalChunks),
	})
}

func (h *Handler) GetRestoreJob(c *fiber.Ctx) error {
	t := tenantFromCtx(c)
	id := c.Params("id")

	job, err := h.backupRepo.GetRestoreJob(c.Context(), t.ID, id)
	if err != nil {
		if err == backup.ErrNotFound {
			return fiber.NewError(fiber.StatusNotFound, "restore job not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get restore job")
	}
	return c.JSON(job)
}
