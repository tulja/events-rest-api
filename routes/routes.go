package routes

import (
	"events-rest-api/db"
	"events-rest-api/middlewares"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(server *gin.Engine) {
	server.GET("/health", health)

	server.GET("/events", getEvents)

	// Static path before /events/:id so "registrations" is never captured as an id.
	authenticated := server.Group("/")
	authenticated.Use(middlewares.Authenticate)
	authenticated.GET("/events/registrations", getAllRegistrations)
	authenticated.POST("/events", createEvent)
	authenticated.PUT("/events/:id", updateEvent)
	authenticated.DELETE("/events/:id", deleteEvent)
	authenticated.POST("/events/:id/register", registerForEvent)
	authenticated.DELETE("/events/:id/register", cancelEventRegistration)

	server.GET("/events/:id", getEventById)

	server.POST("/signup", createUser)
	server.POST("/login", login)
}

func health(c *gin.Context) {
	if err := db.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
