package services

import (
	"context"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"backend/internal/models"
)

var ErrInvalidLearningChat = errors.New("choose a farming category and enter a question of up to 1,200 characters")

type LearningChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type LearningChatRequest struct {
	FarmingType string                `json:"farmingType"`
	Messages    []LearningChatMessage `json:"messages"`
}

func LearningCategories() []string {
	return []string{"Crop farming", "Poultry farming", "Livestock farming", "Fish farming", "Vegetable farming", "Soil management", "Starting a farm"}
}

// PublicChat keeps the legacy learning chat independent of registration and
// saved journeys. The caller supplies recent conversation, never system roles.
func (s *LearningService) PublicChat(ctx context.Context, request LearningChatRequest) (string, error) {
	category := strings.TrimSpace(request.FarmingType)
	if category == "" || utf8.RuneCountInString(category) > 80 || strings.IndexFunc(category, unicode.IsControl) >= 0 || len(request.Messages) == 0 || len(request.Messages) > 12 {
		return "", ErrInvalidLearningChat
	}
	history := make([]models.LearningMessage, 0, len(request.Messages))
	for _, message := range request.Messages {
		content := strings.TrimSpace(message.Content)
		if (message.Role != "user" && message.Role != "assistant") || content == "" || utf8.RuneCountInString(content) > 1200 {
			return "", ErrInvalidLearningChat
		}
		sender := "user"
		if message.Role == "assistant" {
			sender = "ai"
		}
		history = append(history, models.LearningMessage{SenderType: sender, Content: content})
	}
	if request.Messages[len(request.Messages)-1].Role != "user" {
		return "", ErrInvalidLearningChat
	}
	if s == nil || s.provider == nil {
		return "", ErrLearningUnavailable
	}
	answer, err := s.provider.Learn(ctx, LearningContext{FarmingType: category, Stage: "learning", Messages: history})
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", ErrLearningUnavailable
	}
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return "", ErrLearningUnavailable
	}
	return answer, nil
}
