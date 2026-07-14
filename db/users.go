package db

import (
	"database/sql"
	"errors"
	"events-rest-api/models"
	"events-rest-api/utils"
	"strings"

	"github.com/mattn/go-sqlite3"
)

var (
	ErrUserNotFound   = errors.New("User not found")
	ErrDuplicateEmail = errors.New("Email already exists")
)

func InsertUser(user models.User) (models.User, error) {
	insertQuery := `INSERT INTO users (email, password) VALUES (?,?)`

	hashedPassword, err := utils.GenerateHash(user.Password)
	if err != nil {
		return models.User{}, err
	}

	result, err := db.Exec(insertQuery, user.Email, hashedPassword)
	if err != nil {
		if isUniqueConstraint(err) {
			return models.User{}, ErrDuplicateEmail
		}
		return models.User{}, err
	}
	lastId, err := result.LastInsertId()
	if err != nil {
		return models.User{}, err
	}
	user.ID = lastId
	return user, nil
}

func GetUserByEmail(email string) (models.User, error) {
	query := "SELECT id, email, password FROM users WHERE email = ?"
	row := db.QueryRow(query, email)
	var user models.User
	if err := row.Scan(&user.ID, &user.Email, &user.Password); err != nil {
		return models.User{}, mapNoRows(err, ErrUserNotFound)
	}
	return user, nil
}

func isUniqueConstraint(err error) bool {
	var se sqlite3.Error
	if errors.As(err, &se) {
		return se.ExtendedCode == sqlite3.ErrConstraintUnique || se.Code == sqlite3.ErrConstraint
	}
	// Fallback for wrapped/driver message variants
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func mapNoRows(err error, notFound error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return notFound
	}
	return err
}
