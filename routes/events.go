package routes

import (
	"events-rest-api/db"
	"events-rest-api/models"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func createEvent(ginContext *gin.Context) {
	var event models.Event
	var err error
	if err = ginContext.ShouldBindJSON(&event); err != nil {
		slog.Warn("create event bind failed", "err", err)
		ginContext.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userId := ginContext.GetInt64("userId")
	event.UserID = userId
	if err = db.InsertEvent(&event); err != nil {
		slog.Error("failed to create event", "userId", userId, "err", err)
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	slog.Info("event created", "eventId", event.ID, "userId", userId)
	ginContext.JSON(http.StatusCreated, event)
}

func getEvents(ginContext *gin.Context) {
	events, err := db.GetAllEvents()
	if err != nil {
		slog.Error("failed to list events", "err", err)
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	slog.Debug("listed events", "count", len(events))
	ginContext.JSON(http.StatusOK, events)
}

func getEventById(ginContext *gin.Context) {
	id, err := strconv.ParseInt(ginContext.Param("id"), 10, 64)
	if err != nil {
		slog.Warn("invalid event id", "err", err)
		ginContext.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var event models.Event
	if event, err = db.GetEventById(id); err != nil {
		writeDomainOrInternalError(ginContext, err, "failed to get event", "eventId", id)
		return
	}
	ginContext.JSON(http.StatusOK, event)
}

func updateEvent(ginContext *gin.Context) {
	id, err := strconv.ParseInt(ginContext.Param("id"), 10, 64)
	if err != nil {
		slog.Warn("invalid event id", "err", err)
		ginContext.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var event models.Event
	if err = ginContext.ShouldBindJSON(&event); err != nil {
		slog.Warn("update event bind failed", "eventId", id, "err", err)
		ginContext.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userIdFromToken := ginContext.GetInt64("userId")

	if event, err = db.UpdateEvent(id, event, userIdFromToken); err != nil {
		writeDomainOrInternalError(ginContext, err, "event update failed", "eventId", id, "userId", userIdFromToken)
		return
	}
	slog.Info("event updated", "eventId", event.ID, "userId", userIdFromToken)
	ginContext.JSON(http.StatusOK, event)
}

func deleteEvent(ginContext *gin.Context) {
	id, err := strconv.ParseInt(ginContext.Param("id"), 10, 64)
	if err != nil {
		slog.Warn("invalid event id", "err", err)
		ginContext.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userIdFromToken := ginContext.GetInt64("userId")
	if err = db.DeleteEvent(id, userIdFromToken); err != nil {
		writeDomainOrInternalError(ginContext, err, "event delete failed", "eventId", id, "userId", userIdFromToken)
		return
	}
	slog.Info("event deleted", "eventId", id, "userId", userIdFromToken)
	ginContext.JSON(http.StatusOK, gin.H{"message": "Event deleted successfully!"})
}
