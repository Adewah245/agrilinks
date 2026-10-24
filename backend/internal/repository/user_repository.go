package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"backend/internal/models"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lib/pq"
)

const defaultPhotoURL = "/static/assets/images/un.jpeg"

var (
	// ErrPhoneExists means another account already owns this phone number.
	ErrPhoneExists = errors.New("phone number already registered")

	// ErrEmailExists means another account already owns this email address.
	ErrEmailExists = errors.New("email address already registered")

	ErrAccountNotFound          = errors.New("account not found")
	ErrAccountHasBalance        = errors.New("account still has a wallet balance")
	ErrAccountHasPendingRequest = errors.New("account has pending wallet activity")
	ErrAccountHasFundingAccount = errors.New("account has a funding account")
)

// UserRepository provides persistence for user identities and profiles.
//
// Wallet balances, payout bank accounts, deposits, withdrawals, transfers, and
// funding accounts are owned by PaymentRepository. This repository only reads
// the wallet lookup value and refuses to close an account while those payment
// records would strand money.
type UserRepository struct {
	db                       *sql.DB
	configuredAdministrators map[int]bool
}

// NewUserRepository creates a UserRepository using the provided database.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// SetConfiguredAdministrators is called during application setup only.
func (r *UserRepository) SetConfiguredAdministrators(ids []int) {
	r.configuredAdministrators = make(map[int]bool, len(ids))
	for _, id := range ids {
		r.configuredAdministrators[id] = true
	}
}

// uniqueViolationConstraint returns the PostgreSQL constraint involved in a
// unique violation (SQLSTATE 23505), or an empty string for any other error.
func uniqueViolationConstraint(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return pgErr.ConstraintName
	}
	return ""
}

func mapUserUniqueViolation(err error) error {
	switch uniqueViolationConstraint(err) {
	case "users_phone_key":
		return ErrPhoneExists
	case "users_email_key":
		return ErrEmailExists
	default:
		return err
	}
}

// ============================================================
// CREATE USER
// ============================================================

// Create inserts a user and returns the generated id.
//
// The NGN wallet is created by the users trigger installed in migration 043 and
// receives its permanent wallet number from the wallets trigger in migration
// 040. Creating one here as well would duplicate that ownership rule.
func (r *UserRepository) Create(user *models.User) (int, error) {
	return r.create(user, nil)
}

// CreateWithVerifiedPhone consumes a checked signup code and creates the
// account together. A failed insert leaves the code usable for a retry.
func (r *UserRepository) CreateWithVerifiedPhone(user *models.User, proof *models.RegistrationPhoneCode) (int, error) {
	if proof == nil {
		return 0, ErrRegistrationOTPInvalid
	}
	return r.create(user, proof)
}

func (r *UserRepository) create(user *models.User, proof *models.RegistrationPhoneCode) (int, error) {
	if user == nil {
		return 0, fmt.Errorf("user cannot be nil")
	}
	if r == nil || r.db == nil {
		return 0, sql.ErrConnDone
	}

	user.FullName = strings.TrimSpace(user.FullName)
	user.Phone = strings.TrimSpace(user.Phone)
	user.Email = strings.TrimSpace(user.Email)
	user.Location = strings.TrimSpace(user.Location)
	user.LGA = strings.TrimSpace(user.LGA)
	user.State = strings.TrimSpace(user.State)
	user.Country = strings.TrimSpace(user.Country)
	user.Role = strings.ToLower(strings.TrimSpace(user.Role))
	user.AccountType = strings.ToLower(strings.TrimSpace(user.AccountType))
	user.OrganisationName = strings.TrimSpace(user.OrganisationName)
	user.PhotoURL = strings.TrimSpace(user.PhotoURL)

	if user.FullName == "" {
		return 0, fmt.Errorf("user full name is required")
	}
	if user.Phone == "" {
		return 0, fmt.Errorf("user phone is required")
	}
	if strings.TrimSpace(user.PasswordHash) == "" {
		return 0, fmt.Errorf("user password hash is required")
	}
	if user.Role == "" {
		user.Role = "farmer"
	}
	if user.AccountType == "" {
		user.AccountType = "individual"
	}
	if user.Country == "" {
		user.Country = "Nigeria"
	}

	var emailValue any
	if user.Email != "" {
		emailValue = user.Email
	}

	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if proof != nil {
		if err := consumeRegistrationPhone(tx, user.Phone, proof); err != nil {
			return 0, err
		}
	}
	if err = resolveUserLocation(tx, user); err != nil {
		return 0, err
	}
	err = tx.QueryRow(`INSERT INTO users(full_name,phone,email,password_hash,location,lga,state,country,role,account_type,organisation_name,photo_url,state_id,lga_id,community_id)
        VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NULLIF($11,''),$12,$13,$14,$15) RETURNING id`,
		user.FullName, user.Phone, emailValue, user.PasswordHash, user.Location, user.LGA, user.State, user.Country, user.Role, user.AccountType, user.OrganisationName, user.PhotoURL, user.StateID, user.LGAID, user.CommunityID).Scan(&user.ID)
	if err != nil {
		return 0, mapUserUniqueViolation(err)
	}
	if proof != nil {
		if err := tx.QueryRow(`UPDATE users SET phone_verified_at=CURRENT_TIMESTAMP WHERE id=$1 RETURNING phone_verified_at`, user.ID).Scan(&user.PhoneVerifiedAt); err != nil {
			return 0, err
		}
	}
	if err = saveUserFarmingTypes(tx, user.ID, user.FarmingTypes); err != nil {
		return 0, err
	}
	if user.AccountType == "organisation" {
		var organizationID int
		err = tx.QueryRow(`INSERT INTO organizations(name,organization_type,state_id,lga_id,logo_url) VALUES($1,'agricultural_group',$2,$3,$4) RETURNING id`, user.OrganisationName, user.StateID, user.LGAID, user.PhotoURL).Scan(&organizationID)
		if err != nil {
			return 0, err
		}
		if _, err = tx.Exec(`INSERT INTO organization_members(organization_id,user_id,member_role) VALUES($1,$2,'owner')`, organizationID, user.ID); err != nil {
			return 0, err
		}
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return user.ID, nil
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
	COALESCE(is_admin, FALSE) AS is_admin,
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
	updated_at,
 state_id,lga_id,community_id,phone_verified_at
`

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
		&user.IsAdmin,
		&user.OrganisationName,
		&user.PhotoURL,
		pq.Array(&user.FarmingTypes),
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.StateID, &user.LGAID, &user.CommunityID,
		&user.PhoneVerifiedAt,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// ============================================================
// READS
// ============================================================

// GetByPhone returns the active account registered to a phone number.
func (r *UserRepository) GetByPhone(phone string) (*models.User, error) {
	user, err := scanUser(r.db.QueryRow(`
		SELECT `+userSelectColumns+`
		FROM users
		WHERE phone = $1
		AND deleted_at IS NULL
	`, strings.TrimSpace(phone)))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetByID returns the active account with this id.
func (r *UserRepository) GetByID(id int) (*models.User, error) {
	user, err := scanUser(r.db.QueryRow(`
		SELECT `+userSelectColumns+`
		FROM users
		WHERE id = $1
		AND deleted_at IS NULL
	`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if r.configuredAdministrators[id] {
		user.IsAdmin = true
	}
	return user, nil
}

// GetByWalletNumber returns the active member who owns a permanent Agro-Shield
// wallet number. Wallet numbers are generated by the database, not by callers.
func (r *UserRepository) GetByWalletNumber(walletNumber string) (*models.User, error) {
	walletNumber = strings.TrimSpace(walletNumber)
	if walletNumber == "" {
		return nil, nil
	}

	user, err := scanUser(r.db.QueryRow(`
		SELECT `+userSelectColumns+`
		FROM users
		WHERE id IN (
			SELECT owner_id
			FROM wallets
			WHERE owner_type = 'user'
			AND currency = 'NGN'
			AND wallet_number = $1
		)
		AND deleted_at IS NULL
	`, walletNumber))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	user.WalletNumber = walletNumber
	return user, nil
}

// FindByAccountNumber returns users whose saved payout account matches. Bank
// account numbers are not globally unique, so the query stops at two rows and
// callers must treat more than one result as ambiguous.
//
// Bank details are deliberately not selected here; PaymentRepository exposes
// them only through its private payment-profile query.
func (r *UserRepository) FindByAccountNumber(accountNumber string) ([]models.User, error) {
	accountNumber = strings.TrimSpace(accountNumber)
	if accountNumber == "" {
		return []models.User{}, nil
	}

	rows, err := r.db.Query(`
		SELECT `+userSelectColumns+`
		FROM users
		WHERE id IN (
			SELECT user_id
			FROM user_bank_accounts
			WHERE account_number = $1
		)
		AND deleted_at IS NULL
		ORDER BY id
		LIMIT 2
	`, accountNumber)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]models.User, 0, 2)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *user)
	}
	return users, rows.Err()
}

// ListForChat returns active users who can be contacted in general chat.
func (r *UserRepository) ListForChat(excludeID int, search string) ([]models.User, error) {
	search = strings.TrimSpace(search)

	rows, err := r.db.Query(`
		SELECT `+userSelectColumns+`
		FROM users
		WHERE id <> $1
		AND deleted_at IS NULL
		AND (
			$2 = ''
			OR full_name ILIKE '%' || $2 || '%'
			OR role ILIKE '%' || $2 || '%'
			OR COALESCE(location, '') ILIKE '%' || $2 || '%'
			OR COALESCE(state, '') ILIKE '%' || $2 || '%'
			OR COALESCE(lga, '') ILIKE '%' || $2 || '%'
		)
		ORDER BY full_name ASC, id ASC
		LIMIT 50
	`, excludeID, search)
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
	return users, rows.Err()
}

// ============================================================
// PROFILE UPDATES
// ============================================================

// UpdateProfile updates the editable personal details.
func (r *UserRepository) UpdateProfile(id int, fullName, phone, email, location, state, lga, photoURL string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	u := &models.User{ID: id, FullName: fullName, Phone: phone, Email: email, Location: location, State: state, LGA: lga, Country: "Nigeria"}
	if err = resolveUserLocation(tx, u); err != nil {
		return err
	}
	result, err := tx.Exec(`UPDATE users SET full_name=$2,phone=$3,email=NULLIF($4,''),location=$5,state=$6,lga=$7,state_id=$8,lga_id=$9,community_id=$10,photo_url=COALESCE(NULLIF($11,''),photo_url),updated_at=CURRENT_TIMESTAMP WHERE id=$1 AND deleted_at IS NULL`, id, fullName, phone, email, u.Location, u.State, u.LGA, u.StateID, u.LGAID, u.CommunityID, photoURL)
	if err != nil {
		return mapUserUniqueViolation(err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrAccountNotFound
	}
	return tx.Commit()
}

func (r *UserRepository) UpdatePhoto(id int, photoURL string) error {
	result, err := r.db.Exec(`
		UPDATE users
		SET photo_url = $1,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
		AND deleted_at IS NULL
	`, strings.TrimSpace(photoURL), id)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrAccountNotFound
	}
	return nil
}

// UpdatePassword sets a new password hash.
func (r *UserRepository) UpdatePassword(id int, passwordHash string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.Exec(`
		UPDATE users
		SET password_hash = $1,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
		AND deleted_at IS NULL
	`, passwordHash, id)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrAccountNotFound
	}
	if _, err := tx.Exec(`DELETE FROM user_sessions WHERE user_id=$1`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// ============================================================
// FARMING TYPES
// ============================================================

// SaveFarmingTypes records farming activities the user already practises.
func (r *UserRepository) SaveFarmingTypes(userID int, types []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = saveUserFarmingTypes(tx, userID, types); err != nil {
		return err
	}
	return tx.Commit()
}

// ============================================================
// ACCOUNT CLOSURE
// ============================================================

// DeleteAccount closes an account by anonymising the users row rather than
// deleting it. Orders, transactions, deliveries, and wallet history keep a
// stable user id, while profile and chat data are removed.
//
// Closure is refused while money is still held, wallet activity is in flight,
// or a virtual funding account could still receive a transfer. All checks and
// writes run in one transaction so the account is either fully closed or left
// untouched.
func (r *UserRepository) DeleteAccount(ctx context.Context, userID int) error {
	if userID <= 0 {
		return ErrAccountNotFound
	}
	if r == nil || r.db == nil {
		return sql.ErrConnDone
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var id int
	err = tx.QueryRowContext(ctx, `
		SELECT id
		FROM users
		WHERE id = $1
		AND deleted_at IS NULL
		FOR UPDATE
	`, userID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrAccountNotFound
	}
	if err != nil {
		return err
	}

	walletRows, err := tx.QueryContext(ctx, `
		SELECT id, balance::text
		FROM wallets
		WHERE owner_type = 'user'
		AND owner_id = $1
		FOR UPDATE
	`, userID)
	if err != nil {
		return err
	}

	hasBalance := false
	for walletRows.Next() {
		var walletID int
		var balance string
		if err := walletRows.Scan(&walletID, &balance); err != nil {
			_ = walletRows.Close()
			return err
		}

		value, err := strconv.ParseFloat(balance, 64)
		if err != nil {
			_ = walletRows.Close()
			return fmt.Errorf("parse wallet balance: %w", err)
		}
		if value != 0 {
			hasBalance = true
		}
	}
	if err := walletRows.Err(); err != nil {
		_ = walletRows.Close()
		return err
	}
	if err := walletRows.Close(); err != nil {
		return err
	}
	if hasBalance {
		return ErrAccountHasBalance
	}

	var pending int
	if err := tx.QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(*)
			 FROM wallet_deposits d
			 JOIN wallets w ON w.id = d.wallet_id
			 WHERE w.owner_type = 'user'
			   AND w.owner_id = $1
			   AND d.status IN ('pending', 'processing'))
			+
			(SELECT COUNT(*)
			 FROM wallet_withdrawals
			 WHERE user_id = $1
			   AND status IN ('pending', 'processing'))
			+
			(SELECT COUNT(*)
			 FROM wallet_transfers
			 WHERE (sender_id = $1 OR recipient_id = $1)
			   AND status = 'pending')
	`, userID).Scan(&pending); err != nil {
		return err
	}
	if pending > 0 {
		return ErrAccountHasPendingRequest
	}

	var fundingAccounts int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM wallet_virtual_accounts
		WHERE user_id = $1
	`, userID).Scan(&fundingAccounts); err != nil {
		return err
	}
	if fundingAccounts > 0 {
		return ErrAccountHasFundingAccount
	}

	for _, stmt := range []string{
		`DELETE FROM cart_items
		 WHERE buyer_id = $1
		    OR product_id IN (
			    SELECT id FROM products
			    WHERE owner_type = 'user' AND owner_id = $1
		    )`,
		`DELETE FROM negotiations
		 WHERE buyer_id = $1
		    OR product_id IN (
			    SELECT id FROM products
			    WHERE owner_type = 'user' AND owner_id = $1
		    )`,
		`DELETE FROM negotiation_messages WHERE sender_id = $1`,
		`DELETE FROM chat_messages WHERE sender_id = $1`,
		`DELETE FROM conversation_members WHERE user_id = $1`,
		`DELETE FROM notifications WHERE user_id = $1`,
		`DELETE FROM diagnoses WHERE user_id = $1`,
		`DELETE FROM otp_codes WHERE user_id = $1`,
		`DELETE FROM product_favorites WHERE user_id = $1`,
		`DELETE FROM product_reviews WHERE reviewer_id = $1`,
		`DELETE FROM user_reviews WHERE reviewer_id = $1`,
		`DELETE FROM training_enrollments WHERE user_id = $1`,
		`DELETE FROM event_registrations WHERE user_id = $1`,
		`DELETE FROM organization_members WHERE user_id = $1`,
		`DELETE FROM user_bank_accounts WHERE user_id = $1`,
		`DELETE FROM user_verifications WHERE user_id = $1`,
		`DELETE FROM user_farming_types WHERE user_id = $1`,
		`DELETE FROM farming_activities WHERE user_id = $1`,
		`DELETE FROM farm_resources WHERE user_id = $1`,
		`DELETE FROM farm_assets WHERE user_id = $1`,
		`UPDATE market_events SET user_id = NULL WHERE user_id = $1`,
		`UPDATE marketplace_searches SET user_id = NULL WHERE user_id = $1`,
		`UPDATE verification_badges SET issued_by = NULL WHERE issued_by = $1`,
		`UPDATE products
		 SET status = 'inactive', updated_at = CURRENT_TIMESTAMP
		 WHERE owner_type = 'user' AND owner_id = $1`,
	} {
		if _, err := tx.ExecContext(ctx, stmt, userID); err != nil {
			return err
		}
	}

	placeholderPhone := fmt.Sprintf("deleted-%d", userID)
	result, err := tx.ExecContext(ctx, `
		UPDATE users
		SET full_name = 'Deleted account',
			phone = $2,
			email = NULL,
			password_hash = 'deleted',
			location = 'Deleted account',
			lga = NULL,
			state = NULL,
			country = 'Nigeria',
			organisation_name = NULL,
			photo_url = $3,
			is_admin = FALSE,
			deleted_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		AND deleted_at IS NULL
	`, userID, placeholderPhone, defaultPhotoURL)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrAccountNotFound
	}

	return tx.Commit()
}
