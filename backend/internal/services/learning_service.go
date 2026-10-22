package services

import (
	"backend/internal/models"
	"backend/internal/repository"
	"context"
	"errors"
	"github.com/google/uuid"
	"strings"
)

type LearningService struct {
	repo     *repository.LearningRepository
	provider LearningProvider
}

func NewLearningService(repo *repository.LearningRepository) *LearningService {
	return &LearningService{repo: repo}
}
func (s *LearningService) SetProvider(provider LearningProvider) { s.provider = provider }
func (s *LearningService) AIAvailable() bool                     { return s.provider != nil }
func (s *LearningService) GetFarmingTypes() ([]models.FarmingType, error) {
	return s.repo.ListFarmingTypes()
}
func (s *LearningService) GetUserFarmingTypes(userID int) ([]models.UserFarmingType, error) {
	return s.repo.GetUserFarmingTypes(userID)
}
func (s *LearningService) ListJourneys(userID int) ([]models.LearningJourney, error) {
	return s.repo.ListJourneys(userID)
}
func (s *LearningService) GetJourneyForUser(userID, id int) (*models.LearningJourney, error) {
	return s.repo.JourneyForUser(userID, id)
}

func (s *LearningService) GetActiveLearningJourney(userID int) (*models.LearningJourney, error) {
	journeys, err := s.repo.ListJourneys(userID)
	if err != nil {
		return nil, err
	}
	for i := range journeys {
		if journeys[i].Status == "active" || journeys[i].Status == "paused" {
			return &journeys[i], nil
		}
	}
	return nil, nil
}

func (s *LearningService) StartLearningJourney(userID, farmingID int, experience, goal string) (*models.LearningJourney, error) {
	if userID <= 0 || farmingID <= 0 {
		return nil, errors.New("choose a farming activity")
	}
	experience = strings.TrimSpace(experience)
	goal = strings.TrimSpace(goal)
	switch experience {
	case "beginner", "some_experience", "experienced":
	default:
		return nil, errors.New("choose your experience level")
	}
	if len(goal) > 1000 {
		return nil, errors.New("keep your learning goal under 1,000 characters")
	}
	return s.repo.StartJourney(userID, farmingID, experience, goal)
}

func (s *LearningService) GetLearningMessages(userID, journeyID int) ([]models.LearningMessage, error) {
	if _, err := s.repo.JourneyForUser(userID, journeyID); err != nil {
		return nil, err
	}
	return s.repo.GetLearningMessages(journeyID)
}

func (s *LearningService) GetLearningCheckpoints(userID, journeyID int) ([]models.LearningCheckpoint, error) {
	if _, err := s.repo.JourneyForUser(userID, journeyID); err != nil {
		return nil, err
	}
	return s.repo.GetLearningCheckpoints(journeyID)
}

func (s *LearningService) Ask(ctx context.Context, userID, journeyID int, question string) ([]models.LearningMessage, error) {
	journey, err := s.repo.JourneyForUser(userID, journeyID)
	if err != nil {
		return nil, err
	}
	if journey.Status != "active" {
		return nil, errors.New("resume this journey before asking a question")
	}
	question = strings.TrimSpace(question)
	if question == "" || len(question) > 10000 {
		return nil, errors.New("enter a question under 10,000 characters")
	}
	if s.provider == nil {
		return nil, ErrLearningUnavailable
	}
	history, err := s.repo.GetLearningMessages(journeyID)
	if err != nil {
		return nil, err
	}
	checkpoints, err := s.repo.GetLearningCheckpoints(journeyID)
	if err != nil {
		return nil, err
	}
	lastID := 0
	if len(history) > 0 {
		lastID = history[len(history)-1].ID
	}
	history = append(history, models.LearningMessage{SenderType: "user", Content: question})
	answer, err := s.provider.Learn(ctx, LearningContext{FarmingType: journey.FarmingType.Name, Goal: journey.Goal, Experience: journey.ExperienceLevel,
		Stage: journey.CurrentStage, Messages: history, Checkpoints: checkpoints})
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(answer) == "" {
		return nil, ErrLearningUnavailable
	}
	return s.repo.SaveExchange(ctx, userID, journeyID, lastID, question, answer)
}

func (s *LearningService) CreateLearningCheckpoint(userID, journeyID int, key, title, description string) (*models.LearningCheckpoint, error) {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	if title == "" || len(title) > 255 || len(description) > 2000 {
		return nil, errors.New("add a short title and description for your step")
	}
	if key == "" {
		key = uuid.NewString()
	}
	if len(key) > 100 {
		return nil, errors.New("invalid checklist step")
	}
	item := &models.LearningCheckpoint{JourneyID: journeyID, CheckpointKey: key, Title: title, Description: description, Status: "pending"}
	if err := s.repo.CreateCheckpoint(userID, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *LearningService) UpdateLearningCheckpoint(userID int, item *models.LearningCheckpoint) error {
	if item == nil || item.ID <= 0 || item.JourneyID <= 0 {
		return errors.New("choose a checklist step")
	}
	if item.Progress < 0 || item.Progress > 100 || len(item.Notes) > 2000 {
		return errors.New("enter progress from 0 to 100 and a short note")
	}
	switch item.Status {
	case "completed":
		item.Progress = 100
	case "pending", "not_applicable":
		item.Progress = 0
	case "in_progress":
		if item.Progress == 100 {
			item.Status = "completed"
		}
	default:
		return errors.New("choose the step's status")
	}
	return s.repo.UpdateCheckpoint(userID, item)
}

func (s *LearningService) ChangeJourneyStatus(userID, journeyID int, status string) error {
	switch status {
	case "active", "paused", "abandoned", "completed":
	default:
		return errors.New("choose a learning status")
	}
	return s.repo.ChangeJourneyStatus(userID, journeyID, status)
}
