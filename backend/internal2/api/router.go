package api

import "net/http"

func NewRouter(priceHandler *PriceHandler, adminHandler *AdminHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/prices/report", priceHandler.HandleSubmitPrice)
	mux.HandleFunc("/api/v1/prices/current", priceHandler.HandleGetCurrentPrice)

	mux.HandleFunc("POST /api/v1/admin/submitters/{id}/verify", adminHandler.HandleVerifySubmitter)
	mux.HandleFunc("POST /api/v1/admin/reports/{id}/review", adminHandler.HandleReviewReport)
	return mux
}
