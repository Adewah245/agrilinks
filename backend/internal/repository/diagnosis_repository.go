package repository

import (
	"backend/internal/models"
	"database/sql"
	"fmt"
	"strings"
)

// ============================================================
// DIAGNOSIS REPOSITORY
//
// Responsibility:
// - Communicates directly with the diagnoses table.
// - Creates agricultural diagnosis records.
// - Retrieves diagnoses belonging to a user.
// - Counts a user's diagnoses for the current month.
// - Deletes a diagnosis belonging to a user.
//
// CURRENT AGRO-SHIELD ARCHITECTURE:
//
// A diagnosis belongs to a user.
//
// diagnoses:
//
//	id
//	user_id
//	farming_type_id
//	category
//	symptoms
//	result
//	confidence
//	created_at
//	updated_at
//
// IMPORTANT:
//
// The old architecture used:
//
//	description
//
// The new architecture uses:
//
//	symptoms
//
// Do NOT use farmer_id or description here.
// ============================================================

type DiagnosisRepository struct {
	db *sql.DB
}

// ============================================================
// CREATE REPOSITORY
// ============================================================

// NewDiagnosisRepository creates a new repository for
// managing agricultural diagnosis records.
func NewDiagnosisRepository(db *sql.DB) *DiagnosisRepository {
	return &DiagnosisRepository{
		db: db,
	}
}

// ============================================================
// CREATE DIAGNOSIS
//
// Creates a new diagnosis record.
//
// Database-generated fields:
// - id
// - created_at
// - updated_at
// ============================================================

func (r *DiagnosisRepository) Create(d *models.Diagnosis) error {
	if r == nil || r.db == nil {
		return sql.ErrConnDone
	}

	if d == nil {
		return fmt.Errorf("diagnosis cannot be nil")
	}

	if d.UserID <= 0 {
		return fmt.Errorf("invalid diagnosis user")
	}

	if d.FarmingTypeID != nil && *d.FarmingTypeID <= 0 {
		return fmt.Errorf("invalid diagnosis farming type")
	}

	if d.Confidence != nil && (*d.Confidence < 0 || *d.Confidence > 1) {
		return fmt.Errorf("diagnosis confidence must be between 0 and 1")
	}

	d.Category = normalizeDiagnosisText(d.Category)
	d.Symptoms = normalizeDiagnosisText(d.Symptoms)
	d.Result = normalizeDiagnosisText(d.Result)

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin diagnosis transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	query := `
		INSERT INTO diagnoses (
			user_id,
			farming_type_id,
			category,
			symptoms,
			result,
			confidence
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	err = tx.QueryRow(
		query,
		d.UserID,
		d.FarmingTypeID,
		d.Category,
		d.Symptoms,
		d.Result,
		d.Confidence,
	).Scan(
		&d.ID,
		&d.CreatedAt,
		&d.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create diagnosis: %w", err)
	}

	for i := range d.Media {
		media := &d.Media[i]
		if err := normalizeDiagnosisMedia(media); err != nil {
			return err
		}

		if err := tx.QueryRow(`
			INSERT INTO diagnosis_media (
				diagnosis_id,
				media_type,
				media_url,
				image_name
			)
			VALUES ($1, $2, $3, $4)
			RETURNING id, created_at
		`,
			d.ID,
			media.MediaType,
			media.MediaURL,
			media.ImageName,
		).Scan(&media.ID, &media.CreatedAt); err != nil {
			return fmt.Errorf("create diagnosis media: %w", err)
		}

		media.DiagnosisID = d.ID
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit diagnosis: %w", err)
	}

	return nil
}

// ============================================================
// LIST BY USER
//
// Retrieves all diagnoses submitted by a particular user.
//
// Results:
// - newest first
//
// Security:
// - only diagnoses belonging to the supplied user are returned.
// ============================================================

func (r *DiagnosisRepository) ListByUser(
	userID int,
) ([]models.Diagnosis, error) {

	if r == nil || r.db == nil {
		return nil, sql.ErrConnDone
	}

	if userID <= 0 {
		return nil, fmt.Errorf("invalid diagnosis user")
	}

	query := `
		SELECT
			id,
			user_id,
			farming_type_id,
			category,
			symptoms,
			result,
			confidence,
			created_at,
			updated_at
		FROM diagnoses
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC
	`

	rows, err := r.db.Query(
		query,
		userID,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	list := make([]models.Diagnosis, 0)

	for rows.Next() {
		var d models.Diagnosis

		if err := rows.Scan(
			&d.ID,
			&d.UserID,
			&d.FarmingTypeID,
			&d.Category,
			&d.Symptoms,
			&d.Result,
			&d.Confidence,
			&d.CreatedAt,
			&d.UpdatedAt,
		); err != nil {
			return nil, err
		}

		list = append(list, d)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close diagnoses: %w", err)
	}

	if err := r.attachMedia(list); err != nil {
		return nil, fmt.Errorf("load diagnosis media: %w", err)
	}

	return list, nil
}

// ============================================================
// GET BY ID FOR USER
//
// Retrieves one diagnosis only when it belongs to the supplied
// user.
//
// This prevents a user from accessing another user's diagnosis.
// ============================================================

func (r *DiagnosisRepository) GetByIDForUser(
	diagnosisID,
	userID int,
) (*models.Diagnosis, error) {

	if r == nil || r.db == nil {
		return nil, sql.ErrConnDone
	}

	if diagnosisID <= 0 || userID <= 0 {
		return nil, fmt.Errorf("invalid diagnosis request")
	}

	query := `
		SELECT
			id,
			user_id,
			farming_type_id,
			category,
			symptoms,
			result,
			confidence,
			created_at,
			updated_at
		FROM diagnoses
		WHERE id = $1
		  AND user_id = $2
	`

	var d models.Diagnosis

	err := r.db.QueryRow(
		query,
		diagnosisID,
		userID,
	).Scan(
		&d.ID,
		&d.UserID,
		&d.FarmingTypeID,
		&d.Category,
		&d.Symptoms,
		&d.Result,
		&d.Confidence,
		&d.CreatedAt,
		&d.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	diagnoses := []models.Diagnosis{d}
	if err := r.attachMedia(diagnoses); err != nil {
		return nil, fmt.Errorf("load diagnosis media: %w", err)
	}

	return &diagnoses[0], nil
}

// ============================================================
// COUNT THIS MONTH
//
// Counts how many diagnoses a user has submitted during the
// current calendar month.
//
// Example:
//
// If today is September 8:
//
// September 1 → included
// September 8 → included
// Future September dates → naturally excluded
// August → excluded
// ============================================================

func (r *DiagnosisRepository) CountThisMonth(
	userID int,
) (int, error) {

	if r == nil || r.db == nil {
		return 0, sql.ErrConnDone
	}

	if userID <= 0 {
		return 0, fmt.Errorf("invalid diagnosis user")
	}

	query := `
		SELECT COUNT(*)
		FROM diagnoses
		WHERE user_id = $1
		  AND created_at >= date_trunc('month', CURRENT_TIMESTAMP)
		  AND created_at < date_trunc('month', CURRENT_TIMESTAMP)
		      + INTERVAL '1 month'
	`

	var count int

	err := r.db.QueryRow(
		query,
		userID,
	).Scan(&count)

	return count, err
}

// ============================================================
// DELETE DIAGNOSIS
//
// Deletes a diagnosis only when it belongs to the specified user.
//
// Security:
// A user must not be able to delete another user's diagnosis.
// ============================================================

func (r *DiagnosisRepository) DeleteForUser(
	diagnosisID,
	userID int,
) error {

	if r == nil || r.db == nil {
		return sql.ErrConnDone
	}

	if diagnosisID <= 0 || userID <= 0 {
		return fmt.Errorf("invalid diagnosis request")
	}

	result, err := r.db.Exec(`
		DELETE FROM diagnoses
		WHERE id = $1
		  AND user_id = $2
	`,
		diagnosisID,
		userID,
	)

	if err != nil {
		return err
	}

	count, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if count == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// attachMedia loads all media for the supplied diagnoses in one query.
func (r *DiagnosisRepository) attachMedia(diagnoses []models.Diagnosis) error {
	if len(diagnoses) == 0 {
		return nil
	}

	placeholders := make([]string, len(diagnoses))
	args := make([]any, len(diagnoses))
	positions := make(map[int][]int, len(diagnoses))

	for i := range diagnoses {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = diagnoses[i].ID
		positions[diagnoses[i].ID] = append(positions[diagnoses[i].ID], i)
	}

	rows, err := r.db.Query(`
		SELECT
			id,
			diagnosis_id,
			media_type,
			media_url,
			image_name,
			created_at
		FROM diagnosis_media
		WHERE diagnosis_id IN (`+strings.Join(placeholders, ", ")+`)
		ORDER BY diagnosis_id, created_at ASC, id ASC
	`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			media     models.DiagnosisMedia
			imageName sql.NullString
		)

		if err := rows.Scan(
			&media.ID,
			&media.DiagnosisID,
			&media.MediaType,
			&media.MediaURL,
			&imageName,
			&media.CreatedAt,
		); err != nil {
			return err
		}

		if imageName.Valid {
			value := imageName.String
			media.ImageName = &value
		}

		for _, index := range positions[media.DiagnosisID] {
			diagnoses[index].Media = append(diagnoses[index].Media, media)
		}
	}

	return rows.Err()
}

func normalizeDiagnosisText(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func normalizeDiagnosisMedia(media *models.DiagnosisMedia) error {
	if media == nil {
		return fmt.Errorf("diagnosis media cannot be nil")
	}

	media.MediaType = strings.ToLower(strings.TrimSpace(media.MediaType))
	if media.MediaType == "" {
		media.MediaType = "image"
	}

	switch media.MediaType {
	case "image", "video", "audio", "document":
	default:
		return fmt.Errorf("invalid diagnosis media type: %s", media.MediaType)
	}

	media.MediaURL = strings.TrimSpace(media.MediaURL)
	if media.MediaURL == "" {
		return fmt.Errorf("diagnosis media URL is required")
	}

	media.ImageName = normalizeDiagnosisText(media.ImageName)

	return nil
}
