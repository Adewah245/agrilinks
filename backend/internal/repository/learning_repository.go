package repository

import (
	"database/sql"
	"fmt"

	"backend/internal/models"
)

type LearningRepository struct {
	db *sql.DB
}

func NewLearningRepository(db *sql.DB) *LearningRepository {
	return &LearningRepository{db: db}
}

// ============================================================
// FARMING TYPES
// ============================================================

// ListFarmingTypes returns all available farming types.
func (r *LearningRepository) ListFarmingTypes() ([]models.FarmingType, error) {
	const query = `
		SELECT
			id,
			name,
			slug,
			description,
			created_at
		FROM farming_types
		ORDER BY name ASC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("list farming types: %w", err)
	}
	defer rows.Close()

	var farmingTypes []models.FarmingType

	for rows.Next() {
		var farmingType models.FarmingType

		if err := rows.Scan(
			&farmingType.ID,
			&farmingType.Name,
			&farmingType.Slug,
			&farmingType.Description,
			&farmingType.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan farming type: %w", err)
		}

		farmingTypes = append(farmingTypes, farmingType)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate farming types: %w", err)
	}

	return farmingTypes, nil
}

// GetFarmingTypeByID returns one farming type.
func (r *LearningRepository) GetFarmingTypeByID(id int) (*models.FarmingType, error) {
	const query = `
		SELECT
			id,
			name,
			slug,
			description,
			created_at
		FROM farming_types
		WHERE id = $1
	`

	var farmingType models.FarmingType

	err := r.db.QueryRow(query, id).Scan(
		&farmingType.ID,
		&farmingType.Name,
		&farmingType.Slug,
		&farmingType.Description,
		&farmingType.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("farming type not found")
		}

		return nil, fmt.Errorf("get farming type: %w", err)
	}

	return &farmingType, nil
}

// ============================================================
// USER FARMING TYPES
// ============================================================

// GetUserFarmingTypes returns farming activities associated
// with a user.
func (r *LearningRepository) GetUserFarmingTypes(userID int) ([]models.UserFarmingType, error) {
	const query = `
		SELECT
			uft.id,
			uft.user_id,
			uft.farming_type_id,
			uft.status,
			uft.source,
			uft.confirmed_at,
			uft.created_at,
			uft.updated_at,

			ft.id,
			ft.name,
			ft.slug,
			ft.description,
			ft.created_at

		FROM user_farming_types uft
		INNER JOIN farming_types ft
			ON ft.id = uft.farming_type_id

		WHERE uft.user_id = $1
		ORDER BY uft.created_at DESC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("get user farming types: %w", err)
	}
	defer rows.Close()

	var result []models.UserFarmingType

	for rows.Next() {
		var item models.UserFarmingType
		var farmingType models.FarmingType

		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.FarmingTypeID,
			&item.Status,
			&item.Source,
			&item.ConfirmedAt,
			&item.CreatedAt,
			&item.UpdatedAt,

			&farmingType.ID,
			&farmingType.Name,
			&farmingType.Slug,
			&farmingType.Description,
			&farmingType.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user farming type: %w", err)
		}

		item.FarmingType = &farmingType
		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user farming types: %w", err)
	}

	return result, nil
}

// GetUserFarmingType returns the user's relationship with a farming type.
func (r *LearningRepository) GetUserFarmingType(
	userID int,
	farmingTypeID int,
) (*models.UserFarmingType, error) {
	const query = `
		SELECT
			uft.id,
			uft.user_id,
			uft.farming_type_id,
			uft.status,
			uft.source,
			uft.confirmed_at,
			uft.created_at,
			uft.updated_at,

			ft.id,
			ft.name,
			ft.slug,
			ft.description,
			ft.created_at

		FROM user_farming_types uft
		INNER JOIN farming_types ft
			ON ft.id = uft.farming_type_id

		WHERE uft.user_id = $1
		  AND uft.farming_type_id = $2
	`

	var item models.UserFarmingType
	var farmingType models.FarmingType

	err := r.db.QueryRow(query, userID, farmingTypeID).Scan(
		&item.ID,
		&item.UserID,
		&item.FarmingTypeID,
		&item.Status,
		&item.Source,
		&item.ConfirmedAt,
		&item.CreatedAt,
		&item.UpdatedAt,

		&farmingType.ID,
		&farmingType.Name,
		&farmingType.Slug,
		&farmingType.Description,
		&farmingType.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("get user farming type: %w", err)
	}

	item.FarmingType = &farmingType

	return &item, nil
}

// CreateUserFarmingType creates the user's interest in a farming type.
func (r *LearningRepository) CreateUserFarmingType(
	userID int,
	farmingTypeID int,
	status string,
	source string,
) (*models.UserFarmingType, error) {
	const query = `
		INSERT INTO user_farming_types (
			user_id,
			farming_type_id,
			status,
			source
		)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, farming_type_id)
		DO UPDATE SET
			status = EXCLUDED.status,
			source = EXCLUDED.source,
			updated_at = CURRENT_TIMESTAMP
		RETURNING
			id,
			user_id,
			farming_type_id,
			status,
			source,
			confirmed_at,
			created_at,
			updated_at
	`

	var item models.UserFarmingType

	err := r.db.QueryRow(
		query,
		userID,
		farmingTypeID,
		status,
		source,
	).Scan(
		&item.ID,
		&item.UserID,
		&item.FarmingTypeID,
		&item.Status,
		&item.Source,
		&item.ConfirmedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("create user farming type: %w", err)
	}

	return &item, nil
}

// ============================================================
// LEARNING JOURNEYS
// ============================================================

// CreateLearningJourney creates a new learning journey.
func (r *LearningRepository) CreateLearningJourney(
	journey *models.LearningJourney,
) error {
	const query = `
		INSERT INTO learning_journeys (
			user_id,
			farming_type_id,
			user_farming_type_id,
			status,
			experience_level,
			goal,
			current_stage,
			readiness_score
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8
		)
		RETURNING
			id,
			started_at,
			last_activity_at,
			created_at,
			updated_at
	`

	err := r.db.QueryRow(
		query,
		journey.UserID,
		journey.FarmingTypeID,
		journey.UserFarmingTypeID,
		journey.Status,
		journey.ExperienceLevel,
		journey.Goal,
		journey.CurrentStage,
		journey.ReadinessScore,
	).Scan(
		&journey.ID,
		&journey.StartedAt,
		&journey.LastActivityAt,
		&journey.CreatedAt,
		&journey.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("create learning journey: %w", err)
	}

	return nil
}

// GetLearningJourney returns a user's journey for a farming type.
func (r *LearningRepository) GetLearningJourney(
	userID int,
	farmingTypeID int,
) (*models.LearningJourney, error) {
	const query = `
		SELECT
			lj.id,
			lj.user_id,
			lj.farming_type_id,
			lj.user_farming_type_id,
			lj.status,
			lj.experience_level,
			lj.goal,
			lj.current_stage,
			lj.readiness_score,
			lj.started_at,
			lj.last_activity_at,
			lj.completed_at,
			lj.confirmed_at,
			lj.created_at,
			lj.updated_at,

			ft.id,
			ft.name,
			ft.slug,
			ft.description,
			ft.created_at

		FROM learning_journeys lj
		INNER JOIN farming_types ft
			ON ft.id = lj.farming_type_id

		WHERE lj.user_id = $1
		  AND lj.farming_type_id = $2

		ORDER BY lj.created_at DESC
		LIMIT 1
	`

	var journey models.LearningJourney
	var farmingType models.FarmingType

	err := r.db.QueryRow(query, userID, farmingTypeID).Scan(
		&journey.ID,
		&journey.UserID,
		&journey.FarmingTypeID,
		&journey.UserFarmingTypeID,
		&journey.Status,
		&journey.ExperienceLevel,
		&journey.Goal,
		&journey.CurrentStage,
		&journey.ReadinessScore,
		&journey.StartedAt,
		&journey.LastActivityAt,
		&journey.CompletedAt,
		&journey.ConfirmedAt,
		&journey.CreatedAt,
		&journey.UpdatedAt,

		&farmingType.ID,
		&farmingType.Name,
		&farmingType.Slug,
		&farmingType.Description,
		&farmingType.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("get learning journey: %w", err)
	}

	journey.FarmingType = &farmingType

	return &journey, nil
}

// GetActiveLearningJourney returns the user's active or paused journey.
func (r *LearningRepository) GetActiveLearningJourney(
	userID int,
) (*models.LearningJourney, error) {
	const query = `
		SELECT
			lj.id,
			lj.user_id,
			lj.farming_type_id,
			lj.user_farming_type_id,
			lj.status,
			lj.experience_level,
			lj.goal,
			lj.current_stage,
			lj.readiness_score,
			lj.started_at,
			lj.last_activity_at,
			lj.completed_at,
			lj.confirmed_at,
			lj.created_at,
			lj.updated_at,

			ft.id,
			ft.name,
			ft.slug,
			ft.description,
			ft.created_at

		FROM learning_journeys lj
		INNER JOIN farming_types ft
			ON ft.id = lj.farming_type_id

		WHERE lj.user_id = $1
		  AND lj.status IN ('active', 'paused')

		ORDER BY lj.last_activity_at DESC
		LIMIT 1
	`

	var journey models.LearningJourney
	var farmingType models.FarmingType

	err := r.db.QueryRow(query, userID).Scan(
		&journey.ID,
		&journey.UserID,
		&journey.FarmingTypeID,
		&journey.UserFarmingTypeID,
		&journey.Status,
		&journey.ExperienceLevel,
		&journey.Goal,
		&journey.CurrentStage,
		&journey.ReadinessScore,
		&journey.StartedAt,
		&journey.LastActivityAt,
		&journey.CompletedAt,
		&journey.ConfirmedAt,
		&journey.CreatedAt,
		&journey.UpdatedAt,

		&farmingType.ID,
		&farmingType.Name,
		&farmingType.Slug,
		&farmingType.Description,
		&farmingType.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("get active learning journey: %w", err)
	}

	journey.FarmingType = &farmingType

	return &journey, nil
}

// UpdateLearningJourney updates progress/stage information.
func (r *LearningRepository) UpdateLearningJourney(
	journey *models.LearningJourney,
) error {
	const query = `
		UPDATE learning_journeys
		SET
			status = $1,
			experience_level = $2,
			goal = $3,
			current_stage = $4,
			readiness_score = $5,
			last_activity_at = CURRENT_TIMESTAMP,
			completed_at = $6,
			confirmed_at = $7,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $8
		  AND user_id = $9
		RETURNING
			last_activity_at,
			updated_at
	`

	err := r.db.QueryRow(
		query,
		journey.Status,
		journey.ExperienceLevel,
		journey.Goal,
		journey.CurrentStage,
		journey.ReadinessScore,
		journey.CompletedAt,
		journey.ConfirmedAt,
		journey.ID,
		journey.UserID,
	).Scan(
		&journey.LastActivityAt,
		&journey.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("update learning journey: %w", err)
	}

	return nil
}

// ============================================================
// LEARNING MESSAGES
// ============================================================

// CreateLearningMessage stores a user/AI/system message.
func (r *LearningRepository) CreateLearningMessage(
	message *models.LearningMessage,
) error {
	const query = `
		INSERT INTO learning_messages (
			journey_id,
			sender_type,
			message_type,
			content
		)
		VALUES ($1, $2, $3, $4)
		RETURNING
			id,
			created_at
	`

	return r.db.QueryRow(
		query,
		message.JourneyID,
		message.SenderType,
		message.MessageType,
		message.Content,
	).Scan(
		&message.ID,
		&message.CreatedAt,
	)
}

// GetLearningMessages returns messages for a journey.
func (r *LearningRepository) GetLearningMessages(
	journeyID int,
) ([]models.LearningMessage, error) {
	const query = `
		SELECT
			id,
			journey_id,
			sender_type,
			message_type,
			content,
			created_at
		FROM learning_messages
		WHERE journey_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(query, journeyID)
	if err != nil {
		return nil, fmt.Errorf("get learning messages: %w", err)
	}
	defer rows.Close()

	var messages []models.LearningMessage

	for rows.Next() {
		var message models.LearningMessage

		if err := rows.Scan(
			&message.ID,
			&message.JourneyID,
			&message.SenderType,
			&message.MessageType,
			&message.Content,
			&message.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan learning message: %w", err)
		}

		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate learning messages: %w", err)
	}

	return messages, nil
}

// ============================================================
// LEARNING CHECKPOINTS
// ============================================================

// CreateLearningCheckpoint creates a checkpoint.
func (r *LearningRepository) CreateLearningCheckpoint(
	checkpoint *models.LearningCheckpoint,
) error {
	const query = `
		INSERT INTO learning_checkpoints (
			journey_id,
			checkpoint_key,
			title,
			description,
			status,
			progress,
			notes
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (journey_id, checkpoint_key)
		DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			updated_at = CURRENT_TIMESTAMP
		RETURNING
			id,
			created_at,
			updated_at
	`

	return r.db.QueryRow(
		query,
		checkpoint.JourneyID,
		checkpoint.CheckpointKey,
		checkpoint.Title,
		checkpoint.Description,
		checkpoint.Status,
		checkpoint.Progress,
		checkpoint.Notes,
	).Scan(
		&checkpoint.ID,
		&checkpoint.CreatedAt,
		&checkpoint.UpdatedAt,
	)
}

// GetLearningCheckpoints returns all checkpoints for a journey.
func (r *LearningRepository) GetLearningCheckpoints(
	journeyID int,
) ([]models.LearningCheckpoint, error) {
	const query = `
		SELECT
			id,
			journey_id,
			checkpoint_key,
			title,
			description,
			status,
			progress,
			notes,
			completed_at,
			created_at,
			updated_at
		FROM learning_checkpoints
		WHERE journey_id = $1
		ORDER BY id ASC
	`

	rows, err := r.db.Query(query, journeyID)
	if err != nil {
		return nil, fmt.Errorf("get learning checkpoints: %w", err)
	}
	defer rows.Close()

	var checkpoints []models.LearningCheckpoint

	for rows.Next() {
		var checkpoint models.LearningCheckpoint

		if err := rows.Scan(
			&checkpoint.ID,
			&checkpoint.JourneyID,
			&checkpoint.CheckpointKey,
			&checkpoint.Title,
			&checkpoint.Description,
			&checkpoint.Status,
			&checkpoint.Progress,
			&checkpoint.Notes,
			&checkpoint.CompletedAt,
			&checkpoint.CreatedAt,
			&checkpoint.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan learning checkpoint: %w", err)
		}

		checkpoints = append(checkpoints, checkpoint)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate learning checkpoints: %w", err)
	}

	return checkpoints, nil
}

// UpdateLearningCheckpoint updates checkpoint progress.
func (r *LearningRepository) UpdateLearningCheckpoint(
	checkpoint *models.LearningCheckpoint,
) error {
	const query = `
		UPDATE learning_checkpoints
		SET
			status = $1,
			progress = $2,
			notes = $3,
			completed_at = $4,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $5
		  AND journey_id = $6
		RETURNING
			updated_at
	`

	return r.db.QueryRow(
		query,
		checkpoint.Status,
		checkpoint.Progress,
		checkpoint.Notes,
		checkpoint.CompletedAt,
		checkpoint.ID,
		checkpoint.JourneyID,
	).Scan(&checkpoint.UpdatedAt)
}
