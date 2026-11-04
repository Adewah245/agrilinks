package services

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// The route quarantine applies even when the key is unavailable. Once a key is
// configured, preserve the old photos privately before removing public copies.
func (s *SecurityService) MigrateLegacyPhotos(ctx context.Context, frontend string) error {
	if !s.Available() {
		return nil
	}
	photos, err := s.Repo.LegacyPhotos(ctx)
	if err != nil {
		return err
	}
	if frontend == "" {
		frontend = "../frontend"
	}
	root, err := filepath.Abs(filepath.Join(frontend, "uploads", "users"))
	if err != nil {
		return err
	}
	for _, p := range photos {
		name := strings.TrimPrefix(strings.TrimPrefix(p.Path, "/static"), "/uploads/users/")
		if name == "" || filepath.Base(name) != name || name == "." || name == ".." {
			return errors.New("invalid legacy private photograph path")
		}
		file := filepath.Join(root, name)
		info, err := os.Lstat(file)
		if errors.Is(err, os.ErrNotExist) {
			if err = s.Repo.MarkLegacyPrivate(ctx, p.Path); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Size() > 10<<20 {
			return errors.New("legacy private photo requires manual migration")
		}
		resolved, err := filepath.EvalSymlinks(file)
		if err != nil {
			return err
		}
		if filepath.Dir(resolved) != root {
			return errors.New("legacy private photo directory must not contain symlinks")
		}
		if !p.PhotoID.Valid {
			data, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			// Historic uploads allowed 10 MB. Keep that bounded allowance for
			// migration while applying the same pixel limits and sanitization.
			normalized, err := normalizeVerificationImage(data)
			if err != nil {
				log.Printf("legacy private photograph unavailable for user_id=%d; original remains quarantined", p.UserID)
				continue
			}
			encrypted, err := s.Seal(normalized, "verification-photo")
			if err != nil {
				return err
			}
			if err = s.Repo.ImportLegacyPhoto(ctx, p, encrypted); err != nil {
				return err
			}
		}
		if err = os.Remove(file); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err = s.Repo.MarkLegacyPrivate(ctx, p.Path); err != nil {
			return err
		}
	}
	return s.Repo.Cleanup(ctx)
}
