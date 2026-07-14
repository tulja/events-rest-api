package routes

import (
	"events-rest-api/db"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func registerForEvent(ginContext *gin.Context) {
	userId := ginContext.GetInt64("userId")
	eventId, err := strconv.ParseInt(ginContext.Param("id"), 10, 64)
	if err != nil {
		slog.Warn("invalid event id", "err", err)
		ginContext.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err = db.RegisterForEvent(eventId, userId); err != nil {
		writeDomainOrInternalError(ginContext, err, "event registration failed", "eventId", eventId, "userId", userId)
		return
	}
	slog.Info("registered for event", "eventId", eventId, "userId", userId)
	ginContext.JSON(http.StatusOK, gin.H{"message": "Registered for event successfully!"})
}

func cancelEventRegistration(ginContext *gin.Context) {
	userId := ginContext.GetInt64("userId")
	eventId, err := strconv.ParseInt(ginContext.Param("id"), 10, 64)
	if err != nil {
		slog.Warn("invalid event id", "err", err)
		ginContext.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err = db.DeleteRegistration(eventId, userId); err != nil {
		writeDomainOrInternalError(ginContext, err, "cancel registration failed", "eventId", eventId, "userId", userId)
		return
	}
	slog.Info("cancelled event registration", "eventId", eventId, "userId", userId)
	ginContext.JSON(http.StatusOK, gin.H{"message": "Cancelled event registration successfully!"})
}

func getAllRegistrations(ginContext *gin.Context) {
	userId := ginContext.GetInt64("userId")
	events, err := db.GetAllRegisteredEventsForUser(userId)
	if err != nil {
		slog.Error("failed to list registrations", "userId", userId, "err", err)
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	slog.Debug("listed registrations", "userId", userId, "count", len(events))
	ginContext.JSON(http.StatusOK, events)
}
