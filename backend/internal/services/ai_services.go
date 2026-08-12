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
	imageName string,
	imageBytes []byte,
) (*models.Diagnosis, error) {
	if len(imageBytes) == 0 {
		return nil, errors.New("no image was uploaded")
	}

	aiResult, err := s.provider.AnalyzeImage(imageBytes)
	if err != nil {
		return nil, err
	}

	diagnosis := &models.Diagnosis{
		FarmerID:  farmerID,
		ImageName: imageName,
		Result:    aiResult.Result,
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
