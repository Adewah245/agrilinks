package handlers

import "backend/internal/services"

// Register handles user registration requests.
type Register struct {
	service  *services.AuthService
	learning *services.LearningService
}

// NewRegisterHandler creates a new Register handler.
func NewRegisterHandler(service *services.AuthService, learning *services.LearningService) *Register {
	return &Register{
		service:  service,
		learning: learning,
	}
}
