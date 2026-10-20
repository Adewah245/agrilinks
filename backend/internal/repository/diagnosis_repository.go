package repository

import (
	"backend/internal/models"
	"database/sql"
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
		return sql.ErrNoRows
	}

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

	return r.db.QueryRow(
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

	return &d, nil
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
