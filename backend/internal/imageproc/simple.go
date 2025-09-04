package imageproc

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"bytes"
	"github.com/hackclub/format/internal/util"
)

// Simple processor without libvips dependency for initial testing
type SimpleProcessor struct {
	maxWidth  int
	maxHeight int
	jpegQuality int
}

func NewSimpleProcessor(maxWidth, maxHeight, jpegQuality int) *SimpleProcessor {
	return &SimpleProcessor{
		maxWidth:    maxWidth,
		maxHeight:   maxHeight,
		jpegQuality: jpegQuality,
	}
}

func (p *SimpleProcessor) Process(data []byte, originalContentType string) (*ProcessResult, error) {
	// Validate input is an image
	if !util.IsImageMIME(originalContentType) {
		detectedType := util.DetectContentType(data)
		if !util.IsImageMIME(detectedType) {
			return nil, fmt.Errorf("input is not a valid image format")
		}
		originalContentType = detectedType
	}

	// Decode the image
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %v", err)
	}

	originalSize := len(data)
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// For now, just re-encode without resizing
	var processedData []byte
	var outputContentType string

	// Simple format decision - convert large PNGs to JPEG
	shouldConvertToJPEG := format == "png" && originalSize > 1024*1024 // > 1MB

	if shouldConvertToJPEG || format == "jpeg" {
		// Encode as JPEG
		var buf bytes.Buffer
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: p.jpegQuality})
		if err != nil {
			return nil, fmt.Errorf("failed to encode as JPEG: %v", err)
		}
		processedData = buf.Bytes()
		outputContentType = "image/jpeg"
	} else {
		// Keep as PNG
		var buf bytes.Buffer
		err = png.Encode(&buf, img)
		if err != nil {
			return nil, fmt.Errorf("failed to encode as PNG: %v", err)
		}
		processedData = buf.Bytes()
		outputContentType = "image/png"
	}

	return &ProcessResult{
		Data:           processedData,
		ContentType:    outputContentType,
		Width:          width,
		Height:         height,
		HasAlpha:       format == "png", // Simplified
		OriginalSize:   originalSize,
		CompressedSize: len(processedData),
	}, nil
}
