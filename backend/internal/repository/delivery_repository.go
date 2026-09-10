package repository

import (
	"backend/internal/models"
	"database/sql"
	"fmt"
)

type DeliveryRepository struct{ db *sql.DB }

func NewDeliveryRepository(db *sql.DB) *DeliveryRepository { return &DeliveryRepository{db: db} }

// ListByBuyer returns only deliveries for orders owned by the supplied buyer.
func (r *DeliveryRepository) ListByBuyer(buyerID int) ([]models.Delivery, error) {
	rows, err := r.db.Query(`SELECT d.id,d.order_id,d.delivery_partner_id,d.pickup_address,d.destination_address,d.delivery_fee,d.distance_km,d.status,COALESCE(d.tracking_code,''),d.picked_up_at,d.delivered_at,d.created_at,d.updated_at,COALESCE(p.name,'') FROM deliveries d JOIN orders o ON o.id=d.order_id LEFT JOIN delivery_partners p ON p.id=d.delivery_partner_id WHERE o.buyer_id=$1 ORDER BY d.created_at DESC`, buyerID)
	if err != nil {
		return nil, fmt.Errorf("list deliveries: %w", err)
	}
	defer rows.Close()
	var out []models.Delivery
	for rows.Next() {
		var d models.Delivery
		if err := rows.Scan(&d.ID, &d.OrderID, &d.DeliveryPartnerID, &d.PickupAddress, &d.DestinationAddress, &d.DeliveryFee, &d.DistanceKM, &d.Status, &d.TrackingCode, &d.PickedUpAt, &d.DeliveredAt, &d.CreatedAt, &d.UpdatedAt, &d.PartnerName); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
func (r *DeliveryRepository) TrackingForBuyer(buyerID, deliveryID int) ([]models.DeliveryTracking, error) {
	rows, err := r.db.Query(`SELECT t.id,t.delivery_id,t.latitude,t.longitude,COALESCE(t.location_name,''),COALESCE(t.note,''),t.created_at FROM delivery_tracking t JOIN deliveries d ON d.id=t.delivery_id JOIN orders o ON o.id=d.order_id WHERE t.delivery_id=$1 AND o.buyer_id=$2 ORDER BY t.created_at DESC`, deliveryID, buyerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.DeliveryTracking
	for rows.Next() {
		var t models.DeliveryTracking
		if err := rows.Scan(&t.ID, &t.DeliveryID, &t.Latitude, &t.Longitude, &t.LocationName, &t.Note, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
