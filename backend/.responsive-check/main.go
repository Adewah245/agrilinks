package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"backend/internal/render"
)

var fields = regexp.MustCompile(`\.?([A-Z][A-Za-z0-9]*)(?:\.[A-Z][A-Za-z0-9]*)*`)
var paths = regexp.MustCompile(`(?:\$)?\.([A-Z][A-Za-z0-9]*(?:\.[A-Z][A-Za-z0-9]*)*)`)
var ranges = regexp.MustCompile(`{{\s*range\s+(?:\$)?\.([A-Z][A-Za-z0-9.]*)\s*}}`)

func value(key string) any {
	switch key {
	case "CreatedAt", "UpdatedAt", "ExpiresAt", "LastMessageAt", "JoinedAt", "LastUpdated", "IssuedAt":
		return time.Date(2026, 10, 27, 14, 30, 0, 0, time.UTC)
	case "ID", "UserID", "OwnerID", "SenderID", "BuyerID", "SellerID", "ProductID", "OrderID", "OtherUserID", "CurrentUserID", "ConversationID", "SelectedType", "SelectedID", "RoundCount", "MaxRounds":
		return 1
	case "Quantity", "Price", "PricePerUnit", "OfferPrice", "AgreedPrice", "TotalPrice", "TotalAmount", "Amount", "Balance", "AveragePrice", "AverageRating", "AverageOfferPrice", "Latitude", "Longitude", "Progress", "ReadinessScore":
		return float64(1250)
	case "Error", "Message", "Notice", "EditProduct":
		return nil
	case "IsBuyer", "IsRead", "IsOrganisation", "IsMinistry":
		return false
	case "Status":
		return "active"
	case "CategoryText":
		return "Crops"
	case "Category":
		return "crops"
	case "Unit", "ProductUnit":
		return "kg"
	case "Currency":
		return "NGN"
	case "ImageURL", "PhotoURL", "PrimaryImageURL", "SellerPhotoURL", "BuyerPhotoURL", "OtherPhotoURL":
		return "/static/assets/images/Farmer1.jpeg"
	case "Email", "SellerEmail", "BuyerEmail":
		return "farmer@example.test"
	case "Phone", "SellerPhone", "BuyerPhone":
		return "2348012345678"
	case "FullName", "Name", "SellerName", "BuyerName", "OtherName", "ReviewerName", "RecipientName", "SenderName":
		return "Sample Agricultural Cooperative"
	case "ProductName", "ProductNames":
		return "Fresh harvest from the community farm"
	case "Location", "State", "SellerLocation", "BuyerLocation", "PickupAddress", "DestinationAddress":
		return "Community market, Makurdi, Benue State"
	case "Reference", "TrackingCode":
		return "agroshield-reference-123456789012345678901234567890"
	case "ConversationType":
		return "private"
	case "MessageType":
		return "offer"
	case "OfferStatus":
		return "pending"
	case "SenderType":
		return "ai"
	}
	if strings.HasPrefix(key, "Is") || strings.HasSuffix(key, "Available") || key == "Joined" || key == "Ready" || key == "Verified" || key == "Seen" {
		return true
	}
	if strings.HasSuffix(key, "Count") || strings.HasSuffix(key, "Listings") || strings.HasSuffix(key, "Orders30") || strings.HasSuffix(key, "Negotiations30") || key == "Views" || key == "Orders" || key == "CartAdds" || key == "UnreadNotifications" {
		return 3
	}
	if strings.Contains(key, "Price") || strings.Contains(key, "Amount") || strings.HasPrefix(key, "Revenue") || key == "Total" { return float64(1250) }
	return "Sample farming information for this page"
}

func put(data map[string]any, path string, v any) {
	parts := strings.Split(path, ".")
	for _, key := range parts[:len(parts)-1] {
		next, ok := data[key].(map[string]any)
		if !ok {
			next = map[string]any{}
			data[key] = next
		}
		data = next
	}
	data[parts[len(parts)-1]] = v
}

func main() {
	files, _ := filepath.Glob("../frontend/pages/*.html")
	_ = os.MkdirAll("/tmp/agrilinks-responsive/pages", 0755)
	for _, path := range files {
		text, _ := os.ReadFile(path)
		data := map[string]any{}
		for _, match := range fields.FindAllStringSubmatch(string(text), -1) {
			data[match[1]] = value(match[1])
		}
		for _, match := range paths.FindAllStringSubmatch(string(text), -1) {
			parts := strings.Split(match[1], ".")
			if strings.Contains(match[1], ".Format") || strings.Contains(match[1], ".UTC") { continue }
			put(data, match[1], value(parts[len(parts)-1]))
		}
		item := map[string]any{}
		for key, v := range data {
			item[key] = v
		}
		for _, match := range ranges.FindAllStringSubmatch(string(text), -1) {
			put(data, match[1], []any{item, item, item})
			put(item, match[1], []any{map[string]any{"ImageURL": "/static/assets/images/Farmer1.jpeg", "IsPrimary": true, "Name": "Crop Farming", "ID": 1, "Quantity": float64(12), "ProductName": "Maize"}})
		}
		data["HealthMetrics"] = []any{item, item, item, item}
		data["EditProduct"] = nil
		data["FarmingType"] = map[string]any{"Name": "Crop Farming"}
		item["FarmingType"] = data["FarmingType"]
		if journey, ok := data["Journey"].(map[string]any); ok { journey["FarmingType"] = data["FarmingType"] }
		if filepath.Base(path) == "dashboard.html" || filepath.Base(path) == "profile.html" {
			data["FarmingTypes"] = []string{"Crop Farming", "Poultry Farming"}
		}

		if filepath.Base(path) == "product.html" {
			put(data, "Product.Images", []any{item, item, item})
			put(data, "SellerTrust.Badges", []any{item})
		}
		var output bytes.Buffer
		if err := render.TMPL.ExecuteTemplate(&output, filepath.Base(path), data); err != nil {
			fmt.Printf("FAIL %s: %v\n", filepath.Base(path), err)
			continue
		}
		_ = os.WriteFile(filepath.Join("/tmp/agrilinks-responsive/pages", filepath.Base(path)), output.Bytes(), 0644)
		fmt.Println("OK", filepath.Base(path))
	}
}
