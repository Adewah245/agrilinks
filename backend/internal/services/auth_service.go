package services

import (
	"errors"
	"strings"

	"backend/internal/models"
	"backend/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

// AuthService handles user registration, authentication,
// profile updates, and password management.
type AuthService struct {
	repo *repository.UserRepository
}

// NewAuthService creates a new AuthService using the provided
// user repository.
func NewAuthService(repo *repository.UserRepository) *AuthService {
	return &AuthService{
		repo: repo,
	}
}

// ============================================================
// REGISTER
//
// Creates a new Agro-Shield user account.
//
// Important architecture:
//
// - role is the user's platform identity.
// - learning is NOT a role.
// - account_type describes the account structure.
// - farmingTypes represent farming activities the user
//   already practices.
// - NIN is OPTIONAL for now because the verification API
//   has not yet been integrated.
// - Profile picture IS REQUIRED.
// ============================================================

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

	// --------------------------------------------------------
	// Normalize input
	// --------------------------------------------------------

	firstName = strings.TrimSpace(firstName)
	lastName = strings.TrimSpace(lastName)
	phone = strings.TrimSpace(phone)
	email = strings.TrimSpace(email)
	password = strings.TrimSpace(password)
	location = strings.TrimSpace(location)
	lga = strings.TrimSpace(lga)
	state = strings.TrimSpace(state)
	country = strings.TrimSpace(country)
	role = strings.ToLower(strings.TrimSpace(role))
	photoURL = strings.TrimSpace(photoURL)
	accountType = strings.ToLower(strings.TrimSpace(accountType))
	nin = strings.TrimSpace(nin)
	organisationName = strings.TrimSpace(organisationName)

	// --------------------------------------------------------
	// Required personal information
	// --------------------------------------------------------

	if firstName == "" || lastName == "" {
		return errors.New("first name and last name are required")
	}

	if phone == "" {
		return errors.New("phone is required")
	}

	if password == "" {
		return errors.New("password is required")
	}

	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	// --------------------------------------------------------
	// Profile picture
	//
	// Profile picture is compulsory.
	// --------------------------------------------------------

	if photoURL == "" {
		return errors.New("profile picture is required")
	}

	// --------------------------------------------------------
	// Location
	// --------------------------------------------------------

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

	// --------------------------------------------------------
	// Account type
	//
	// Must match users.account_type constraint:
	//
	// individual
	// organisation
	// --------------------------------------------------------

	if accountType == "" {
		accountType = "individual"
	}

	if accountType != "individual" &&
		accountType != "organisation" {
		return errors.New("invalid account type")
	}

	// --------------------------------------------------------
	// Role
	//
	// Learning is NOT a role.
	// --------------------------------------------------------

	switch role {
	case "farmer", "buyer", "ministry", "organisation":
		// Valid role.
	default:
		return errors.New("invalid user role")
	}

	// --------------------------------------------------------
	// Organisation validation
	// --------------------------------------------------------

	if accountType == "organisation" {

		if organisationName == "" {
			return errors.New("organisation name is required")
		}

	} else {
		organisationName = ""
	}

	// Organisation role must use organisation account type.
	if role == "organisation" &&
		accountType != "organisation" {

		return errors.New(
			"organisation accounts must use organisation account type",
		)
	}

	// --------------------------------------------------------
	// NIN
	//
	// NIN is intentionally OPTIONAL for now.
	//
	// We keep receiving it here so that the registration
	// interface is ready for the future NIN API.
	//
	// For now we DO NOT:
	//
	// - require NIN
	// - verify NIN
	// - store raw NIN in users
	//
	// When the NIN API is available, this section can become:
	//
	// NIN
	//   ↓
	// API verification
	//   ↓
	// successful verification
	//   ↓
	// identity verification record
	//
	// --------------------------------------------------------

	_ = nin

	// --------------------------------------------------------
	// Check phone uniqueness
	// --------------------------------------------------------

	existing, err := s.repo.GetByPhone(phone)
	if err != nil {
		return err
	}

	if existing != nil {
		return errors.New("phone number already registered")
	}

	// --------------------------------------------------------
	// Hash password
	// --------------------------------------------------------

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return err
	}

	// --------------------------------------------------------
	// Build user
	// --------------------------------------------------------

	user := &models.User{
		FullName: firstName + " " + lastName,

		Phone: phone,
		Email: email,

		PasswordHash: string(hash),

		Location: location,
		LGA:      lga,
		State:    state,
		Country:  country,

		Role:        role,
		AccountType: accountType,

		OrganisationName: organisationName,

		PhotoURL: photoURL,

		FarmingTypes: farmingTypes,
	}

	// --------------------------------------------------------
	// Create user
	// --------------------------------------------------------

	userID, err := s.repo.Create(user)
	if err != nil {
		return err
	}

	// --------------------------------------------------------
	// Save existing farming activities
	//
	// These are NOT learning journeys.
	// --------------------------------------------------------

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

// ============================================================
// LOGIN
// ============================================================

// Login verifies a user's phone number and password.
//
// The stored password hash is compared using bcrypt.
// On success, the complete user record is returned.
func (s *AuthService) Login(
	phone,
	password string,
) (*models.User, error) {

	phone = strings.TrimSpace(phone)

	if phone == "" || password == "" {
		return nil, errors.New(
			"phone and password are required",
		)
	}

	user, err := s.repo.GetByPhone(phone)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, errors.New(
			"invalid phone number or password",
		)
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(password),
	); err != nil {
		return nil, errors.New(
		"invalid phone number or password",
		)
	}

	return user, nil
}

// ============================================================
// GET USER BY ID
// ============================================================

// GetUserByID retrieves a user's complete profile information.
func (s *AuthService) GetUserByID(
	id int,
) (*models.User, error) {

	if id <= 0 {
		return nil, errors.New("invalid user ID")
	}

	return s.repo.GetByID(id)
}

// ============================================================
// UPDATE PROFILE
// ============================================================

// UpdateProfile updates a user's editable personal information.
func (s *AuthService) UpdateProfile(
	id int,
	fullName,
	phone,
	email,
	location string,
) error {

	if id <= 0 {
		return errors.New("invalid user ID")
	}

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

// ============================================================
// UPDATE PHOTO
// ============================================================

// UpdatePhoto updates the user's profile photograph URL.
func (s *AuthService) UpdatePhoto(
	id int,
	photoURL string,
) error {

	if id <= 0 {
		return errors.New("invalid user ID")
	}

	photoURL = strings.TrimSpace(photoURL)

	if photoURL == "" {
		return errors.New("photo URL is required")
	}

	return s.repo.UpdatePhoto(
		id,
		photoURL,
	)
}

// ============================================================
// RESET PASSWORD
// ============================================================

// ResetPassword changes the password associated with a user's
// phone number.
func (s *AuthService) ResetPassword(
	phone,
	newPassword string,
) error {

	phone = strings.TrimSpace(phone)

	if phone == "" {
		return errors.New("phone number is required")
	}

	if newPassword == "" {
		return errors.New("new password is required")
	}

	if len(newPassword) < 8 {
		return errors.New(
			"new password must be at least 8 characters",
		)
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
