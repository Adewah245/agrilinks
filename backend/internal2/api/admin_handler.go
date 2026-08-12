package api

import (
	"encoding/json"
	"errors"
	"net/http"

    "backend/internal2/domain"
	"backend/internal2/service"
)

type AdminHandler struct {
	svc *service.PriceService
}

func NewAdminHandler(svc *service.PriceService) *AdminHandler {
	return &AdminHandler{svc: svc}
}

type verifySubmitterRequest struct {
	TrustScore float64 `json:"trust_score"`
}

type submitterResponse struct {
	ID              string  `json:"id"`
	Verified        bool    `json:"verified"`
	TrustScore      float64 `json:"trust_score"`
	TotalReports    int     `json:"total_reports"`
	AcceptedReports int     `json:"accepted_reports"`
}

// HandleVerifySubmitter handles POST /api/v1/admin/submitters/{id}/verify
//
// NOTE: this endpoint has no auth/role check yet — in production this must
// sit behind admin-only authentication before going anywhere near real
// users. It's left open here to keep the module focused on the
// verification/review logic itself.
func (h *AdminHandler) HandleVerifySubmitter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	submitterID := r.PathValue("id")
	if submitterID == "" {
		writeError(w, http.StatusBadRequest, "submitter id is required")
		return
	}

	var req verifySubmitterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.TrustScore <= 0 {
		// Default a sensible starting trust for a newly verified submitter
		// if the caller didn't specify one.
		req.TrustScore = 0.8
	}

	submitter, err := h.svc.VerifySubmitter(r.Context(), submitterID, req.TrustScore)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify submitter")
		return
	}

	writeJSON(w, http.StatusOK, submitterResponse{
		ID:              submitter.ID,
		Verified:        submitter.Verified,
		TrustScore:      submitter.TrustScore,
		TotalReports:    submitter.TotalReports,
		AcceptedReports: submitter.AcceptedReports,
	})
}

type reviewReportRequest struct {
	Decision string `json:"decision"` // "accepted" or "rejected"
	Reason   string `json:"reason"`
}

type reportResponse struct {
	ReportID string `json:"report_id"`
	Status   string `json:"status"`
	Reason   string `json:"reason"`
}

// HandleReviewReport handles POST /api/v1/admin/reports/{id}/review
//
// Same auth caveat as HandleVerifySubmitter above — needs to be locked down
// to admins/moderators before this touches real data.
func (h *AdminHandler) HandleReviewReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	reportID := r.PathValue("id")
	if reportID == "" {
		writeError(w, http.StatusBadRequest, "report id is required")
		return
	}

	var req reviewReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var decision domain.ReportStatus
	switch req.Decision {
	case "accepted":
		decision = domain.StatusAccepted
	case "rejected":
		decision = domain.StatusRejected
	default:
		writeError(w, http.StatusBadRequest, `decision must be "accepted" or "rejected"`)
		return
	}

	updated, err := h.svc.ReviewReport(r.Context(), reportID, decision, req.Reason)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			writeError(w, http.StatusNotFound, "report not found")
		case errors.Is(err, domain.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "failed to review report")
		}
		return
	}

	writeJSON(w, http.StatusOK, reportResponse{
		ReportID: updated.ID,
		Status:   string(updated.Status),
		Reason:   updated.FlagReason,
	})
}
