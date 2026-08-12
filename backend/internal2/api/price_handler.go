package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"backend/internal2/domain"
	"backend/internal2/service"
)

type PriceHandler struct {
	svc *service.PriceService
}

func NewPriceHandler(svc *service.PriceService) *PriceHandler {
	return &PriceHandler{svc: svc}
}

type submitPriceRequest struct {
	CommodityID string  `json:"commodity_id"`
	MarketID    string  `json:"market_id"`
	SubmitterID string  `json:"submitter_id"`
	Price       float64 `json:"price"`
	Currency    string  `json:"currency"`
	Unit        string  `json:"unit"`
}

type submitPriceResponse struct {
	ReportID string `json:"report_id"`
	Status   string `json:"status"`
	Reason   string `json:"reason"`
}

// HandleSubmitPrice handles POST /api/v1/prices/report
func (h *PriceHandler) HandleSubmitPrice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req submitPriceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.svc.SubmitPrice(r.Context(), service.SubmitPriceInput{
		CommodityID: req.CommodityID,
		MarketID:    req.MarketID,
		SubmitterID: req.SubmitterID,
		Price:       req.Price,
		Currency:    req.Currency,
		Unit:        req.Unit,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidInput), errors.Is(err, domain.ErrInvalidPrice):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "failed to submit price report")
		}
		return
	}

	// Reports that were rejected outright are still "successfully
	// processed" from an HTTP standpoint — 200 with a clear status, not
	// a 4xx/5xx, since the client did nothing wrong.
	writeJSON(w, http.StatusOK, submitPriceResponse{
		ReportID: result.Report.ID,
		Status:   string(result.Report.Status),
		Reason:   result.Reason,
	})
}

type currentPriceResponse struct {
	CommodityID string  `json:"commodity_id"`
	MarketID    string  `json:"market_id"`
	AvgPrice    float64 `json:"avg_price"`
	MinPrice    float64 `json:"min_price"`
	MaxPrice    float64 `json:"max_price"`
	SampleSize  int     `json:"sample_size"`
}

// HandleGetCurrentPrice handles GET /api/v1/prices/current?commodity_id=&market_id=
func (h *PriceHandler) HandleGetCurrentPrice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	commodityID := r.URL.Query().Get("commodity_id")
	marketID := r.URL.Query().Get("market_id")
	if commodityID == "" || marketID == "" {
		writeError(w, http.StatusBadRequest, "commodity_id and market_id are required")
		return
	}

	agg, err := h.svc.GetCurrentPrice(r.Context(), commodityID, marketID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "no accepted price data yet for this commodity/market")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch current price")
		return
	}

	writeJSON(w, http.StatusOK, currentPriceResponse{
		CommodityID: agg.CommodityID,
		MarketID:    agg.MarketID,
		AvgPrice:    agg.AvgPrice,
		MinPrice:    agg.MinPrice,
		MaxPrice:    agg.MaxPrice,
		SampleSize:  agg.SampleSize,
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
