package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"backend/internal/models"
	"backend/internal/repository"
)

type LearningService struct {
	repo *repository.LearningRepository
}

func NewLearningService(repo *repository.LearningRepository) *LearningService {
	return &LearningService{
		repo: repo,
	}
}

// ============================================================
// CONSTANTS
// ============================================================

const (
	LearningStatusActive    = "active"
	LearningStatusPaused    = "paused"
	LearningStatusCompleted = "completed"
	LearningStatusAbandoned = "abandoned"

	ExperienceBeginner     = "beginner"
	ExperienceSome         = "some_experience"
	ExperienceExperienced  = "experienced"

	StageInterest     = "interest"
	StageLearning     = "learning"
	StagePreparation  = "preparation"
	StageReadiness    = "readiness"
	StageConfirmation = "confirmation"
	StageConfirmed    = "confirmed"

	FarmingStatusInterested  = "interested"
	FarmingStatusLearning    = "learning"
	FarmingStatusPreparing   = "preparing"
	FarmingStatusConfirmed   = "confirmed"

	FarmingSourceUserSelection = "user_selection"
	FarmingSourceLearning      = "learning_journey"
)

// ============================================================
// FARMING TYPES
// ============================================================

// GetFarmingTypes returns all farming types available for learning.
func (s *LearningService) GetFarmingTypes() ([]models.FarmingType, error) {
	farmingTypes, err := s.repo.ListFarmingTypes()
	if err != nil {
		return nil, fmt.Errorf("get farming types: %w", err)
	}

	return farmingTypes, nil
}

// GetFarmingType returns one farming type.
func (s *LearningService) GetFarmingType(id int) (*models.FarmingType, error) {
	if id <= 0 {
		return nil, errors.New("invalid farming type")
	}

	farmingType, err := s.repo.GetFarmingTypeByID(id)
	if err != nil {
		return nil, fmt.Errorf("get farming type: %w", err)
	}

	return farmingType, nil
}

// ============================================================
// USER FARMING TYPES
// ============================================================

// GetUserFarmingTypes returns the farming activities associated
// with the current user.
func (s *LearningService) GetUserFarmingTypes(
	userID int,
) ([]models.UserFarmingType, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user")
	}

	result, err := s.repo.GetUserFarmingTypes(userID)
	if err != nil {
		return nil, fmt.Errorf("get user farming types: %w", err)
	}

	return result, nil
}

// RegisterLearningInterest records that a user wants to learn
// a particular farming activity.
func (s *LearningService) RegisterLearningInterest(
	userID int,
	farmingTypeID int,
) (*models.UserFarmingType, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user")
	}

	if farmingTypeID <= 0 {
		return nil, errors.New("invalid farming type")
	}

	// Make sure the farming type actually exists.
	farmingType, err := s.repo.GetFarmingTypeByID(farmingTypeID)
	if err != nil {
		return nil, fmt.Errorf("validate farming type: %w", err)
	}

	if farmingType == nil {
		return nil, errors.New("farming type not found")
	}

	item, err := s.repo.CreateUserFarmingType(
		userID,
		farmingTypeID,
		FarmingStatusInterested,
		FarmingSourceUserSelection,
	)
	if err != nil {
		return nil, fmt.Errorf("register learning interest: %w", err)
	}

	item.FarmingType = farmingType

	return item, nil
}

// ============================================================
// LEARNING JOURNEY
// ============================================================

// StartLearningJourney starts a new learning journey.
func (s *LearningService) StartLearningJourney(
	userID int,
	farmingTypeID int,
	experienceLevel string,
	goal string,
) (*models.LearningJourney, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user")
	}

	if farmingTypeID <= 0 {
		return nil, errors.New("invalid farming type")
	}

	experienceLevel = strings.TrimSpace(experienceLevel)
	goal = strings.TrimSpace(goal)

	if !validExperienceLevel(experienceLevel) {
		return nil, errors.New(
			"experience level must be beginner, some_experience, or experienced",
		)
	}

	if len(goal) > 1000 {
		return nil, errors.New("learning goal is too long")
	}

	// Verify farming type.
	farmingType, err := s.repo.GetFarmingTypeByID(farmingTypeID)
	if err != nil {
		return nil, fmt.Errorf("validate farming type: %w", err)
	}

	if farmingType == nil {
		return nil, errors.New("farming type not found")
	}

	// Ensure the user has a learning relationship with this
	// farming type.
	userFarmingType, err := s.repo.GetUserFarmingType(
		userID,
		farmingTypeID,
	)
	if err != nil {
		return nil, fmt.Errorf("check learning interest: %w", err)
	}

	if userFarmingType == nil {
		userFarmingType, err = s.repo.CreateUserFarmingType(
			userID,
			farmingTypeID,
			FarmingStatusInterested,
			FarmingSourceLearning,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"create learning interest: %w",
				err,
			)
		}
	}

	// Prevent accidentally creating another active journey
	// for the same user.
	activeJourney, err := s.repo.GetActiveLearningJourney(userID)
	if err != nil {
		return nil, fmt.Errorf(
			"check active learning journey: %w",
			err,
		)
	}

	if activeJourney != nil {
		if activeJourney.FarmingTypeID == farmingTypeID {
			return activeJourney, nil
		}

		return nil, errors.New(
			"user already has an active learning journey",
		)
	}

	journey := &models.LearningJourney{
		UserID:            userID,
		FarmingTypeID:     farmingTypeID,
		UserFarmingTypeID: userFarmingType.ID,
		Status:            LearningStatusActive,
		ExperienceLevel:   experienceLevel,
		Goal:              goal,
		CurrentStage:     StageLearning,
		ReadinessScore:    0,
	}

	if err := s.repo.CreateLearningJourney(journey); err != nil {
		return nil, fmt.Errorf("start learning journey: %w", err)
	}

	journey.FarmingType = farmingType

	// Move the user's farming-type status from interest to learning.
	userFarmingType.Status = FarmingStatusLearning
	userFarmingType.Source = FarmingSourceLearning

	if _, err := s.repo.CreateUserFarmingType(
		userID,
		farmingTypeID,
		FarmingStatusLearning,
		FarmingSourceLearning,
	); err != nil {
		return nil, fmt.Errorf(
			"update learning status: %w",
			err,
		)
	}

	return journey, nil
}

// GetLearningJourney returns a user's journey for a farming type.
func (s *LearningService) GetLearningJourney(
	userID int,
	farmingTypeID int,
) (*models.LearningJourney, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user")
	}

	if farmingTypeID <= 0 {
		return nil, errors.New("invalid farming type")
	}

	journey, err := s.repo.GetLearningJourney(
		userID,
		farmingTypeID,
	)
	if err != nil {
		return nil, fmt.Errorf("get learning journey: %w", err)
	}

	return journey, nil
}

// GetActiveLearningJourney returns the user's current journey.
func (s *LearningService) GetActiveLearningJourney(
	userID int,
) (*models.LearningJourney, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user")
	}

	journey, err := s.repo.GetActiveLearningJourney(userID)
	if err != nil {
		return nil, fmt.Errorf(
			"get active learning journey: %w",
			err,
		)
	}

	return journey, nil
}

// ============================================================
// JOURNEY PROGRESS
// ============================================================

// UpdateJourneyProgress updates the learning stage and readiness.
func (s *LearningService) UpdateJourneyProgress(
	userID int,
	journeyID int,
	stage string,
	readinessScore int,
	status string,
) error {
	if userID <= 0 {
		return errors.New("invalid user")
	}

	if journeyID <= 0 {
		return errors.New("invalid learning journey")
	}

	stage = strings.TrimSpace(stage)
	status = strings.TrimSpace(status)

	if !validLearningStage(stage) {
		return errors.New("invalid learning stage")
	}

	if !validLearningStatus(status) {
		return errors.New("invalid learning status")
	}

	if readinessScore < 0 || readinessScore > 100 {
		return errors.New("readiness score must be between 0 and 100")
	}

	journey := &models.LearningJourney{
		ID:             journeyID,
		UserID:         userID,
		CurrentStage:   stage,
		ReadinessScore: readinessScore,
		Status:         status,
	}

	// Confirmation should only happen when the readiness
	// requirement has been reached.
	if stage == StageConfirmed {
		if readinessScore < 80 {
			return errors.New(
				"readiness score must be at least 80 before confirmation",
			)
		}

		now := time.Now()
		journey.ConfirmedAt = &now
		journey.Status = LearningStatusCompleted
	}

	if stage == StagePreparation {
		// Preparation means the learner has moved beyond
		// theoretical learning.
	}

	if err := s.repo.UpdateLearningJourney(journey); err != nil {
		return fmt.Errorf("update journey progress: %w", err)
	}

	// Keep the user's farming-type status synchronized.
	farmingStatus := FarmingStatusLearning

	switch stage {
	case StagePreparation:
		farmingStatus = FarmingStatusPreparing

	case StageConfirmed:
		farmingStatus = FarmingStatusConfirmed
	}

	if _, err := s.repo.CreateUserFarmingType(
		userID,
		journey.FarmingTypeID,
		farmingStatus,
		FarmingSourceLearning,
	); err != nil {
		return fmt.Errorf(
			"update farming type learning status: %w",
			err,
		)
	}

	return nil
}

// ============================================================
// LEARNING MESSAGES
// ============================================================

// SendLearningMessage stores a message in a learning journey.
func (s *LearningService) SendLearningMessage(
	userID int,
	journeyID int,
	senderType string,
	messageType string,
	content string,
) (*models.LearningMessage, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user")
	}

	if journeyID <= 0 {
		return nil, errors.New("invalid learning journey")
	}

	content = strings.TrimSpace(content)

	if content == "" {
		return nil, errors.New("message cannot be empty")
	}

	if len(content) > 10000 {
		return nil, errors.New("message is too long")
	}

	if !validSenderType(senderType) {
		return nil, errors.New("invalid message sender type")
	}

	if !validMessageType(messageType) {
		return nil, errors.New("invalid message type")
	}

	message := &models.LearningMessage{
		JourneyID:  journeyID,
		SenderType: senderType,
		MessageType: messageType,
		Content:    content,
	}

	if err := s.repo.CreateLearningMessage(message); err != nil {
		return nil, fmt.Errorf("send learning message: %w", err)
	}

	return message, nil
}

// GetLearningMessages returns the conversation history.
func (s *LearningService) GetLearningMessages(
	userID int,
	journeyID int,
) ([]models.LearningMessage, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user")
	}

	if journeyID <= 0 {
		return nil, errors.New("invalid learning journey")
	}

	messages, err := s.repo.GetLearningMessages(journeyID)
	if err != nil {
		return nil, fmt.Errorf("get learning messages: %w", err)
	}

	return messages, nil
}

// ============================================================
// CHECKPOINTS
// ============================================================

// GetLearningCheckpoints returns checkpoints for a journey.
func (s *LearningService) GetLearningCheckpoints(
	userID int,
	journeyID int,
) ([]models.LearningCheckpoint, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user")
	}

	if journeyID <= 0 {
		return nil, errors.New("invalid learning journey")
	}

	checkpoints, err := s.repo.GetLearningCheckpoints(journeyID)
	if err != nil {
		return nil, fmt.Errorf(
			"get learning checkpoints: %w",
			err,
		)
	}

	return checkpoints, nil
}

// CreateLearningCheckpoint creates a checkpoint for a journey.
func (s *LearningService) CreateLearningCheckpoint(
	userID int,
	journeyID int,
	key string,
	title string,
	description string,
) (*models.LearningCheckpoint, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user")
	}

	if journeyID <= 0 {
		return nil, errors.New("invalid learning journey")
	}

	key = strings.TrimSpace(key)
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)

	if key == "" {
		return nil, errors.New("checkpoint key is required")
	}

	if title == "" {
		return nil, errors.New("checkpoint title is required")
	}

	if len(key) > 100 {
		return nil, errors.New("checkpoint key is too long")
	}

	if len(title) > 255 {
		return nil, errors.New("checkpoint title is too long")
	}

	checkpoint := &models.LearningCheckpoint{
		JourneyID:     journeyID,
		CheckpointKey: key,
		Title:         title,
		Description:   description,
		Status:        "pending",
		Progress:      0,
	}

	if err := s.repo.CreateLearningCheckpoint(checkpoint); err != nil {
		return nil, fmt.Errorf(
			"create learning checkpoint: %w",
			err,
		)
	}

	return checkpoint, nil
}

// UpdateLearningCheckpoint updates checkpoint progress.
func (s *LearningService) UpdateLearningCheckpoint(
	userID int,
	checkpoint *models.LearningCheckpoint,
) error {
	if userID <= 0 {
		return errors.New("invalid user")
	}

	if checkpoint == nil {
		return errors.New("checkpoint is required")
	}

	if checkpoint.ID <= 0 || checkpoint.JourneyID <= 0 {
		return errors.New("invalid checkpoint")
	}

	if checkpoint.Progress < 0 || checkpoint.Progress > 100 {
		return errors.New("checkpoint progress must be between 0 and 100")
	}

	if !validCheckpointStatus(checkpoint.Status) {
		return errors.New("invalid checkpoint status")
	}

	if checkpoint.Progress == 100 {
		checkpoint.Status = "completed"

		if checkpoint.CompletedAt == nil {
			now := time.Now()
			checkpoint.CompletedAt = &now
		}
	}

	if err := s.repo.UpdateLearningCheckpoint(checkpoint); err != nil {
		return fmt.Errorf(
			"update learning checkpoint: %w",
			err,
		)
	}

	return nil
}

// ============================================================
// VALIDATION HELPERS
// ============================================================

func validExperienceLevel(value string) bool {
	switch value {
	case ExperienceBeginner,
		ExperienceSome,
		ExperienceExperienced:
		return true
	default:
		return false
	}
}

func validLearningStatus(value string) bool {
	switch value {
	case LearningStatusActive,
		LearningStatusPaused,
		LearningStatusCompleted,
		LearningStatusAbandoned:
		return true
	default:
		return false
	}
}

func validLearningStage(value string) bool {
	switch value {
	case StageInterest,
		StageLearning,
		StagePreparation,
		StageReadiness,
		StageConfirmation,
		StageConfirmed:
		return true
	default:
		return false
	}
}

func validSenderType(value string) bool {
	switch value {
	case "user", "ai", "system":
		return true
	default:
		return false
	}
}

func validMessageType(value string) bool {
	switch value {
	case "text",
		"guidance",
		"recommendation",
		"question",
		"milestone",
		"warning":
		return true
	default:
		return false
	}
}

func validCheckpointStatus(value string) bool {
	switch value {
	case "pending",
		"in_progress",
		"completed",
		"not_applicable":
		return true
	default:
		return false
	}
}
