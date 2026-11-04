package models

import "time"

type WithdrawalAuthorization struct{ TokenHash, SessionHash string }

type WithdrawalEvidence struct {
	WithdrawalID                   int64
	UserID                         int
	PhotoID                        int64
	Latitude, Longitude, AccuracyM float64
	CapturedAt, ReceivedAt         time.Time
}

type WithdrawalReviewPage struct {
	Withdrawal          *Withdrawal
	Evidence            *WithdrawalEvidence
	RegistrationPhotoID int64
	CSRF, Error         string
}
