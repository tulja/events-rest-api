package db

import (
	"errors"
	"events-rest-api/models"
)

var (
	ErrEventNotFound        = errors.New("Event not found")
	ErrNotAuthorized        = errors.New("You are not authorized to update this event")
	ErrDeleteNotAuthorized  = errors.New("You are not authorized to delete this event")
	ErrAlreadyRegistered    = errors.New("You have already registered for this event")
	ErrRegistrationNotFound = errors.New("Registration not found")
)

const eventColumns = "id, name, description, location, date_time, userid"

func InsertEvent(event *models.Event) error {
	insertQuery := `INSERT INTO events (name, description, location, date_time, userid) VALUES (?,?,?,?,?)`
	result, err := db.Exec(insertQuery, event.Name, event.Description, event.Location, event.DateTime, event.UserID)
	if err != nil {
		return err
	}
	lastId, _ := result.LastInsertId()
	event.ID = lastId
	return nil
}

func GetAllEvents() ([]models.Event, error) {
	rows, err := db.Query("SELECT " + eventColumns + " FROM events")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []models.Event
	for rows.Next() {
		var event models.Event
		if err := rows.Scan(&event.ID, &event.Name, &event.Description, &event.Location, &event.DateTime, &event.UserID); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

func GetEventById(id int64) (models.Event, error) {
	query := "SELECT " + eventColumns + " FROM events WHERE id = ?"
	row := db.QueryRow(query, id)
	var event models.Event
	if err := row.Scan(&event.ID, &event.Name, &event.Description, &event.Location, &event.DateTime, &event.UserID); err != nil {
		return models.Event{}, mapNoRows(err, ErrEventNotFound)
	}
	return event, nil
}

func UpdateEvent(id int64, event models.Event, userIdFromToken int64) (models.Event, error) {
	eventFromDb, err := GetEventById(id)
	if err != nil {
		return models.Event{}, err
	}

	if eventFromDb.UserID != userIdFromToken {
		return models.Event{}, ErrNotAuthorized
	}

	query := "UPDATE events SET name = ?, description = ?, location = ?, date_time = ?, userid = ? WHERE id = ?"
	_, err = db.Exec(query, event.Name, event.Description, event.Location, event.DateTime, userIdFromToken, id)
	if err != nil {
		return models.Event{}, err
	}
	event.ID = id
	event.UserID = userIdFromToken
	return event, nil
}

func DeleteEvent(id int64, userIdFromToken int64) error {
	eventFromDb, err := GetEventById(id)
	if err != nil {
		return err
	}

	if eventFromDb.UserID != userIdFromToken {
		return ErrDeleteNotAuthorized
	}

	query := "DELETE FROM events WHERE id = ?"
	_, err = db.Exec(query, id)
	if err != nil {
		return err
	}
	return nil
}
