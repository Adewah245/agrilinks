package services

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

var ErrRegistrationPhotoUnavailable = errors.New("we could not save your profile picture; please try again")

// saveRegistrationProfilePhoto accepts only an already normalized JPEG. Stage
// it outside the public directories, then publish the complete file under an
// unpredictable name. The caller removes it if account creation rolls back.
func saveRegistrationProfilePhoto(frontendDir string, image []byte) (string, func(), error) {
	if frontendDir == "" {
		for _, candidate := range []string{"../frontend", "frontend", "../../../frontend"} {
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				frontendDir = candidate
				break
			}
		}
	}
	info, err := os.Stat(frontendDir)
	if err != nil {
		return "", nil, err
	}
	if !info.IsDir() {
		return "", nil, fmt.Errorf("frontend root is not a directory")
	}
	uploadRoot := filepath.Join(frontendDir, "uploads")
	profileDir := filepath.Join(uploadRoot, "profiles")
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		return "", nil, err
	}
	staged, err := os.CreateTemp(uploadRoot, ".registration-*")
	if err != nil {
		return "", nil, err
	}
	defer func() {
		staged.Close()
		removeRegistrationPhoto(staged.Name())
	}()
	if _, err := staged.Write(image); err != nil {
		return "", nil, err
	}
	if err := staged.Chmod(0644); err != nil {
		return "", nil, err
	}
	if err := staged.Sync(); err != nil {
		return "", nil, err
	}
	if err := staged.Close(); err != nil {
		return "", nil, err
	}
	name, err := securityToken(16)
	if err != nil {
		return "", nil, err
	}
	name += ".jpg"
	destination := filepath.Join(profileDir, name)
	// Linking within the upload filesystem publishes atomically and refuses
	// to overwrite an existing file, even in the event of a name collision.
	if err := os.Link(staged.Name(), destination); err != nil {
		return "", nil, err
	}
	return "/static/uploads/profiles/" + name, func() { removeRegistrationPhoto(destination) }, nil
}

func removeRegistrationPhoto(path string) {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("[registration] profile_photo.cleanup_failed: %v", err)
	}
}
