package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/novaacs/go-acs/internal/acs"
	"github.com/novaacs/go-acs/internal/api"
	"github.com/novaacs/go-acs/internal/auth"
	"github.com/novaacs/go-acs/internal/backup"
	"github.com/novaacs/go-acs/internal/config"
	"github.com/novaacs/go-acs/internal/connreq"
	"github.com/novaacs/go-acs/internal/credprofile"
	"github.com/novaacs/go-acs/internal/cwmp"
	"github.com/novaacs/go-acs/internal/db"
	"github.com/novaacs/go-acs/internal/device"
	"github.com/novaacs/go-acs/internal/events"
	"github.com/novaacs/go-acs/internal/logger"
	"github.com/novaacs/go-acs/internal/provisioning"
	"github.com/novaacs/go-acs/internal/task"
	"github.com/novaacs/go-acs/internal/tenant"
	"go.uber.org/zap"
)

func main() {
	cfgPath := ""
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.New(cfg.Logging.Level, cfg.Logging.Format)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	ctx := context.Background()

	pgPool, err := db.NewPostgresPool(ctx, cfg.Database.PostgresDSN)
	if err != nil {
		log.Fatal("postgres connect", zap.Error(err))
	}
	defer pgPool.Close()

	if err := db.RunMigrations(ctx, pgPool, "migrations"); err != nil {
		log.Fatal("migrations", zap.Error(err))
	}

	tenantRepo := tenant.NewPostgresRepository(pgPool)
	t, err := tenantRepo.GetBySlug(ctx, "default")
	if err != nil {
		log.Fatal("default tenant missing — check migrations", zap.Error(err))
	}
	log.Info("default tenant loaded", zap.String("id", t.ID))

	mongoClient, err := db.NewMongoClient(ctx, cfg.Database.MongoURI)
	if err != nil {
		log.Fatal("mongodb connect", zap.Error(err))
	}
	defer mongoClient.Disconnect(ctx)

	mongoDB := mongoClient.Database(cfg.Database.MongoDB)
	deviceRepo := device.NewMongoRepository(mongoDB, log)
	if err := deviceRepo.EnsureIndexes(ctx); err != nil {
		log.Fatal("device indexes", zap.Error(err))
	}
	log.Info("mongodb connected", zap.String("db", cfg.Database.MongoDB))

	taskRepo := task.NewMongoRepository(mongoDB, log)
	if err := taskRepo.EnsureIndexes(ctx); err != nil {
		log.Fatal("task indexes", zap.Error(err))
	}
	log.Info("task repository ready")

	provRulesRepo := provisioning.NewPostgresRepository(pgPool, log)
	provEngine := provisioning.NewEngine(provRulesRepo, taskRepo, log)
	log.Info("provisioning engine ready")

	userRepo := auth.NewPostgresRepository(pgPool, log)
	authCfg := auth.Config{
		AccessSecret:  cfg.Auth.AccessSecret,
		RefreshSecret: cfg.Auth.RefreshSecret,
		AccessTTL:     cfg.Auth.AccessTTL,
		RefreshTTL:    cfg.Auth.RefreshTTL,
	}
	if authCfg.AccessTTL == 0 {
		authCfg.AccessTTL = 15 * time.Minute
	}
	if authCfg.RefreshTTL == 0 {
		authCfg.RefreshTTL = 168 * time.Hour
	}
	authSvc := auth.NewService(userRepo, tenantRepo, authCfg, log)
	log.Info("auth service ready")

	sessions := cwmp.NewSessionManager(cfg.ACS.SessionTimeout, cfg.ACS.MaxConcurrentSessions)

	hub := events.NewHub(log)
	go hub.Run()
	log.Info("event hub started")

	credProfileRepo := credprofile.NewPostgresRepository(pgPool, log)
	credResolver := credprofile.NewResolver(credProfileRepo, tenantRepo, log)
	log.Info("credential resolver ready")

	connreqClient := connreq.NewClient(deviceRepo, tenantRepo, credResolver, hub, log)
	log.Info("connection request client ready")

	backupRepo := backup.NewMongoRepository(mongoDB, log)
	if err := backupRepo.EnsureIndexes(ctx); err != nil {
		log.Fatal("backup indexes", zap.Error(err))
	}
	backupSvc := backup.NewService(backupRepo, deviceRepo, taskRepo, log)
	log.Info("backup service ready")

	acsSrv, err := acs.NewServer(cfg.ACS.CWMPPort, deviceRepo, taskRepo, sessions, log, tenantRepo, provEngine, hub)
	if err != nil {
		log.Fatal("acs server", zap.Error(err))
	}

	apiSrv := api.NewServer(cfg, log, deviceRepo, taskRepo, provRulesRepo, tenantRepo, authSvc, hub, connreqClient, backupRepo, backupSvc, credProfileRepo)

	apiAddr := cfg.API.Addr
	if apiAddr == "" {
		apiAddr = ":8080"
	}

	go func() {
		addr := fmt.Sprintf(":%d", cfg.ACS.CWMPPort)
		log.Info("ACS CWMP listening", zap.String("addr", addr))
		if err := acsSrv.Listen(addr); err != nil {
			log.Error("acs listen", zap.Error(err))
		}
	}()

	go func() {
		if err := apiSrv.Start(apiAddr); err != nil {
			log.Error("api listen", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutting down")
	_ = acsSrv.Shutdown()
	_ = apiSrv.Shutdown()
}
