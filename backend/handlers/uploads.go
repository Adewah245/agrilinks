package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// saveUploadedFile reads a file from the given multipart form field, saves
// it under frontend/uploads/<subdir>/, and returns the public URL to store
// on the record. Files are served through the existing /static/ file
// server (which points at ../frontend), so the returned URL looks like
// /static/uploads/<subdir>/<filename>.
//
// If the field was left empty, it returns ("", nil) rather than an error —
// callers should treat an empty return as "no new file was uploaded."
func saveUploadedFile(r *http.Request, field, subdir string) (string, error) {
	file, header, err := r.FormFile(field)
	if err != nil {
		if err == http.ErrMissingFile {
			return "", nil
		}
		return "", err
	}
	defer file.Close()

	dir := filepath.Join("..", "frontend", "uploads", subdir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	name := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	path := filepath.Join(dir, name)

	dst, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return "", err
	}

	return "/static/uploads/" + subdir + "/" + name, nil
}
