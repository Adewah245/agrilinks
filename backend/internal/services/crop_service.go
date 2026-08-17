package services

import (
	"errors"

	"backend/internal/models"
	"backend/internal/repository"
)

type CropService struct {
	repo *repository.CropRepository
}

func NewCropService(repo *repository.CropRepository) *CropService {
	return &CropService{repo: repo}
}

func (s *CropService) AddCrop(farmerID int, name, unit, location string, quantity, price float64, listForSale bool, imageURL string) error {
	if name == "" {
		return errors.New("crop name is required")
	}
	if quantity <= 0 {
		return errors.New("quantity must be greater than zero")
	}
	if listForSale && price <= 0 {
		return errors.New("price must be greater than zero to list for sale")
	}

	crop := &models.Crop{
		FarmerID:      farmerID,
		Name:          name,
		Quantity:      quantity,
		Unit:          unit,
		Location:      location,
		PricePerUnit:  price,
		ListedForSale: listForSale,
		ImageURL:      imageURL,
	}

	return s.repo.Create(crop)
}

func (s *CropService) MyCrops(farmerID int) ([]models.Crop, error) {
	return s.repo.ListByFarmer(farmerID)
}

func (s *CropService) AvailableCrops() ([]models.Crop, error) {
	return s.repo.ListAvailable()
}
func (s *CropService) GetCrop(cropID int) (*models.Crop, error) {
	return s.repo.GetByID(cropID)
}
