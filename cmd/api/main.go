package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/louisealberti/onboarding-api/docs"
	"github.com/louisealberti/onboarding-api/internal/config"
	"github.com/louisealberti/onboarding-api/internal/database"
	"github.com/louisealberti/onboarding-api/internal/handler"
	"github.com/louisealberti/onboarding-api/internal/middleware"
	"github.com/louisealberti/onboarding-api/internal/repository"
	"github.com/louisealberti/onboarding-api/internal/service"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Injected at build time via:
// go build -ldflags "-X main.version=$(git rev-parse --short HEAD) -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration error", slog.Any("error", err))
		os.Exit(1)
	}

	db, err := database.NewPostgresConnection(cfg)
	if err != nil {
		logger.Error("critical failure connecting to database", slog.Any("error", err))
		os.Exit(1)
	}
	defer db.Close()

	repo := repository.NewCustomerRepository(db)
	idempotencyRepo := repository.NewIdempotencyRepository(db)
	svc := service.NewCustomerService(repo)
	h := handler.NewCustomerHandler(svc)
	hh := handler.NewHealthHandler(db, handler.BuildInfo{
		Version:   version,
		BuildTime: buildTime,
	})

	corsOrigins := os.Getenv("CORS_ORIGINS")
	if corsOrigins == "" {
		corsOrigins = "*"
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS([]string{corsOrigins}))
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(logger))

	// Swagger UI — available at /swagger/index.html
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.GET("/health", hh.Health)

	v1 := r.Group("/v1")
	// When /v2 is introduced, uncomment to signal deprecation:
	// v1.Use(middleware.Deprecated("2027-01-01", "https://api.example.com/v2"))
	v1.POST("/customers", middleware.Idempotency(idempotencyRepo), h.CreateCustomer)
	v1.PUT("/customers/:id", h.UpdateCustomer)
	v1.PATCH("/customers/:id/status", h.UpdateStatus)
	v1.GET("/customers/:id", h.GetCustomerByID)
	v1.GET("/customers", h.ListCustomers)
	v1.DELETE("/customers/:id", h.DeleteCustomer)

	srvHttp := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	go func() {
		logger.Info("server starting",
			slog.String("port", cfg.ServerPort),
			slog.String("version", version),
			slog.String("swagger", "http://localhost:"+cfg.ServerPort+"/swagger/index.html"),
		)
		if err := srvHttp.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("failed to start server", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	logger.Info("signal received, starting graceful shutdown", slog.String("signal", sig.String()))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srvHttp.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("server shutdown completed")
}
