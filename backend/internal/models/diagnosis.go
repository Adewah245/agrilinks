package models

import "time"

type Diagnosis struct {
	ID        int
	FarmerID  int
	ImageName string
	Result    string
	CreatedAt time.Time
}
