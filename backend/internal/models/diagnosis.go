package models

import "time"

// Diagnosis represents an agricultural diagnosis submitted by a user.
type Diagnosis struct {
	ID int

	UserID int

	Category    string
	Description string
	Result      string

	CreatedAt time.Time
}