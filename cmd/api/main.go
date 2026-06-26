package main

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/louisealberti/onboarding-api/docs"
	"github.com/louisealberti/onboarding-api/internal/config"
	"github.com/louisealberti/onboarding-api/internal/database"
	"github.com/louisealberti/onboarding-api/internal/handler"
	"github.com/louisealberti/onboarding-api/internal/metrics"
	"github.com/louisealberti/onboarding-api/internal/middleware"
	"github.com/louisealberti/onboarding-api/internal/repository"
	"github.com/louisealberti/onboarding-api/internal/service"
	"github.com/louisealberti/onboarding-api/internal/webhook"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

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

	privateKey, err := loadPrivateKey(cfg.JWTPrivateKey)
	if err != nil {
		logger.Error("failed to load JWT private key", slog.Any("error", err))
		os.Exit(1)
	}

	publicKey, err := loadPublicKey(cfg.JWTPublicKey)
	if err != nil {
		logger.Error("failed to load JWT public key", slog.Any("error", err))
		os.Exit(1)
	}

	db, err := database.NewPostgresConnection(cfg)
	if err != nil {
		logger.Error("critical failure connecting to database", slog.Any("error", err))
		os.Exit(1)
	}
	defer db.Close()

	appCtx, stopAppCtx := context.WithCancel(context.Background())
	defer stopAppCtx()
	go metrics.StartDBStatsCollector(appCtx, db)

	repo := repository.NewCustomerRepository(db)
	idempotencyRepo := repository.NewIdempotencyRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	auditSvc := service.NewAuditService(auditRepo)
	svc := service.NewCustomerService(repo).WithAudit(auditSvc)

	if cfg.WebhookEnabled() {
		notifier := webhook.NewNotifier(cfg.WebhookURL, cfg.WebhookSecret, logger)
		svc = svc.WithWebhook(notifier)
		logger.Info("webhook notifications enabled", slog.String("url", cfg.WebhookURL))
	}

	h := handler.NewCustomerHandler(svc)
	ah := handler.NewAuditHandler(auditSvc)
	hh := handler.NewHealthHandler(db, handler.BuildInfo{Version: version, BuildTime: buildTime})
	authH := handler.NewAuthHandler(privateKey)

	corsOrigins := os.Getenv("CORS_ORIGINS")
	if corsOrigins == "" {
		corsOrigins = "*"
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS([]string{corsOrigins}))
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(logger))
	r.Use(middleware.Metrics())

	// Public routes
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/health", hh.Health)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	r.POST("/auth/token", authH.Token)

	// Protected routes
	authMiddleware := middleware.Auth(publicKey)

	v1 := r.Group("/v1")
	v1.Use(authMiddleware)
	// When /v2 is introduced, uncomment to signal deprecation:
	// v1.Use(middleware.Deprecated("2027-01-01", "https://api.example.com/v2"))

	// admin only
	v1.POST("/customers", middleware.RequireRole("admin"), middleware.Idempotency(idempotencyRepo), h.CreateCustomer)
	v1.PUT("/customers/:id", middleware.RequireRole("admin"), h.UpdateCustomer)
	v1.DELETE("/customers/:id", middleware.RequireRole("admin"), h.DeleteCustomer)

	// admin + operator
	v1.GET("/customers/:id", middleware.RequireRole("admin", "operator"), h.GetCustomerByID)
	v1.GET("/customers", middleware.RequireRole("admin", "operator"), h.ListCustomers)
	v1.PATCH("/customers/:id/status", middleware.RequireRole("admin", "operator"), h.UpdateStatus)
	v1.GET("/customers/:id/audit", middleware.RequireRole("admin", "operator"), ah.GetAuditLog)

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

// loadPrivateKey loads an RSA private key. If pathOrPEM looks like inline
// PEM content (starts with "-----BEGIN"), it is parsed directly — this is
// how the key arrives in ECS, injected from SSM Parameter Store as an
// environment variable, since Fargate tasks have no persistent filesystem
// to mount a keys/ directory into. Otherwise pathOrPEM is treated as a file
// path, which is how local development and Docker Compose provide it.
func loadPrivateKey(pathOrPEM string) (*rsa.PrivateKey, error) {
	data, err := pemContentOrFile(pathOrPEM)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to decode PEM block for private key")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

// loadPublicKey loads an RSA public key. See loadPrivateKey for the
// inline-PEM-vs-file-path resolution rule.
func loadPublicKey(pathOrPEM string) (*rsa.PublicKey, error) {
	data, err := pemContentOrFile(pathOrPEM)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to decode PEM block for public key")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}
	return rsaPub, nil
}

// pemContentOrFile resolves a config value that is either raw PEM content
// or a file path. ECS injects keys as env vars (their content, not a path);
// local/Docker Compose pass a path on disk (keys/private.pem). Distinguish
// by checking for the PEM header — a real file path never starts with it.
func pemContentOrFile(value string) ([]byte, error) {
	if strings.HasPrefix(strings.TrimSpace(value), "-----BEGIN") {
		return []byte(value), nil
	}
	return os.ReadFile(value)
}
