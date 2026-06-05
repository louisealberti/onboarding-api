package main

import (
	"log"

	"github.com/louisealberti/onboarding-api/internal/config"
	"github.com/louisealberti/onboarding-api/internal/database"
	"github.com/louisealberti/onboarding-api/internal/handler"
	"github.com/louisealberti/onboarding-api/internal/repository"
	"github.com/louisealberti/onboarding-api/internal/service"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("erro de configuração: %v", err)
	}

	db, err := database.NewPostgresConnection(cfg)
	if err != nil {
		log.Fatalf("falha crítica ao iniciar o banco: %v", err)
	}
	defer db.Close()

	repo := repository.NewCustomerRepository(db)
	srv := service.NewCustomerService(repo)
	h := handler.NewCustomerHandler(srv)

	r := gin.Default()

	v1 := r.Group("/v1")
	v1.POST("/customers", h.CreateCustomer)
	v1.PUT("/customers/:id", h.UpdateCustomer)
	v1.GET("/customers/:id", h.GetCustomerByID)
	v1.DELETE("/customers/:id", h.DeleteCustomer)

	log.Printf("server listening on port %s...", cfg.ServerPort)
	r.Run(":" + cfg.ServerPort)
}
