package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/labstack/echo/v5"
	echomw "github.com/labstack/echo/v5/middleware"
	"github.com/vasti/gdc-backend-test/internal/config"
	"github.com/vasti/gdc-backend-test/internal/db"
	appmw "github.com/vasti/gdc-backend-test/internal/middleware"
	"github.com/vasti/gdc-backend-test/internal/repository"
	"github.com/vasti/gdc-backend-test/internal/service"

	"github.com/vasti/gdc-backend-test/internal/handler"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Create root context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize OpenTelemetry (optional — only if OTEL_EXPORTER_OTLP_ENDPOINT is set)
	otelShutdown := initOTEL(ctx)
	defer otelShutdown()

	// Database connection pool
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to create database pool: %v", err)
	}
	defer pool.Close()

	// Run migrations
	if err := db.RunMigrations(ctx, pool); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	// Repositories
	userRepo := repository.NewUserRepository(pool)
	taskRepo := repository.NewTaskRepository(pool)
	idemRepo := repository.NewIdempotencyRepository(pool)

	// Services
	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret)
	taskSvc := service.NewTaskService(taskRepo)
	taskAssignSvc := service.NewTaskAssignService(pool, userRepo, taskRepo)

	// Handlers
	authH := handler.NewAuthHandler(authSvc)
	taskH := handler.NewTaskHandler(taskSvc, taskAssignSvc)

	// Echo instance
	e := echo.New()

	// Global error handler (catches errors from all middleware and routes)
	e.HTTPErrorHandler = appmw.GlobalErrorHandler

	// Middleware (order matters)
	e.Use(echomw.Recover())
	e.Use(appmw.RequestIDMiddleware())
	e.Use(appmw.TraceMiddleware())
	e.Use(appmw.LoggerMiddleware())
	e.Use(echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Authorization", "Content-Type", "Idempotency-Key", "X-Request-ID"},
	}))
	e.Use(appmw.ErrorHandlerMiddleware())

	// Health check
	e.GET("/health", func(c *echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// Public routes
	e.POST("/register", authH.Register)
	e.POST("/login", authH.Login)

	// Protected routes
	protected := e.Group("")
	protected.Use(appmw.AuthMiddleware(cfg.JWTSecret))

	// Idempotent task creation
	protected.POST("/tasks", taskH.Create, appmw.IdempotencyMiddleware(idemRepo))

	// Other task routes
	protected.GET("/tasks", taskH.List)
	protected.GET("/tasks/:id", taskH.GetByID)
	protected.PUT("/tasks/:id", taskH.Update)
	protected.DELETE("/tasks/:id", taskH.Delete)
	protected.POST("/tasks/:id/assign", taskH.Assign)

	// Start server in a goroutine
	go func() {
		if err := e.Start(":" + cfg.Port); err != nil {
			log.Printf("server stopped: %v", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")
	cancel()
}
