package repository

import (
	"backend/internal/models"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// DeliveryRepository owns delivery records and their tracking history.
//
// A delivery belongs to an order, not directly to a buyer, seller, or product.
// Buyer and seller access is therefore derived through orders and order_items.
type DeliveryRepository struct {
	db *sql.DB
}

func NewDeliveryRepository(db *sql.DB) *DeliveryRepository {
	return &DeliveryRepository{db: db}
}

const deliverySelectColumns = `
	d.id,
	d.order_id,
	o.buyer_id,
	d.delivery_partner_id,
	d.pickup_address,
	d.pickup_state_id,
	d.pickup_lga_id,
	d.pickup_community_id,
	d.destination_address,
	d.destination_state_id,
	d.destination_lga_id,
	d.destination_community_id,
	d.delivery_fee,
	d.distance_km,
	d.status,
	COALESCE(d.tracking_code, ''),
	d.picked_up_at,
	d.delivered_at,
	d.created_at,
	d.updated_at,
	COALESCE(p.name, ''),
	COALESCE(b.full_name, '')
`

const deliveryJoins = `
	FROM deliveries d
	JOIN orders o ON o.id = d.order_id
	LEFT JOIN users b ON b.id = o.buyer_id
	LEFT JOIN delivery_partners p ON p.id = d.delivery_partner_id
`

// Create inserts a delivery for an order and sets its generated fields.
func (r *DeliveryRepository) Create(delivery *models.Delivery) error {
	if delivery == nil {
		return fmt.Errorf("delivery cannot be nil")
	}
	if delivery.OrderID <= 0 {
		return fmt.Errorf("invalid delivery order")
	}

	delivery.PickupAddress = strings.TrimSpace(delivery.PickupAddress)
	if delivery.PickupAddress == "" {
		return fmt.Errorf("pickup address is required")
	}

	delivery.DestinationAddress = strings.TrimSpace(delivery.DestinationAddress)
	if delivery.DestinationAddress == "" {
		return fmt.Errorf("destination address is required")
	}

	status, err := normalizeDeliveryStatus(delivery.Status)
	if err != nil {
		return err
	}
	delivery.Status = status

	if delivery.DeliveryFee < 0 {
		return fmt.Errorf("delivery fee cannot be negative")
	}
	if delivery.DistanceKM != nil && *delivery.DistanceKM < 0 {
		return fmt.Errorf("delivery distance cannot be negative")
	}
	if delivery.DeliveryPartnerID != nil && *delivery.DeliveryPartnerID <= 0 {
		return fmt.Errorf("invalid delivery partner")
	}
	if err := validateDeliveryLocationIDs(delivery); err != nil {
		return err
	}

	if err := prepareDeliveryTimestamps(delivery); err != nil {
		return err
	}

	return r.db.QueryRow(`
		INSERT INTO deliveries (
			order_id,
			delivery_partner_id,
			pickup_address,
			pickup_state_id,
			pickup_lga_id,
			pickup_community_id,
			destination_address,
			destination_state_id,
			destination_lga_id,
			destination_community_id,
			delivery_fee,
			distance_km,
			status,
			tracking_code,
			picked_up_at,
			delivered_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14, $15, $16
		)
		RETURNING id, created_at, updated_at
	`,
		delivery.OrderID,
		nullableInt(delivery.DeliveryPartnerID),
		delivery.PickupAddress,
		nullableInt(delivery.PickupStateID),
		nullableInt(delivery.PickupLGAID),
		nullableInt(delivery.PickupCommunityID),
		delivery.DestinationAddress,
		nullableInt(delivery.DestinationStateID),
		nullableInt(delivery.DestinationLGAID),
		nullableInt(delivery.DestinationCommunityID),
		delivery.DeliveryFee,
		nullableFloat(delivery.DistanceKM),
		delivery.Status,
		nullableString(delivery.TrackingCode),
		nullableTime(delivery.PickedUpAt),
		nullableTime(delivery.DeliveredAt),
	).Scan(&delivery.ID, &delivery.CreatedAt, &delivery.UpdatedAt)
}

// GetByID returns one delivery, or nil when it does not exist.
func (r *DeliveryRepository) GetByID(deliveryID int) (*models.Delivery, error) {
	if deliveryID <= 0 {
		return nil, fmt.Errorf("invalid delivery ID")
	}

	var delivery models.Delivery

	err := scanDelivery(r.db.QueryRow(`
		SELECT `+deliverySelectColumns+deliveryJoins+`
		WHERE d.id = $1
	`, deliveryID), &delivery)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get delivery: %w", err)
	}

	return &delivery, nil
}

// GetByOrder returns the delivery for an order, or nil when none exists.
func (r *DeliveryRepository) GetByOrder(orderID int) (*models.Delivery, error) {
	if orderID <= 0 {
		return nil, fmt.Errorf("invalid order ID")
	}

	var delivery models.Delivery

	err := scanDelivery(r.db.QueryRow(`
		SELECT `+deliverySelectColumns+deliveryJoins+`
		WHERE d.order_id = $1
	`, orderID), &delivery)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get order delivery: %w", err)
	}

	return &delivery, nil
}

// ListByBuyer returns deliveries for orders owned by the supplied buyer.
func (r *DeliveryRepository) ListByBuyer(buyerID int) ([]models.Delivery, error) {
	if buyerID <= 0 {
		return nil, fmt.Errorf("invalid buyer ID")
	}

	return r.queryDeliveries(`
		SELECT `+deliverySelectColumns+deliveryJoins+`
		WHERE o.buyer_id = $1
		ORDER BY d.created_at DESC, d.id DESC
	`, buyerID)
}

// ListByOwner returns deliveries containing products owned by a user or
// organization. DISTINCT is required because an order may contain several
// products from the same owner, while deliveries are unique per order.
func (r *DeliveryRepository) ListByOwner(ownerType string, ownerID int) ([]models.Delivery, error) {
	var err error
	ownerType, err = normalizeOwnerType(ownerType)
	if err != nil {
		return nil, err
	}
	if ownerID <= 0 {
		return nil, fmt.Errorf("invalid product owner")
	}

	return r.queryDeliveries(`
		SELECT DISTINCT `+deliverySelectColumns+deliveryJoins+`
		JOIN order_items oi ON oi.order_id = d.order_id
		JOIN products pr ON pr.id = oi.product_id
		WHERE pr.owner_type = $1
		AND pr.owner_id = $2
		ORDER BY d.created_at DESC, d.id DESC
	`, ownerType, ownerID)
}

// ListByPartner returns deliveries assigned to one logistics provider.
func (r *DeliveryRepository) ListByPartner(partnerID int) ([]models.Delivery, error) {
	if partnerID <= 0 {
		return nil, fmt.Errorf("invalid delivery partner")
	}

	return r.queryDeliveries(`
		SELECT `+deliverySelectColumns+deliveryJoins+`
		WHERE d.delivery_partner_id = $1
		ORDER BY d.created_at DESC, d.id DESC
	`, partnerID)
}

// UpdateStatus moves a delivery through its lifecycle and records the matching
// timestamps. The database status constraint remains the final authority.
func (r *DeliveryRepository) UpdateStatus(deliveryID int, status string) error {
	if deliveryID <= 0 {
		return fmt.Errorf("invalid delivery ID")
	}

	status, err := normalizeDeliveryStatus(status)
	if err != nil {
		return err
	}

	result, err := r.db.Exec(`
		UPDATE deliveries
		SET
			status = $1,
			picked_up_at = CASE
				WHEN $1 IN ('picked_up', 'in_transit', 'delivered')
				THEN COALESCE(picked_up_at, CURRENT_TIMESTAMP)
				ELSE picked_up_at
			END,
			delivered_at = CASE
				WHEN $1 = 'delivered'
				THEN COALESCE(delivered_at, CURRENT_TIMESTAMP)
				ELSE delivered_at
			END,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, status, deliveryID)
	if err != nil {
		return fmt.Errorf("update delivery status: %w", err)
	}

	return requireDeliveryUpdated(result)
}

// AssignPartner assigns or reassigns a logistics provider.
func (r *DeliveryRepository) AssignPartner(deliveryID, partnerID int) error {
	if deliveryID <= 0 {
		return fmt.Errorf("invalid delivery ID")
	}
	if partnerID <= 0 {
		return fmt.Errorf("invalid delivery partner")
	}

	result, err := r.db.Exec(`
		UPDATE deliveries
		SET delivery_partner_id = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, partnerID, deliveryID)
	if err != nil {
		return fmt.Errorf("assign delivery partner: %w", err)
	}

	return requireDeliveryUpdated(result)
}

// UpdateTracking updates the public tracking code for a delivery.
func (r *DeliveryRepository) UpdateTracking(deliveryID int, trackingCode string) error {
	if deliveryID <= 0 {
		return fmt.Errorf("invalid delivery ID")
	}

	result, err := r.db.Exec(`
		UPDATE deliveries
		SET tracking_code = NULLIF(BTRIM($1), ''), updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, strings.TrimSpace(trackingCode), deliveryID)
	if err != nil {
		return fmt.Errorf("update delivery tracking: %w", err)
	}

	return requireDeliveryUpdated(result)
}

// MarkDelivered completes a delivery and records its delivery time.
func (r *DeliveryRepository) MarkDelivered(deliveryID int) error {
	return r.UpdateStatus(deliveryID, "delivered")
}

// MarkCancelled closes a delivery that will not be fulfilled. The schema has
// no separate failed status, so cancellation is the terminal failure state.
func (r *DeliveryRepository) MarkCancelled(deliveryID int) error {
	return r.UpdateStatus(deliveryID, "cancelled")
}

// CreateTracking appends one immutable location update.
func (r *DeliveryRepository) CreateTracking(update *models.DeliveryTracking) error {
	if update == nil {
		return fmt.Errorf("delivery tracking cannot be nil")
	}
	if update.DeliveryID <= 0 {
		return fmt.Errorf("invalid delivery ID")
	}
	if update.Latitude < -90 || update.Latitude > 90 {
		return fmt.Errorf("invalid tracking latitude")
	}
	if update.Longitude < -180 || update.Longitude > 180 {
		return fmt.Errorf("invalid tracking longitude")
	}

	update.LocationName = strings.TrimSpace(update.LocationName)
	update.Note = strings.TrimSpace(update.Note)

	return r.db.QueryRow(`
		INSERT INTO delivery_tracking (
			delivery_id,
			latitude,
			longitude,
			location_name,
			note
		)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''))
		RETURNING id, created_at
	`,
		update.DeliveryID,
		update.Latitude,
		update.Longitude,
		update.LocationName,
		update.Note,
	).Scan(&update.ID, &update.CreatedAt)
}

// TrackingForBuyer returns tracking history only when the delivery belongs to
// an order owned by the supplied buyer.
func (r *DeliveryRepository) TrackingForBuyer(buyerID, deliveryID int) ([]models.DeliveryTracking, error) {
	if buyerID <= 0 || deliveryID <= 0 {
		return nil, fmt.Errorf("invalid delivery tracking request")
	}

	rows, err := r.db.Query(`
		SELECT
			t.id,
			t.delivery_id,
			t.latitude,
			t.longitude,
			COALESCE(t.location_name, ''),
			COALESCE(t.note, ''),
			t.created_at
		FROM delivery_tracking t
		JOIN deliveries d ON d.id = t.delivery_id
		JOIN orders o ON o.id = d.order_id
		WHERE t.delivery_id = $1
		AND o.buyer_id = $2
		ORDER BY t.created_at DESC, t.id DESC
	`, deliveryID, buyerID)
	if err != nil {
		return nil, fmt.Errorf("list delivery tracking: %w", err)
	}
	defer rows.Close()

	tracking := make([]models.DeliveryTracking, 0)

	for rows.Next() {
		var update models.DeliveryTracking

		if err := rows.Scan(
			&update.ID,
			&update.DeliveryID,
			&update.Latitude,
			&update.Longitude,
			&update.LocationName,
			&update.Note,
			&update.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan delivery tracking: %w", err)
		}

		tracking = append(tracking, update)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate delivery tracking: %w", err)
	}

	return tracking, nil
}

func (r *DeliveryRepository) queryDeliveries(query string, args ...any) ([]models.Delivery, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list deliveries: %w", err)
	}
	defer rows.Close()

	deliveries := make([]models.Delivery, 0)

	for rows.Next() {
		var delivery models.Delivery
		if err := scanDelivery(rows, &delivery); err != nil {
			return nil, fmt.Errorf("scan delivery: %w", err)
		}
		deliveries = append(deliveries, delivery)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate deliveries: %w", err)
	}

	return deliveries, nil
}

type deliveryScanner interface {
	Scan(dest ...any) error
}

func scanDelivery(scanner deliveryScanner, delivery *models.Delivery) error {
	var (
		partnerID, pickupStateID, pickupLGAID, pickupCommunityID sql.NullInt64
		destinationStateID, destinationLGAID                     sql.NullInt64
		destinationCommunityID                                   sql.NullInt64
		distanceKM                                               sql.NullFloat64
		pickedUpAt, deliveredAt                                  sql.NullTime
	)

	if err := scanner.Scan(
		&delivery.ID,
		&delivery.OrderID,
		&delivery.BuyerID,
		&partnerID,
		&delivery.PickupAddress,
		&pickupStateID,
		&pickupLGAID,
		&pickupCommunityID,
		&delivery.DestinationAddress,
		&destinationStateID,
		&destinationLGAID,
		&destinationCommunityID,
		&delivery.DeliveryFee,
		&distanceKM,
		&delivery.Status,
		&delivery.TrackingCode,
		&pickedUpAt,
		&deliveredAt,
		&delivery.CreatedAt,
		&delivery.UpdatedAt,
		&delivery.PartnerName,
		&delivery.BuyerName,
	); err != nil {
		return err
	}

	delivery.DeliveryPartnerID = intPointer(partnerID)
	delivery.PickupStateID = intPointer(pickupStateID)
	delivery.PickupLGAID = intPointer(pickupLGAID)
	delivery.PickupCommunityID = intPointer(pickupCommunityID)
	delivery.DestinationStateID = intPointer(destinationStateID)
	delivery.DestinationLGAID = intPointer(destinationLGAID)
	delivery.DestinationCommunityID = intPointer(destinationCommunityID)
	delivery.DistanceKM = floatPointer(distanceKM)
	delivery.PickedUpAt = timePointer(pickedUpAt)
	delivery.DeliveredAt = timePointer(deliveredAt)

	return nil
}

func normalizeDeliveryStatus(status string) (string, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		return "pending", nil
	}

	switch status {
	case "pending", "assigned", "picked_up", "in_transit", "delivered", "cancelled":
		return status, nil
	default:
		return "", fmt.Errorf("invalid delivery status: %s", status)
	}
}

func validateDeliveryLocationIDs(delivery *models.Delivery) error {
	ids := []struct {
		name  string
		value *int
	}{
		{"pickup state", delivery.PickupStateID},
		{"pickup LGA", delivery.PickupLGAID},
		{"pickup community", delivery.PickupCommunityID},
		{"destination state", delivery.DestinationStateID},
		{"destination LGA", delivery.DestinationLGAID},
		{"destination community", delivery.DestinationCommunityID},
	}

	for _, id := range ids {
		if id.value != nil && *id.value <= 0 {
			return fmt.Errorf("invalid %s", id.name)
		}
	}

	return nil
}

func prepareDeliveryTimestamps(delivery *models.Delivery) error {
	now := time.Now()

	switch delivery.Status {
	case "picked_up", "in_transit", "delivered":
		if delivery.PickedUpAt == nil {
			delivery.PickedUpAt = &now
		}
	default:
		delivery.PickedUpAt = nil
	}

	if delivery.Status == "delivered" {
		if delivery.DeliveredAt == nil {
			delivery.DeliveredAt = delivery.PickedUpAt
		}
		if delivery.PickedUpAt != nil && delivery.DeliveredAt.Before(*delivery.PickedUpAt) {
			return fmt.Errorf("delivery time cannot be before pickup time")
		}
	} else {
		delivery.DeliveredAt = nil
	}

	return nil
}

func requireDeliveryUpdated(result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func nullableString(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableFloat(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

func intPointer(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	result := int(value.Int64)
	return &result
}

func floatPointer(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	result := value.Float64
	return &result
}

func timePointer(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}
