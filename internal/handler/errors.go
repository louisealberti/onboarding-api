package handler

import (
	"errors"
	"net/http"

	"github.com/louisealberti/onboarding-api/internal/service"
	"github.com/gin-gonic/gin"
)

func handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrCustomerNotRegistered):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})

	case errors.Is(err, service.ErrDuplicatedEmail):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})

	case errors.Is(err, service.ErrDuplicatedTaxID):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})

	case errors.Is(err, service.ErrMissingEmail),
		errors.Is(err, service.ErrMissingTaxID),
		errors.Is(err, service.ErrMissingCountryCode):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

	case errors.Is(err, service.ErrCustomerIsBlocked):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})

	default:
		// Unexpected Error — does not expose internaldetails
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
