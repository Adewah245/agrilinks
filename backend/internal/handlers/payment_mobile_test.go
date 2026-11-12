package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMobilePaymentCancellationReturnsWithoutClaimingSettlement(t *testing.T) {
	handler := NewPaymentHandler(nil)
	request := httptest.NewRequest(http.MethodGet, "/payments/callback?app_return=wallet&status=successful", nil)
	response := httptest.NewRecorder()
	handler.CallbackHandler(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `href="agrilinks://wallet"`) || strings.Contains(response.Body.String(), "successful") {
		t.Fatal("an unverified return must only offer navigation")
	}
}

func TestMobilePaymentReturnCannotBypassProviderVerification(t *testing.T) {
	handler := NewPaymentHandler(nil)
	request := httptest.NewRequest(http.MethodGet, "/payments/callback?app_return=wallet&status=successful&tx_ref=forged&transaction_id=123", nil)
	response := httptest.NewRecorder()
	handler.CallbackHandler(response, request)
	if response.Code != http.StatusBadGateway || strings.Contains(response.Body.String(), "agrilinks://") {
		t.Fatal("a forged success must still fail provider verification")
	}
	request = httptest.NewRequest(http.MethodGet, "/payments/callback?app_return=https://other.example&status=successful", nil)
	response = httptest.NewRecorder()
	handler.CallbackHandler(response, request)
	if response.Code != http.StatusBadRequest || response.Header().Get("Location") != "" {
		t.Fatal("a return parameter must not become an open redirect")
	}
}
