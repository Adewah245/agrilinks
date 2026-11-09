package integration_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/routes"
	"backend/internal/testutil"
)

func TestWalletRecipientLookupWithPostgres(t *testing.T) {
	db, _ := testutil.Database(t, true)
	c := application(t, db, testConfig(t))
	sender := fixtureUser(t, c, 1, "buyer")
	payee := fixtureUser(t, c, 2, "farmer")
	other := fixtureUser(t, c, 3, "farmer")
	if _, err := db.Exec(`UPDATE users SET photo_url='/static/uploads/profiles/payee.jpg' WHERE id=$1`, payee.ID); err != nil {
		t.Fatal(err)
	}
	var wallet string
	if err := db.QueryRow(`SELECT wallet_number FROM wallets WHERE owner_type='user' AND owner_id=$1 AND currency='NGN'`, payee.ID).Scan(&wallet); err != nil {
		t.Fatal(err)
	}
	handler := routes.NewHandler(c)
	cookie := session(t, c, sender)
	lookup := func(input string, status int) *models.WalletRecipient {
		t.Helper()
		response := request(handler, "POST", "/wallet/recipient", url.Values{"recipient": {input}}, cookie)
		requireStatus(t, response, status)
		if response.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("recipient details must not be cached")
		}
		if status != http.StatusOK {
			return nil
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(response.Body.Bytes(), &fields); err != nil {
			t.Fatal(err)
		}
		if len(fields) != 5 {
			t.Fatalf("unexpected recipient fields: %s", response.Body.String())
		}
		var recipient models.WalletRecipient
		if err := json.Unmarshal(response.Body.Bytes(), &recipient); err != nil {
			t.Fatal(err)
		}
		if recipient.UserID != payee.ID || recipient.FullName != payee.FullName || recipient.WalletNumber != wallet || recipient.Currency != "NGN" {
			t.Fatalf("incorrect recipient: %+v", recipient)
		}
		return &recipient
	}
	for _, input := range []string{strings.ToLower(wallet), strconv.Itoa(payee.ID), payee.Phone, "0" + payee.Phone[3:], "+234 " + payee.Phone[3:6] + " " + payee.Phone[6:], payee.Phone[3:]} {
		if recipient := lookup(input, http.StatusOK); recipient.PhotoURL != "/static/uploads/profiles/payee.jpg" {
			t.Fatalf("public profile picture missing: %+v", recipient)
		}
	}
	lookup(strconv.Itoa(sender.ID), http.StatusBadRequest)
	lookup("1 OR 1=1", http.StatusBadRequest)
	lookup("AGW-U-99999999-NGN", http.StatusNotFound)
	lookup("", http.StatusBadRequest)
	for _, photo := range []string{"/private/verification/7", "/static/uploads/users/selfie.jpg"} {
		if _, err := db.Exec(`UPDATE users SET photo_url=$2 WHERE id=$1`, payee.ID, photo); err != nil {
			t.Fatal(err)
		}
		if recipient := lookup(wallet, http.StatusOK); recipient.PhotoURL != "/static/assets/images/un.jpeg" {
			t.Fatalf("private photo exposed: %+v", recipient)
		}
	}
	if _, err := db.Exec(`UPDATE users SET phone=$2 WHERE id=$1`, other.ID, "0"+payee.Phone[3:]); err != nil {
		t.Fatal(err)
	}
	lookup(payee.Phone, http.StatusBadRequest)
	lookup(wallet, http.StatusOK)
	if _, err := db.Exec(`UPDATE users SET account_status='suspended' WHERE id=$1`, payee.ID); err != nil {
		t.Fatal(err)
	}
	lookup(wallet, http.StatusNotFound)
	if err := repository.NewWalletRepository(db, "NGN", 0).RequestTransfer(t.Context(), sender.ID, wallet, "1.00", "recipient-suspended", ""); err == nil {
		t.Fatal("final transfer accepted a suspended recipient")
	}
	if _, err := db.Exec(`UPDATE users SET account_status='active',staff_disabled=TRUE WHERE id=$1`, payee.ID); err != nil {
		t.Fatal(err)
	}
	lookup(wallet, http.StatusNotFound)
	if _, err := db.Exec(`UPDATE users SET staff_disabled=FALSE,deleted_at=CURRENT_TIMESTAMP WHERE id=$1`, payee.ID); err != nil {
		t.Fatal(err)
	}
	lookup(wallet, http.StatusNotFound)

	usd := repository.NewWalletRepository(db, "USD", 0)
	if _, err := usd.Wallet(t.Context(), other.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM wallets WHERE owner_type='user' AND owner_id=$1 AND currency='NGN'`, other.ID); err != nil {
		t.Fatal(err)
	}
	response := request(handler, "POST", "/wallet/recipient", url.Values{"recipient": {strconv.Itoa(other.ID)}}, cookie)
	requireStatus(t, response, http.StatusNotFound)
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM wallets WHERE owner_type='user' AND owner_id=$1 AND currency='NGN'`, other.ID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("lookup created a wallet: %d %v", count, err)
	}
	requireStatus(t, request(handler, "POST", "/wallet/recipient", url.Values{"recipient": {wallet}}, nil), http.StatusSeeOther)
}
