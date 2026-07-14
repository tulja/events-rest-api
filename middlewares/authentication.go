package middlewares

import (
	"events-rest-api/utils"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func Authenticate(ginContext *gin.Context) {
	header := ginContext.Request.Header.Get("Authorization")
	if header == "" {
		slog.Warn("missing authorization header",
			"method", ginContext.Request.Method,
			"path", ginContext.Request.URL.Path,
		)
		ginContext.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	token := extractBearerToken(header)
	userId, err := utils.VerifyToken(token)
	if err != nil {
		slog.Warn("invalid or expired token",
			"method", ginContext.Request.Method,
			"path", ginContext.Request.URL.Path,
			"err", err,
		)
		ginContext.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	ginContext.Set("userId", userId)
	slog.Debug("authenticated", "userId", userId)

	ginContext.Next()
}

// extractBearerToken accepts "Bearer <jwt>" or a raw JWT.
func extractBearerToken(auth string) string {
	auth = strings.TrimSpace(auth)
	if len(auth) > 7 && strings.EqualFold(auth[:7], "Bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return auth
}
