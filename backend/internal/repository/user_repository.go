package repository

import (
	"backend/internal/models"
	"database/sql"
	"strings"

	"github.com/lib/pq"
)

// UserRepository provides database operations for users.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new UserRepository using the provided database.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

// ============================================================
// CREATE USER
// ============================================================

func (r *UserRepository) Create(user *models.User) (int, error) {
	email := strings.TrimSpace(user.Email)

	var emailValue interface{}
	if email != "" {
		emailValue = email
	}

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
			organisation_name,
			photo_url
		)
		VALUES
		(
			$1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, $10, $11, $12
		)
		RETURNING id
	`

	var id int

	err := r.db.QueryRow(
		query,
		strings.TrimSpace(user.FullName),
		strings.TrimSpace(user.Phone),
		emailValue,
		user.PasswordHash,
		strings.TrimSpace(user.Location),
		strings.TrimSpace(user.LGA),
		strings.TrimSpace(user.State),
		strings.TrimSpace(user.Country),
		strings.TrimSpace(user.Role),
		strings.TrimSpace(user.AccountType),
		strings.TrimSpace(user.OrganisationName),
		strings.TrimSpace(user.PhotoURL),
	).Scan(&id)

	return id, err
}

// ============================================================
// COMMON USER SELECT COLUMNS
// ============================================================

const userSelectColumns = `
	id,
	full_name,
	phone,
	COALESCE(email, '') AS email,
	password_hash,
	COALESCE(location, '') AS location,
	COALESCE(lga, '') AS lga,
	COALESCE(state, '') AS state,
	COALESCE(country, 'Nigeria') AS country,
	COALESCE(role, 'farmer') AS role,
	COALESCE(account_type, 'individual') AS account_type,
	COALESCE(organisation_name, '') AS organisation_name,
	COALESCE(photo_url, '') AS photo_url,
	COALESCE(
		(
			SELECT ARRAY_AGG(ft.slug ORDER BY ft.name)
			FROM user_farming_types uft
			JOIN farming_types ft
				ON ft.id = uft.farming_type_id
			WHERE uft.user_id = users.id
		),
		ARRAY[]::TEXT[]
	) AS farming_types,
	created_at,
	updated_at
`

// ============================================================
// SCAN USER
//
// PostgreSQL arrays are scanned through pq.Array() so that
// ARRAY_AGG(...) is correctly converted into []string.
// ============================================================

func scanUser(scanner interface {
	Scan(dest ...any) error
}) (*models.User, error) {

	var user models.User

	err := scanner.Scan(
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
		&user.AccountType,
		&user.OrganisationName,
		&user.PhotoURL,
		pq.Array(&user.FarmingTypes),
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// ============================================================
// GET USER BY PHONE
// ============================================================

func (r *UserRepository) GetByPhone(phone string) (*models.User, error) {

	query := `
		SELECT
			` + userSelectColumns + `
		FROM users
		WHERE phone = $1
	`

	user, err := scanUser(
		r.db.QueryRow(
			query,
			strings.TrimSpace(phone),
		),
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return user, nil
}

// ============================================================
// GET USER BY ID
// ============================================================

func (r *UserRepository) GetByID(id int) (*models.User, error) {

	query := `
		SELECT
			` + userSelectColumns + `
		FROM users
		WHERE id = $1
	`

	user, err := scanUser(
		r.db.QueryRow(query, id),
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return user, nil
}

// ============================================================
// LIST USERS FOR CHAT
// ============================================================

func (r *UserRepository) ListForChat(
	excludeID int,
	search string,
) ([]models.User, error) {

	search = strings.TrimSpace(search)

	query := `
		SELECT
			` + userSelectColumns + `
		FROM users
		WHERE id <> $1
		AND (
			$2 = ''
			OR full_name ILIKE '%' || $2 || '%'
			OR role ILIKE '%' || $2 || '%'
			OR location ILIKE '%' || $2 || '%'
			OR COALESCE(state, '') ILIKE '%' || $2 || '%'
			OR COALESCE(lga, '') ILIKE '%' || $2 || '%'
		)
		ORDER BY full_name ASC
	`

	rows, err := r.db.Query(
		query,
		excludeID,
		search,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	users := make([]models.User, 0)

	for rows.Next() {

		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}

		users = append(users, *user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

// ============================================================
// UPDATE PROFILE
// ============================================================

func (r *UserRepository) UpdateProfile(
	id int,
	fullName string,
	phone string,
	email string,
	location string,
) error {

	email = strings.TrimSpace(email)

	var emailValue interface{}
	if email != "" {
		emailValue = email
	}

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
		strings.TrimSpace(fullName),
		strings.TrimSpace(phone),
		emailValue,
		strings.TrimSpace(location),
		id,
	)

	return err
}

// ============================================================
// UPDATE PROFILE PHOTO
// ============================================================

func (r *UserRepository) UpdatePhoto(
	id int,
	photoURL string,
) error {

	_, err := r.db.Exec(
		`UPDATE users
		 SET
			photo_url = $1,
			updated_at = CURRENT_TIMESTAMP
		 WHERE id = $2`,
		strings.TrimSpace(photoURL),
		id,
	)

	return err
}

// ============================================================
// UPDATE PASSWORD
// ============================================================

func (r *UserRepository) UpdatePassword(
	id int,
	passwordHash string,
) error {

	_, err := r.db.Exec(
		`UPDATE users
		 SET
			password_hash = $1,
			updated_at = CURRENT_TIMESTAMP
		 WHERE id = $2`,
		passwordHash,
		id,
	)

	return err
}

// ============================================================
// SAVE FARMING TYPES
//
// These represent farming activities the user already practices.
//
// They are NOT the farming type the user wants to learn.
// ============================================================

func (r *UserRepository) SaveFarmingTypes(
	userID int,
	farmingTypes []string,
) error {

	query := `
		INSERT INTO user_farming_types (
			user_id,
			farming_type_id
		)
		SELECT
			$1,
			id
		FROM farming_types
		WHERE slug = $2
		ON CONFLICT (user_id, farming_type_id)
		DO NOTHING
	`

	for _, farmingType := range farmingTypes {

		farmingType = strings.TrimSpace(
			strings.ToLower(farmingType),
		)

		if farmingType == "" {
			continue
		}

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
