package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/novaacs/go-acs/internal/task"
)

func (h *Handler) ListTasks(c *fiber.Ctx) error {
	t := tenantFromCtx(c)

	filter := task.Filter{
		DeviceSerial: c.Query("serial"),
		Limit:        queryInt(c, "limit", 50),
		Offset:       queryInt(c, "offset", 0),
	}
	if s := c.Query("status"); s != "" {
		filter.Status = task.Status(s)
	}

	tasks, total, err := h.taskRepo.ListForTenant(c.Context(), t.ID, filter)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list tasks")
	}
	return c.JSON(fiber.Map{
		"data":   tasks,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

func (h *Handler) GetTask(c *fiber.Ctx) error {
	t := tenantFromCtx(c)
	id := c.Params("id")

	tk, err := h.taskRepo.GetByID(c.Context(), t.ID, id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "task not found")
	}
	return c.JSON(tk)
}

func (h *Handler) CancelTask(c *fiber.Ctx) error {
	t := tenantFromCtx(c)
	id := c.Params("id")

	if err := h.taskRepo.Cancel(c.Context(), t.ID, id); err != nil {
		if err == task.ErrNotFound {
			return fiber.NewError(fiber.StatusNotFound, "task not found")
		}
		return fiber.NewError(fiber.StatusBadRequest, "cannot cancel task (already dispatched or complete)")
	}
	return c.Status(fiber.StatusNoContent).Send(nil)
}
