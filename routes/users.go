package routes

import (
	"events-rest-api/db"
	"events-rest-api/models"
	"events-rest-api/utils"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

func createUser(ginContext *gin.Context) {
	var user models.User
	var err error
	if err = ginContext.ShouldBindJSON(&user); err != nil {
		slog.Warn("signup bind failed", "err", err)
		ginContext.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if user, err = db.InsertUser(user); err != nil {
		writeDomainOrInternalError(ginContext, err, "signup failed", "email", user.Email)
		return
	}
	slog.Info("signup successful", "userId", user.ID, "email", user.Email)
	ginContext.JSON(http.StatusCreated, gin.H{"message": "Signup successful!"})
}

func login(ginContext *gin.Context) {
	var user models.User
	var userFromDb models.User
	var err error
	if err = ginContext.ShouldBindJSON(&user); err != nil {
		slog.Warn("login bind failed", "err", err)
		ginContext.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if userFromDb, err = db.GetUserByEmail(user.Email); err != nil {
		slog.Warn("login failed", "email", user.Email, "err", err)
		ginContext.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials!"})
		return
	}

	if ok, err := utils.CompareHash(user.Password, userFromDb.Password); err != nil || !ok {
		slog.Warn("login failed", "email", user.Email, "reason", "invalid password")
		ginContext.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials!"})
		return
	}
	token, err := utils.GenerateToken(userFromDb)
	if err != nil {
		slog.Error("login JWT generation failed", "userId", userFromDb.ID, "err", err)
		ginContext.JSON(http.StatusInternalServerError, gin.H{"error": "could not authenticate user: " + err.Error()})
		return
	}
	slog.Info("login successful", "userId", userFromDb.ID, "email", userFromDb.Email)
	ginContext.JSON(http.StatusOK, gin.H{"message": "Login successful!", "token": token})
}
