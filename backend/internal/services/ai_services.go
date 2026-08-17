package services

import (
	"errors"

	"backend/internal/models"
	"backend/internal/repository"
)

type AIService struct {
	repo     *repository.DiagnosisRepository
	provider AIProvider
}

func NewAIService(
	repo *repository.DiagnosisRepository,
	provider AIProvider,
) *AIService {
	return &AIService{
		repo:     repo,
		provider: provider,
	}
}

func (s *AIService) Diagnose(
	farmerID int,
	request AIRequest,
) (*models.Diagnosis, error) {

	if request.Category == "" &&
		request.Description == "" &&
		len(request.Image) == 0 &&
		len(request.Audio) == 0 &&
		len(request.Video) == 0 {
		return nil, errors.New("no information was provided")
	}

	aiResult, err := s.provider.Analyze(request)
	if err != nil {
		return nil, err
	}

	diagnosis := &models.Diagnosis{
		FarmerID:    farmerID,
		Category:    request.Category,
		Description: request.Description,
		Result:      aiResult.Result,
	}

	if err := s.repo.Create(diagnosis); err != nil {
		return nil, err
	}

	return diagnosis, nil
}

func (s *AIService) History(farmerID int) ([]models.Diagnosis, error) {
	return s.repo.ListByFarmer(farmerID)
}

func (s *AIService) CountThisMonth(farmerID int) (int, error) {
	return s.repo.CountThisMonth(farmerID)
}
