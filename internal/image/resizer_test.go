package image

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

func TestResize(t *testing.T) {
	// Create a simple 100x200 red image
	img := image.NewRGBA(image.Rect(0, 0, 100, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}

	// Encode it to a buffer
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, nil)
	if err != nil {
		t.Fatalf("failed to encode test image: %v", err)
	}

	// Call Resize to fit within 50x50
	// Since original is 100x200 (1:2 ratio), fitting it in 50x50 should result in 25x50
	resizedBytes, err := Resize(&buf, 50, 50)
	if err != nil {
		t.Fatalf("Resize failed: %v", err)
	}

	// Decode the result to check dimensions
	resizedImg, err := jpeg.Decode(bytes.NewReader(resizedBytes))
	if err != nil {
		t.Fatalf("failed to decode resized image: %v", err)
	}

	bounds := resizedImg.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if width != 25 || height != 50 {
		t.Errorf("expected dimensions 25x50, got %dx%d", width, height)
	}
}
