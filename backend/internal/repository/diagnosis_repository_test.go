package repository

import (
	"testing"

	"backend/internal/models"
)

func TestNormalizeDiagnosisText(t *testing.T) {
	empty := "   "
	if got := normalizeDiagnosisText(&empty); got != nil {
		t.Fatalf("blank text = %#v, want nil", got)
	}

	value := "  Leaf spots  "
	got := normalizeDiagnosisText(&value)
	if got == nil || *got != "Leaf spots" {
		t.Fatalf("normalized text = %#v", got)
	}
}

func TestNormalizeDiagnosisMedia(t *testing.T) {
	media := &models.DiagnosisMedia{MediaURL: "  /uploads/leaf.jpg  "}

	if err := normalizeDiagnosisMedia(media); err != nil {
		t.Fatal(err)
	}
	if media.MediaType != "image" {
		t.Fatalf("media type = %q, want image", media.MediaType)
	}
	if media.MediaURL != "/uploads/leaf.jpg" {
		t.Fatalf("media URL = %q", media.MediaURL)
	}

	media.MediaURL = ""
	if err := normalizeDiagnosisMedia(media); err == nil {
		t.Fatal("expected blank media URL to be rejected")
	}
}
