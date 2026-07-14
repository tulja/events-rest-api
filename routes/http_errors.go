package routes

import (
	"errors"
	"events-rest-api/db"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// writeDomainOrInternalError maps known domain errors to 4xx statuses.
// Unknown errors become 500.
func writeDomainOrInternalError(c *gin.Context, err error, logMsg string, attrs ...any) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, db.ErrEventNotFound),
		errors.Is(err, db.ErrUserNotFound),
		errors.Is(err, db.ErrRegistrationNotFound):
		status = http.StatusNotFound
		slog.Warn(logMsg, append(attrs, "err", err, "status", status)...)
	case errors.Is(err, db.ErrNotAuthorized),
		errors.Is(err, db.ErrDeleteNotAuthorized):
		status = http.StatusForbidden
		slog.Warn(logMsg, append(attrs, "err", err, "status", status)...)
	case errors.Is(err, db.ErrAlreadyRegistered),
		errors.Is(err, db.ErrDuplicateEmail):
		status = http.StatusConflict
		slog.Warn(logMsg, append(attrs, "err", err, "status", status)...)
	default:
		slog.Error(logMsg, append(attrs, "err", err, "status", status)...)
	}
	c.JSON(status, gin.H{"error": err.Error()})
}
