package repository

import (
	"backend/internal/models"
	"database/sql"
	"fmt"
	"strings"
)

// DeliveryPartnerRepository persists logistics businesses and the user
// accounts allowed to operate them.
type DeliveryPartnerRepository struct {
	db *sql.DB
}

func NewDeliveryPartnerRepository(db *sql.DB) *DeliveryPartnerRepository {
	return &DeliveryPartnerRepository{db: db}
}

const deliveryPartnerSelectColumns = `
	dp.id,
	dp.organization_id,
	dp.name,
	dp.phone,
	COALESCE(dp.email, ''),
	COALESCE(dp.vehicle_type, ''),
	COALESCE(dp.registration_number, ''),
	dp.state_id,
	dp.lga_id,
	dp.verified,
	dp.status,
	dp.created_at,
	dp.updated_at
`

// Create registers a partner and makes the supplied user its owner. Both
// records are written together so a partner can never be left without a
// controlling account.
func (r *DeliveryPartnerRepository) Create(
	partner *models.DeliveryPartner,
	ownerUserID int,
) error {
	if partner == nil {
		return fmt.Errorf("delivery partner cannot be nil")
	}
	if ownerUserID <= 0 {
		return fmt.Errorf("invalid delivery partner owner")
	}

	partner.Name = strings.TrimSpace(partner.Name)
	partner.Phone = strings.TrimSpace(partner.Phone)
	partner.Email = strings.TrimSpace(partner.Email)
	partner.VehicleType = strings.TrimSpace(partner.VehicleType)
	partner.RegistrationNumber = strings.TrimSpace(partner.RegistrationNumber)

	if partner.Name == "" {
		return fmt.Errorf("delivery partner name is required")
	}
	if partner.Phone == "" {
		return fmt.Errorf("delivery partner phone number is required")
	}
	if partner.StateID == nil && partner.LGAID != nil {
		return fmt.Errorf("choose a state before choosing an LGA")
	}

	if partner.StateID != nil && partner.LGAID != nil {
		var belongs bool
		err := r.db.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM lgas WHERE id = $1 AND state_id = $2
			)
		`, *partner.LGAID, *partner.StateID).Scan(&belongs)
		if err != nil {
			return fmt.Errorf("check delivery partner location: %w", err)
		}
		if !belongs {
			return fmt.Errorf("delivery partner LGA does not belong to the selected state")
		}
	}

	status, err := normalizePartnerStatus(partner.Status)
	if err != nil {
		return err
	}
	partner.Status = status

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin delivery partner registration: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	err = tx.QueryRow(`
		INSERT INTO delivery_partners (
			organization_id,
			name,
			phone,
			email,
			vehicle_type,
			registration_number,
			state_id,
			lga_id,
			verified,
			status
		)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), $7, $8, $9, $10)
		RETURNING id, created_at, updated_at
	`,
		nullableInt(partner.OrganizationID),
		partner.Name,
		partner.Phone,
		partner.Email,
		partner.VehicleType,
		partner.RegistrationNumber,
		nullableInt(partner.StateID),
		nullableInt(partner.LGAID),
		partner.Verified,
		partner.Status,
	).Scan(&partner.ID, &partner.CreatedAt, &partner.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create delivery partner: %w", err)
	}

	_, err = tx.Exec(`
		INSERT INTO delivery_partner_members (delivery_partner_id, user_id, member_role)
		VALUES ($1, $2, 'owner')
	`, partner.ID, ownerUserID)
	if err != nil {
		return fmt.Errorf("add delivery partner owner: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delivery partner registration: %w", err)
	}

	partner.MemberRole = "owner"
	return nil
}

// GetByID returns a partner, including the current member role when the
// partner is loaded for one user account.
func (r *DeliveryPartnerRepository) GetByID(partnerID int) (*models.DeliveryPartner, error) {
	if partnerID <= 0 {
		return nil, fmt.Errorf("invalid delivery partner")
	}

	var partner models.DeliveryPartner
	var organizationID, stateID, lgaID sql.NullInt64

	err := r.db.QueryRow(`
		SELECT `+deliveryPartnerSelectColumns+`
		FROM delivery_partners dp
		WHERE dp.id = $1
	`, partnerID).Scan(
		&partner.ID,
		&organizationID,
		&partner.Name,
		&partner.Phone,
		&partner.Email,
		&partner.VehicleType,
		&partner.RegistrationNumber,
		&stateID,
		&lgaID,
		&partner.Verified,
		&partner.Status,
		&partner.CreatedAt,
		&partner.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get delivery partner: %w", err)
	}

	partner.OrganizationID = intPointer(organizationID)
	partner.StateID = intPointer(stateID)
	partner.LGAID = intPointer(lgaID)
	return &partner, nil
}

// ListForUser returns every partner the user can operate.
func (r *DeliveryPartnerRepository) ListForUser(userID int) ([]models.DeliveryPartner, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user")
	}

	rows, err := r.db.Query(`
		SELECT `+deliveryPartnerSelectColumns+`,
			dpm.member_role
		FROM delivery_partner_members dpm
		JOIN delivery_partners dp ON dp.id = dpm.delivery_partner_id
		WHERE dpm.user_id = $1
		ORDER BY dp.created_at DESC, dp.id DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list user delivery partners: %w", err)
	}
	defer rows.Close()

	partners := make([]models.DeliveryPartner, 0)
	for rows.Next() {
		var partner models.DeliveryPartner
		var organizationID, stateID, lgaID sql.NullInt64

		if err := rows.Scan(
			&partner.ID,
			&organizationID,
			&partner.Name,
			&partner.Phone,
			&partner.Email,
			&partner.VehicleType,
			&partner.RegistrationNumber,
			&stateID,
			&lgaID,
			&partner.Verified,
			&partner.Status,
			&partner.CreatedAt,
			&partner.UpdatedAt,
			&partner.MemberRole,
		); err != nil {
			return nil, fmt.Errorf("scan user delivery partner: %w", err)
		}

		partner.OrganizationID = intPointer(organizationID)
		partner.StateID = intPointer(stateID)
		partner.LGAID = intPointer(lgaID)
		partners = append(partners, partner)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user delivery partners: %w", err)
	}
	return partners, nil
}

// List returns partners for assignment or administration. Callers choose
// whether only active or only approved partners are eligible.
func (r *DeliveryPartnerRepository) List(
	onlyActive bool,
	verifiedOnly bool,
) ([]models.DeliveryPartner, error) {
	rows, err := r.db.Query(`
		SELECT `+deliveryPartnerSelectColumns+`
		FROM delivery_partners dp
		WHERE (NOT $1 OR dp.status = 'active')
		  AND (NOT $2 OR dp.verified)
		ORDER BY dp.name ASC, dp.id ASC
	`, onlyActive, verifiedOnly)
	if err != nil {
		return nil, fmt.Errorf("list delivery partners: %w", err)
	}
	defer rows.Close()

	partners := make([]models.DeliveryPartner, 0)
	for rows.Next() {
		var partner models.DeliveryPartner
		var organizationID, stateID, lgaID sql.NullInt64

		if err := rows.Scan(
			&partner.ID,
			&organizationID,
			&partner.Name,
			&partner.Phone,
			&partner.Email,
			&partner.VehicleType,
			&partner.RegistrationNumber,
			&stateID,
			&lgaID,
			&partner.Verified,
			&partner.Status,
			&partner.CreatedAt,
			&partner.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan delivery partner: %w", err)
		}

		partner.OrganizationID = intPointer(organizationID)
		partner.StateID = intPointer(stateID)
		partner.LGAID = intPointer(lgaID)
		partners = append(partners, partner)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate delivery partners: %w", err)
	}
	return partners, nil
}

// MemberRole returns the user's role on a partner, or sql.ErrNoRows when the
// user does not belong to it.
func (r *DeliveryPartnerRepository) MemberRole(
	partnerID,
	userID int,
) (string, error) {
	if partnerID <= 0 || userID <= 0 {
		return "", fmt.Errorf("invalid delivery partner membership")
	}

	var role string
	err := r.db.QueryRow(`
		SELECT member_role
		FROM delivery_partner_members
		WHERE delivery_partner_id = $1 AND user_id = $2
	`, partnerID, userID).Scan(&role)
	if err == sql.ErrNoRows {
		return "", sql.ErrNoRows
	}
	if err != nil {
		return "", fmt.Errorf("get delivery partner membership: %w", err)
	}
	return role, nil
}

// SetVerified records whether an administrator has approved a partner.
func (r *DeliveryPartnerRepository) SetVerified(partnerID int, verified bool) error {
	if partnerID <= 0 {
		return fmt.Errorf("invalid delivery partner")
	}

	result, err := r.db.Exec(`
		UPDATE delivery_partners
		SET verified = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, verified, partnerID)
	if err != nil {
		return fmt.Errorf("update delivery partner verification: %w", err)
	}
	return requirePartnerUpdated(result)
}

// SetStatus changes whether a partner may receive new deliveries.
func (r *DeliveryPartnerRepository) SetStatus(partnerID int, status string) error {
	if partnerID <= 0 {
		return fmt.Errorf("invalid delivery partner")
	}

	status, err := normalizePartnerStatus(status)
	if err != nil {
		return err
	}

	result, err := r.db.Exec(`
		UPDATE delivery_partners
		SET status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, status, partnerID)
	if err != nil {
		return fmt.Errorf("update delivery partner status: %w", err)
	}
	return requirePartnerUpdated(result)
}

func normalizePartnerStatus(status string) (string, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		return "active", nil
	}

	switch status {
	case "active", "inactive", "suspended":
		return status, nil
	default:
		return "", fmt.Errorf("invalid delivery partner status: %s", status)
	}
}

func requirePartnerUpdated(result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
