package models

import "time"

type Diagnosis struct {
	ID          int
	FarmerID    int
	Category    string
	Description string
	Result      string
	CreatedAt   time.Time
}
