package api

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/novaacs/go-acs/internal/auth"
	"github.com/novaacs/go-acs/internal/backup"
	"github.com/novaacs/go-acs/internal/config"
	"github.com/novaacs/go-acs/internal/connreq"
	"github.com/novaacs/go-acs/internal/credprofile"
	"github.com/novaacs/go-acs/internal/device"
	"github.com/novaacs/go-acs/internal/events"
	"github.com/novaacs/go-acs/internal/provisioning"
	"github.com/novaacs/go-acs/internal/task"
	"github.com/novaacs/go-acs/internal/tenant"
	"go.uber.org/zap"
)

type Server struct {
	app             *fiber.App
	cfg             *config.Config
	log             *zap.Logger
	deviceRepo      device.Repository
	taskRepo        task.Repository
	provRepo        provisioning.Repository
	tenantRepo      tenant.Repository
	authSvc         *auth.Service
	hub             *events.Hub
	connreqClient   *connreq.Client
	backupRepo      backup.Repository
	backupSvc       *backup.Service
	credProfileRepo credprofile.Repository
}

func NewServer(
	cfg *config.Config,
	log *zap.Logger,
	deviceRepo device.Repository,
	taskRepo task.Repository,
	provRepo provisioning.Repository,
	tenantRepo tenant.Repository,
	authSvc *auth.Service,
	hub *events.Hub,
	connreqClient *connreq.Client,
	backupRepo backup.Repository,
	backupSvc *backup.Service,
	credProfileRepo credprofile.Repository,
) *Server {
	s := &Server{
		cfg:             cfg,
		log:             log,
		deviceRepo:      deviceRepo,
		taskRepo:        taskRepo,
		provRepo:        provRepo,
		tenantRepo:      tenantRepo,
		authSvc:         authSvc,
		hub:             hub,
		connreqClient:   connreqClient,
		backupRepo:      backupRepo,
		backupSvc:       backupSvc,
		credProfileRepo: credProfileRepo,
	}

	s.app = fiber.New(fiber.Config{
		AppName:      "NovaACS API",
		ErrorHandler: s.errorHandler,
	})

	s.app.Use(recover.New())
	s.app.Use(requestid.New())
	s.app.Use(corsMiddleware)

	s.registerRoutes()
	return s
}

func (s *Server) Start(addr string) error {
	s.log.Info("REST API listening", zap.String("addr", addr))
	return s.app.Listen(addr)
}

func (s *Server) Shutdown() error {
	return s.app.Shutdown()
}

func (s *Server) errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	msg := "internal server error"
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		msg = e.Message
	}
	return c.Status(code).JSON(fiber.Map{"error": msg})
}

// corsMiddleware sets CORS headers so dashboard on another origin can call the API.
// Runs on every request; for OPTIONS preflight returns 204 with headers only.
func corsMiddleware(c *fiber.Ctx) error {
	origin := c.Get("Origin")
	if origin == "" {
		return c.Next()
	}
	if !corsAllowOrigin(origin) {
		return c.Next()
	}
	c.Set("Access-Control-Allow-Origin", origin)
	c.Set("Access-Control-Allow-Credentials", "true")
	c.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	c.Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-ACS-Key, X-Tenant-ID")
	c.Set("Access-Control-Max-Age", "86400")
	if c.Method() == fiber.MethodOptions {
		return c.SendStatus(fiber.StatusNoContent)
	}
	return c.Next()
}

func corsAllowOrigin(origin string) bool {
	if origin == "https://black.unganishanetworks.com" {
		return true
	}
	if strings.HasSuffix(origin, ".unganishanetworks.com") {
		return true
	}
	if origin == "http://localhost:5173" || origin == "http://127.0.0.1:5173" {
		return true
	}
	return false
}
