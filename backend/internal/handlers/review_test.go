package handlers

import (
	"testing"
	"time"
)

func TestParseBadgeExpiryIncludesWholeSelectedDay(t *testing.T) {
	expiresAt, err := parseBadgeExpiry("2026-12-31")
	if err != nil {
		t.Fatal(err)
	}
	if expiresAt == nil {
		t.Fatal("expected an expiry time")
	}
	if expiresAt.Year() != 2026 || expiresAt.Month() != time.December || expiresAt.Day() != 31 {
		t.Fatalf("expiry date = %s, want 31 December 2026", expiresAt)
	}
	if expiresAt.Hour() != 23 || expiresAt.Minute() != 59 || expiresAt.Second() != 59 {
		t.Fatalf("expiry time = %s, want the end of the selected day", expiresAt)
	}
}

func TestParseBadgeExpiryAllowsNoExpiry(t *testing.T) {
	expiresAt, err := parseBadgeExpiry("  ")
	if err != nil {
		t.Fatal(err)
	}
	if expiresAt != nil {
		t.Fatalf("expiry = %s, want nil", expiresAt)
	}
}

func TestParseBadgeExpiryRejectsInvalidDate(t *testing.T) {
	if _, err := parseBadgeExpiry("31/12/2026"); err == nil {
		t.Fatal("expected an invalid date to be rejected")
	}
}
