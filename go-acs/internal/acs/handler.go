package acs

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/novaacs/go-acs/internal/acs/middleware"
	"github.com/novaacs/go-acs/internal/cwmp"
	"github.com/novaacs/go-acs/internal/device"
	"github.com/novaacs/go-acs/internal/events"
	"github.com/novaacs/go-acs/internal/provisioning"
	"github.com/novaacs/go-acs/internal/task"
	"github.com/novaacs/go-acs/internal/tenant"
	"go.uber.org/zap"
)

// Handler handles CWMP HTTP on port 7547 (TR-069).
type Handler struct {
	Sessions    *cwmp.SessionManager
	Devices     device.Repository
	TaskRepo    task.Repository
	Provisioner *provisioning.Engine
	Hub         *events.Hub
	Log         *zap.Logger
}

// ServeCWMP is the main CWMP endpoint (POST /, /acs, /cwmp, /cwmp/:tenant).
func (h *Handler) ServeCWMP(c *fiber.Ctx) error {
	t := middleware.TenantFromLocals(c)
	if t == nil {
		return c.Status(500).SendString("tenant context missing")
	}

	log := h.Log.With(
		zap.String("tenant_id", t.ID),
		zap.String("tenant_slug", t.Slug),
		zap.String("ip", c.IP()),
	)

	body := c.Body()

	if len(bytes.TrimSpace(body)) == 0 {
		return h.dispatchNextTask(c, t, log)
	}

	rawBody, _, err := cwmp.UnmarshalEnvelope(body)
	if err != nil {
		log.Warn("soap parse error", zap.Error(err))
		return c.Status(http.StatusBadRequest).SendString("Invalid SOAP")
	}

	msgType := cwmp.DetectMessageType(rawBody)

	switch msgType {
	case "Inform":
		return h.handleInform(c, log, rawBody, t)
	case "TransferComplete":
		sess := h.getSessionFromCookie(c)
		if sess == nil {
			return c.SendStatus(http.StatusNoContent)
		}
		return h.handleTransferComplete(c, sess, t, log)
	case "GetParameterValuesResponse":
		sess := h.getSessionFromCookie(c)
		if sess == nil {
			return c.SendStatus(http.StatusNoContent)
		}
		return h.handleGetParameterValuesResponse(c, sess, t, log)
	case "SetParameterValuesResponse":
		sess := h.getSessionFromCookie(c)
		if sess == nil {
			return c.SendStatus(http.StatusNoContent)
		}
		return h.handleSetParameterValuesResponse(c, sess, t, log)
	case "GetParameterNamesResponse":
		sess := h.getSessionFromCookie(c)
		if sess == nil {
			return c.SendStatus(http.StatusNoContent)
		}
		return h.handleGetParameterNamesResponse(c, sess, t, log)
	case "RebootResponse", "FactoryResetResponse", "DownloadResponse":
		sess := h.getSessionFromCookie(c)
		if sess == nil {
			return c.SendStatus(http.StatusNoContent)
		}
		return h.handleSimpleResponse(c, sess, t, log)
	case "Fault":
		sess := h.getSessionFromCookie(c)
		if sess == nil {
			return c.SendStatus(http.StatusNoContent)
		}
		return h.handleFault(c, sess, t, log)
	default:
		log.Warn("unknown CWMP message", zap.String("type", msgType))
		return h.dispatchNextTask(c, t, log)
	}
}

func (h *Handler) getSessionFromCookie(c *fiber.Ctx) *cwmp.Session {
	sessionID := h.sessionID(c)
	if sessionID == "" {
		return nil
	}
	sess := h.Sessions.Get(sessionID)
	if sess == nil || sess.Expired() {
		return nil
	}
	return sess
}

func (h *Handler) dispatchNextTask(c *fiber.Ctx, t *tenant.Tenant, log *zap.Logger) error {
	sess := h.getSessionFromCookie(c)
	if sess == nil {
		c.Status(fiber.StatusNoContent)
		return nil
	}
	sess.SetTenant(t)
	serial := sess.GetDeviceSerial()
	if serial == "" {
		c.Status(fiber.StatusNoContent)
		return nil
	}

	nextTask, err := h.TaskRepo.NextForDevice(c.Context(), t.ID, serial)
	if err != nil {
		log.Error("task queue error", zap.Error(err))
		c.Status(fiber.StatusNoContent)
		return nil
	}
	if nextTask == nil {
		c.Status(fiber.StatusNoContent)
		sess.Transition(cwmp.StateIdle)
		return nil
	}

	sess.SetCurrentTask(nextTask)
	sess.Transition(cwmp.StateWaitingTask)

	log.Info("dispatching task",
		zap.String("serial", serial),
		zap.String("task_id", nextTask.ID),
		zap.String("type", string(nextTask.Type)),
	)

	if h.Hub != nil {
		h.Hub.Broadcast(t.ID, events.EventTaskDispatched, fiber.Map{
			"task_id": nextTask.ID,
			"type":    string(nextTask.Type),
			"serial":  serial,
		})
	}

	return h.buildRPCRequest(c, sess, nextTask)
}

func (h *Handler) buildRPCRequest(c *fiber.Ctx, sess *cwmp.Session, t *task.Task) error {
	c.Set("Content-Type", "text/xml; charset=utf-8")
	c.Set("SOAPAction", "")

	cwmpID := sess.GetID()

	switch t.Type {
	case task.TypeGetParameterValues:
		env, err := cwmp.BuildGetParameterValues(cwmpID, t.ParameterNames)
		if err != nil {
			return err
		}
		return c.Send(env)
	case task.TypeSetParameterValues:
		env, err := cwmp.BuildSetParameterValues(cwmpID, t.ParameterValues)
		if err != nil {
			return err
		}
		return c.Send(env)
	case task.TypeGetParameterNames:
		path := ""
		if len(t.ParameterNames) > 0 {
			path = t.ParameterNames[0]
		}
		env, err := cwmp.BuildGetParameterNames(cwmpID, path, true)
		if err != nil {
			return err
		}
		return c.Send(env)
	case task.TypeReboot:
		env, err := cwmp.BuildReboot(cwmpID, t.ID)
		if err != nil {
			return err
		}
		return c.Send(env)
	case task.TypeFactoryReset:
		env, err := cwmp.BuildFactoryReset(cwmpID)
		if err != nil {
			return err
		}
		return c.Send(env)
	case task.TypeDownload:
		env, err := cwmp.BuildDownload(cwmpID, t.Download)
		if err != nil {
			return err
		}
		return c.Send(env)
	default:
		_ = h.TaskRepo.Fail(c.Context(), t.TenantID, t.ID, "unknown task type")
		c.Status(fiber.StatusNoContent)
		return nil
	}
}

func (h *Handler) sessionID(c *fiber.Ctx) string {
	// Prefer cookie set by ACS on InformResponse
	sess := c.Cookies("session")
	if sess != "" {
		return sess
	}
	return ""
}

func (h *Handler) handleInform(c *fiber.Ctx, log *zap.Logger, rawBody []byte, t *tenant.Tenant) error {
	_, cwmpID, _ := cwmp.UnmarshalEnvelope(c.Body())
	ip := c.IP()

	inform, err := cwmp.ParseInform(rawBody)
	if err != nil {
		log.Warn("inform parse error", zap.Error(err))
		return c.Status(http.StatusBadRequest).SendString("Invalid Inform")
	}

	log = log.With(
		zap.String("serial", inform.DeviceID.SerialNumber),
		zap.String("oui", inform.DeviceID.OUI),
		zap.Strings("events", inform.EventCodes()),
	)
	log.Info("inform received")

	if strings.TrimSpace(inform.DeviceID.SerialNumber) == "" {
		log.Warn("inform rejected: missing device serial number")
		return c.Status(http.StatusBadRequest).SendString("Inform missing device serial number")
	}

	params := make(map[string]string, len(inform.ParameterList.Params))
	for _, p := range inform.ParameterList.Params {
		params[p.Name] = p.Value
	}

	now := time.Now()
	dev := &device.Device{
		TenantID:     t.ID,
		SerialNumber: inform.DeviceID.SerialNumber,
		Manufacturer: inform.DeviceID.Manufacturer,
		OUI:          inform.DeviceID.OUI,
		ProductClass: inform.DeviceID.ProductClass,
		IPAddress:    ip,
		CWMPURL:      h.connectionURL(c),
		Online:       true,
		LastInform:   now,
		LastEvents:   inform.EventCodes(),
		Parameters:   params,

		SoftwareVersion: inform.GetParam("Device.DeviceInfo.SoftwareVersion"),
		HardwareVersion: inform.GetParam("Device.DeviceInfo.HardwareVersion"),
		ModelName:       inform.GetParam("Device.DeviceInfo.ModelName"),
		MACAddress: firstNonEmpty(
			inform.GetParam("Device.Ethernet.Interface.1.MACAddress"),
			inform.GetParam("InternetGatewayDevice.LANDevice.1.LANEthernetInterfaceConfig.1.MACAddress"),
		),
		ConnectionRequestURL: firstNonEmpty(
			inform.GetParam("Device.ManagementServer.ConnectionRequestURL"),
			inform.GetParam("InternetGatewayDevice.ManagementServer.ConnectionRequestURL"),
		),
		ConnectionRequestUsername: firstNonEmpty(
			inform.GetParam("Device.ManagementServer.ConnectionRequestUsername"),
			inform.GetParam("InternetGatewayDevice.ManagementServer.ConnectionRequestUsername"),
		),
		ConnectionRequestPassword: firstNonEmpty(
			inform.GetParam("Device.ManagementServer.ConnectionRequestPassword"),
			inform.GetParam("InternetGatewayDevice.ManagementServer.ConnectionRequestPassword"),
		),
	}
	if dev.SoftwareVersion == "" {
		dev.SoftwareVersion = inform.GetParam("InternetGatewayDevice.DeviceInfo.SoftwareVersion")
	}
	if dev.HardwareVersion == "" {
		dev.HardwareVersion = inform.GetParam("InternetGatewayDevice.DeviceInfo.HardwareVersion")
	}

	dev.PON = detectPON(inform)
	if inform.HasEvent("1 BOOT") || inform.HasEvent("0 BOOTSTRAP") {
		dev.LastBoot = now
	}

	if err := h.Devices.Upsert(c.Context(), dev); err != nil {
		log.Error("upsert device failed", zap.Error(err))
	} else {
		log.Info("device upserted",
			zap.String("sw_version", dev.SoftwareVersion),
			zap.String("ip", dev.IPAddress),
		)
		// Merge Inform params without replacing the full parameters map
		// (so GetParameterValues results like WiFi params are preserved).
		if len(params) > 0 {
			if err := h.Devices.UpdateParameters(c.Context(), t.ID, dev.SerialNumber, params); err != nil {
				log.Warn("failed to merge inform parameters", zap.Error(err))
			}
		}
		if h.Hub != nil {
			h.Hub.Broadcast(t.ID, events.EventDeviceInform, fiber.Map{
				"serial":           dev.SerialNumber,
				"manufacturer":     dev.Manufacturer,
				"model":            dev.ModelName,
				"software_version": dev.SoftwareVersion,
				"ip":               dev.IPAddress,
				"events":           dev.LastEvents,
			})
		}
	}

	eventCodes := inform.EventCodes()
	if h.Provisioner != nil {
		go h.Provisioner.OnInform(context.Background(), dev, eventCodes)
	}

	sessionID := h.sessionID(c)
	if sessionID == "" {
		sessionID = t.ID + "-" + inform.DeviceID.SerialNumber + "-" + cwmpID
	}
	sess := h.Sessions.GetOrCreate(sessionID)
	sess.SetTenant(t)
	sess.SetDeviceSerial(inform.DeviceID.SerialNumber)
	sess.SetCWMPID(cwmpID)
	sess.SetParameterTree(params)
	sess.Transition(cwmp.StateInformed)

	respBody := cwmp.BuildInformResponseBody()
	env, err := cwmp.BuildEnvelope(respBody, cwmpID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString("Internal error")
	}

	c.Set("Content-Type", "text/xml; charset=utf-8")
	c.Cookie(&fiber.Cookie{
		Name:  "session",
		Value: sessionID,
		Path:  "/",
	})
	return c.Send(env)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func detectPON(inform *cwmp.Inform) *device.PONInfo {
	for _, p := range inform.ParameterList.Params {
		if strings.Contains(p.Name, "PON") ||
			strings.Contains(p.Name, "Optical") ||
			strings.Contains(p.Name, "GPON") ||
			strings.Contains(p.Name, "EPON") {
			return &device.PONInfo{
				PONType:     detectPONType(inform),
				SignalLevel: inform.GetParam("Device.Optical.Interface.1.Stats.X_ReceivePower"),
			}
		}
	}
	return nil
}

func detectPONType(inform *cwmp.Inform) string {
	for _, p := range inform.ParameterList.Params {
		if strings.Contains(p.Name, "GPON") {
			return "GPON"
		}
		if strings.Contains(p.Name, "EPON") {
			return "EPON"
		}
		if strings.Contains(p.Name, "XGPON") {
			return "XGPON"
		}
	}
	return "unknown"
}

func (h *Handler) handleGetParameterValuesResponse(c *fiber.Ctx, sess *cwmp.Session, t *tenant.Tenant, log *zap.Logger) error {
	rawBody, _, _ := cwmp.UnmarshalEnvelope(c.Body())
	resp, err := cwmp.ParseGetParameterValuesResponse(rawBody)
	if err != nil {
		log.Error("parse GetParameterValuesResponse", zap.Error(err))
		return h.dispatchNextTask(c, t, log)
	}

	values := make(map[string]string, len(resp.ParameterList.Params))
	for _, p := range resp.ParameterList.Params {
		values[p.Name] = p.Value
	}

	serial := sess.GetDeviceSerial()
	if err := h.Devices.UpdateParameters(c.Context(), t.ID, serial, values); err != nil {
		log.Error("update parameters", zap.Error(err))
	} else if h.Hub != nil {
		h.Hub.Broadcast(t.ID, events.EventParametersUpdated, fiber.Map{
			"serial": serial,
			"count":  len(values),
		})
	}

	taskID := sess.GetCurrentTaskID()
	if taskID != "" {
		_ = h.TaskRepo.Complete(c.Context(), t.ID, taskID, task.Result{
			Success:     true,
			Values:      values,
			CompletedAt: time.Now(),
		})
		sess.ClearCurrentTask()
	}

	log.Info("GetParameterValues complete",
		zap.String("serial", serial),
		zap.Int("count", len(values)),
	)
	return h.dispatchNextTask(c, t, log)
}

func (h *Handler) handleSetParameterValuesResponse(c *fiber.Ctx, sess *cwmp.Session, t *tenant.Tenant, log *zap.Logger) error {
	taskID := sess.GetCurrentTaskID()
	serial := sess.GetDeviceSerial()
	var paramNames []string
	if taskID != "" {
		tk, err := h.TaskRepo.GetByID(c.Context(), t.ID, taskID)
		if err == nil && tk.ParameterValues != nil {
			updates := make(map[string]string)
			if u, ok := tk.ParameterValues["Device.ManagementServer.ConnectionRequestUsername"]; ok && u != "" {
				updates["connection_request_username"] = u
			}
			if p, ok := tk.ParameterValues["Device.ManagementServer.ConnectionRequestPassword"]; ok && p != "" {
				updates["connection_request_password"] = p
			}
			if u, ok := tk.ParameterValues["InternetGatewayDevice.ManagementServer.ConnectionRequestUsername"]; ok && u != "" {
				updates["connection_request_username"] = u
			}
			if p, ok := tk.ParameterValues["InternetGatewayDevice.ManagementServer.ConnectionRequestPassword"]; ok && p != "" {
				updates["connection_request_password"] = p
			}
			if len(updates) > 0 {
				if err := h.Devices.UpdateConnectionRequest(c.Context(), t.ID, serial, updates); err != nil {
					log.Warn("store connection request credentials", zap.Error(err), zap.String("serial", serial))
				} else {
					log.Info("stored connection request credentials", zap.String("serial", serial))
				}
			}
			// Refetch the params we just set so DB and dashboard stay in sync
			for k := range tk.ParameterValues {
				paramNames = append(paramNames, k)
			}
		}
		_ = h.TaskRepo.Complete(c.Context(), t.ID, taskID, task.Result{
			Success:     true,
			CompletedAt: time.Now(),
		})
		sess.ClearCurrentTask()
	}
	if len(paramNames) > 0 {
		refetch := &task.Task{
			TenantID:       t.ID,
			DeviceSerial:   serial,
			Type:           task.TypeGetParameterValues,
			Status:         task.StatusPending,
			Priority:       15,
			ParameterNames: paramNames,
			CreatedAt:      time.Now(),
			Timeout:        int64(5 * time.Minute),
			CreatedBy:      "acs",
		}
		if err := h.TaskRepo.Enqueue(c.Context(), refetch); err != nil {
			log.Warn("enqueue refetch after SetParameterValues", zap.Error(err))
		}
	}
	log.Info("SetParameterValues complete", zap.String("serial", serial))
	return h.dispatchNextTask(c, t, log)
}

func (h *Handler) handleGetParameterNamesResponse(c *fiber.Ctx, sess *cwmp.Session, t *tenant.Tenant, log *zap.Logger) error {
	taskID := sess.GetCurrentTaskID()
	if taskID != "" {
		_ = h.TaskRepo.Complete(c.Context(), t.ID, taskID, task.Result{
			Success:     true,
			CompletedAt: time.Now(),
		})
		sess.ClearCurrentTask()
	}
	return h.dispatchNextTask(c, t, log)
}

func (h *Handler) handleSimpleResponse(c *fiber.Ctx, sess *cwmp.Session, t *tenant.Tenant, log *zap.Logger) error {
	taskID := sess.GetCurrentTaskID()
	if taskID != "" {
		_ = h.TaskRepo.Complete(c.Context(), t.ID, taskID, task.Result{
			Success:     true,
			CompletedAt: time.Now(),
		})
		if h.Hub != nil {
			h.Hub.Broadcast(t.ID, events.EventTaskComplete, fiber.Map{
				"task_id": taskID,
				"type":    string(sess.GetCurrentTaskType()),
				"serial":  sess.GetDeviceSerial(),
				"success": true,
			})
		}
		sess.ClearCurrentTask()
	}
	return h.dispatchNextTask(c, t, log)
}

func (h *Handler) handleTransferComplete(c *fiber.Ctx, sess *cwmp.Session, t *tenant.Tenant, log *zap.Logger) error {
	rawBody, _, _ := cwmp.UnmarshalEnvelope(c.Body())
	tc, err := cwmp.ParseTransferComplete(rawBody)
	if err != nil {
		log.Error("parse TransferComplete", zap.Error(err))
	} else {
		success := tc.FaultStruct == nil || tc.FaultStruct.FaultCode == 0
		log.Info("TransferComplete",
			zap.String("serial", sess.GetDeviceSerial()),
			zap.String("command_key", tc.CommandKey),
			zap.Bool("success", success),
		)
		if tc.FaultStruct != nil {
			log.Info("TransferComplete fault", zap.String("fault", tc.FaultStruct.FaultString))
		}
		taskID := sess.GetCurrentTaskID()
		if taskID != "" {
			if success {
				_ = h.TaskRepo.Complete(c.Context(), t.ID, taskID, task.Result{
					Success:     true,
					CompletedAt: time.Now(),
				})
				if h.Hub != nil {
					h.Hub.Broadcast(t.ID, events.EventTaskComplete, fiber.Map{
						"task_id": taskID,
						"type":    string(sess.GetCurrentTaskType()),
						"serial":  sess.GetDeviceSerial(),
						"success": true,
					})
				}
			} else {
				reason := ""
				if tc.FaultStruct != nil {
					reason = tc.FaultStruct.FaultString
				}
				_ = h.TaskRepo.Fail(c.Context(), t.ID, taskID, reason)
				if h.Hub != nil {
					h.Hub.Broadcast(t.ID, events.EventTaskFailed, fiber.Map{
						"task_id": taskID,
						"serial":  sess.GetDeviceSerial(),
						"fault":   reason,
					})
				}
			}
			sess.ClearCurrentTask()
		}
	}

	env, err := cwmp.BuildTransferCompleteResponse(sess.GetID())
	if err != nil {
		return err
	}
	c.Set("Content-Type", "text/xml; charset=utf-8")
	return c.Send(env)
}

func (h *Handler) handleFault(c *fiber.Ctx, sess *cwmp.Session, t *tenant.Tenant, log *zap.Logger) error {
	rawBody, _, _ := cwmp.UnmarshalEnvelope(c.Body())
	fault, err := cwmp.ParseFault(rawBody)
	taskID := sess.GetCurrentTaskID()

	if err == nil {
		log.Warn("CWMP fault from device",
			zap.String("serial", sess.GetDeviceSerial()),
			zap.String("code", fault.Detail.FaultCode),
			zap.String("message", fault.Detail.FaultString),
		)
		if taskID != "" {
			_ = h.TaskRepo.Fail(c.Context(), t.ID, taskID,
				fmt.Sprintf("%s: %s", fault.Detail.FaultCode, fault.Detail.FaultString))
			if h.Hub != nil {
				h.Hub.Broadcast(t.ID, events.EventTaskFailed, fiber.Map{
					"task_id": taskID,
					"serial":  sess.GetDeviceSerial(),
					"fault":   fault.Detail.FaultString,
				})
			}
		}
	}
	sess.ClearCurrentTask()
	return h.dispatchNextTask(c, t, log)
}

func (h *Handler) connectionURL(c *fiber.Ctx) string {
	scheme := "http"
	if c.Protocol() == "https" {
		scheme = "https"
	}
	host := c.Get("Host", c.Hostname())
	return scheme + "://" + strings.TrimSpace(host) + c.Path()
}
