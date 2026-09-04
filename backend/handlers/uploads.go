package handlers

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// saveUploadedFiles saves one or more uploaded image files.
//
// The function reads all files submitted under the specified form field,
// validates and saves each file, and returns browser-accessible URLs for
// the successfully saved files.
//
// The caller is responsible for deciding whether at least one uploaded
// file is required.
func saveUploadedFiles(
	r *http.Request,
	field string,
	subdir string,
) ([]string, error) {
	if r == nil {
		return nil, fmt.Errorf("request is nil")
	}

	if r.MultipartForm == nil {
		return nil, fmt.Errorf("multipart form is not available")
	}

	files := r.MultipartForm.File[field]

	if len(files) == 0 {
		return nil, nil
	}

	var uploadedURLs []string

	for _, header := range files {
		if header == nil {
			continue
		}

		file, err := header.Open()
		if err != nil {
			return nil, fmt.Errorf(
				"could not open uploaded file %q: %w",
				header.Filename,
				err,
			)
		}

		url, saveErr := saveOneUploadedFile(
			file,
			header,
			subdir,
		)

		closeErr := file.Close()

		if saveErr != nil {
			return nil, saveErr
		}

		if closeErr != nil {
			return nil, fmt.Errorf(
				"could not close uploaded file %q: %w",
				header.Filename,
				closeErr,
			)
		}

		if strings.TrimSpace(url) != "" {
			uploadedURLs = append(uploadedURLs, url)
		}
	}

	return uploadedURLs, nil
}

// saveOneUploadedFile validates and saves one uploaded image.
//
// The uploaded image must be no larger than 10 MB and must use one of
// the supported image MIME types: JPEG, PNG, or WEBP.
//
// The returned path is relative to the web server's static directory
// and can therefore be stored as the image URL used by Agro-Shield.
func saveOneUploadedFile(
	file multipart.File,
	header *multipart.FileHeader,
	subdir string,
) (string, error) {
	if file == nil {
		return "", fmt.Errorf("uploaded file is nil")
	}

	if header == nil {
		return "", fmt.Errorf("uploaded file information is missing")
	}

	// --------------------------------------------------------
	// Validate upload directory
	// --------------------------------------------------------

	subdir = strings.TrimSpace(subdir)

	if subdir == "" {
		return "", fmt.Errorf("upload subdirectory is empty")
	}

	// Prevent absolute paths and directory traversal.
	cleanSubdir := filepath.Clean(subdir)

	if cleanSubdir == "." ||
		cleanSubdir == ".." ||
		filepath.IsAbs(cleanSubdir) ||
		strings.HasPrefix(cleanSubdir, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid upload subdirectory")
	}

	// --------------------------------------------------------
	// Validate file size
	// --------------------------------------------------------

	const maxFileSize = 10 << 20 // 10 MB

	if header.Size > maxFileSize {
		return "", fmt.Errorf(
			"file %q is too large; maximum size is 10 MB",
			header.Filename,
		)
	}

	// --------------------------------------------------------
	// Validate file type
	// --------------------------------------------------------

	contentType := strings.ToLower(
		strings.TrimSpace(
			header.Header.Get("Content-Type"),
		),
	)

	allowedTypes := map[string]string{
		"image/jpeg": ".jpg",
		"image/jpg":  ".jpg",
		"image/png":  ".png",
		"image/webp": ".webp",
	}

	extension, ok := allowedTypes[contentType]

	if !ok {
		return "", fmt.Errorf(
			"unsupported image type %q; allowed types are JPG, PNG and WEBP",
			contentType,
		)
	}

	// --------------------------------------------------------
	// Create upload directory
	// --------------------------------------------------------

	uploadDir := filepath.Join(
		"frontend",
		"static",
		"uploads",
		cleanSubdir,
	)

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return "", fmt.Errorf(
			"could not create upload directory: %w",
			err,
		)
	}

	// --------------------------------------------------------
	// Generate unique filename
	// --------------------------------------------------------

	filename := fmt.Sprintf(
		"%d_%d%s",
		time.Now().UnixNano(),
		os.Getpid(),
		extension,
	)

	destination := filepath.Join(
		uploadDir,
		filename,
	)

	// --------------------------------------------------------
	// Create destination file
	// --------------------------------------------------------

	output, err := os.Create(destination)

	if err != nil {
		return "", fmt.Errorf(
			"could not create destination file: %w",
			err,
		)
	}

	// Make sure the destination file is closed if an error occurs.
	defer output.Close()

	// --------------------------------------------------------
	// Copy uploaded file
	// --------------------------------------------------------

	if _, err := io.Copy(output, file); err != nil {
		// Remove a partially written file so that incomplete uploads
		// are not left inside the uploads directory.
		_ = os.Remove(destination)

		return "", fmt.Errorf(
			"could not save uploaded file: %w",
			err,
		)
	}

	// --------------------------------------------------------
	// Return browser-accessible path
	// --------------------------------------------------------

	return "/static/uploads/" +
		cleanSubdir +
		"/" +
		filename, nil
}

// saveUploadedFile saves a single uploaded image.
//
// This function is retained for handlers that only accept one image.
// Internally it uses saveUploadedFiles and returns the first saved image.
func saveUploadedFile(
	r *http.Request,
	field string,
	subdir string,
) (string, error) {
	files, err := saveUploadedFiles(
		r,
		field,
		subdir,
	)

	if err != nil {
		return "", err
	}

	if len(files) == 0 {
		return "", nil
	}

	return files[0], nil
}
