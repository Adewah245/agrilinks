package services

import (
	"errors"
	"log"
	"strings"
	"time"

	"backend/internal/models"
	"backend/internal/repository"
)

// AIProvider defines the contract that an AI provider must implement
// to analyze agricultural requests.
type AIProvider interface {
	Analyze(request models.AIRequest) (*models.DiagnosisResult, error)
}

// AIService coordinates agricultural diagnosis requests between
// the application, AI provider, and diagnosis repository.
type AIService struct {
	repo     *repository.DiagnosisRepository
	provider AIProvider
}

// NewAIService creates a new AIService with the provided diagnosis
// repository and AI provider.
func NewAIService(
	repo *repository.DiagnosisRepository,
	provider AIProvider,
) *AIService {
	return &AIService{
		repo:     repo,
		provider: provider,
	}
}

// Diagnose sends a user's agricultural information to the configured
// AI provider and stores the resulting diagnosis in the database.
func (s *AIService) Diagnose(
	userID int,
	request models.AIRequest,
) (*models.Diagnosis, error) {

	if strings.TrimSpace(request.Category) == "" &&
		strings.TrimSpace(request.Description) == "" &&
		len(request.Image) == 0 &&
		len(request.Audio) == 0 &&
		len(request.Video) == 0 {
		return nil, errors.New("no information was provided")
	}

	aiStart := time.Now()
	log.Println("⏱️ AI analysis started")

	aiResult, err := s.provider.Analyze(request)

	log.Printf("⏱️ AI analysis took: %v", time.Since(aiStart))

	if err != nil {
		return nil, err
	}

	if aiResult == nil {
		return nil, errors.New("AI provider returned no result")
	}

	diagnosis := &models.Diagnosis{
		UserID:   userID,
		Category: stringPointer(request.Category),
		Symptoms: stringPointer(request.Description),
		Result:   stringPointer(aiResult.Result),
	}

	dbStart := time.Now()
	log.Println("⏱️ Saving diagnosis to database")

	if err := s.repo.Create(diagnosis); err != nil {
		return nil, err
	}

	log.Printf("⏱️ Database save took: %v", time.Since(dbStart))

	return diagnosis, nil
}

func stringPointer(value string) *string { return &value }

// DeleteDiagnosis deletes a diagnosis belonging to the specified user.
func (s *AIService) DeleteDiagnosis(
	userID int,
	diagnosisID int,
) error {
	return s.repo.DeleteForUser(diagnosisID, userID)
}

// History returns the diagnosis history belonging to a specific user.
func (s *AIService) History(
	userID int,
) ([]models.Diagnosis, error) {
	return s.repo.ListByUser(userID)
}

// CountThisMonth returns the number of diagnoses created by a user
// during the current calendar month.
func (s *AIService) CountThisMonth(
	userID int,
) (int, error) {
	return s.repo.CountThisMonth(userID)
}
