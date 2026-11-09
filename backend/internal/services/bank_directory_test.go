package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBankDirectoryIncludesBanksFintechsAndAllProviderCodes(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != "GET" || r.URL.Path != "/banks/NG" {
			t.Errorf("incorrect directory request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": []Bank{
			{Name: " Access Bank ", Code: "044"}, {Name: "Paycom Nigeria Limited", Code: "100004"},
			{Name: "PalmPay Limited", Code: "palmpay-provider-code"}, {Name: "Kuda Microfinance Bank", Code: "50211"},
			{Name: "TeamApt Microfinance Bank", Code: "50515"}, {Name: "Community Microfinance Bank", Code: "community-provider-code"},
			{Name: "Duplicate Access", Code: "044"}, {Name: "Missing code"}, {Code: "missing-name"},
		}})
	}))
	defer server.Close()
	provider := NewFlutterwaveClient("test-key", "", "")
	provider.SetBaseURL(server.URL)
	service := NewWalletService(nil, nil, provider, nil, "NGN")
	banks, err := service.Banks(t.Context())
	if err != nil || len(banks) != 6 {
		t.Fatalf("directory omitted an institution: %+v %v", banks, err)
	}
	for _, code := range []string{"044", "100004", "palmpay-provider-code", "50211", "50515", "community-provider-code"} {
		found := false
		for _, bank := range banks {
			if bank.Code == code {
				found = true
			}
		}
		if !found {
			t.Fatalf("provider code %s was filtered out", code)
		}
	}
	if banks[0].Name != "Access Bank" {
		t.Fatal("directory should be trimmed and alphabetized")
	}
	banks[0].Name = "modified"
	list, err := service.Banks(t.Context())
	if err != nil || list[0].Name != "Access Bank" || requests != 1 {
		t.Fatalf("directory cache was mutated or fetched twice: %+v %v", list, err)
	}
}
