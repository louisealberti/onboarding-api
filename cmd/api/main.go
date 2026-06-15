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

	repo := repository.NewCustomerRepository(db)
	idempotencyRepo := repository.NewIdempotencyRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	auditSvc := service.NewAuditService(auditRepo)
	svc := service.NewCustomerService(repo).WithAudit(auditSvc)

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

	// Public routes
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/health", hh.Health)
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

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to decode PEM block for private key")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func loadPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
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