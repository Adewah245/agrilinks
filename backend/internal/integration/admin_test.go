package integration_test

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/routes"
	"backend/internal/testutil"
)

func adminForm(t *testing.T, handler http.Handler, cookie *http.Cookie, path string, values url.Values) url.Values {
	t.Helper()
	page := request(handler, "GET", path, nil, cookie)
	requireStatus(t, page, 200)
	match := regexp.MustCompile(`name="csrf" value="([a-f0-9]+)"`).FindStringSubmatch(page.Body.String())
	if len(match) != 2 {
		t.Fatal("admin form is missing CSRF protection")
	}
	values.Set("csrf", match[1])
	values.Set("return_to", path)
	return values
}

func TestAdminWorkspacePermissionsAndWorkflows(t *testing.T) {
	db, _ := testutil.Database(t, true)
	c := application(t, db, testConfig(t))
	ctx := context.Background()
	root := fixtureUser(t, c, 1, "buyer")
	farmer := fixtureUser(t, c, 2, "farmer")
	care := fixtureUser(t, c, 3, "buyer")
	ops := fixtureUser(t, c, 4, "buyer")
	if err := c.Admin.Bootstrap(ctx, root.Phone); err != nil {
		t.Fatal(err)
	}
	if err := c.Admin.Bootstrap(ctx, farmer.Phone); err == nil {
		t.Fatal("bootstrap replaced an existing Super Admin")
	}
	root, err := c.UserRepo.GetByID(root.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range []struct {
		user *models.User
		role string
	}{{care, "customer_care"}, {ops, "operations_manager"}} {
		if err := c.Admin.Apply(ctx, root, map[string]string{"action": "staff_role", "id": strconv.Itoa(entry.user.ID), "role": entry.role, "reason": "Assign work responsibilities"}); err != nil {
			t.Fatal(err)
		}
	}
	handler := routes.NewHandler(c)
	rootCookie := session(t, c, root)
	careCookie := session(t, c, care)
	opsCookie := session(t, c, ops)
	farmerCookie := session(t, c, farmer)
	var productID, orderID, deliveryID, walletID int
	if err := db.QueryRow(`INSERT INTO products(owner_type,owner_id,name,category,quantity,unit,price) VALUES('user',$1,'Test maize','Grains',20,'kg',10) RETURNING id`, farmer.ID).Scan(&productID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`INSERT INTO orders(buyer_id,total_amount) VALUES($1,20) RETURNING id`, root.ID).Scan(&orderID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO order_items(order_id,product_id,quantity,unit_price,total_price) VALUES($1,$2,2,10,20)`, orderID, productID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`INSERT INTO deliveries(order_id,pickup_address,destination_address) VALUES($1,'Farm','Buyer address') RETURNING id`, orderID).Scan(&deliveryID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO delivery_tracking(delivery_id,latitude,longitude,location_name) VALUES($1,7.7,8.5,'Makurdi')`, deliveryID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT id FROM wallets WHERE owner_type='user' AND owner_id=$1 AND currency='NGN'`, root.ID).Scan(&walletID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO wallet_deposits(wallet_id,reference,amount) VALUES($1,'admin-test-deposit',50)`, walletID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO wallet_transactions(wallet_id,type,amount,reference) VALUES($1,'credit',50,'admin-test-ledger')`, walletID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO transactions(order_id,buyer_id,seller_type,seller_id,amount,status,paid_at,payment_reference) VALUES($1,$2,'user',$3,20,'successful',CURRENT_TIMESTAMP,'admin-test-payment')`, orderID, root.ID, farmer.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO platform_revenue(order_id,reference,gross_amount,amount,rate_percent) VALUES($1,'admin-test-commission',20,1,5)`, orderID); err != nil {
		t.Fatal(err)
	}

	for _, section := range models.AdminSections {
		t.Run("page_"+section.Key, func(t *testing.T) {
			requireStatus(t, request(handler, "GET", section.Path, nil, rootCookie), 200)
			requireStatus(t, request(handler, "GET", section.Path+"?q=%27%20OR%201%3D1%20--", nil, rootCookie), 200)
			requireStatus(t, request(handler, "GET", section.Path, nil, farmerCookie), 403)
		})
	}
	for _, path := range []string{"/admin", "/admin/users", "/admin/orders", "/admin/support"} {
		requireStatus(t, request(handler, "GET", path, nil, careCookie), 200)
	}
	for _, path := range []string{"/admin/finance", "/admin/marketplace", "/admin/reports", "/admin/audit", "/admin/staff", "/admin/settings"} {
		requireStatus(t, request(handler, "GET", path, nil, careCookie), 403)
	}
	for _, path := range []string{"/admin/staff", "/admin/settings", "/admin/audit"} {
		requireStatus(t, request(handler, "GET", path, nil, opsCookie), 403)
	}
	if strings.Contains(request(handler, "GET", "/admin", nil, careCookie).Body.String(), "Recorded platform commissions") {
		t.Fatal("finance metric leaked to Customer Care")
	}
	requireStatus(t, request(handler, "POST", "/admin/action", url.Values{"action": {"staff_access"}, "id": {strconv.Itoa(care.ID)}, "value": {"disabled"}}, rootCookie), 403)
	forbidden := adminForm(t, handler, careCookie, "/admin/support", url.Values{"action": {"staff_role"}, "id": {strconv.Itoa(care.ID)}, "role": {"super_admin"}, "reason": {"Escalate access"}})
	requireStatus(t, request(handler, "POST", "/admin/action", forbidden, careCookie), 403)
	post := func(cookie *http.Cookie, path string, form url.Values) {
		t.Helper()
		form = adminForm(t, handler, cookie, path, form)
		requireStatus(t, request(handler, "POST", "/admin/action", form, cookie), http.StatusSeeOther)
	}
	post(rootCookie, "/admin/settings", url.Values{"action": {"setting_update"}, "key": {"support_default_priority"}, "value": {"high"}, "reason": {"Prioritize current support needs"}})
	post(careCookie, "/admin/support", url.Values{"action": {"ticket_create"}, "member_id": {strconv.Itoa(root.ID)}, "order_id": {strconv.Itoa(orderID)}, "subject": {"Delivery complaint"}, "description": {"Please investigate the delivery"}, "kind": {"dispute"}})
	var ticketID int
	var priority string
	if err := db.QueryRow(`SELECT id,priority FROM admin_support_tickets`).Scan(&ticketID, &priority); err != nil || priority != "high" {
		t.Fatalf("ticket did not use configured priority: %s %v", priority, err)
	}
	post(careCookie, "/admin/support", url.Values{"action": {"ticket_update"}, "id": {strconv.Itoa(ticketID)}, "value": {"resolved"}, "assignee": {strconv.Itoa(care.ID)}, "reason": {"Buyer confirmed the delivery arrived"}})
	var history, notifications int
	if err := db.QueryRow(`SELECT count(*) FROM admin_ticket_history WHERE ticket_id=$1`, ticketID).Scan(&history); err != nil || history != 2 {
		t.Fatalf("resolution history: %d %v", history, err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM notifications WHERE user_id=$1 AND notification_type='system'`, root.ID).Scan(&notifications); err != nil || notifications != 2 {
		t.Fatalf("ticket notifications: %d %v", notifications, err)
	}
	post(opsCookie, "/admin/marketplace", url.Values{"action": {"product_status"}, "id": {strconv.Itoa(productID)}, "value": {"inactive"}, "reason": {"Investigating listing quality"}})
	post(opsCookie, "/admin/users", url.Values{"action": {"user_verification"}, "id": {strconv.Itoa(farmer.ID)}, "value": {"verified"}, "reason": {"Reviewed identity evidence"}})
	post(careCookie, "/admin/users", url.Values{"action": {"report_create"}, "target_kind": {"user"}, "target_id": {strconv.Itoa(farmer.ID)}, "reason": {"Buyer reported account conduct"}})
	post(rootCookie, "/admin/staff", url.Values{"action": {"staff_create"}, "name": {"New Care Staff"}, "phone": {"08010000005"}, "email": {"newcare@example.test"}, "password": {"a secure staff password"}, "role": {"customer_care"}})
	newStaff, err := c.Auth.Login("08010000005", "a secure staff password")
	if err != nil || newStaff.StaffRole != "customer_care" {
		t.Fatalf("new staff login failed: %v", err)
	}
	// Reserve money using the existing wallet repository, then review through HTTP.
	wallets := repository.NewWalletRepository(db, "NGN", 0)
	if _, err := db.Exec(`UPDATE wallets SET balance=50 WHERE id=$1`, walletID); err != nil {
		t.Fatal(err)
	}
	var recipientNumber string
	if err := db.QueryRow(`SELECT wallet_number FROM wallets WHERE owner_type='user' AND owner_id=$1 AND currency='NGN'`, farmer.ID).Scan(&recipientNumber); err != nil {
		t.Fatal(err)
	}
	if err := wallets.RequestTransfer(ctx, root.ID, recipientNumber, "10.00", "admin-test-transfer", "Test staff review"); err != nil {
		t.Fatal(err)
	}
	var transferID int
	if err := db.QueryRow(`SELECT id FROM wallet_transfers WHERE reference='admin-test-transfer'`).Scan(&transferID); err != nil {
		t.Fatal(err)
	}
	forbidden = adminForm(t, handler, careCookie, "/admin/support", url.Values{"action": {"wallet_transfer"}, "id": {strconv.Itoa(transferID)}, "value": {"completed"}, "reason": {"Attempt unauthorized payment approval"}})
	requireStatus(t, request(handler, "POST", "/admin/action", forbidden, careCookie), 403)
	post(opsCookie, "/admin/finance", url.Values{"action": {"wallet_transfer"}, "id": {strconv.Itoa(transferID)}, "value": {"completed"}, "reason": {"Verified transfer request"}})
	if err := wallets.RequestWithdrawal(ctx, root.ID, "5.00", "admin-test-withdrawal", &models.UserBankAccount{BankName: "Test bank", BankCode: "044", AccountName: root.FullName, AccountNumber: "1234567890"}); err != nil {
		t.Fatal(err)
	}
	var withdrawalID int
	if err := db.QueryRow(`SELECT id FROM wallet_withdrawals WHERE reference='admin-test-withdrawal'`).Scan(&withdrawalID); err != nil {
		t.Fatal(err)
	}
	post(opsCookie, "/admin/finance", url.Values{"action": {"wallet_withdrawal"}, "id": {strconv.Itoa(withdrawalID)}, "value": {"rejected"}, "reason": {"Reject request and release reserved funds"}})
	var balance string
	if err := db.QueryRow(`SELECT balance::text FROM wallets WHERE id=$1`, walletID).Scan(&balance); err != nil || balance != "40.00" {
		t.Fatalf("wallet review changed balance incorrectly: %s %v", balance, err)
	}
	post(opsCookie, "/admin/users", url.Values{"action": {"user_status"}, "id": {strconv.Itoa(farmer.ID)}, "value": {"suspended"}, "reason": {"Suspended pending investigation"}})
	requireStatus(t, request(handler, "GET", "/dashboard", nil, farmerCookie), http.StatusSeeOther)
	if _, err := c.Auth.Login(farmer.Phone, "test password with spaces"); err == nil {
		t.Fatal("suspended account logged in")
	}
	post(rootCookie, "/admin/staff", url.Values{"action": {"staff_access"}, "id": {strconv.Itoa(care.ID)}, "value": {"disabled"}, "reason": {"Compromised staff account"}})
	requireStatus(t, request(handler, "GET", "/admin", nil, careCookie), http.StatusSeeOther)
	if _, err := c.Auth.Login(care.Phone, "test password with spaces"); err == nil {
		t.Fatal("disabled staff logged in")
	}
	self := adminForm(t, handler, rootCookie, "/admin/staff", url.Values{"action": {"staff_access"}, "id": {strconv.Itoa(root.ID)}, "value": {"disabled"}, "reason": {"Attempt self disable"}})
	requireStatus(t, request(handler, "POST", "/admin/action", self, rootCookie), 400)
	for _, section := range models.AdminSections {
		requireStatus(t, request(handler, "GET", section.Path, nil, rootCookie), 200)
	}
	var successes, failures int
	if err := db.QueryRow(`SELECT count(*) FILTER(WHERE succeeded),count(*) FILTER(WHERE NOT succeeded) FROM admin_audit_logs`).Scan(&successes, &failures); err != nil || successes < 10 || failures < 3 {
		t.Fatalf("audit history missing: %d %d %v", successes, failures, err)
	}
	var leaked bool
	if err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM admin_audit_logs WHERE before_value::text LIKE '%password%' OR after_value::text LIKE '%password%')`).Scan(&leaked); err != nil || leaked {
		t.Fatal("credentials appeared in audit values")
	}
	if _, err := db.Exec(`UPDATE admin_audit_logs SET detail='tampered' WHERE id=(SELECT min(id) FROM admin_audit_logs)`); err == nil {
		t.Fatal("audit history could be rewritten")
	}
	if err := c.UserRepo.DeleteAccount(ctx, root.ID); err == nil {
		t.Fatal("staff identity could be closed outside the staff workspace")
	}
	if _, err := db.Exec(`INSERT INTO users(full_name,phone,password_hash,role) SELECT 'Pagination Farmer '||n,'page-test-'||n,$1,'farmer' FROM generate_series(1,28) n`, root.PasswordHash); err != nil {
		t.Fatal(err)
	}
	page := request(handler, "GET", "/admin/users?q=Pagination+Farmer", nil, rootCookie)
	requireStatus(t, page, 200)
	if !strings.Contains(page.Body.String(), "Next →") {
		t.Fatal("pagination did not expose remaining records")
	}
	requireStatus(t, request(handler, "GET", "/admin/users?q=Pagination+Farmer&page=2", nil, rootCookie), 200)
	if _, err := db.Exec(`UPDATE users SET full_name='<script>alert(1)</script>' WHERE id=$1`, farmer.ID); err != nil {
		t.Fatal(err)
	}
	page = request(handler, "GET", "/admin/users?q="+strconv.Itoa(farmer.ID), nil, rootCookie)
	requireStatus(t, page, 200)
	if strings.Contains(page.Body.String(), "<script>alert(1)</script>") || !strings.Contains(page.Body.String(), "&lt;script&gt;") {
		t.Fatal("member name was not safely escaped")
	}
	legacyAdmin := fixtureUser(t, c, 6, "buyer")
	c.UserRepo.SetConfiguredAdministrators([]int{legacyAdmin.ID})
	legacyAdmin, err = c.UserRepo.GetByID(legacyAdmin.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Admin.Apply(ctx, legacyAdmin, map[string]string{"action": "staff_access", "id": strconv.Itoa(root.ID), "value": "disabled", "reason": "Attempt to disable last Super Admin"}); err == nil {
		t.Fatal("last persisted Super Admin could be disabled")
	}
	login := request(handler, "POST", "/login", url.Values{"phone": {root.Phone}, "password": {"test password with spaces"}}, nil)
	requireStatus(t, login, http.StatusSeeOther)
	if login.Header().Get("Location") != "/admin" {
		t.Fatal("staff login did not open admin")
	}
	legacy := request(handler, "GET", "/wallet/admin", nil, rootCookie)
	requireStatus(t, legacy, http.StatusSeeOther)
	if legacy.Header().Get("Location") != "/admin/finance" {
		t.Fatal("legacy admin link did not open finance")
	}
}
