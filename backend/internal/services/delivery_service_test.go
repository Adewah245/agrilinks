package services

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"backend/internal/models"
)

type fakeDeliveryStore struct {
	delivery        *models.Delivery
	orderContext    *models.DeliveryOrderContext
	existing        *models.Delivery
	managed         bool
	created         *models.Delivery
	assignedWith    [2]int
	statusFrom      string
	statusTo        string
	createdTracking *models.DeliveryTracking
}

func (s *fakeDeliveryStore) Create(delivery *models.Delivery) error {
	delivery.ID = 99
	s.created = delivery
	return nil
}

func (s *fakeDeliveryStore) GetByID(int) (*models.Delivery, error) {
	return s.delivery, nil
}

func (s *fakeDeliveryStore) GetByOrder(int) (*models.Delivery, error) {
	return s.existing, nil
}

func (s *fakeDeliveryStore) ListByBuyer(int) ([]models.Delivery, error)       { return nil, nil }
func (s *fakeDeliveryStore) ListForSeller(int) ([]models.Delivery, error)     { return nil, nil }
func (s *fakeDeliveryStore) ListByPartnerUser(int) ([]models.Delivery, error) { return nil, nil }
func (s *fakeDeliveryStore) ListAll() ([]models.Delivery, error)              { return nil, nil }
func (s *fakeDeliveryStore) ListDeliveryOrderOptions(int) ([]models.DeliveryOrderOption, error) {
	return nil, nil
}

func (s *fakeDeliveryStore) GetOrderForBuyerDelivery(int, int) (*models.DeliveryOrderContext, error) {
	return s.orderContext, nil
}

func (s *fakeDeliveryStore) CanManageDelivery(int, int) (bool, error) {
	return s.managed, nil
}

func (s *fakeDeliveryStore) AssignPartner(deliveryID, partnerID int) error {
	s.assignedWith = [2]int{deliveryID, partnerID}
	return nil
}

func (s *fakeDeliveryStore) UpdateStatusFrom(_ int, from, to string) error {
	s.statusFrom = from
	s.statusTo = to
	return nil
}

func (s *fakeDeliveryStore) CreateTracking(update *models.DeliveryTracking) error {
	s.createdTracking = update
	return nil
}

func (s *fakeDeliveryStore) TrackingForBuyer(int, int) ([]models.DeliveryTracking, error) {
	return nil, nil
}

func (s *fakeDeliveryStore) TrackingForDelivery(int) ([]models.DeliveryTracking, error) {
	return nil, nil
}

type fakeDeliveryPartnerStore struct {
	partner      *models.DeliveryPartner
	created      *models.DeliveryPartner
	createdOwner int
	memberRoles  map[string]string
}

func (s *fakeDeliveryPartnerStore) Create(partner *models.DeliveryPartner, ownerUserID int) error {
	partner.ID = 42
	partner.MemberRole = "owner"
	s.created = partner
	s.createdOwner = ownerUserID
	return nil
}

func (s *fakeDeliveryPartnerStore) GetByID(int) (*models.DeliveryPartner, error) {
	return s.partner, nil
}

func (s *fakeDeliveryPartnerStore) ListForUser(int) ([]models.DeliveryPartner, error) {
	return nil, nil
}

func (s *fakeDeliveryPartnerStore) List(bool, bool) ([]models.DeliveryPartner, error) {
	return nil, nil
}

func (s *fakeDeliveryPartnerStore) MemberRole(partnerID, userID int) (string, error) {
	role, ok := s.memberRoles[fmt.Sprintf("%d:%d", partnerID, userID)]
	if !ok {
		return "", sql.ErrNoRows
	}
	return role, nil
}

func (s *fakeDeliveryPartnerStore) SetVerified(int, bool) error { return nil }
func (s *fakeDeliveryPartnerStore) SetStatus(int, string) error { return nil }

func TestRequestDeliveryRejectsUnpaidOrder(t *testing.T) {
	deliveries := &fakeDeliveryStore{orderContext: &models.DeliveryOrderContext{
		OrderID: 11,
		BuyerID: 7,
		Status:  "pending",
	}}
	service := NewDeliveryService(deliveries, &fakeDeliveryPartnerStore{})

	_, err := service.RequestDelivery(&models.User{ID: 7, Role: "buyer"}, 11, "12 Market Road")
	if !errors.Is(err, ErrDeliveryNotReady) {
		t.Fatalf("error = %v, want %v", err, ErrDeliveryNotReady)
	}
}

func TestRequestDeliverySnapshotsOrderAndAddress(t *testing.T) {
	deliveries := &fakeDeliveryStore{orderContext: &models.DeliveryOrderContext{
		OrderID:           11,
		BuyerID:           7,
		Status:            "confirmed",
		PickupAddress:     "Kano market (pickup from Ada Farms)",
		PickupStateID:     intTestPointer(3),
		PickupCommunityID: intTestPointer(5),
	}}
	service := NewDeliveryService(deliveries, &fakeDeliveryPartnerStore{})

	delivery, err := service.RequestDelivery(
		&models.User{ID: 7, Role: "buyer"},
		11,
		"  12 Market Road  ",
	)
	if err != nil {
		t.Fatal(err)
	}
	if deliveries.created == nil {
		t.Fatal("expected delivery to be stored")
	}
	if delivery.DestinationAddress != "12 Market Road" {
		t.Fatalf("destination = %q", delivery.DestinationAddress)
	}
	if delivery.PickupAddress != "Kano market (pickup from Ada Farms)" {
		t.Fatalf("pickup = %q", delivery.PickupAddress)
	}
	if delivery.PickupStateID == nil || *delivery.PickupStateID != 3 {
		t.Fatal("pickup state was not copied from the order")
	}
	if delivery.Status != "pending" || delivery.TrackingCode == "" {
		t.Fatalf("delivery state = %#v", delivery)
	}
}

func TestRequestDeliveryRejectsDuplicate(t *testing.T) {
	deliveries := &fakeDeliveryStore{
		orderContext: &models.DeliveryOrderContext{OrderID: 11, BuyerID: 7, Status: "confirmed"},
		existing:     &models.Delivery{ID: 3, OrderID: 11},
	}
	service := NewDeliveryService(deliveries, &fakeDeliveryPartnerStore{})

	_, err := service.RequestDelivery(&models.User{ID: 7, Role: "buyer"}, 11, "12 Market Road")
	if !errors.Is(err, ErrDeliveryAlreadyExists) {
		t.Fatalf("error = %v, want %v", err, ErrDeliveryAlreadyExists)
	}
	if deliveries.created != nil {
		t.Fatal("duplicate delivery must not be stored")
	}
}

func TestAssignPartnerRequiresApproval(t *testing.T) {
	deliveries := &fakeDeliveryStore{
		delivery: &models.Delivery{ID: 4, BuyerID: 7, Status: "pending"},
		managed:  true,
	}
	partners := &fakeDeliveryPartnerStore{partner: &models.DeliveryPartner{
		ID:       9,
		Status:   "active",
		Verified: false,
	}}
	service := NewDeliveryService(deliveries, partners)

	err := service.AssignPartner(&models.User{ID: 8, Role: "farmer"}, 4, 9)
	if !errors.Is(err, ErrDeliveryPartnerNotApproved) {
		t.Fatalf("error = %v, want %v", err, ErrDeliveryPartnerNotApproved)
	}
	if deliveries.assignedWith != [2]int{} {
		t.Fatal("unapproved partner must not be assigned")
	}
}

func TestAssignedPartnerCanProgressDelivery(t *testing.T) {
	deliveries := &fakeDeliveryStore{delivery: &models.Delivery{
		ID:                4,
		BuyerID:           7,
		Status:            "assigned",
		DeliveryPartnerID: intTestPointer(10),
	}}
	partners := &fakeDeliveryPartnerStore{memberRoles: map[string]string{"10:8": "driver"}}
	service := NewDeliveryService(deliveries, partners)

	if err := service.UpdateStatus(&models.User{ID: 8, Role: "farmer"}, 4, "picked_up"); err != nil {
		t.Fatal(err)
	}
	if deliveries.statusFrom != "assigned" || deliveries.statusTo != "picked_up" {
		t.Fatalf("transition = %s -> %s", deliveries.statusFrom, deliveries.statusTo)
	}
}

func TestPartnerCannotSkipPickup(t *testing.T) {
	deliveries := &fakeDeliveryStore{delivery: &models.Delivery{
		ID:                4,
		BuyerID:           7,
		Status:            "assigned",
		DeliveryPartnerID: intTestPointer(10),
	}}
	partners := &fakeDeliveryPartnerStore{memberRoles: map[string]string{"10:8": "driver"}}
	service := NewDeliveryService(deliveries, partners)

	err := service.UpdateStatus(&models.User{ID: 8, Role: "farmer"}, 4, "delivered")
	if !errors.Is(err, ErrDeliveryForbidden) {
		t.Fatalf("error = %v, want %v", err, ErrDeliveryForbidden)
	}
	if deliveries.statusTo != "" {
		t.Fatal("invalid transition must not reach storage")
	}
}

func TestBuyerCannotWriteTracking(t *testing.T) {
	deliveries := &fakeDeliveryStore{delivery: &models.Delivery{ID: 4, BuyerID: 7, Status: "assigned"}}
	service := NewDeliveryService(deliveries, &fakeDeliveryPartnerStore{})

	err := service.AddTracking(&models.User{ID: 7, Role: "buyer"}, 4, 8.5, 9.2, "Market", "")
	if !errors.Is(err, ErrDeliveryForbidden) {
		t.Fatalf("error = %v, want %v", err, ErrDeliveryForbidden)
	}
	if deliveries.createdTracking != nil {
		t.Fatal("buyer tracking update must not be stored")
	}
}

func TestRegisterPartnerCreatesOwnerAndWaitsForApproval(t *testing.T) {
	partners := &fakeDeliveryPartnerStore{}
	service := NewDeliveryService(&fakeDeliveryStore{}, partners)

	partner, err := service.RegisterPartner(
		&models.User{ID: 8, Role: "farmer"},
		models.DeliveryPartnerInput{Name: "  Swift Farms Logistics ", Phone: " 08012345678 "},
	)
	if err != nil {
		t.Fatal(err)
	}
	if partners.createdOwner != 8 {
		t.Fatalf("owner = %d, want 8", partners.createdOwner)
	}
	if partner.Verified {
		t.Fatal("new partner must wait for administrator approval")
	}
	if partner.MemberRole != "owner" {
		t.Fatalf("member role = %q, want owner", partner.MemberRole)
	}
}

func TestAdministratorCanReviewPartner(t *testing.T) {
	partners := &fakeDeliveryPartnerStore{}
	service := NewDeliveryService(&fakeDeliveryStore{}, partners)

	if err := service.SetPartnerVerified(&models.User{ID: 1, IsAdmin: true}, 42, true); err != nil {
		t.Fatal(err)
	}
	if err := service.SetPartnerVerified(&models.User{ID: 8, Role: "farmer"}, 42, true); !errors.Is(err, ErrDeliveryForbidden) {
		t.Fatalf("non-admin error = %v, want %v", err, ErrDeliveryForbidden)
	}
}

func intTestPointer(value int) *int { return &value }
