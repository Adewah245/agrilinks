package services

import (
	"bytes"
	"encoding/base32"
	"encoding/base64"
	"image"
	"image/jpeg"
	"testing"
)

func TestPrivateEncryptionAuthenticatesPurposeAndCiphertext(t *testing.T) {
	s, err := NewSecurityService(nil, base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte("private evidence that must not appear in storage")
	a, err := s.Seal(plain, "verification-photo")
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Seal(plain, "verification-photo")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(a, plain) || bytes.Equal(a, b) {
		t.Fatal("plaintext or repeated encryption nonce")
	}
	got, err := s.Open(a, "verification-photo")
	if err != nil || !bytes.Equal(got, plain) {
		t.Fatal("ciphertext cannot be decrypted")
	}
	if _, err = s.Open(a, "admin-mfa:1"); err == nil {
		t.Fatal("ciphertext accepted in another security context")
	}
	a[len(a)-1] ^= 1
	if _, err = s.Open(a, "verification-photo"); err == nil {
		t.Fatal("tampered ciphertext accepted")
	}
	if _, err = s.Open([]byte{1, 2}, "verification-photo"); err == nil {
		t.Fatal("truncated ciphertext accepted")
	}
}

func TestAuthenticatorRFC6238Vectors(t *testing.T) {
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte("12345678901234567890"))
	// RFC 6238 SHA-1 vectors reduced to the configured six digits.
	for _, tt := range []struct {
		unix int64
		code string
	}{{59, "287082"}, {1111111109, "081804"}, {1111111111, "050471"}, {1234567890, "005924"}, {2000000000, "279037"}, {20000000000, "353130"}} {
		got, err := totpCode(secret, tt.unix/30)
		if err != nil || got != tt.code {
			t.Fatalf("time %d: %s %v", tt.unix, got, err)
		}
	}
}

func TestVerificationImagesStripAppendedPayloadAndRejectInvalidData(t *testing.T) {
	var b bytes.Buffer
	if err := jpeg.Encode(&b, image.NewRGBA(image.Rect(0, 0, 32, 32)), nil); err != nil {
		t.Fatal(err)
	}
	data := append(b.Bytes(), []byte("<script>private metadata</script>")...)
	clean, err := NormalizeVerificationImage(data)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(clean, []byte("<script>")) {
		t.Fatal("appended payload survived re-encoding")
	}
	if _, _, err = image.Decode(bytes.NewReader(clean)); err != nil {
		t.Fatal(err)
	}
	for _, bad := range [][]byte{nil, []byte("<svg onload='alert(1)'/>"), bytes.Repeat([]byte{0}, (2<<20)+1)} {
		if _, err := NormalizeVerificationImage(bad); err == nil {
			t.Fatal("invalid or oversized image accepted")
		}
	}
	missing, _ := NewSecurityService(nil, "")
	if _, err := missing.PrivateImage(b.Bytes()); err != ErrSecurityUnavailable {
		t.Fatal("missing encryption key did not fail closed")
	}
}
