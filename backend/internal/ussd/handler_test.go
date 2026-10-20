package ussd

import (
	"context"
	"strings"
	"testing"

	"backend/internal/models"
)

type fakeUsers struct {
	user *models.User
}

func (f fakeUsers) GetByPhone(string) (*models.User, error) {
	return f.user, nil
}

type fakeProducts struct {
	products []models.Product
}

func (f fakeProducts) AvailableProducts() ([]models.Product, error) {
	return f.products, nil
}

func (f fakeProducts) MyProducts(string, int) ([]models.Product, error) {
	return f.products, nil
}

type fakeWeather struct{}

func (fakeWeather) Current(context.Context, string) (models.Weather, error) {
	return models.Weather{
		Available:      true,
		Location:       "Makurdi",
		Temperature:    "30C",
		Summary:        "Clear",
		RainChance:     "10%",
		Recommendation: "Water crops early.",
	}, nil
}

func testHandler() *Handler {
	return &Handler{
		users: fakeUsers{
			user: &models.User{
				ID:       1,
				FullName: "Amina",
				Phone:    "+2341",
				Location: "Makurdi",
			},
		},
		products: fakeProducts{
			products: []models.Product{
				{
					ID:       1,
					Name:     "Maize",
					Price:    500,
					Unit:     "kg",
					Quantity: 100,
					Status:   "active",
				},
			},
		},
		weather: fakeWeather{},
	}
}

func TestMainMenuContinuesSession(t *testing.T) {
	got := testHandler().respond(
		context.Background(),
		"+2341",
		"",
	)

	if !strings.HasPrefix(got, "CON Agro-Shield") {
		t.Fatalf("got %q", got)
	}
}

func TestStorageEndsSession(t *testing.T) {
	got := testHandler().respond(
		context.Background(),
		"+2341",
		"1",
	)

	if !strings.HasPrefix(got, "END Storage:") {
		t.Fatalf("got %q", got)
	}
}

func TestMarketPricesEndSession(t *testing.T) {
	got := testHandler().respond(
		context.Background(),
		"+2341",
		"2",
	)

	if got != "END Marketplace:\n1. Maize ₦500.00/kg" {
		t.Fatalf("got %q", got)
	}
}

func TestUnknownCallerCannotViewProfile(t *testing.T) {
	h := testHandler()

	h.users = fakeUsers{}

	got := h.respond(
		context.Background(),
		"+000",
		"3",
	)

	if !strings.HasPrefix(got, "END This number is not registered") {
		t.Fatalf("got %q", got)
	}
}

func TestUnknownCallerIsRejectedBeforeMenu(t *testing.T) {
	h := testHandler()

	h.users = fakeUsers{}

	got := h.respond(
		context.Background(),
		"+000",
		"",
	)

	if !strings.HasPrefix(got, "END This number is not registered") {
		t.Fatalf("got %q", got)
	}
}
