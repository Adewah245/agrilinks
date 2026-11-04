package services

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"log"
	"math"
	"net/url"
	"strings"
	"time"

	"backend/internal/models"
	"backend/internal/repository"
	"github.com/disintegration/imaging"
	"golang.org/x/crypto/bcrypt"
)

var ErrSecurityUnavailable = errors.New("private verification is unavailable; contact support")
var ErrSecurityLimit = errors.New("too many sensitive requests; wait before trying again")
var ErrAuthenticatorCode = errors.New("the authenticator code is invalid, expired, or already used")

type SecurityService struct {
	Repo *repository.SecurityRepository
	box  cipher.AEAD
}

func NewSecurityService(repo *repository.SecurityRepository, key string) (*SecurityService, error) {
	s := &SecurityService{Repo: repo}
	if key == "" {
		return s, nil
	}
	raw, err := base64.StdEncoding.DecodeString(key)
	if err != nil || len(raw) != 32 {
		return nil, errors.New("SECURITY_ENCRYPTION_KEY must be a base64-encoded 32-byte key")
	}
	block, err := aes.NewCipher(raw)
	if err != nil {
		return nil, err
	}
	s.box, err = cipher.NewGCM(block)
	return s, err
}
func (s *SecurityService) Available() bool { return s != nil && s.box != nil }
func SecurityHash(value string) string {
	h := sha256.Sum256([]byte(value))
	return hex.EncodeToString(h[:])
}
func securityToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
func (s *SecurityService) Seal(data []byte, purpose string) ([]byte, error) {
	if !s.Available() {
		return nil, ErrSecurityUnavailable
	}
	nonce := make([]byte, s.box.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return s.box.Seal(nonce, nonce, data, []byte(purpose)), nil
}
func (s *SecurityService) Open(data []byte, purpose string) ([]byte, error) {
	if !s.Available() {
		return nil, ErrSecurityUnavailable
	}
	if len(data) < s.box.NonceSize() {
		return nil, ErrSecurityUnavailable
	}
	return s.box.Open(nil, data[:s.box.NonceSize()], data[s.box.NonceSize():], []byte(purpose))
}

// Decode and re-encode the image: filenames, metadata, and polyglot payloads
// never reach storage or a browser. Check dimensions before allocating pixels.
func NormalizeVerificationImage(data []byte) ([]byte, error) {
	if len(data) == 0 || len(data) > 2<<20 {
		return nil, errors.New("use a selfie smaller than 2 MB")
	}
	return normalizeVerificationImage(data)
}

func normalizeVerificationImage(data []byte) ([]byte, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png") || cfg.Width < 8 || cfg.Height < 8 || cfg.Width > 6000 || cfg.Height > 6000 || int64(cfg.Width)*int64(cfg.Height) > 16000000 {
		return nil, errors.New("use a valid JPEG or PNG selfie")
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, errors.New("could not read the selfie")
	}
	if cfg.Width > 1280 || cfg.Height > 1280 {
		img = imaging.Fit(img, 1280, 1280, imaging.Lanczos)
	}
	var out bytes.Buffer
	if err = jpeg.Encode(&out, img, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
func (s *SecurityService) PrivateImage(data []byte) ([]byte, error) {
	image, err := NormalizeVerificationImage(data)
	if err != nil {
		return nil, err
	}
	return s.Seal(image, "verification-photo")
}
func (s *SecurityService) Limit(ctx context.Context, userID int, scope string, limit int, window time.Duration) error {
	if s == nil || s.Repo == nil {
		return ErrSecurityUnavailable
	}
	ok, err := s.Repo.Allow(ctx, SecurityHash(fmt.Sprintf("%s:%d", scope, userID)), limit, window)
	if err != nil {
		return ErrSecurityUnavailable
	}
	if !ok {
		s.Event(ctx, userID, "rate_limit_"+scope, "high")
		return ErrSecurityLimit
	}
	return nil
}
func (s *SecurityService) Event(ctx context.Context, userID int, kind, severity string) {
	if s == nil || s.Repo == nil {
		return
	}
	if err := s.Repo.Event(ctx, userID, kind, severity); err != nil {
		log.Printf("security event persistence failed: kind=%s user_id=%d", kind, userID)
	}
	log.Printf("security_event kind=%s severity=%s user_id=%d", kind, severity, userID)
}
func (s *SecurityService) FreshMFA(ctx context.Context, userID int, session string) (bool, error) {
	if !s.Available() {
		return false, ErrSecurityUnavailable
	}
	return s.Repo.FreshMFA(ctx, userID, SecurityHash(session))
}
func (s *SecurityService) BeginMFA(ctx context.Context, u *models.User, session, password string, replace bool) (string, string, error) {
	if !s.Available() {
		return "", "", ErrSecurityUnavailable
	}
	if err := s.Limit(ctx, u.ID, "mfa_setup", 5, 15*time.Minute); err != nil {
		return "", "", err
	}
	if u.AdminRole() == "" || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return "", "", repository.ErrMFARequired
	}
	if replace {
		fresh, err := s.FreshMFA(ctx, u.ID, session)
		if err != nil || !fresh {
			return "", "", repository.ErrMFARequired
		}
	}
	raw := make([]byte, 20)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)
	encrypted, err := s.Seal([]byte(secret), fmt.Sprintf("admin-mfa:%d", u.ID))
	if err != nil {
		return "", "", err
	}
	if err = s.Repo.BeginMFA(ctx, u.ID, SecurityHash(session), encrypted, replace); err != nil {
		return "", "", err
	}
	uri := "otpauth://totp/" + url.PathEscape("Agrilinks Nigeria:"+u.Phone) + "?" + url.Values{"secret": {secret}, "issuer": {"Agrilinks Nigeria"}, "algorithm": {"SHA1"}, "digits": {"6"}, "period": {"30"}}.Encode()
	s.Event(ctx, u.ID, "mfa_enrollment_started", "info")
	return secret, uri, nil
}
func totpCode(secret string, counter int64) (string, error) {
	raw, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return "", err
	}
	var message [8]byte
	binary.BigEndian.PutUint64(message[:], uint64(counter))
	mac := hmac.New(sha1.New, raw)
	mac.Write(message[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 15
	n := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", n%1000000), nil
}
func (s *SecurityService) VerifyMFA(ctx context.Context, u *models.User, session, code string) ([]string, error) {
	if err := s.Limit(ctx, u.ID, "mfa_verify", 5, 5*time.Minute); err != nil {
		return nil, err
	}
	m, err := s.Repo.MFA(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	if m == nil || len(code) != 6 {
		return nil, ErrAuthenticatorCode
	}
	if !m.Enabled && (m.SetupSession != SecurityHash(session) || time.Now().After(m.SetupExpires)) {
		return nil, ErrAuthenticatorCode
	}
	secret := m.Secret
	lastCounter := m.LastCounter
	if m.Enabled && len(m.ReplacementSecret) > 0 && m.SetupSession == SecurityHash(session) && time.Now().Before(m.SetupExpires) {
		m.UseReplacement = true
		secret = m.ReplacementSecret
		lastCounter = -1
	}
	raw, err := s.Open(secret, fmt.Sprintf("admin-mfa:%d", u.ID))
	if err != nil {
		return nil, err
	}
	counter := time.Now().Unix() / 30
	matched := int64(-1)
	for c := counter - 1; c <= counter+1; c++ {
		want, err := totpCode(string(raw), c)
		if err != nil {
			return nil, err
		}
		if subtle.ConstantTimeCompare([]byte(want), []byte(code)) == 1 && c > lastCounter {
			matched = c
		}
	}
	if matched < 0 {
		s.Event(ctx, u.ID, "mfa_verification_failed", "high")
		return nil, ErrAuthenticatorCode
	}
	var codes, hashes []string
	if !m.Enabled || m.UseReplacement {
		for i := 0; i < 10; i++ {
			code, err := securityToken(10)
			if err != nil {
				return nil, err
			}
			codes = append(codes, code)
			hashes = append(hashes, SecurityHash(code))
		}
	}
	if err = s.Repo.ConfirmMFA(ctx, u.ID, SecurityHash(session), m, matched, hashes); err != nil {
		return nil, ErrAuthenticatorCode
	}
	s.Event(ctx, u.ID, "mfa_verified", "info")
	return codes, nil
}
func (s *SecurityService) RecoverMFA(ctx context.Context, u *models.User, session, password, code string) error {
	if err := s.Limit(ctx, u.ID, "mfa_verify", 5, 5*time.Minute); err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return ErrAuthenticatorCode
	}
	code = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(code), "-", ""))
	if err := s.Repo.RecoverMFA(ctx, u.ID, SecurityHash(session), SecurityHash(code)); err != nil {
		s.Event(ctx, u.ID, "mfa_recovery_failed", "high")
		return ErrAuthenticatorCode
	}
	s.Event(ctx, u.ID, "mfa_recovery_used", "high")
	return nil
}
func (s *SecurityService) StartEvidence(ctx context.Context, userID int, session, quote string) (string, error) {
	if !s.Available() {
		return "", ErrSecurityUnavailable
	}
	if !requestToken.MatchString(quote) {
		return "", repository.ErrPrivateEvidence
	}
	if err := s.Limit(ctx, userID, "evidence_capture", 10, time.Hour); err != nil {
		return "", err
	}
	token, err := securityToken(32)
	if err != nil {
		return "", err
	}
	reference := fmt.Sprintf("WITHDRAW-%d-%s", userID, quote)
	if err = s.Repo.Challenge(ctx, userID, SecurityHash(session), quote, reference, SecurityHash(token)); err != nil {
		return "", err
	}
	return token, nil
}
func (s *SecurityService) SaveEvidence(ctx context.Context, userID int, session, token string, data []byte, e models.WithdrawalEvidence) error {
	if err := s.Limit(ctx, userID, "evidence_upload", 12, time.Hour); err != nil {
		return err
	}
	now := time.Now()
	if len(token) != 64 || math.IsNaN(e.Latitude) || math.IsNaN(e.Longitude) || math.IsNaN(e.AccuracyM) || math.IsInf(e.Latitude, 0) || math.IsInf(e.Longitude, 0) || math.IsInf(e.AccuracyM, 0) || e.Latitude < -90 || e.Latitude > 90 || e.Longitude < -180 || e.Longitude > 180 || e.AccuracyM < 0 || e.AccuracyM > 1000 || e.CapturedAt.Before(now.Add(-2*time.Minute)) || e.CapturedAt.After(now.Add(30*time.Second)) {
		return errors.New("capture a fresh, accurate location together with your selfie")
	}
	encrypted, err := s.PrivateImage(data)
	if err != nil {
		return err
	}
	return s.Repo.SaveEvidence(ctx, userID, SecurityHash(session), SecurityHash(token), encrypted, e)
}
