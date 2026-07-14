package models

type Registration struct {
	ID      int64 `json:"id"`
	EventID int64 `json:"event_id"`
	UserID  int64 `json:"user_id"`
}
