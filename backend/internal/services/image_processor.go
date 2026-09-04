package services

import (
	"bytes"
	"errors"
	"image/jpeg"

	"github.com/disintegration/imaging"
)

// prepareImage resizes and compresses an uploaded image before it
// is sent to the AI provider.
//
// Large images from mobile phones can increase upload size and
// AI processing time. The image is resized so that its longest
// side is at most 1600 pixels and then encoded as a JPEG at
// quality 80.
//
// The original image is not modified.
func prepareImage(imageData []byte) ([]byte, error) {
	// ----------------------------------------------------------
	// 1. Validate the uploaded image data.
	// ----------------------------------------------------------

	if len(imageData) == 0 {
		return nil, errors.New("image data is empty")
	}

	// ----------------------------------------------------------
	// 2. Decode the uploaded image.
	// ----------------------------------------------------------

	img, err := imaging.Decode(
		bytes.NewReader(imageData),
	)
	if err != nil {
		return nil, err
	}

	// ----------------------------------------------------------
	// 3. Resize the image.
	//
	// The longest side will be at most 1600 pixels.
	//
	// The original aspect ratio is preserved.
	// ----------------------------------------------------------

	img = imaging.Fit(
		img,
		1600,
		1600,
		imaging.Lanczos,
	)

	// ----------------------------------------------------------
	// 4. Create a buffer for the compressed image.
	// ----------------------------------------------------------

	var output bytes.Buffer

	// ----------------------------------------------------------
	// 5. Encode the image as JPEG.
	//
	// Quality 80 provides a reasonable balance between:
	//
	// - Image quality
	// - File size
	// - Upload speed
	// - AI processing efficiency
	// ----------------------------------------------------------

	err = jpeg.Encode(
		&output,
		img,
		&jpeg.Options{
			Quality: 80,
		},
	)
	if err != nil {
		return nil, err
	}

	// ----------------------------------------------------------
	// 6. Return the processed image.
	// ----------------------------------------------------------

	return output.Bytes(), nil
}
