package acs

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/novaacs/go-acs/internal/acs/middleware"
	"github.com/novaacs/go-acs/internal/cwmp"
	"github.com/novaacs/go-acs/internal/device"
	"github.com/novaacs/go-acs/internal/events"
	"github.com/novaacs/go-acs/internal/provisioning"
	"github.com/novaacs/go-acs/internal/task"
	"github.com/novaacs/go-acs/internal/tenant"
	"go.uber.org/zap"
)

// Server runs the ACS CWMP HTTP server (port 7547).
type Server struct {
	app        *fiber.App
	handler    *Handler
	log        *zap.Logger
	tenantRepo tenant.Repository
}

// NewServer creates the ACS Fiber app and CWMP handler.
func NewServer(_ int, devRepo device.Repository, taskRepo task.Repository, sessions *cwmp.SessionManager, log *zap.Logger, tenantRepo tenant.Repository, provEngine *provisioning.Engine, hub *events.Hub) (*Server, error) {
	if sessions == nil {
		sessions = cwmp.NewSessionManager(30*time.Second, 10000)
	}
	h := &Handler{
		Sessions:    sessions,
		Devices:     devRepo,
		TaskRepo:    taskRepo,
		Provisioner: provEngine,
		Hub:         hub,
		Log:         log,
	}

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		BodyLimit:             4 * 1024 * 1024, // 4MB for SOAP
	})
	app.Use(recover.New())
	app.Use(logger.New())

	// Tenant from URL slug: POST /cwmp/:tenant
	app.Post("/cwmp/:tenant", middleware.FromSlug(tenantRepo, log), h.ServeCWMP)
	// Tenant from X-ACS-Key header: POST /cwmp
	app.Post("/cwmp", middleware.FromAPIKey(tenantRepo, log), h.ServeCWMP)
	// Legacy fallback (API key, dev default): POST / and POST /acs
	app.Post("/", middleware.FromAPIKey(tenantRepo, log), h.ServeCWMP)
	app.Post("/acs", middleware.FromAPIKey(tenantRepo, log), h.ServeCWMP)

	return &Server{app: app, handler: h, log: log, tenantRepo: tenantRepo}, nil
}

// Listen starts the server on the given port.
func (s *Server) Listen(addr string) error {
	return s.app.Listen(addr)
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown() error {
	return s.app.Shutdown()
}
