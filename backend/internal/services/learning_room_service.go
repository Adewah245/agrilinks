package services

import (
	"backend/internal/models"
	"context"
	"errors"
	"strings"
)

func (s *LearningService) ListRooms(ctx context.Context, userID, farmingID int) ([]models.LearningRoom, error) {
	if userID <= 0 || farmingID < 0 {
		return nil, errors.New("choose a farming activity")
	}
	return s.repo.ListRooms(ctx, userID, farmingID)
}

func (s *LearningService) CreateRoom(ctx context.Context, userID, farmingID int, title, description string) (int, error) {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	if userID <= 0 || farmingID <= 0 || title == "" || len(title) > 100 || len(description) > 2000 {
		return 0, errors.New("enter a farming activity, a short room name, and a description")
	}
	return s.repo.CreateRoom(ctx, userID, farmingID, title, description)
}

func (s *LearningService) JoinRoom(ctx context.Context, userID, roomID int) error {
	if userID <= 0 || roomID <= 0 {
		return errors.New("choose a learning room")
	}
	return s.repo.JoinRoom(ctx, userID, roomID)
}
