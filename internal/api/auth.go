package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"locator/internal/service"
	pkgjwt "locator/pkg/jwt"
)

const authUserContextKey = "auth_user"

type authService interface {
	Enabled() bool
	Login(ctx context.Context, input service.LoginInput) (*service.LoginResult, error)
	ParseToken(token string) (*pkgjwt.Claims, error)
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type authUser struct {
	UserID   uint64 `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

func authEnabled(authSvc authService) bool {
	return authSvc != nil && authSvc.Enabled()
}

func requireAuth(authSvc authService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !authEnabled(authSvc) {
			c.Next()
			return
		}

		token := extractBearerToken(c.GetHeader("Authorization"))
		if token == "" {
			fail(c, http.StatusUnauthorized, "missing bearer token")
			c.Abort()
			return
		}

		claims, err := authSvc.ParseToken(token)
		if err != nil {
			switch {
			case errors.Is(err, pkgjwt.ErrExpiredToken):
				fail(c, http.StatusUnauthorized, "token expired")
			default:
				fail(c, http.StatusUnauthorized, "invalid token")
			}
			c.Abort()
			return
		}

		c.Set(authUserContextKey, authUser{
			UserID:   claims.UserID,
			Username: claims.Username,
			Role:     claims.Role,
		})
		c.Next()
	}
}

func requireRole(authSvc authService, roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		trimmed := strings.TrimSpace(role)
		if trimmed == "" {
			continue
		}
		allowed[trimmed] = struct{}{}
	}

	return func(c *gin.Context) {
		if !authEnabled(authSvc) {
			c.Next()
			return
		}

		user, ok := currentUser(c)
		if !ok {
			fail(c, http.StatusUnauthorized, "authentication required")
			c.Abort()
			return
		}

		if _, exists := allowed[user.Role]; !exists {
			fail(c, http.StatusForbidden, "insufficient permissions")
			c.Abort()
			return
		}

		c.Next()
	}
}

func currentUser(c *gin.Context) (authUser, bool) {
	value, ok := c.Get(authUserContextKey)
	if !ok {
		return authUser{}, false
	}

	user, ok := value.(authUser)
	return user, ok
}

func authorizeWebSocket(c *gin.Context, authSvc authService) bool {
	if !authEnabled(authSvc) {
		return true
	}

	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		token = extractBearerToken(c.GetHeader("Authorization"))
	}
	if token == "" {
		fail(c, http.StatusUnauthorized, "missing websocket token")
		return false
	}

	claims, err := authSvc.ParseToken(token)
	if err != nil {
		switch {
		case errors.Is(err, pkgjwt.ErrExpiredToken):
			fail(c, http.StatusUnauthorized, "token expired")
		default:
			fail(c, http.StatusUnauthorized, "invalid token")
		}
		return false
	}

	c.Set(authUserContextKey, authUser{
		UserID:   claims.UserID,
		Username: claims.Username,
		Role:     claims.Role,
	})

	return true
}

func extractBearerToken(header string) string {
	trimmed := strings.TrimSpace(header)
	if trimmed == "" {
		return ""
	}

	parts := strings.SplitN(trimmed, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return strings.TrimSpace(parts[1])
}
