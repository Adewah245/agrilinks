package services

import (
	"errors"
	"strings"

	"backend/internal/models"
	"backend/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	repo *repository.UserRepository
}

func NewAuthService(repo *repository.UserRepository) *AuthService {
	return &AuthService{
		repo: repo,
	}
}

// Register creates a new user account.
// role should normally be "farmer" or "buyer".
func (s *AuthService) Register(
	firstName,
	lastName,
	phone,
	email,
	password,
	location,
	lga,
	state,
	country,
	role,
	photoURL,
	accountType,
	nin,
	organisationName string,
	farmingTypes []string,
) error {

	if firstName == "" || lastName == "" {
		return errors.New("name is required")
	}

	if phone == "" {
		return errors.New("phone is required")
	}

	if password == "" {
		return errors.New("password is required")
	}

	if photoURL == "" {
		return errors.New("profile picture is required")
	}

	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	if location == "" {
		return errors.New("community is required")
	}

	if lga == "" {
		return errors.New("LGA is required")
	}

	if state == "" {
		return errors.New("state is required")
	}

	if country == "" {
		return errors.New("country is required")
	}

	// Default account settings.
	if accountType == "" {
		accountType = "individual"
	}

	verificationStatus := "pending"

	if role == "partner" {
		verificationStatus = "pending"
	}

	if accountType != "organization" {
		organisationName = ""
	}

	existing, err := s.repo.GetByPhone(phone)
	if err != nil {
		return err
	}

	if existing != nil {
		return errors.New("phone number already registered")
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return err
	}

	user := &models.User{
		FullName: firstName + " " + lastName,

		Phone: phone,
		Email: email,

		PasswordHash: string(hash),

		// Location information.
		Location: location,
		LGA:      lga,
		State:    state,
		Country:  country,

		// Account information.
		Role:        role,
		AccountType: accountType,

		// Verification.
		NIN:                nin,
		VerificationStatus: verificationStatus,

		// Organisation / Partner.
		OrganisationName: organisationName,

		// Profile.
		PhotoURL: photoURL,

		// Farming types.
		FarmingTypes: farmingTypes,
	}

	userID, err := s.repo.Create(user)
	if err != nil {
		return err
	}

	if len(user.FarmingTypes) > 0 {
		if err := s.repo.SaveFarmingTypes(
			userID,
			user.FarmingTypes,
		); err != nil {
			return err
		}
	}

	return nil
}

// Login verifies a phone number + password against the stored hash
// and returns the matching user on success.
func (s *AuthService) Login(phone, password string) (*models.User, error) {

	if phone == "" || password == "" {
		return nil, errors.New("phone and password are required")
	}

	user, err := s.repo.GetByPhone(phone)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, errors.New("invalid phone number or password")
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(password),
	); err != nil {
		return nil, errors.New("invalid phone number or password")
	}

	return user, nil
}

// GetUserByID loads a user's full details, e.g. for the profile page.
func (s *AuthService) GetUserByID(id int) (*models.User, error) {
	return s.repo.GetByID(id)
}

// UpdateProfile edits a user's name, phone and location.
func (s *AuthService) UpdateProfile(
	id int,
	fullName,
	phone,
	email,
	location string,
) error {

	fullName = strings.TrimSpace(fullName)
	phone = strings.TrimSpace(phone)
	email = strings.TrimSpace(email)
	location = strings.TrimSpace(location)

	if fullName == "" {
		return errors.New("full name is required")
	}

	if phone == "" {
		return errors.New("phone is required")
	}

	if location == "" {
		return errors.New("location is required")
	}

	return s.repo.UpdateProfile(
		id,
		fullName,
		phone,
		email,
		location,
	)
}

// UpdatePhoto sets the user's passport photograph.
func (s *AuthService) UpdatePhoto(id int, photoURL string) error {
	if photoURL == "" {
		return errors.New("photo URL is required")
	}

	return s.repo.UpdatePhoto(id, photoURL)
}

// ResetPassword changes a user's password.
func (s *AuthService) ResetPassword(
	phone,
	newPassword string,
) error {

	if phone == "" {
		return errors.New("phone number is required")
	}

	if newPassword == "" {
		return errors.New("new password is required")
	}

	if len(newPassword) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	user, err := s.repo.GetByPhone(phone)
	if err != nil {
		return err
	}

	if user == nil {
		return errors.New("account not found")
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(newPassword),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return err
	}

	return s.repo.UpdatePassword(
		user.ID,
		string(hash),
	)
}