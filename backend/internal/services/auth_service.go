package services

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"backend/internal/models"
	"backend/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

// AuthService handles user registration, authentication,
// profile updates, and password management.
type AuthService struct {
	repo            *repository.UserRepository
	otp             *OTPService
	registrationOTP *RegistrationOTPService
}

func (s *AuthService) SetOTP(otp *OTPService) { s.otp = otp }

func (s *AuthService) SetRegistrationOTP(otp *RegistrationOTPService) { s.registrationOTP = otp }

func (s *AuthService) RequestRegistrationCode(ctx context.Context, phone string) (string, error) {
	if !s.registrationOTP.Available() {
		return "", ErrRegistrationOTPUnavailable
	}
	if NormalizeNigerianPhone(phone) == "" {
		return "", errors.New("enter a valid Nigerian phone number")
	}
	user, err := s.FindByPhone(phone)
	if err != nil {
		return "", errors.New("we could not start phone confirmation; please try again")
	}
	if user != nil {
		return "", errors.New("that phone number already has an account; please sign in")
	}
	return s.registrationOTP.Issue(ctx, phone)
}

func (s *AuthService) RegistrationCodeTTL() time.Duration      { return s.registrationOTP.ttl }
func (s *AuthService) RegistrationCodeCooldown() time.Duration { return s.registrationOTP.cooldown }

var ErrRecoveryUnavailable = errors.New("password recovery is unavailable; please contact support")

// FindByPhone accepts the phone formats used by the old application while
// new accounts store one canonical Nigerian number.
func (s *AuthService) FindByPhone(phone string) (*models.User, error) {
	phone = strings.TrimSpace(phone)
	canonical := NormalizeNigerianPhone(phone)
	if canonical == "" {
		return nil, nil
	}
	seen := make(map[string]bool)
	for _, value := range []string{phone, canonical, "+" + canonical, "0" + canonical[3:]} {
		if seen[value] {
			continue
		}
		seen[value] = true
		user, err := s.repo.GetByPhone(value)
		if err != nil || user != nil {
			return user, err
		}
	}
	return nil, nil
}

func (s *AuthService) RequestPasswordReset(ctx context.Context, phone string) error {
	if s.otp == nil || !s.otp.Available() {
		return ErrRecoveryUnavailable
	}
	user, err := s.FindByPhone(phone)
	if err != nil {
		return err
	}
	if user == nil {
		return nil
	}
	return s.otp.Issue(ctx, user.ID, user.Phone, user.Email, models.OTPPurposePasswordReset)
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
// - learner is a valid platform role.
// - account_type describes the account structure.
// - farmingTypes represent farming activities the user
//   already practices.
// - Phone confirmation is required before an account is created.
// - Profile picture IS REQUIRED.
// ============================================================

func (s *AuthService) Register(
	ctx context.Context,
	registrationToken, registrationCode string,
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
	organisationName string,
	farmingTypes []string,
) error {
	if !s.registrationOTP.Available() {
		return ErrRegistrationOTPUnavailable
	}

	// --------------------------------------------------------
	// Normalize input
	// --------------------------------------------------------

	firstName = strings.TrimSpace(firstName)
	lastName = strings.TrimSpace(lastName)
	phone = NormalizeNigerianPhone(phone)
	email = strings.TrimSpace(email)
	location = strings.TrimSpace(location)
	lga = strings.TrimSpace(lga)
	state = strings.TrimSpace(state)
	country = strings.TrimSpace(country)
	role = strings.ToLower(strings.TrimSpace(role))
	photoURL = strings.TrimSpace(photoURL)
	accountType = strings.ToLower(strings.TrimSpace(accountType))
	if accountType == "organization" {
		accountType = "organisation"
	}
	organisationName = strings.TrimSpace(organisationName)

	// --------------------------------------------------------
	// Required personal information
	// --------------------------------------------------------

	if firstName == "" || lastName == "" {
		return errors.New("first name and last name are required")
	}

	if phone == "" {
		return errors.New("enter a valid Nigerian phone number")
	}
	if email != "" {
		if parsed, err := mail.ParseAddress(email); err != nil || parsed.Address != email {
			return errors.New("enter a valid email address")
		}
	}

	if password == "" {
		return errors.New("password is required")
	}

	if len(password) < 8 || len(password) > 72 {
		return errors.New("password must be between 8 and 72 bytes")
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
	// Learner is a valid platform role.
	// --------------------------------------------------------

	switch role {
	case "farmer", "buyer", "learner", "ministry", "organisation":
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
	// Check phone uniqueness
	// --------------------------------------------------------

	existing, err := s.FindByPhone(phone)
	if err != nil {
		return err
	}

	if existing != nil {
		return errors.New("phone number already registered")
	}
	proof, err := s.registrationOTP.Check(ctx, registrationToken, phone, registrationCode)
	if err != nil {
		return err
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

	_, err = s.repo.CreateWithVerifiedPhone(user, proof)
	if err != nil {
		return err
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

	user, err := s.FindByPhone(phone)
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
func (s *AuthService) UpdateProfile(id int, fullName, phone, email, location, state, lga, currentPassword, photoURL string) error {
	fullName = strings.TrimSpace(fullName)
	phone = NormalizeNigerianPhone(phone)
	email = strings.TrimSpace(email)
	location = strings.TrimSpace(location)
	if id <= 0 || fullName == "" || phone == "" || location == "" {
		return errors.New("enter your name, phone number, and community")
	}
	if email != "" {
		if address, err := mail.ParseAddress(email); err != nil || address.Address != email {
			return errors.New("enter a valid email address")
		}
	}
	current, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if current == nil {
		return errors.New("account was not found")
	}
	if NormalizeNigerianPhone(current.Phone) != phone || current.Email != email {
		if err = bcrypt.CompareHashAndPassword([]byte(current.PasswordHash), []byte(currentPassword)); err != nil {
			return errors.New("enter your current password to change your phone number or email")
		}
	}
	if state == "" {
		state = current.State
	}
	if lga == "" {
		lga = current.LGA
	}
	return s.repo.UpdateProfile(id, fullName, phone, email, location, state, lga, photoURL)
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
	ctx context.Context,
	phone,
	newPassword, code string,
) error {
	if s.otp == nil || !s.otp.Available() {
		return ErrRecoveryUnavailable
	}

	phone = strings.TrimSpace(phone)

	if phone == "" {
		return errors.New("phone number is required")
	}

	if newPassword == "" {
		return errors.New("new password is required")
	}

	if len(newPassword) < 8 || len(newPassword) > 72 {
		return errors.New(
			"new password must be between 8 and 72 bytes",
		)
	}

	user, err := s.FindByPhone(phone)
	if err != nil {
		return err
	}

	if user == nil {
		return errors.New("that code is incorrect or has expired")
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(newPassword),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.otp.Verify(user.ID, models.OTPPurposePasswordReset, code); err != nil {
		return errors.New("that code is incorrect, expired, or already used")
	}

	return s.repo.UpdatePassword(
		user.ID,
		string(hash),
	)
}
