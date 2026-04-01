package api

import (
	"github.com/novaacs/go-acs/internal/api/middleware"
	"github.com/novaacs/go-acs/internal/auth"
)

func (s *Server) registerRoutes() {
	h := &Handler{
		log:             s.log,
		deviceRepo:      s.deviceRepo,
		taskRepo:        s.taskRepo,
		provRepo:        s.provRepo,
		tenantRepo:      s.tenantRepo,
		authSvc:         s.authSvc,
		connreqClient:   s.connreqClient,
		backupRepo:      s.backupRepo,
		backupSvc:       s.backupSvc,
		credProfileRepo: s.credProfileRepo,
	}

	authMW := middleware.Auth(s.authSvc, s.tenantRepo)

	s.app.Get("/health", h.Health)
	s.app.Get("/ready", h.Ready)
	s.app.Post("/api/v1/auth/login", h.Login)
	s.app.Post("/api/v1/auth/refresh", h.RefreshToken)

	v1 := s.app.Group("/api/v1", authMW)

	v1.Get("/ws", h.WSUpgrade, h.WSHandler(s.hub))
	v1.Get("/auth/me", h.GetMe)

	v1.Get("/devices", h.ListDevices)
	v1.Post("/devices/preregister", middleware.RequireRole(auth.RoleAdmin), h.PreRegister)
	// More specific /devices/:serial/... routes before generic /devices/:serial
	v1.Get("/devices/:serial/parameters", h.GetDeviceParameters)
	v1.Get("/devices/:serial/tasks", h.GetDeviceTasks)
	v1.Get("/devices/:serial/backups", h.ListBackups)
	v1.Post("/devices/:serial/backups", h.CreateBackup)
	v1.Get("/devices/:serial/backups/:id", h.GetBackup)
	v1.Delete("/devices/:serial/backups/:id", middleware.RequireRole(auth.RoleAdmin), h.DeleteBackup)
	v1.Post("/devices/:serial/backups/:id/restore", middleware.RequireRole(auth.RoleAdmin), h.RestoreBackup)

	v1.Get("/devices/:serial", h.GetDevice)
	v1.Delete("/devices/:serial", middleware.RequireRole(auth.RoleAdmin), h.DeleteDevice)

	v1.Post("/devices/:serial/reboot", h.Reboot)
	v1.Post("/devices/:serial/factory-reset", middleware.RequireRole(auth.RoleAdmin), h.FactoryReset)
	v1.Post("/devices/:serial/get-parameters", h.GetParameters)
	v1.Post("/devices/:serial/set-parameters", h.SetParameters)
	v1.Post("/devices/:serial/download", middleware.RequireRole(auth.RoleAdmin), h.Download)
	v1.Post("/devices/:serial/get-names", h.GetParameterNames)
	v1.Post("/devices/:serial/wake", h.Wake)

	v1.Get("/restore-jobs/:id", h.GetRestoreJob)

	creds := v1.Group("/credential-profiles", middleware.RequireRole(auth.RoleAdmin))
	creds.Get("", h.ListCredProfiles)
	creds.Post("", h.CreateCredProfile)
	creds.Put("/:id", h.UpdateCredProfile)
	creds.Delete("/:id", h.DeleteCredProfile)
	v1.Patch("/tenant/credentials", middleware.RequireRole(auth.RoleAdmin), h.UpdateTenantDefaults)

	v1.Get("/tasks", h.ListTasks)
	v1.Get("/tasks/:id", h.GetTask)
	v1.Delete("/tasks/:id", h.CancelTask)

	prov := v1.Group("/provisioning", middleware.RequireRole(auth.RoleAdmin))
	prov.Get("/rules", h.ListRules)
	prov.Post("/rules", h.CreateRule)
	prov.Get("/rules/:id", h.GetRule)
	prov.Put("/rules/:id", h.UpdateRule)
	prov.Delete("/rules/:id", h.DeleteRule)

	v1.Get("/stats", h.Stats)
}
