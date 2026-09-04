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
// IMPORTANT:
//
// The new Agro-Shield architecture uses the users table.
//
// Therefore:
//
// OLD:
// farmer_id
//
// NEW:
// user_id
//
// A diagnosis belongs to the user who submitted it.
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
// Creates a new diagnosis record for a user.
//
// The database generates:
// - ID
// - CreatedAt
// ============================================================

// Create saves a new agricultural diagnosis and updates the
// model with the generated ID and creation timestamp.
func (r *DiagnosisRepository) Create(d *models.Diagnosis) error {
	query := `
		INSERT INTO diagnoses (
			user_id,
			category,
			description,
			result
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`

	return r.db.QueryRow(
		query,
		d.UserID,
		d.Category,
		d.Description,
		d.Result,
	).Scan(
		&d.ID,
		&d.CreatedAt,
	)
}

// ============================================================
// LIST BY USER
//
// Retrieves all diagnoses submitted by a particular user.
//
// Results are ordered from newest to oldest.
// ============================================================

// ListByUser retrieves all diagnoses belonging to a user.
func (r *DiagnosisRepository) ListByUser(
	userID int,
) ([]models.Diagnosis, error) {

	query := `
		SELECT
			id,
			user_id,
			category,
			description,
			result,
			created_at
		FROM diagnoses
		WHERE user_id = $1
		ORDER BY created_at DESC
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
			&d.Category,
			&d.Description,
			&d.Result,
			&d.CreatedAt,
		); err != nil {
			return nil, err
		}

		list = append(
			list,
			d,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return list, nil
}

// ============================================================
// COUNT THIS MONTH
//
// Counts how many diagnoses a user has submitted during
// the current calendar month.
//
// Example:
//
// If today is September 4:
//
// September 1 → included
// September 4 → included
// September 30 → included when reached
//
// August → not included.
// ============================================================

// CountThisMonth returns the number of diagnoses submitted
// by a user during the current calendar month.
func (r *DiagnosisRepository) CountThisMonth(
	userID int,
) (int, error) {

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
// This is important for security.
//
// A user must not be able to delete another user's diagnosis.
// ============================================================

// DeleteForUser deletes a diagnosis belonging to a specific user.
func (r *DiagnosisRepository) DeleteForUser(
	diagnosisID,
	userID int,
) error {

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