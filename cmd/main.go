package main

import (
	"OrgAPI/internal/config"
	"OrgAPI/internal/handler"
	"OrgAPI/internal/logger"
	"OrgAPI/internal/repository"
	"OrgAPI/internal/service"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel)
	log.Info("service starting",
		"addr", cfg.ServerAddress,
		"log_level", cfg.LogLevel,
	)

	db, err := openDB(cfg, log)
	if err != nil {
		log.Error("failed to connect database", "error", err)
		os.Exit(1)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Error("failed to get sql database", "error", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	sqlDB.SetMaxOpenConns(cfg.DBMaxConns)
	sqlDB.SetMaxIdleConns(cfg.DBMinConns)
	sqlDB.SetConnMaxIdleTime(cfg.DBConnMaxIdleTime)

	if err := goose.SetDialect("postgres"); err != nil {
		log.Error("failed to configure goose", "error", err)
		os.Exit(1)
	}
	if err := goose.Up(sqlDB, cfg.MigrationsDir); err != nil {
		log.Error("failed to apply database migrations", "error", err)
		os.Exit(1)
	}

	departmentRepo := repository.NewDepartmentRepo(db)
	employeeRepo := repository.NewEmployeeRepo(db)
	departmentService := service.NewDepartmentService(departmentRepo)
	employeeService := service.NewEmployeeService(employeeRepo)
	departmentHandler := handler.NewDepartmentHandler(departmentService, log)
	employeeHandler := handler.NewEmployeeHandler(employeeService, log)

	server := &http.Server{
		Addr:         cfg.ServerAddress,
		Handler:      routes(departmentHandler, employeeHandler),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		log.Info("http server listening", "addr", server.Addr)
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error("http server shutdown failed", "error", err)
			os.Exit(1)
		}
		log.Info("service stopped")
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server failed", "error", err)
			os.Exit(1)
		}
	}
}

func openDB(cfg *config.Config, log *slog.Logger) (*gorm.DB, error) {
	gormLogLevel := gormLogger.Warn
	if cfg.LogLevel == "debug" {
		gormLogLevel = gormLogger.Info
	}

	return gorm.Open(postgres.Open(cfg.DatabaseDSN), &gorm.Config{
		Logger: gormLogger.New(slog.NewLogLogger(log.Handler(), slog.LevelDebug), gormLogger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  gormLogLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		}),
	})
}

func routes(departments *handler.DepartmentHandler, employees *handler.EmployeeHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	mux.HandleFunc("POST /departments/", departments.CreateDepartment)
	mux.HandleFunc("POST /departments/{department_id}/employees/", employees.CreateEmployee)
	mux.HandleFunc("GET /departments/{id}", departments.GetByID)
	mux.HandleFunc("PATCH /departments/{id}", departments.UpdateParent)
	mux.HandleFunc("DELETE /departments/{id}", departments.Delete)

	return mux
}
