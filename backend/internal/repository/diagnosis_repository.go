package repository

import (
	"backend/internal/models"
	"database/sql"
)

type DiagnosisRepository struct {
	db *sql.DB
}

func NewDiagnosisRepository(db *sql.DB) *DiagnosisRepository {
	return &DiagnosisRepository{db: db}
}

func (r *DiagnosisRepository) Create(d *models.Diagnosis) error {
	query := `INSERT INTO diagnoses (farmer_id, image_name, result) VALUES ($1, $2, $3)`
	_, err := r.db.Exec(query, d.FarmerID, d.ImageName, d.Result)
	return err
}

func (r *DiagnosisRepository) ListByFarmer(farmerID int) ([]models.Diagnosis, error) {
	query := `
	SELECT id, farmer_id, image_name, result, created_at
	FROM diagnoses WHERE farmer_id = $1 ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query, farmerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Diagnosis
	for rows.Next() {
		var d models.Diagnosis
		if err := rows.Scan(&d.ID, &d.FarmerID, &d.ImageName, &d.Result, &d.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

func (r *DiagnosisRepository) CountThisMonth(farmerID int) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM diagnoses
		WHERE farmer_id = $1
		  AND created_at >= date_trunc('month', CURRENT_TIMESTAMP)
		  AND created_at < date_trunc('month', CURRENT_TIMESTAMP) + INTERVAL '1 month'
	`

	var count int

	err := r.db.QueryRow(query, farmerID).Scan(&count)

	return count, err
}
