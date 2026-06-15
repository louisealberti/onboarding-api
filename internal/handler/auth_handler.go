package handler

import (
	"crypto/rsa"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	privateKey *rsa.PrivateKey
}

func NewAuthHandler(privateKey *rsa.PrivateKey) *AuthHandler {
	return &AuthHandler{privateKey: privateKey}
}

type tokenRequest struct {
	Subject string `json:"sub" binding:"required"`
	Role    string `json:"role" binding:"required,oneof=admin operator"`
}

// Token godoc
//
//	@Summary		Generate a JWT token
//	@Description	Issues a signed RS256 JWT for the given subject and role. For demonstration purposes only — in production this would validate credentials against an identity provider.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		tokenRequest		true	"Subject and role"
//	@Success		200		{object}	map[string]string	"JWT token"
//	@Failure		400		{object}	map[string]string
//	@Router			/auth/token [post]
func (h *AuthHandler) Token(c *gin.Context) {
	var req tokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"sub":  req.Subject,
		"role": req.Role,
		"iat":  now.Unix(),
		"exp":  now.Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(h.privateKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sign token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": signed})
}
