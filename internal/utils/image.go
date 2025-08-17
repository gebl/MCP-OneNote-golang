// Copyright (c) 2025 Gabriel Lawrence
//
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

// image.go - Image processing utilities for OneNote operations.
//
// This file contains image processing and manipulation utilities used for
// handling images in OneNote pages, including scaling and encoding functions.
//
// Key Features:
// - Image scaling for large images
// - Base64 encoding for image data
// - Support for multiple image formats (JPEG, PNG, GIF)
// - Quality optimization for web display
//
// Usage Example:
//   scaledImage, wasScaled, err := ScaleImageIfNeeded(imageData, "image/jpeg", 1024, 768)
//   if err != nil {
//       // Handle error
//   }
//
//   base64Data := EncodeToBase64(imageData)

package utils

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"strings"

	"golang.org/x/image/draw"

	"github.com/gebl/onenote-mcp-server/internal/logging"
)

// ScaleImageIfNeeded scales down large images to a more manageable size.
// maxWidth and maxHeight define the maximum dimensions for the scaled image.
// Returns the scaled image bytes and whether scaling was performed.
func ScaleImageIfNeeded(imageData []byte, contentType string, maxWidth, maxHeight int) ([]byte, bool, error) {
	// Check if this is an image type we can process
	if !strings.HasPrefix(contentType, "image/") {
		return imageData, false, nil
	}

	// Decode the image
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		logging.GetLogger("utils").Debug("Failed to decode image for scaling", "error", err)
		return imageData, false, nil // Return original data if decoding fails
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Check if image needs scaling
	if width <= maxWidth && height <= maxHeight {
		logging.GetLogger("utils").Debug("Image size within limits, no scaling needed", "width", width, "height", height, "max_width", maxWidth, "max_height", maxHeight)
		return imageData, false, nil
	}

	logging.GetLogger("utils").Debug("Scaling image to fit within limits", "original_width", width, "original_height", height, "max_width", maxWidth, "max_height", maxHeight)

	// Calculate new dimensions while maintaining aspect ratio
	scaleX := float64(maxWidth) / float64(width)
	scaleY := float64(maxHeight) / float64(height)
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	newWidth := int(float64(width) * scale)
	newHeight := int(float64(height) * scale)

	// Create a new image with the scaled dimensions
	scaledImg := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// Scale the image using high-quality interpolation
	draw.ApproxBiLinear.Scale(scaledImg, scaledImg.Bounds(), img, img.Bounds(), draw.Over, nil)

	// Encode the scaled image back to bytes
	var buf bytes.Buffer
	switch contentType {
	case "image/jpeg", "image/jpg":
		err = jpeg.Encode(&buf, scaledImg, &jpeg.Options{Quality: 85})
	case "image/png":
		err = png.Encode(&buf, scaledImg)
	case "image/gif":
		err = gif.Encode(&buf, scaledImg, nil)
	default:
		logging.GetLogger("utils").Debug("Unsupported image format for scaling", "content_type", contentType)
		return imageData, false, nil
	}

	if err != nil {
		logging.GetLogger("utils").Debug("Failed to encode scaled image", "error", err)
		return imageData, false, nil
	}

	scaledData := buf.Bytes()
	logging.GetLogger("utils").Debug("Successfully scaled image",
		"original_bytes", len(imageData), "scaled_bytes", len(scaledData),
		"original_width", width, "original_height", height,
		"new_width", newWidth, "new_height", newHeight)

	return scaledData, true, nil
}

// EncodeToBase64 encodes bytes to a base64 string with a data URI prefix.
func EncodeToBase64(data []byte) string {
	return "data:;base64," + strings.TrimRight(strings.TrimSpace(fmt.Sprintf("%x", data)), "=")
}
