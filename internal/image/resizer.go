package image

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/disintegration/imaging"
)

const (
	// MaxCoverWidth and MaxCoverHeight bound per-request work and cache churn.
	MaxCoverWidth  = 600
	MaxCoverHeight = 900

	maxCoverInputBytes = 25 << 20
	maxCoverPixels     = 24_000_000
)

var (
	ErrInvalidDimensions = errors.New("invalid resize dimensions")
	ErrImageTooLarge     = errors.New("image is too large")
	ErrUnsupportedFormat = errors.New("unsupported image format")
)

// ValidateDimensions returns an error when requested cover dimensions exceed
// the supported bounds for OPDS thumbnail/cover generation.
func ValidateDimensions(width, height int) error {
	if width <= 0 || height <= 0 {
		return fmt.Errorf("%w: dimensions must be positive", ErrInvalidDimensions)
	}
	if width > MaxCoverWidth || height > MaxCoverHeight {
		return fmt.Errorf("%w: maximum is %dx%d", ErrInvalidDimensions, MaxCoverWidth, MaxCoverHeight)
	}
	return nil
}

// Resize takes an image from the provided io.Reader, resizes it to fit within the
// specified width and height while preserving the aspect ratio using Lanczos resampling,
// and returns the resulting image as a JPEG byte slice.
func Resize(src io.Reader, width, height int) ([]byte, error) {
	if err := ValidateDimensions(width, height); err != nil {
		return nil, err
	}

	data, err := io.ReadAll(io.LimitReader(src, maxCoverInputBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxCoverInputBytes {
		return nil, ErrImageTooLarge
	}

	img, err := decodeCover(data)
	if err != nil {
		return nil, err
	}

	resizedImg := imaging.Fit(img, width, height, imaging.Lanczos)

	var buf bytes.Buffer
	err = jpeg.Encode(&buf, resizedImg, &jpeg.Options{Quality: 90})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func decodeCover(data []byte) (image.Image, error) {
	if isJPEG(data) {
		cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		if err := validateSourceSize(cfg.Width, cfg.Height); err != nil {
			return nil, err
		}
		return jpeg.Decode(bytes.NewReader(data))
	}

	if isPNG(data) {
		cfg, err := png.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		if err := validateSourceSize(cfg.Width, cfg.Height); err != nil {
			return nil, err
		}
		return png.Decode(bytes.NewReader(data))
	}

	return nil, ErrUnsupportedFormat
}

func isJPEG(data []byte) bool {
	return len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff
}

func isPNG(data []byte) bool {
	return len(data) >= 8 &&
		data[0] == 0x89 &&
		data[1] == 0x50 &&
		data[2] == 0x4e &&
		data[3] == 0x47 &&
		data[4] == 0x0d &&
		data[5] == 0x0a &&
		data[6] == 0x1a &&
		data[7] == 0x0a
}

func validateSourceSize(width, height int) error {
	if width <= 0 || height <= 0 {
		return ErrImageTooLarge
	}
	if width > maxCoverPixels/height {
		return ErrImageTooLarge
	}
	return nil
}
