package main

import (
	"events-rest-api/db"
	"events-rest-api/routes"
	"events-rest-api/utils"
	"log/slog"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func main() {
	setupLogger()

	db.InitDB()
	slog.Info("database ready")

	if err := utils.EnsureJWTSigningKey(); err != nil {
		slog.Error("JWT signing key required at startup", "err", err)
		os.Exit(1)
	}

	server := gin.Default()
	// CORS — must be first, before any routes
	server.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	routes.RegisterRoutes(server)

	// Vercel and other platforms inject PORT; default to 8080 for local dev.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	slog.Info("starting HTTP server", "addr", addr)
	if err := server.Run(addr); err != nil {
		slog.Error("HTTP server failed", "err", err)
		os.Exit(1)
	}
}

func setupLogger() {
	level := slog.LevelInfo
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})))
	slog.Info("logger initialized", "level", level.String())
}
