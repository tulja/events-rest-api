package db

import (
	"events-rest-api/models"
)

func RegisterForEvent(eventId int64, userId int64) error {
	// Check if the event exists
	_, err := GetEventById(eventId)
	if err != nil {
		return err
	}

	// Check if the user has already registered for the event
	selectQuery := `SELECT id FROM registrations WHERE event_id = ? AND user_id = ?`
	rows, err := db.Query(selectQuery, eventId, userId)
	if err != nil {
		return err
	}
	defer rows.Close()
	if rows.Next() {
		return ErrAlreadyRegistered
	}

	insertQuery := `INSERT INTO registrations (event_id, user_id) VALUES (?,?)`
	_, err = db.Exec(insertQuery, eventId, userId)
	if err != nil {
		return err
	}
	return nil
}

func GetAllRegisteredEventsForUser(userId int64) ([]models.Event, error) {
	query := "SELECT e.id, e.name, e.description, e.location, e.date_time, e.userid FROM events e JOIN registrations r ON e.id = r.event_id WHERE r.user_id = ?"
	rows, err := db.Query(query, userId)
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

func DeleteRegistration(eventId int64, userId int64) error {
	// Verify event exists
	_, err := GetEventById(eventId)
	if err != nil {
		return err
	}

	query := "DELETE FROM registrations WHERE event_id = ? and user_id = ?"
	result, err := db.Exec(query, eventId, userId)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrRegistrationNotFound
	}
	return nil
}
