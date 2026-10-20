package handlers

import "backend/internal/services"

// Register handles user registration requests.
type Register struct {
	service *services.AuthService
}

// NewRegisterHandler creates a new Register handler.
func NewRegisterHandler(service *services.AuthService) *Register {
	return &Register{
		service: service,
	}
}
