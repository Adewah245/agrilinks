package handlers

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"testing"
)

// TestUploadRejectsSpoofedContentType verifies that uploads are validated by
// their bytes rather than a client-controlled multipart header.
func TestUploadRejectsSpoofedContentType(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("image", "not-an-image.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("not an image")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest("POST", "/", body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if err := request.ParseMultipartForm(1024); err != nil {
		t.Fatal(err)
	}

	file := request.MultipartForm.File["image"][0]
	opened, err := file.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()

	if _, err := saveOneUploadedFile(opened, file, "products"); err == nil {
		t.Fatal("expected disguised non-image upload to be rejected")
	}
}
