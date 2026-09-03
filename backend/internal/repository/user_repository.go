package repository

import (
	"backend/internal/models"
	"database/sql"
	"strings"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

func (r *UserRepository) Create(user *models.User) (int, error) {
	query := `
		INSERT INTO users
		(
			full_name,
			phone,
			email,
			password_hash,
			location,
			lga,
			state,
			country,
			role,
			account_type,
			nin,
			verification_status,
			organisation_name,
			photo_url
		)
		VALUES
		(
			$1,$2,$3,$4,
			$5,$6,$7,$8,
			$9,$10,$11,$12,
			$13,$14
		)
		RETURNING id
	`

	var id int

	err := r.db.QueryRow(
		query,

		user.FullName,
		user.Phone,
		user.Email,
		user.PasswordHash,

		user.Location,
		user.LGA,
		user.State,
		user.Country,

		user.Role,
		user.AccountType,

		user.NIN,
		user.VerificationStatus,

		user.OrganisationName,

		user.PhotoURL,
	).Scan(&id)

	return id, err
}

func (r *UserRepository) GetByPhone(phone string) (*models.User, error) {
	query := `
		SELECT
			id,
			full_name,
			phone,
			COALESCE(email, '') AS email,
			password_hash,
			location,
			COALESCE(lga, location),
			COALESCE(state, ''),
			COALESCE(country, 'Nigeria'),
			role,
			COALESCE(photo_url, '') AS photo_url,
			created_at,
			updated_at
		FROM users
		WHERE phone = $1
	`

	var user models.User

	err := r.db.QueryRow(query, phone).Scan(
		&user.ID,
		&user.FullName,
		&user.Phone,
		&user.Email,
		&user.PasswordHash,
		&user.Location,
		&user.LGA,
		&user.State,
		&user.Country,
		&user.Role,
		&user.PhotoURL,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetByID(id int) (*models.User, error) {
	query := `
		SELECT
			id,
			full_name,
			phone,
			COALESCE(email, '') AS email,
			password_hash,
			location,
			COALESCE(lga, location),
			COALESCE(state, ''),
			COALESCE(country, 'Nigeria'),
			role,
			COALESCE(photo_url, '') AS photo_url,
			created_at,
			updated_at
		FROM users
		WHERE id = $1
	`

	var user models.User

	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.FullName,
		&user.Phone,
		&user.Email,
		&user.PasswordHash,
		&user.Location,
		&user.LGA,
		&user.State,
		&user.Country,
		&user.Role,
		&user.PhotoURL,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// ListForChat returns registered users that can be contacted in general chat.
func (r *UserRepository) ListForChat(excludeID int, search string) ([]models.User, error) {
	rows, err := r.db.Query(`
		SELECT
			id,
			full_name,
			phone,
			COALESCE(email, ''),
			password_hash,
			location,
			COALESCE(lga, location),
			COALESCE(state, ''),
			COALESCE(country, 'Nigeria'),
			role,
			COALESCE(photo_url, ''),
			created_at,
			updated_at
		FROM users
		WHERE id <> $1
		AND (
			$2 = ''
			OR full_name ILIKE '%' || $2 || '%'
			OR role ILIKE '%' || $2 || '%'
			OR location ILIKE '%' || $2 || '%'
		)
		ORDER BY full_name ASC
	`, excludeID, strings.TrimSpace(search))

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var users []models.User

	for rows.Next() {
		var user models.User

		if err := rows.Scan(
			&user.ID,
			&user.FullName,
			&user.Phone,
			&user.Email,
			&user.PasswordHash,
			&user.Location,
			&user.LGA,
			&user.State,
			&user.Country,
			&user.Role,
			&user.PhotoURL,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, err
		}

		users = append(users, user)
	}

	return users, rows.Err()
}

// UpdateProfile updates the editable personal details.
func (r *UserRepository) UpdateProfile(
	id int,
	fullName,
	phone,
	email,
	location string,
) error {
	query := `
		UPDATE users
		SET
			full_name = $1,
			phone = $2,
			email = $3,
			location = $4,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $5
	`

	_, err := r.db.Exec(
		query,
		fullName,
		phone,
		email,
		location,
		id,
	)

	return err
}

// UpdatePhoto sets the user's profile photograph URL.
func (r *UserRepository) UpdatePhoto(id int, photoURL string) error {
	_, err := r.db.Exec(
		`UPDATE users
		 SET photo_url = $1,
		     updated_at = CURRENT_TIMESTAMP
		 WHERE id = $2`,
		photoURL,
		id,
	)

	return err
}

// UpdatePassword sets a new password hash.
func (r *UserRepository) UpdatePassword(id int, passwordHash string) error {
	_, err := r.db.Exec(
		`UPDATE users
		 SET password_hash = $1,
		     updated_at = CURRENT_TIMESTAMP
		 WHERE id = $2`,
		passwordHash,
		id,
	)

	return err
}

// SaveFarmingTypes connects a user to their selected farming types.
func (r *UserRepository) SaveFarmingTypes(
	userID int,
	farmingTypes []string,
) error {
	if len(farmingTypes) == 0 {
		return nil
	}

	query := `
		INSERT INTO user_farming_types (user_id, farming_type_id)
		SELECT $1, id
		FROM farming_types
		WHERE slug = $2
		ON CONFLICT (user_id, farming_type_id) DO NOTHING
	`

	for _, farmingType := range farmingTypes {
		_, err := r.db.Exec(
			query,
			userID,
			farmingType,
		)

		if err != nil {
			return err
		}
	}

	return nil
}
