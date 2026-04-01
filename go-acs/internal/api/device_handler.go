package api

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/novaacs/go-acs/internal/api/middleware"
	"github.com/novaacs/go-acs/internal/device"
	"github.com/novaacs/go-acs/internal/task"
	"go.uber.org/zap"
)

func queryInt(c *fiber.Ctx, key string, defaultVal int) int64 {
	s := c.Query(key, strconv.Itoa(defaultVal))
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return int64(defaultVal)
	}
	return v
}

func (h *Handler) ListDevices(c *fiber.Ctx) error {
	t := tenantFromCtx(c)

	filter := device.DeviceFilter{
		Manufacturer: c.Query("manufacturer"),
		Search:       c.Query("search"),
		Limit:        queryInt(c, "limit", 50),
		Offset:       queryInt(c, "offset", 0),
	}
	if c.Query("online") == "true" {
		v := true
		filter.Online = &v
	}
	if c.Query("online") == "false" {
		v := false
		filter.Online = &v
	}

	devices, total, err := h.deviceRepo.List(c.Context(), t.ID, filter)
	if err != nil {
		h.log.Error("list devices", zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list devices")
	}

	return c.JSON(fiber.Map{
		"data":   devices,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

func (h *Handler) GetDevice(c *fiber.Ctx) error {
	t := tenantFromCtx(c)
	serial := c.Params("serial")

	dev, err := h.deviceRepo.GetBySerial(c.Context(), t.ID, serial)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "device not found")
	}
	return c.JSON(dev)
}

func (h *Handler) GetDeviceParameters(c *fiber.Ctx) error {
	t := tenantFromCtx(c)
	serial := c.Params("serial")
	prefix := c.Query("prefix", "")

	dev, err := h.deviceRepo.GetBySerial(c.Context(), t.ID, serial)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "device not found")
	}

	if prefix == "" {
		return c.JSON(dev.Parameters)
	}
	filtered := make(map[string]string)
	for k, v := range dev.Parameters {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			filtered[k] = v
		}
	}
	return c.JSON(filtered)
}

func (h *Handler) GetDeviceTasks(c *fiber.Ctx) error {
	t := tenantFromCtx(c)
	serial := c.Params("serial")
	limit := queryInt(c, "limit", 20)

	tasks, err := h.taskRepo.ListForDevice(c.Context(), t.ID, serial, limit)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list tasks")
	}
	return c.JSON(fiber.Map{"data": tasks, "count": len(tasks)})
}

func (h *Handler) DeleteDevice(c *fiber.Ctx) error {
	t := tenantFromCtx(c)
	serial := c.Params("serial")

	if err := h.deviceRepo.Delete(c.Context(), t.ID, serial); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to delete device")
	}
	return c.Status(fiber.StatusNoContent).Send(nil)
}

func (h *Handler) enqueueTask(c *fiber.Ctx, serial string, taskType task.Type, customize func(*task.Task)) error {
	ten := tenantFromCtx(c)

	_, err := h.deviceRepo.GetBySerial(c.Context(), ten.ID, serial)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "device not found")
	}

	tk := &task.Task{
		TenantID:     ten.ID,
		DeviceSerial: serial,
		Type:         taskType,
		Status:       task.StatusPending,
		Priority:     10,
		CreatedAt:    time.Now(),
		Timeout:      int64(5 * time.Minute),
		CreatedBy:    "api",
	}
	if customize != nil {
		customize(tk)
	}

	if err := h.taskRepo.Enqueue(c.Context(), tk); err != nil {
		h.log.Error("enqueue task", zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "failed to enqueue task")
	}

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"task_id": tk.ID,
		"type":    tk.Type,
		"status":  tk.Status,
		"message": "task queued — will execute on next device contact",
	})
}

func (h *Handler) Reboot(c *fiber.Ctx) error {
	return h.enqueueTask(c, c.Params("serial"), task.TypeReboot, nil)
}

func (h *Handler) FactoryReset(c *fiber.Ctx) error {
	return h.enqueueTask(c, c.Params("serial"), task.TypeFactoryReset, nil)
}

func (h *Handler) GetParameters(c *fiber.Ctx) error {
	var body struct {
		Names []string `json:"names"`
	}
	if err := c.BodyParser(&body); err != nil || len(body.Names) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "names array required")
	}
	return h.enqueueTask(c, c.Params("serial"), task.TypeGetParameterValues,
		func(t *task.Task) { t.ParameterNames = body.Names })
}

func (h *Handler) SetParameters(c *fiber.Ctx) error {
	var body struct {
		Values map[string]string `json:"values"`
	}
	if err := c.BodyParser(&body); err != nil || len(body.Values) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "values map required")
	}
	return h.enqueueTask(c, c.Params("serial"), task.TypeSetParameterValues,
		func(t *task.Task) { t.ParameterValues = body.Values })
}

func (h *Handler) GetParameterNames(c *fiber.Ctx) error {
	var body struct {
		Path      string `json:"path"`
		NextLevel bool   `json:"next_level"`
	}
	_ = c.BodyParser(&body)
	if body.Path == "" {
		body.Path = "Device."
	}
	return h.enqueueTask(c, c.Params("serial"), task.TypeGetParameterNames,
		func(t *task.Task) { t.ParameterNames = []string{body.Path} })
}

func (h *Handler) Download(c *fiber.Ctx) error {
	var body task.DownloadArgs
	if err := c.BodyParser(&body); err != nil || body.URL == "" {
		return fiber.NewError(fiber.StatusBadRequest, "url required")
	}
	if body.FileType == "" {
		body.FileType = "1 Firmware Upgrade Image"
	}
	return h.enqueueTask(c, c.Params("serial"), task.TypeDownload,
		func(t *task.Task) { t.Download = &body })
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func (h *Handler) Wake(c *fiber.Ctx) error {
	t := tenantFromCtx(c)
	serial := c.Params("serial")

	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	_ = c.BodyParser(&body)

	result, err := h.connreqClient.Wake(c.Context(), t.ID, serial, body.Username, body.Password)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(result)
	}
	return c.JSON(result)
}

// PreRegister adds a device record before it contacts the ACS (e.g. xPON from OLT).
func (h *Handler) PreRegister(c *fiber.Ctx) error {
	t := tenantFromCtx(c)
	claims := middleware.ClaimsFromCtx(c)
	createdBy := "api"
	if claims != nil && claims.Email != "" {
		createdBy = claims.Email
	}

	var body struct {
		SerialNumber         string   `json:"serial_number"`
		Manufacturer         string   `json:"manufacturer"`
		OUI                  string   `json:"oui"`
		ModelName            string   `json:"model_name"`
		ConnectionRequestURL string   `json:"connection_request_url"`
		CRUsername           string   `json:"cr_username"`
		CRPassword           string   `json:"cr_password"`
		Tags                 []string `json:"tags"`
		Notes                string   `json:"notes"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	if body.SerialNumber == "" {
		return fiber.NewError(fiber.StatusBadRequest, "serial_number required")
	}

	dev := &device.Device{
		TenantID:                  t.ID,
		SerialNumber:              body.SerialNumber,
		Manufacturer:              body.Manufacturer,
		OUI:                       body.OUI,
		ModelName:                 body.ModelName,
		ConnectionRequestURL:      body.ConnectionRequestURL,
		ConnectionRequestUsername: body.CRUsername,
		ConnectionRequestPassword: body.CRPassword,
		Tags:                      body.Tags,
		Online:                    false,
		Parameters:                make(map[string]string),
	}
	if dev.Tags == nil {
		dev.Tags = []string{}
	}

	if err := h.deviceRepo.Upsert(c.Context(), dev); err != nil {
		h.log.Error("pre-register device failed", zap.Error(err), zap.String("serial", body.SerialNumber))
		return fiber.NewError(fiber.StatusInternalServerError, "failed to pre-register device")
	}

	h.log.Info("device pre-registered",
		zap.String("serial", body.SerialNumber),
		zap.String("tenant", t.ID),
		zap.String("by", createdBy),
	)
	if body.Notes != "" {
		h.log.Debug("preregister notes", zap.String("serial", body.SerialNumber), zap.String("notes", body.Notes))
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":       "device pre-registered",
		"serial_number": body.SerialNumber,
		"note":          "configure ACS URL on device as http://<acs-host>:7547/cwmp/" + t.Slug + " — no username/password needed on device side",
	})
}
