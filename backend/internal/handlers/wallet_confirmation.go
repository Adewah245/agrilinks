package handlers

import (
	"database/sql"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"backend/internal/middleware"
	"backend/internal/models"
	"backend/internal/services"
	"github.com/jackc/pgx/v5/pgconn"
)

func walletConfirmationResponse(w http.ResponseWriter, value any, err error) {
	var databaseError *pgconn.PgError
	if errors.As(err, &databaseError) || errors.Is(err, sql.ErrNoRows) || errors.Is(err, sql.ErrConnDone) {
		err = services.ErrSecurityUnavailable
	}
	if errors.Is(err, services.ErrPaymentProvider) {
		err = errors.New("the bank provider is unavailable; please try again")
	}
	securityResponse(w, value, err)
}
func confirmationOwner(w http.ResponseWriter, r *http.Request) (*models.User, string, bool) {
	user, ok := middleware.UserFromContext(r)
	session, _ := middleware.SessionIDFromRequest(r)
	if !ok || user == nil || session == "" {
		http.Error(w, "Please sign in", http.StatusUnauthorized)
		return nil, "", false
	}
	return user, session, true
}
func (h *WalletHandler) ConfirmationStart(w http.ResponseWriter, r *http.Request) {
	user, session, ok := confirmationOwner(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		walletConfirmationResponse(w, nil, errors.New("could not read this confirmation"))
		return
	}
	token, err := h.service.StartConfirmation(r.Context(), user, session, services.WalletConfirmationInput{
		Kind: r.PostFormValue("kind"), Recipient: r.PostFormValue("recipient"), Amount: r.PostFormValue("amount"), Note: r.PostFormValue("note"), PIN: r.PostFormValue("transaction_pin"), Key: r.PostFormValue("request_key"), Quote: r.PostFormValue("quote"),
	})
	walletConfirmationResponse(w, map[string]string{"token": token}, err)
}
func (h *WalletHandler) ConfirmationVerify(w http.ResponseWriter, r *http.Request) {
	user, session, ok := confirmationOwner(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		walletConfirmationResponse(w, nil, errors.New("could not read this code"))
		return
	}
	err := h.service.VerifyConfirmation(r.Context(), user.ID, session, r.PostFormValue("token"), r.PostFormValue("code"))
	walletConfirmationResponse(w, map[string]bool{"verified": err == nil}, err)
}
func (h *WalletHandler) ConfirmationComplete(w http.ResponseWriter, r *http.Request) {
	user, session, ok := confirmationOwner(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, (2<<20)+(64<<10))
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		walletConfirmationResponse(w, nil, errors.New("the selfie upload is too large or invalid"))
		return
	}
	defer r.MultipartForm.RemoveAll()
	f, _, err := r.FormFile("selfie")
	if err != nil {
		walletConfirmationResponse(w, nil, errors.New("take a live selfie to continue"))
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, (2<<20)+1))
	if err != nil {
		walletConfirmationResponse(w, nil, errors.New("could not read your selfie"))
		return
	}
	evidence := models.WithdrawalEvidence{}
	for name, dest := range map[string]*float64{"latitude": &evidence.Latitude, "longitude": &evidence.Longitude, "accuracy": &evidence.AccuracyM} {
		value, err := strconv.ParseFloat(r.PostFormValue(name), 64)
		if err != nil {
			walletConfirmationResponse(w, nil, errors.New("capture your current location"))
			return
		}
		*dest = value
	}
	evidence.CapturedAt, err = time.Parse(time.RFC3339Nano, r.PostFormValue("captured_at"))
	if err != nil {
		walletConfirmationResponse(w, nil, errors.New("capture a fresh selfie and location"))
		return
	}
	result, err := h.service.CompleteConfirmation(r.Context(), user, session, r.PostFormValue("token"), data, evidence)
	walletConfirmationResponse(w, result, err)
}
