package integration_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"

	"backend/internal/models"
	"backend/internal/repository"
	"backend/internal/routes"
	"backend/internal/testutil"
)

func fixtureProduct(t *testing.T, db *sql.DB, sellerID int, name string, quantity, price float64) int {
	t.Helper()
	var id int
	err := db.QueryRow(`INSERT INTO products(owner_type,owner_id,name,category,quantity,initial_listed_quantity,unit,price,status)
		VALUES('user',$1,$2,'crop',$3,$3,'kg',$4,'active') RETURNING id`, sellerID, name, quantity, price).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestBuyingPaymentDeliveryAndReviewsWithPostgres(t *testing.T) {
	db, _ := testutil.Database(t, true)
	cfg := testConfig(t)
	var providerMu sync.Mutex
	var reference string
	amount := 159.52
	initializations := 0
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		providerMu.Lock()
		defer providerMu.Unlock()
		if r.Header.Get("Authorization") != "Bearer test-payment-key" {
			t.Error("payment provider did not receive configured key")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/payments":
			initializations++
			var body struct{ TxRef, Amount, Currency, RedirectURL string }
			var raw map[string]string
			var payload map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			raw = map[string]string{}
			for _, key := range []string{"tx_ref", "amount", "currency", "redirect_url"} {
				var value string
				_ = json.Unmarshal(payload[key], &value)
				raw[key] = value
			}
			body.TxRef, body.Amount, body.Currency, body.RedirectURL = raw["tx_ref"], raw["amount"], raw["currency"], raw["redirect_url"]
			if body.Amount != "159.52" || body.Currency != "NGN" || body.RedirectURL != cfg.PaymentRedirectURL {
				t.Errorf("wrong payment request: %+v", body)
			}
			reference = body.TxRef
			fmt.Fprint(w, `{"status":"success","data":{"link":"https://checkout.example.test/checkout"}}`)
		case "/transactions/7001/verify":
			json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]any{"id": 7001, "tx_ref": reference, "amount": amount, "currency": "NGN", "status": "successful"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()
	cfg.FlutterwaveSecretKey = "test-payment-key"
	cfg.FlutterwaveSecretHash = "test-webhook-hash"
	cfg.FlutterwaveBaseURL = provider.URL
	cfg.PaymentRedirectURL = "http://example.test/payments/callback"
	c := application(t, db, cfg)
	seller := fixtureUser(t, c, 1, "farmer")
	buyer := fixtureUser(t, c, 2, "buyer")
	other := fixtureUser(t, c, 3, "buyer")
	driver := fixtureUser(t, c, 4, "farmer")
	p1 := fixtureProduct(t, db, seller.ID, "Maize", 10, 125.55)
	p2 := fixtureProduct(t, db, seller.ID, "Beans", 10, 12.25)

	n, err := c.Negotiation.StartNegotiation(buyer.ID, p1, 1.5, 90.01, "")
	if err != nil {
		t.Fatal(err)
	}
	var offerID int
	if err = db.QueryRow(`SELECT id FROM negotiation_messages WHERE negotiation_id=$1`, n.ID).Scan(&offerID); err != nil {
		t.Fatal(err)
	}
	if err = c.Negotiation.Accept(n.ID, offerID, other.ID); err == nil {
		t.Fatal("outsider accepted a negotiation")
	}
	if err = c.Negotiation.Accept(n.ID, offerID, buyer.ID); err == nil {
		t.Fatal("buyer accepted own offer")
	}
	if err = c.Negotiation.Accept(n.ID, offerID, seller.ID); err != nil {
		t.Fatal(err)
	}
	if err = c.Negotiation.Accept(n.ID, offerID, seller.ID); err == nil {
		t.Fatal("negotiation was accepted twice")
	}
	if _, err = db.Exec(`UPDATE products SET price=999 WHERE id=$1`, p1); err != nil {
		t.Fatal(err)
	}
	handler := routes.NewHandler(c)
	cookie := session(t, c, buyer)
	requireStatus(t, request(handler, "POST", "/cart", url.Values{"action": {"add"}, "product_id": {strconv.Itoa(p2)}, "quantity": {"2"}, "price_per_unit": {"0.01"}}, cookie), http.StatusSeeOther)
	requireStatus(t, request(handler, "GET", "/cart", nil, cookie), http.StatusOK)
	for i := 0; i < 2; i++ {
		requireStatus(t, request(handler, "POST", "/orders/place", url.Values{"checkout_key": {"test-cart-request"}}, cookie), http.StatusSeeOther)
	}
	var orderID, count int
	var total, status string
	if err = db.QueryRow(`SELECT id,total_amount::text,status FROM orders WHERE buyer_id=$1`, buyer.ID).Scan(&orderID, &total, &status); err != nil {
		t.Fatal(err)
	}
	if total != "159.52" || status != "pending" {
		t.Fatalf("wrong saved order: %s %s", total, status)
	}
	if err = db.QueryRow(`SELECT count(*) FROM cart_items WHERE buyer_id=$1`, buyer.ID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("cart was not cleared: %d %v", count, err)
	}
	payable, err := c.Payment.PayableOrders(buyer.ID)
	if err != nil || len(payable) != 1 || len(payable[0].Items) != 2 || payable[0].TotalAmount != 159.52 {
		t.Fatalf("payment page duplicated or lost order items: %+v %v", payable, err)
	}
	if _, err = c.Delivery.RequestDelivery(buyer, orderID, "Test destination"); err == nil {
		t.Fatal("unpaid order allowed delivery")
	}
	if _, err = c.Review.ReviewProduct(buyer.ID, p1, orderID, 5, "Not yet delivered"); err == nil {
		t.Fatal("unpaid order allowed a review")
	}
	if _, err = c.Payment.Initiate(context.Background(), other.ID, orderID); err == nil {
		t.Fatal("outsider started payment")
	}
	for i := 0; i < 2; i++ {
		requireStatus(t, request(handler, "POST", "/payments", url.Values{"order_id": {strconv.Itoa(orderID)}}, cookie), http.StatusSeeOther)
	}
	providerMu.Lock()
	calls, ref := initializations, reference
	amount = 1
	providerMu.Unlock()
	if calls != 1 {
		t.Fatalf("repeated payment created %d checkouts", calls)
	}
	callback := "/payments/callback?transaction_id=7001&tx_ref=" + url.QueryEscape(ref)
	requireStatus(t, request(handler, "GET", callback, nil, nil), http.StatusBadRequest)
	providerMu.Lock()
	amount = 159.52
	providerMu.Unlock()
	// Model another pending attempt left by an older application version.
	if _, err = db.Exec(`INSERT INTO transactions(order_id,buyer_id,seller_type,seller_id,amount,currency,payment_reference,payment_provider,status)
		VALUES($1,$2,'user',$3,159.52,'NGN','old-extra-attempt','flutterwave','pending')`, orderID, buyer.ID, seller.ID); err != nil {
		t.Fatal(err)
	}
	requireStatus(t, request(handler, "GET", callback, nil, nil), http.StatusSeeOther)
	webhook := func(hash string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "http://example.test/webhooks/flutterwave", strings.NewReader(fmt.Sprintf(`{"event":"charge.completed","data":{"id":7001,"tx_ref":%q}}`, ref)))
		r.Header.Set("verif-hash", hash)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	requireStatus(t, webhook("wrong"), http.StatusUnauthorized)
	for i := 0; i < 3; i++ {
		requireStatus(t, webhook(cfg.FlutterwaveSecretHash), http.StatusOK)
	}
	paidAgain, err := repository.NewPaymentRepository(db).FinalizePayment("old-extra-attempt", "successful", "7002", nil)
	if err != nil || paidAgain {
		t.Fatalf("duplicate attempt paid order again: %v %v", paidAgain, err)
	}
	var balance string
	if err = db.QueryRow(`SELECT balance::text FROM wallets WHERE owner_type='user' AND owner_id=$1 AND currency='NGN'`, seller.ID).Scan(&balance); err != nil || balance != "159.52" {
		t.Fatalf("wrong seller balance %s %v", balance, err)
	}
	if err = db.QueryRow(`SELECT count(*) FROM wallet_transactions WHERE type='sale_credit'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("sale recorded more than once: %d %v", count, err)
	}
	payments, err := c.Payment.MyPayments(buyer.ID)
	if err != nil || len(payments) != 2 {
		t.Fatalf("multiple items duplicated payment history: %+v %v", payments, err)
	}
	var flagged bool
	if err = db.QueryRow(`SELECT review_required FROM transactions WHERE payment_reference='old-extra-attempt'`).Scan(&flagged); err != nil || !flagged {
		t.Fatal("extra payment was not marked for review")
	}

	delivery, err := c.Delivery.RequestDelivery(buyer, orderID, "Test buyer address")
	if err != nil {
		t.Fatal(err)
	}
	partner, err := c.Delivery.RegisterPartner(driver, models.DeliveryPartnerInput{Name: "Test delivery", Phone: driver.Phone, Email: driver.Email, VehicleType: "truck", RegistrationNumber: "TEST-1"})
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Delivery.AssignPartner(seller, delivery.ID, partner.ID); err == nil {
		t.Fatal("unapproved partner was assigned")
	}
	admin := *seller
	admin.IsAdmin = true
	if err = c.Delivery.SetPartnerVerified(&admin, partner.ID, true); err != nil {
		t.Fatal(err)
	}
	if err = c.Delivery.AssignPartner(seller, delivery.ID, partner.ID); err != nil {
		t.Fatal(err)
	}
	if err = c.Delivery.UpdateStatus(other, delivery.ID, "delivered"); err == nil {
		t.Fatal("outsider updated delivery")
	}
	if err = c.Delivery.UpdateStatus(driver, delivery.ID, "delivered"); err == nil {
		t.Fatal("delivery skipped required steps")
	}
	if err = c.Delivery.UpdateStatus(driver, delivery.ID, "picked_up"); err != nil {
		t.Fatal(err)
	}
	if err = c.Delivery.AddTracking(driver, delivery.ID, 7.2, 8.3, "Test road", "On the way"); err != nil {
		t.Fatal(err)
	}
	if err = c.Delivery.UpdateStatus(driver, delivery.ID, "in_transit"); err != nil {
		t.Fatal(err)
	}
	if err = c.Delivery.UpdateStatus(driver, delivery.ID, "delivered"); err != nil {
		t.Fatal(err)
	}
	if _, err = c.Review.ReviewProduct(other.ID, p1, orderID, 5, "Not my purchase"); err == nil {
		t.Fatal("outsider reviewed purchase")
	}
	if _, err = c.Review.ReviewProduct(buyer.ID, p1, orderID, 5, "Good delivery"); err != nil {
		t.Fatal(err)
	}
	if _, err = c.Review.ReviewProduct(buyer.ID, p1, orderID, 4, "Second review"); err == nil {
		t.Fatal("purchase reviewed twice")
	}
	for _, path := range []string{"/orders", "/payments", "/deliveries", "/product?id=" + strconv.Itoa(p1)} {
		requireStatus(t, request(handler, "GET", path, nil, cookie), http.StatusOK)
	}
}

func TestCheckoutAndNegotiationConcurrentRequests(t *testing.T) {
	db, _ := testutil.Database(t, true)
	c := application(t, db, testConfig(t))
	seller := fixtureUser(t, c, 1, "farmer")
	a := fixtureUser(t, c, 2, "buyer")
	b := fixtureUser(t, c, 3, "buyer")
	p := fixtureProduct(t, db, seller.ID, "Last bag", 1, 10)
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, buyer := range []int{a.ID, b.ID} {
		go func(id int) { <-start; results <- c.Order.PlaceOrder(id, p, .75) }(buyer)
	}
	close(start)
	success := 0
	for i := 0; i < 2; i++ {
		if <-results == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("%d buyers bought the last stock", success)
	}
	var quantity float64
	if err := db.QueryRow(`SELECT quantity FROM products WHERE id=$1`, p).Scan(&quantity); err != nil || quantity != .25 {
		t.Fatalf("wrong remaining stock %v %v", quantity, err)
	}
	if err := c.Order.PlaceOrder(a.ID, p, math.NaN()); err == nil {
		t.Fatal("NaN quantity accepted")
	}
	p2 := fixtureProduct(t, db, seller.ID, "Small amount", .3, 10)
	if err := c.Order.PlaceOrder(a.ID, p2, .1, "small-one"); err != nil {
		t.Fatal(err)
	}
	if err := c.Order.PlaceOrder(a.ID, p2, .1, "small-one"); err != nil {
		t.Fatal(err)
	}
	if err := c.Order.PlaceOrder(b.ID, p2, .2, "small-two"); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT quantity FROM products WHERE id=$1`, p2).Scan(&quantity); err != nil || quantity != 0 {
		t.Fatalf("decimal stock subtraction failed: %v %v", quantity, err)
	}

	p3 := fixtureProduct(t, db, seller.ID, "Negotiated bag", 10, 10)
	n, err := c.Negotiation.StartNegotiation(a.ID, p3, 1, 8, "First offer")
	if err != nil {
		t.Fatal(err)
	}
	results = make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func() { results <- c.Negotiation.SendOffer(n.ID, seller.ID, 9, "") }()
	}
	for i := 0; i < 10; i++ {
		<-results
	}
	var rounds, messages int
	if err = db.QueryRow(`SELECT round_count,(SELECT count(*) FROM negotiation_messages WHERE negotiation_id=$1 AND message_type='offer') FROM negotiations WHERE id=$1`, n.ID).Scan(&rounds, &messages); err != nil {
		t.Fatal(err)
	}
	if rounds != 5 || messages != 5 {
		t.Fatalf("round limit lost under repeated requests: rounds=%d messages=%d", rounds, messages)
	}
}
