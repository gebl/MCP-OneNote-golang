// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package utils

import (
	"bytes"
	"encoding/hex"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
)

// createTestImage generates a test image with specified dimensions and format
func createTestImage(width, height int, format string) ([]byte, error) {
	// Create a simple colored rectangle
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	
	// Fill with a simple pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Simple gradient pattern
			r := uint8((x * 255) / width)
			g := uint8((y * 255) / height)
			b := uint8(128)
			a := uint8(255)
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: a})
		}
	}
	
	var buf bytes.Buffer
	switch format {
	case "image/jpeg", "image/jpg":
		err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
		return buf.Bytes(), err
	case "image/png":
		err := png.Encode(&buf, img)
		return buf.Bytes(), err
	case "image/gif":
		err := gif.Encode(&buf, img, nil)
		return buf.Bytes(), err
	default:
		return nil, nil
	}
}

func TestScaleImageIfNeeded(t *testing.T) {
	tests := []struct {
		name              string
		imageWidth        int
		imageHeight       int
		contentType       string
		maxWidth          int
		maxHeight         int
		expectScaled      bool
		expectError       bool
		skipImageCreation bool
		customImageData   []byte
	}{
		{
			name:         "Image within limits - no scaling needed",
			imageWidth:   500,
			imageHeight:  400,
			contentType:  "image/jpeg",
			maxWidth:     1024,
			maxHeight:    768,
			expectScaled: false,
			expectError:  false,
		},
		{
			name:         "Image too wide - needs scaling",
			imageWidth:   1500,
			imageHeight:  600,
			contentType:  "image/jpeg",
			maxWidth:     1024,
			maxHeight:    768,
			expectScaled: true,
			expectError:  false,
		},
		{
			name:         "Image too tall - needs scaling",
			imageWidth:   800,
			imageHeight:  1000,
			contentType:  "image/jpeg",
			maxWidth:     1024,
			maxHeight:    768,
			expectScaled: true,
			expectError:  false,
		},
		{
			name:         "Image too large in both dimensions",
			imageWidth:   2000,
			imageHeight:  1500,
			contentType:  "image/jpeg",
			maxWidth:     1024,
			maxHeight:    768,
			expectScaled: true,
			expectError:  false,
		},
		{
			name:         "PNG format scaling",
			imageWidth:   1500,
			imageHeight:  1200,
			contentType:  "image/png",
			maxWidth:     1024,
			maxHeight:    768,
			expectScaled: true,
			expectError:  false,
		},
		{
			name:         "GIF format scaling",
			imageWidth:   1200,
			imageHeight:  1000,
			contentType:  "image/gif",
			maxWidth:     1024,
			maxHeight:    768,
			expectScaled: true,
			expectError:  false,
		},
		{
			name:              "Non-image content type",
			contentType:       "text/plain",
			maxWidth:          1024,
			maxHeight:         768,
			expectScaled:      false,
			expectError:       false,
			skipImageCreation: true,
			customImageData:   []byte("not an image"),
		},
		{
			name:              "Unsupported image format",
			contentType:       "image/webp",
			maxWidth:          1024,
			maxHeight:         768,
			expectScaled:      false,
			expectError:       false,
			skipImageCreation: true,
			customImageData:   []byte("fake webp data"),
		},
		{
			name:              "Invalid image data",
			contentType:       "image/jpeg",
			maxWidth:          1024,
			maxHeight:         768,
			expectScaled:      false,
			expectError:       false,
			skipImageCreation: true,
			customImageData:   []byte("invalid jpeg data"),
		},
		{
			name:         "Very small image - no scaling",
			imageWidth:   50,
			imageHeight:  50,
			contentType:  "image/png",
			maxWidth:     1024,
			maxHeight:    768,
			expectScaled: false,
			expectError:  false,
		},
		{
			name:         "Exact size match - no scaling",
			imageWidth:   1024,
			imageHeight:  768,
			contentType:  "image/jpeg",
			maxWidth:     1024,
			maxHeight:    768,
			expectScaled: false,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var imageData []byte
			var err error

			if tt.skipImageCreation {
				imageData = tt.customImageData
			} else {
				imageData, err = createTestImage(tt.imageWidth, tt.imageHeight, tt.contentType)
				if err != nil {
					t.Fatalf("Failed to create test image: %v", err)
				}
			}

			scaledData, wasScaled, err := ScaleImageIfNeeded(imageData, tt.contentType, tt.maxWidth, tt.maxHeight)

			if tt.expectError && err == nil {
				t.Error("Expected an error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if wasScaled != tt.expectScaled {
				t.Errorf("Expected wasScaled=%v, got %v", tt.expectScaled, wasScaled)
			}

			if tt.expectScaled {
				// Verify that scaling actually occurred by checking the image dimensions
				if len(scaledData) == 0 {
					t.Error("Expected scaled image data but got empty result")
				}

				// For valid image formats, verify the dimensions changed
				if strings.HasPrefix(tt.contentType, "image/") && !tt.skipImageCreation {
					img, _, decodeErr := image.Decode(bytes.NewReader(scaledData))
					if decodeErr != nil {
						t.Errorf("Failed to decode scaled image: %v", decodeErr)
					} else {
						bounds := img.Bounds()
						scaledWidth := bounds.Dx()
						scaledHeight := bounds.Dy()

						if scaledWidth > tt.maxWidth {
							t.Errorf("Scaled width %d exceeds max width %d", scaledWidth, tt.maxWidth)
						}
						if scaledHeight > tt.maxHeight {
							t.Errorf("Scaled height %d exceeds max height %d", scaledHeight, tt.maxHeight)
						}
						if scaledWidth == tt.imageWidth && scaledHeight == tt.imageHeight {
							t.Error("Image dimensions didn't change despite scaling flag being true")
						}
					}
				}
			} else {
				// If not scaled, data should be identical
				if !bytes.Equal(imageData, scaledData) {
					t.Error("Expected identical data when not scaling, but data changed")
				}
			}
		})
	}
}

func TestScaleImageIfNeeded_AspectRatioPreservation(t *testing.T) {
	// Test that aspect ratio is preserved during scaling
	originalWidth := 1600
	originalHeight := 900 // 16:9 aspect ratio
	maxWidth := 1024
	maxHeight := 768

	imageData, err := createTestImage(originalWidth, originalHeight, "image/jpeg")
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	scaledData, wasScaled, err := ScaleImageIfNeeded(imageData, "image/jpeg", maxWidth, maxHeight)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !wasScaled {
		t.Fatal("Expected image to be scaled")
	}

	img, _, err := image.Decode(bytes.NewReader(scaledData))
	if err != nil {
		t.Fatalf("Failed to decode scaled image: %v", err)
	}

	bounds := img.Bounds()
	scaledWidth := bounds.Dx()
	scaledHeight := bounds.Dy()

	// Calculate aspect ratios
	originalAspect := float64(originalWidth) / float64(originalHeight)
	scaledAspect := float64(scaledWidth) / float64(scaledHeight)

	// Allow small tolerance for floating point precision
	if abs(originalAspect-scaledAspect) > 0.01 {
		t.Errorf("Aspect ratio not preserved: original=%.3f, scaled=%.3f", originalAspect, scaledAspect)
	}

	// Ensure at least one dimension reaches the limit
	if scaledWidth != maxWidth && scaledHeight != maxHeight {
		t.Error("Neither dimension reaches its maximum limit")
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func TestEncodeToBase64(t *testing.T) {
	tests := []struct {
		name         string
		input        []byte
		expectedPrefix string
	}{
		{
			name:           "Empty data",
			input:          []byte{},
			expectedPrefix: "data:;base64,",
		},
		{
			name:           "Simple text data",
			input:          []byte("Hello, World!"),
			expectedPrefix: "data:;base64,",
		},
		{
			name:           "Binary data",
			input:          []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD},
			expectedPrefix: "data:;base64,",
		},
		{
			name:           "JPEG header bytes",
			input:          []byte{0xFF, 0xD8, 0xFF, 0xE0},
			expectedPrefix: "data:;base64,",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EncodeToBase64(tt.input)

			// Check that result starts with the expected prefix
			if !strings.HasPrefix(result, tt.expectedPrefix) {
				t.Errorf("Expected result to start with %q, got %q", tt.expectedPrefix, result)
			}

			// Check that result has content after the prefix (unless input is empty)
			if len(tt.input) > 0 && len(result) == len(tt.expectedPrefix) {
				t.Error("Expected base64 encoded content after prefix")
			}

			// Verify we can decode the hex part back to original bytes
			hexPart := strings.TrimPrefix(result, tt.expectedPrefix)
			if len(hexPart) > 0 {
				decoded, err := hex.DecodeString(hexPart)
				if err != nil {
					t.Errorf("Failed to decode hex part: %v", err)
				} else if !bytes.Equal(decoded, tt.input) {
					t.Errorf("Decoded data doesn't match input. Expected %v, got %v", tt.input, decoded)
				}
			}
		})
	}
}

func TestEncodeToBase64_ConsistentResults(t *testing.T) {
	// Test that the same input produces the same output
	testData := []byte("Consistent test data")
	
	result1 := EncodeToBase64(testData)
	result2 := EncodeToBase64(testData)
	
	if result1 != result2 {
		t.Errorf("EncodeToBase64 should produce consistent results. Got %q and %q", result1, result2)
	}
}

// Benchmark tests for performance-critical functions
func BenchmarkScaleImageIfNeeded_NoScaling(b *testing.B) {
	imageData, err := createTestImage(800, 600, "image/jpeg")
	if err != nil {
		b.Fatalf("Failed to create test image: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = ScaleImageIfNeeded(imageData, "image/jpeg", 1024, 768)
	}
}

func BenchmarkScaleImageIfNeeded_WithScaling(b *testing.B) {
	imageData, err := createTestImage(2000, 1500, "image/jpeg")
	if err != nil {
		b.Fatalf("Failed to create test image: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = ScaleImageIfNeeded(imageData, "image/jpeg", 1024, 768)
	}
}

func BenchmarkEncodeToBase64(b *testing.B) {
	testData := make([]byte, 1024) // 1KB of data
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EncodeToBase64(testData)
	}
}