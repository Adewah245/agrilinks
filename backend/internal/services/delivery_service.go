package services

import (
	"backend/internal/models"
	"backend/internal/repository"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrDeliveryNotFound           = errors.New("delivery not found")
	ErrDeliveryForbidden          = errors.New("you cannot make that delivery change")
	ErrDeliveryAlreadyExists      = repository.ErrDeliveryAlreadyExists
	ErrDeliveryNotReady           = errors.New("this order is not ready for delivery")
	ErrDeliveryStatusConflict     = errors.New("this delivery was already updated")
	ErrDeliveryAddressRequired    = errors.New("enter the delivery address")
	ErrDeliveryPartnerNotApproved = errors.New("choose an approved, active delivery partner")
	ErrDeliveryAssignmentClosed   = errors.New("a partner can only be assigned before pickup")
	ErrDeliveryTrackingClosed     = errors.New("tracking is closed for this delivery")
)

// deliveryStore is the persistence boundary used by DeliveryService.
type deliveryStore interface {
	Create(*models.Delivery) error
	GetByID(int) (*models.Delivery, error)
	GetByOrder(int) (*models.Delivery, error)
	ListByBuyer(int) ([]models.Delivery, error)
	ListForSeller(int) ([]models.Delivery, error)
	ListByPartnerUser(int) ([]models.Delivery, error)
	ListAll() ([]models.Delivery, error)
	ListDeliveryOrderOptions(int) ([]models.DeliveryOrderOption, error)
	GetOrderForBuyerDelivery(int, int) (*models.DeliveryOrderContext, error)
	CanManageDelivery(int, int) (bool, error)
	AssignPartner(int, int) error
	UpdateStatusFrom(int, string, string) error
	CreateTracking(*models.DeliveryTracking) error
	TrackingForBuyer(int, int) ([]models.DeliveryTracking, error)
	TrackingForDelivery(int) ([]models.DeliveryTracking, error)
}

type deliveryPartnerStore interface {
	Create(*models.DeliveryPartner, int) error
	GetByID(int) (*models.DeliveryPartner, error)
	ListForUser(int) ([]models.DeliveryPartner, error)
	List(bool, bool) ([]models.DeliveryPartner, error)
	MemberRole(int, int) (string, error)
	SetVerified(int, bool) error
	SetStatus(int, string) error
}

// DeliveryService owns delivery requests, partner assignment, progress, and
// tracking authorization. Buyers, sellers, and partner accounts all pass
// through the same rules instead of relying on UI visibility.
type DeliveryService struct {
	deliveries deliveryStore
	partners   deliveryPartnerStore
}

func NewDeliveryService(
	deliveries deliveryStore,
	partners deliveryPartnerStore,
) *DeliveryService {
	return &DeliveryService{deliveries: deliveries, partners: partners}
}

// MyDeliveries returns the authenticated buyer's deliveries.
func (s *DeliveryService) MyDeliveries(buyerID int) ([]models.Delivery, error) {
	if buyerID <= 0 {
		return nil, errors.New("invalid buyer")
	}
	return s.deliveries.ListByBuyer(buyerID)
}

// MyTracking preserves the buyer-specific tracking boundary used by the
// existing deliveries page.
func (s *DeliveryService) MyTracking(buyerID, deliveryID int) ([]models.DeliveryTracking, error) {
	if buyerID <= 0 || deliveryID <= 0 {
		return nil, errors.New("invalid delivery request")
	}
	return s.deliveries.TrackingForBuyer(buyerID, deliveryID)
}

// DeliveryTracking returns tracking history after checking that the viewer is
// the buyer, a seller in the order, a partner member, or an administrator.
func (s *DeliveryService) DeliveryTracking(
	viewer *models.User,
	deliveryID int,
) ([]models.DeliveryTracking, error) {
	allowed, err := s.CanView(viewer, deliveryID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrDeliveryForbidden
	}
	return s.deliveries.TrackingForDelivery(deliveryID)
}

// CanView reports whether the viewer may see a delivery and its tracking.
func (s *DeliveryService) CanView(viewer *models.User, deliveryID int) (bool, error) {
	if viewer == nil || viewer.ID <= 0 || deliveryID <= 0 {
		return false, ErrDeliveryForbidden
	}

	delivery, err := s.deliveries.GetByID(deliveryID)
	if err != nil {
		return false, err
	}
	if delivery == nil {
		return false, ErrDeliveryNotFound
	}
	if viewer.IsAdministrator() || delivery.BuyerID == viewer.ID {
		return true, nil
	}

	managed, err := s.deliveries.CanManageDelivery(viewer.ID, deliveryID)
	if err != nil {
		return false, err
	}
	if managed {
		return true, nil
	}

	if delivery.DeliveryPartnerID == nil {
		return false, nil
	}
	_, member, err := s.partnerMembership(viewer.ID, *delivery.DeliveryPartnerID)
	if err != nil {
		return false, err
	}
	return member, nil
}

// EligibleOrders lists paid orders for which the buyer has not requested a
// delivery yet.
func (s *DeliveryService) EligibleOrders(buyerID int) ([]models.DeliveryOrderOption, error) {
	if buyerID <= 0 {
		return nil, errors.New("invalid buyer")
	}
	return s.deliveries.ListDeliveryOrderOptions(buyerID)
}

// RequestDelivery creates the address snapshot and first delivery state for a
// paid order. It is deliberately separate from payment: a buyer should not be
// asked to arrange delivery before the order is confirmed.
func (s *DeliveryService) RequestDelivery(
	buyer *models.User,
	orderID int,
	destinationAddress string,
) (*models.Delivery, error) {
	if buyer == nil || buyer.ID <= 0 || orderID <= 0 {
		return nil, ErrDeliveryForbidden
	}

	destinationAddress = strings.TrimSpace(destinationAddress)
	if destinationAddress == "" {
		return nil, ErrDeliveryAddressRequired
	}

	order, err := s.deliveries.GetOrderForBuyerDelivery(orderID, buyer.ID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrDeliveryNotFound
	}
	if !deliveryEligibleOrderStatus(order.Status) {
		return nil, ErrDeliveryNotReady
	}

	existing, err := s.deliveries.GetByOrder(orderID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrDeliveryAlreadyExists
	}

	trackingCode, err := newDeliveryTrackingCode()
	if err != nil {
		return nil, err
	}

	delivery := &models.Delivery{
		OrderID:            orderID,
		BuyerID:            buyer.ID,
		PickupAddress:      order.PickupAddress,
		PickupStateID:      order.PickupStateID,
		PickupLGAID:        order.PickupLGAID,
		PickupCommunityID:  order.PickupCommunityID,
		DestinationAddress: destinationAddress,
		Status:             "pending",
		TrackingCode:       trackingCode,
	}
	if err := s.deliveries.Create(delivery); err != nil {
		return nil, err
	}
	return delivery, nil
}

// SellerDeliveries returns deliveries for sales owned directly by the user or
// through one of their organizations.
func (s *DeliveryService) SellerDeliveries(userID int) ([]models.Delivery, error) {
	if userID <= 0 {
		return nil, errors.New("invalid seller")
	}
	return s.deliveries.ListForSeller(userID)
}

// PartnerDeliveries returns deliveries assigned to any partner the user can
// operate.
func (s *DeliveryService) PartnerDeliveries(userID int) ([]models.Delivery, error) {
	if userID <= 0 {
		return nil, errors.New("invalid delivery partner account")
	}
	return s.deliveries.ListByPartnerUser(userID)
}

// AllDeliveries is available only to administrators.
func (s *DeliveryService) AllDeliveries(actor *models.User) ([]models.Delivery, error) {
	if actor == nil || !actor.IsAdministrator() {
		return nil, ErrDeliveryForbidden
	}
	return s.deliveries.ListAll()
}

// PartnerAccounts lists the delivery partner accounts operated by a user.
func (s *DeliveryService) PartnerAccounts(userID int) ([]models.DeliveryPartner, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user")
	}
	return s.partners.ListForUser(userID)
}

// RegisterPartner creates a partner business owned by the signed-in user.
func (s *DeliveryService) RegisterPartner(
	user *models.User,
	input models.DeliveryPartnerInput,
) (*models.DeliveryPartner, error) {
	if user == nil || user.ID <= 0 {
		return nil, ErrDeliveryForbidden
	}

	partner := &models.DeliveryPartner{
		Name:               input.Name,
		Phone:              input.Phone,
		Email:              input.Email,
		VehicleType:        input.VehicleType,
		RegistrationNumber: input.RegistrationNumber,
		StateID:            input.StateID,
		LGAID:              input.LGAID,
		Status:             "active",
		Verified:           false,
	}
	if err := s.partners.Create(partner, user.ID); err != nil {
		return nil, err
	}
	return partner, nil
}

// AvailablePartners returns approved, active partners that a seller can assign.
func (s *DeliveryService) AvailablePartners() ([]models.DeliveryPartner, error) {
	return s.partners.List(true, true)
}

// PartnersForReview lists all partners for an administrator.
func (s *DeliveryService) PartnersForReview(actor *models.User) ([]models.DeliveryPartner, error) {
	if actor == nil || !actor.IsAdministrator() {
		return nil, ErrDeliveryForbidden
	}
	return s.partners.List(false, false)
}

// SetPartnerVerified lets an administrator approve or revoke a partner.
func (s *DeliveryService) SetPartnerVerified(
	actor *models.User,
	partnerID int,
	verified bool,
) error {
	if actor == nil || !actor.IsAdministrator() {
		return ErrDeliveryForbidden
	}
	if partnerID <= 0 {
		return ErrDeliveryNotFound
	}
	return s.partners.SetVerified(partnerID, verified)
}

// SetPartnerStatus lets an administrator activate, pause, or suspend a partner.
func (s *DeliveryService) SetPartnerStatus(
	actor *models.User,
	partnerID int,
	status string,
) error {
	if actor == nil || !actor.IsAdministrator() {
		return ErrDeliveryForbidden
	}
	if partnerID <= 0 {
		return ErrDeliveryNotFound
	}
	return s.partners.SetStatus(partnerID, status)
}

// AssignPartner assigns one approved partner to a delivery. Only a seller in
// the order or an administrator may assign it.
func (s *DeliveryService) AssignPartner(
	actor *models.User,
	deliveryID,
	partnerID int,
) error {
	delivery, err := s.manageableDelivery(actor, deliveryID)
	if err != nil {
		return err
	}
	if delivery.Status != "pending" && delivery.Status != "assigned" {
		return ErrDeliveryAssignmentClosed
	}

	partner, err := s.partners.GetByID(partnerID)
	if err != nil {
		return err
	}
	if partner == nil {
		return ErrDeliveryNotFound
	}
	if partner.Status != "active" || !partner.Verified {
		return ErrDeliveryPartnerNotApproved
	}

	if err := s.deliveries.AssignPartner(deliveryID, partnerID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrDeliveryStatusConflict
		}
		return err
	}
	return nil
}

// UpdateStatus moves a delivery along its lifecycle. Assignment itself is a
// separate operation because it also selects a partner.
func (s *DeliveryService) UpdateStatus(
	actor *models.User,
	deliveryID int,
	status string,
) error {
	delivery, err := s.deliveries.GetByID(deliveryID)
	if err != nil {
		return err
	}
	if delivery == nil {
		return ErrDeliveryNotFound
	}
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		return errors.New("choose the next delivery step")
	}

	allowed, err := s.canChangeStatus(actor, delivery, status)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrDeliveryForbidden
	}

	if err := s.deliveries.UpdateStatusFrom(deliveryID, delivery.Status, status); err != nil {
		if errors.Is(err, repository.ErrDeliveryStatusConflict) {
			return ErrDeliveryStatusConflict
		}
		return err
	}
	return nil
}

// AddTracking appends a location update. Only the assigned partner's members
// and administrators may add tracking; buyers read it but cannot write it.
func (s *DeliveryService) AddTracking(
	actor *models.User,
	deliveryID int,
	latitude,
	longitude float64,
	locationName,
	note string,
) error {
	delivery, err := s.deliveries.GetByID(deliveryID)
	if err != nil {
		return err
	}
	if delivery == nil {
		return ErrDeliveryNotFound
	}
	if delivery.Status == "delivered" || delivery.Status == "cancelled" {
		return ErrDeliveryTrackingClosed
	}

	allowed := actor != nil && actor.IsAdministrator()
	if !allowed && actor != nil && delivery.DeliveryPartnerID != nil {
		_, member, membershipErr := s.partnerMembership(actor.ID, *delivery.DeliveryPartnerID)
		if membershipErr != nil {
			return membershipErr
		}
		allowed = member
	}
	if !allowed {
		return ErrDeliveryForbidden
	}

	return s.deliveries.CreateTracking(&models.DeliveryTracking{
		DeliveryID:   deliveryID,
		Latitude:     latitude,
		Longitude:    longitude,
		LocationName: locationName,
		Note:         note,
	})
}

func (s *DeliveryService) manageableDelivery(
	actor *models.User,
	deliveryID int,
) (*models.Delivery, error) {
	if actor == nil || actor.ID <= 0 || deliveryID <= 0 {
		return nil, ErrDeliveryForbidden
	}
	delivery, err := s.deliveries.GetByID(deliveryID)
	if err != nil {
		return nil, err
	}
	if delivery == nil {
		return nil, ErrDeliveryNotFound
	}
	if actor.IsAdministrator() {
		return delivery, nil
	}

	managed, err := s.deliveries.CanManageDelivery(actor.ID, deliveryID)
	if err != nil {
		return nil, err
	}
	if !managed {
		return nil, ErrDeliveryForbidden
	}
	return delivery, nil
}

func (s *DeliveryService) canChangeStatus(
	actor *models.User,
	delivery *models.Delivery,
	status string,
) (bool, error) {
	if actor == nil || actor.ID <= 0 {
		return false, ErrDeliveryForbidden
	}
	if !validDeliveryTransition(delivery.Status, status) {
		return false, nil
	}
	if actor.IsAdministrator() {
		return true, nil
	}

	if status == "cancelled" && delivery.BuyerID == actor.ID && delivery.Status == "pending" {
		return true, nil
	}

	managed, err := s.deliveries.CanManageDelivery(actor.ID, delivery.ID)
	if err != nil {
		return false, err
	}
	if managed && status == "cancelled" && (delivery.Status == "pending" || delivery.Status == "assigned") {
		return true, nil
	}

	if delivery.DeliveryPartnerID != nil {
		_, member, err := s.partnerMembership(actor.ID, *delivery.DeliveryPartnerID)
		if err != nil {
			return false, err
		}
		if member && partnerCanSetStatus(delivery.Status, status) {
			return true, nil
		}
	}

	return false, nil
}

func (s *DeliveryService) partnerMembership(userID, partnerID int) (string, bool, error) {
	role, err := s.partners.MemberRole(partnerID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return role, true, nil
}

func deliveryEligibleOrderStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "confirmed", "processing", "ready_for_delivery", "shipped":
		return true
	default:
		return false
	}
}

func validDeliveryTransition(from, to string) bool {
	next, ok := deliveryTransitions[from]
	return ok && next[to]
}

func partnerCanSetStatus(from, to string) bool {
	switch from {
	case "assigned":
		return to == "picked_up"
	case "picked_up":
		return to == "in_transit" || to == "delivered"
	case "in_transit":
		return to == "delivered"
	default:
		return false
	}
}

var deliveryTransitions = map[string]map[string]bool{
	"pending":    {"assigned": true, "cancelled": true},
	"assigned":   {"picked_up": true, "cancelled": true},
	"picked_up":  {"in_transit": true, "delivered": true, "cancelled": true},
	"in_transit": {"delivered": true, "cancelled": true},
}

func newDeliveryTrackingCode() (string, error) {
	random := make([]byte, 6)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("create delivery tracking code: %w", err)
	}
	return "AGD-" + strings.ToUpper(hex.EncodeToString(random)), nil
}
