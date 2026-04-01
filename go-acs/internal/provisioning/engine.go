package provisioning

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/novaacs/go-acs/internal/device"
	"github.com/novaacs/go-acs/internal/task"
	"go.uber.org/zap"
)

const defaultTaskTimeout = 5 * time.Minute

// Engine applies provisioning rules after device Inform.
type Engine struct {
	rules Repository
	tasks task.Repository
	log   *zap.Logger
}

// NewEngine returns a new provisioning engine.
func NewEngine(rules Repository, tasks task.Repository, log *zap.Logger) *Engine {
	return &Engine{rules: rules, tasks: tasks, log: log}
}

// OnInform is called async from handleInform after device upsert.
// It loads active rules for the tenant, matches them against the device and event codes,
// and enqueues matching tasks.
func (e *Engine) OnInform(ctx context.Context, dev *device.Device, events []string) {
	rules, err := e.rules.ListActive(ctx, dev.TenantID)
	if err != nil {
		e.log.Error("load provisioning rules",
			zap.String("tenant_id", dev.TenantID),
			zap.Error(err),
		)
		return
	}

	e.log.Debug("provisioning check",
		zap.String("serial", dev.SerialNumber),
		zap.Int("rules", len(rules)),
		zap.Strings("events", events),
	)

	for _, rule := range rules {
		if !e.matchesEvents(rule, events) {
			continue
		}
		if !e.matchesDevice(rule, dev) {
			continue
		}

		e.log.Info("provisioning rule matched",
			zap.String("serial", dev.SerialNumber),
			zap.String("rule", rule.Name),
			zap.String("trigger", rule.Trigger),
		)

		e.applyRule(ctx, rule, dev)
	}
}

// matchesEvents returns true if the rule trigger matches any event code.
func (e *Engine) matchesEvents(rule *Rule, events []string) bool {
	if rule.Trigger == "ANY" {
		return true
	}
	for _, ev := range events {
		if ev == rule.Trigger {
			return true
		}
	}
	return false
}

// matchesDevice returns true if all non-empty match criteria pass.
func (e *Engine) matchesDevice(rule *Rule, dev *device.Device) bool {
	if rule.MatchManufacturer != "" &&
		!strings.EqualFold(rule.MatchManufacturer, dev.Manufacturer) {
		return false
	}
	if rule.MatchOUI != "" &&
		!strings.EqualFold(rule.MatchOUI, dev.OUI) {
		return false
	}
	if rule.MatchProductClass != "" &&
		!strings.EqualFold(rule.MatchProductClass, dev.ProductClass) {
		return false
	}
	if rule.MatchModelName != "" &&
		!strings.EqualFold(rule.MatchModelName, dev.ModelName) {
		return false
	}
	if rule.MatchSWVersion != "" {
		matched, err := regexp.MatchString(rule.MatchSWVersion, dev.SoftwareVersion)
		if err != nil || !matched {
			return false
		}
	}
	return true
}

// applyRule enqueues one task per action in the rule.
func (e *Engine) applyRule(ctx context.Context, rule *Rule, dev *device.Device) {
	for i, action := range rule.Actions {
		t := &task.Task{
			TenantID:        dev.TenantID,
			DeviceSerial:    dev.SerialNumber,
			Type:            action.Type,
			Status:          task.StatusPending,
			Priority:        rule.Priority + i,
			ParameterNames:  action.ParameterNames,
			ParameterValues: action.ParameterValues,
			Download:        action.Download,
			CreatedAt:       time.Now(),
			Timeout:         int64(defaultTaskTimeout),
			CreatedBy:       "provisioning:" + rule.ID,
		}

		if err := e.tasks.Enqueue(ctx, t); err != nil {
			e.log.Error("enqueue provisioning task",
				zap.String("serial", dev.SerialNumber),
				zap.String("rule", rule.Name),
				zap.String("task_type", string(action.Type)),
				zap.Error(err),
			)
		} else {
			e.log.Info("provisioning task enqueued",
				zap.String("serial", dev.SerialNumber),
				zap.String("rule", rule.Name),
				zap.String("task_type", string(action.Type)),
				zap.String("task_id", t.ID),
			)
		}
	}
}
