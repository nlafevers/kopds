package image

import (
	"bytes"
	"image/jpeg"
	"io"

	"github.com/disintegration/imaging"
)

// Resize takes an image from the provided io.Reader, resizes it to fit within the
// specified width and height while preserving the aspect ratio using Lanczos resampling,
// and returns the resulting image as a JPEG byte slice.
func Resize(src io.Reader, width, height int) ([]byte, error) {
	// Decode the image from the reader
	img, err := imaging.Decode(src)
	if err != nil {
		return nil, err
	}

	// Resize the image to fit the specified dimensions while preserving aspect ratio
	// imaging.Fit is better than imaging.Resize if we want to ensure it fits within bounds
	// without stretching, which is usually what's desired for covers.
	resizedImg := imaging.Fit(img, width, height, imaging.Lanczos)

	// Encode the result to JPEG
	var buf bytes.Buffer
	err = jpeg.Encode(&buf, resizedImg, &jpeg.Options{Quality: 90})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
