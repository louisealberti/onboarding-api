package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/louisealberti/onboarding-api/internal/domain"
	"github.com/louisealberti/onboarding-api/internal/service"
)

// AuditHandler handles audit log endpoints.
type AuditHandler struct {
	audit *service.AuditService
}

func NewAuditHandler(audit *service.AuditService) *AuditHandler {
	return &AuditHandler{audit: audit}
}

// GetAuditLog godoc
//
//	@Summary		Get audit log for a customer
//	@Description	Returns all recorded changes for a customer, newest first.
//	@Tags			customers
//	@Produce		json
//	@Param			id	path		string	true	"Customer UUID"
//	@Success		200	{array}		domain.AuditLog
//	@Failure		400	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Router			/v1/customers/{id}/audit [get]
func (h *AuditHandler) GetAuditLog(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid UUID format"})
		return
	}

	logs, err := h.audit.ListByCustomer(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch audit log"})
		return
	}

	if logs == nil {
		logs = []domain.AuditLog{}
	}

	c.JSON(http.StatusOK, logs)
}