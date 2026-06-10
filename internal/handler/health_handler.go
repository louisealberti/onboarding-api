package handler

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

// BuildInfo holds version metadata injected at build time via ldflags.
type BuildInfo struct {
	Version   string
	BuildTime string
}

// HealthHandler handles the GET /health endpoint.
type HealthHandler struct {
	db    *sql.DB
	build BuildInfo
}

func NewHealthHandler(db *sql.DB, build BuildInfo) *HealthHandler {
	return &HealthHandler{db: db, build: build}
}

func (h *HealthHandler) Health(c *gin.Context) {
	dbStatus := "healthy"
	httpStatus := http.StatusOK

	if err := h.db.PingContext(c.Request.Context()); err != nil {
		dbStatus = "unhealthy"
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, gin.H{
		"status":    overallStatus(dbStatus),
		"database":  dbStatus,
		"version":   h.build.Version,
		"buildTime": h.build.BuildTime,
	})
}

func overallStatus(dbStatus string) string {
	if dbStatus == "healthy" {
		return "healthy"
	}
	return "unhealthy"
}