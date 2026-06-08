package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/louisealberti/onboarding-api/internal/config"
	"github.com/louisealberti/onboarding-api/internal/database"
	"github.com/louisealberti/onboarding-api/internal/handler"
	"github.com/louisealberti/onboarding-api/internal/middleware"
	"github.com/louisealberti/onboarding-api/internal/repository"
	"github.com/louisealberti/onboarding-api/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	db, err := database.NewPostgresConnection(cfg)
	if err != nil {
		log.Fatalf("critical failure connecting to database: %v", err)
	}
	defer db.Close()

	repo := repository.NewCustomerRepository(db)
	srv := service.NewCustomerService(repo)
	h := handler.NewCustomerHandler(srv)

	r := gin.Default()
	r.Use(middleware.RequestID())

	v1 := r.Group("/v1")
	v1.POST("/customers", h.CreateCustomer)
	v1.PUT("/customers/:id", h.UpdateCustomer)
	v1.GET("/customers/:id", h.GetCustomerByID)
	v1.GET("/customers", h.ListCustomers)
	v1.DELETE("/customers/:id", h.DeleteCustomer)

	// 1. Configure the HTTP server using Gin as the router
	srvHttp := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	// 2. Start the server in a separate goroutine to avoid blocking main
	go func() {
		log.Printf("server listening on port %s...", cfg.ServerPort)
		if err := srvHttp.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	// 3. Create a channel to listen for OS signals
	// SIGINT = Ctrl+C | SIGTERM = shutdown signal sent by Docker/Kubernetes
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Main goroutine blocks here until a signal is received
	sig := <-quit
	log.Printf("signal received (%v), starting graceful shutdown...", sig)

	// 4. Create a context with timeout to allow in-flight requests to complete
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 5. Stop accepting new requests and wait for active ones to finish
	if err := srvHttp.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown before completing pending requests: %v", err)
	}

	log.Println("server shutdown completed.")
}
