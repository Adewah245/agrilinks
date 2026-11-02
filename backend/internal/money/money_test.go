package money

import "testing"

func TestApprovedWithdrawalTiers(t *testing.T) {
	for _, tt := range []struct{ amount, fee int64 }{{100, 5000}, {500000, 5000}, {500100, 20000}, {999900, 20000}, {1000000, 30000}, {500000000, 30000}} {
		fee, err := WithdrawalFee(tt.amount)
		if err != nil || fee != tt.fee {
			t.Errorf("%s: fee=%d want=%d error=%v", Format(tt.amount), fee, tt.fee, err)
		}
	}
	for _, n := range []int64{0, -100, 99, 500001} {
		if _, err := WithdrawalFee(n); err == nil {
			t.Errorf("invalid payout accepted: %d", n)
		}
	}
}
func TestMoneyExactAndOverflow(t *testing.T) {
	for _, tt := range []struct {
		s string
		n int64
	}{{"10000", 1000000}, {"0.01", 1}, {"123456789.99", 12345678999}, {"0005.1", 510}} {
		n, err := Parse(tt.s)
		if err != nil || n != tt.n {
			t.Errorf("parse %s: %d %v", tt.s, n, err)
		}
	}
	for _, s := range []string{"1.001", "NaN", "1e3", "-1", "92233720368547759"} {
		if _, err := Parse(s); err == nil {
			t.Errorf("invalid amount accepted: %s", s)
		}
	}
	if _, err := Add(9223372036854775807, 1); err == nil {
		t.Fatal("overflow accepted")
	}
}

func TestFormatNegativeLimit(t *testing.T) {
	if got := Format(-9223372036854775808); got != "-92233720368547758.08" {
		t.Fatal(got)
	}
}
