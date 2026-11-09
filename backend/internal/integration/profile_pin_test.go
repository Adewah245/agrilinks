package integration_test

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"backend/internal/routes"
	"backend/internal/testui"
	"backend/internal/testutil"
)

func TestPINEnrolmentBelongsToTheOwnersProfile(t *testing.T) {
	db, _ := testutil.Database(t, true)
	c := application(t, db, testConfig(t))
	owner := fixtureUser(t, c, 1, "buyer")
	viewer := fixtureUser(t, c, 2, "buyer")
	handler := routes.NewHandler(c)
	cookie := session(t, c, owner)
	page := request(handler, "GET", "/profile", nil, cookie)
	requireStatus(t, page, http.StatusOK)
	if !strings.Contains(testui.HTML(t, page.Body.String()), `action="/profile/pin"`) {
		t.Fatal("owner without a PIN needs the enrolment form in their profile")
	}
	form := url.Values{"current_password": {"incorrect password"}, "transaction_pin": {"582491"}, "transaction_pin_confirmation": {"582491"}}
	requireStatus(t, request(handler, "POST", "/profile/pin", form, cookie), http.StatusBadRequest)
	if exists, err := c.UserRepo.HasTransactionPIN(t.Context(), owner.ID); err != nil || exists {
		t.Fatal("incorrect account password must not enrol a PIN")
	}
	form.Set("current_password", "test password with spaces")
	requireStatus(t, request(handler, "POST", "/profile/pin", form, cookie), http.StatusSeeOther)
	if err := c.UserRepo.VerifyTransactionPIN(t.Context(), owner.ID, "582491"); err != nil {
		t.Fatal(err)
	}
	page = request(handler, "GET", "/profile", nil, cookie)
	if !strings.Contains(testui.HTML(t, page.Body.String()), "Your PIN is set") {
		t.Fatal("existing PIN should show its status without its value")
	}
	public := request(handler, "GET", "/profile?id="+strconv.Itoa(owner.ID), nil, session(t, c, viewer))
	requireStatus(t, public, http.StatusOK)
	if strings.Contains(testui.HTML(t, public.Body.String()), `id="profile-pin"`) {
		t.Fatal("another member must not see the private PIN controls")
	}
	requireStatus(t, request(handler, "POST", "/profile/pin", form, nil), http.StatusSeeOther)
}
