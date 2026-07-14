package db

import (
	"database/sql"
	"errors"
	"events-rest-api/models"
	"events-rest-api/utils"
	"strings"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
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

// isUniqueConstraint detects SQLite unique-constraint failures using modernc.org/sqlite
// (pure Go; works without CGO / on Vercel).
func isUniqueConstraint(err error) bool {
	if err == nil {
		return false
	}
	var se *sqlite.Error
	if errors.As(err, &se) {
		code := se.Code()
		// SQLITE_CONSTRAINT = 19; extended unique codes keep low byte as CONSTRAINT.
		return code == sqlite3.SQLITE_CONSTRAINT || (code&0xff) == sqlite3.SQLITE_CONSTRAINT
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint failed")
}

func mapNoRows(err error, notFound error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return notFound
	}
	return err
}
