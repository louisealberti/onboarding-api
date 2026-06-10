package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/louisealberti/onboarding-api/internal/domain"
	"github.com/louisealberti/onboarding-api/internal/service"
)

// Gin Routes

type CustomerHandler struct {
	srv *service.CustomerService
}

func NewCustomerHandler(srv *service.CustomerService) *CustomerHandler {
	return &CustomerHandler{srv: srv}
}

func (h *CustomerHandler) CreateCustomer(c *gin.Context) {
	var customer domain.Customer
	if err := c.ShouldBindJSON(&customer); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}
	if err := h.srv.CreateCustomer(c.Request.Context(), &customer); err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, customer)
}

func (h *CustomerHandler) UpdateCustomer(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid UUID format"})
		return
	}
	var customer domain.Customer
	if err := c.ShouldBindJSON(&customer); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "details": err.Error()})
		return
	}
	customer.ID = id
	if err := h.srv.UpdateCustomer(c.Request.Context(), &customer); err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "customer updated successfully"})
}

func (h *CustomerHandler) GetCustomerByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid UUID format"})
		return
	}
	customer, err := h.srv.SearchCustomer(c.Request.Context(), id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, customer)
}

func (h *CustomerHandler) SearchByTaxID(c *gin.Context) {
	taxID := c.Query("taxId")
	if taxID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "taxId query param is required"})
		return
	}

	customer, err := h.srv.SearchByTaxID(c.Request.Context(), taxID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, customer)
}

func (h *CustomerHandler) DeleteCustomer(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid UUID format"})
		return
	}
	if err := h.srv.DeleteCustomer(c.Request.Context(), id); err != nil {
		handleServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *CustomerHandler) UpdateStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer ID"})
		return
	}

	var body struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status is required"})
		return
	}

	if err := h.srv.UpdateStatus(c.Request.Context(), id, body.Status); err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "status updated successfully"})
}
