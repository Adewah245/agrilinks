package handlers

import "backend/internal/services"

type Register struct {
	service *services.AuthService
}

func NewRegisterHandler(service *services.AuthService) *Register {

	return &Register{
		service: service,
	}

}
