package imageproc

import (
	"fmt"
	"github.com/h2non/bimg"
	"github.com/hackclub/format/internal/util"
)

type Processor struct {
	jpegQuality int
	jpegProgressive bool
	pngStrip bool
}

type ProcessResult struct {
	Data         []byte
	ContentType  string
	Width        int
	Height       int
	HasAlpha     bool
	OriginalSize int
	CompressedSize int
}

func NewProcessor(jpegQuality int, jpegProgressive, pngStrip bool) *Processor {
	return &Processor{
		jpegQuality: jpegQuality,
		jpegProgressive: jpegProgressive,
		pngStrip:    pngStrip,
	}
}

func (p *Processor) Process(data []byte, originalContentType string) (*ProcessResult, error) {
	// Validate input is an image
	if !util.IsImageMIME(originalContentType) {
		detectedType := util.DetectContentType(data)
		if !util.IsImageMIME(detectedType) {
			return nil, fmt.Errorf("input is not a valid image format")
		}
		originalContentType = detectedType
	}

	// Get image metadata
	metadata, err := bimg.NewImage(data).Metadata()
	if err != nil {
		return nil, fmt.Errorf("failed to read image metadata: %v", err)
	}

	originalSize := len(data)
	hasAlpha := metadata.Alpha

	// Determine if we need to resize
	// Resize if dimensions exceed 3840px OR file size > 5MB
	const maxFileSize = 5 * 1024 * 1024 // 5MB
	needsResize := metadata.Size.Width > 3840 || 
	              metadata.Size.Height > 3840 || 
	              originalSize > maxFileSize
	
	// Log resize decision
	if needsResize {
		fmt.Printf("🔄 Image resize triggered: %dx%d pixels, %d bytes (%.1fMB) - max: 3840px or 5MB\n", 
			metadata.Size.Width, metadata.Size.Height, originalSize, float64(originalSize)/(1024*1024))
	} else {
		fmt.Printf("✅ Image resize skipped: %dx%d pixels, %d bytes (%.1fMB) - within limits\n",
			metadata.Size.Width, metadata.Size.Height, originalSize, float64(originalSize)/(1024*1024))
	}

	// Create processing options
	options := bimg.Options{
		Quality:       p.jpegQuality,
		StripMetadata: true,
	}

	// Resize if needed
	if needsResize {
		// Calculate new dimensions maintaining aspect ratio with 3840px max
		newWidth, newHeight := calculateDimensionsWithMax(metadata.Size.Width, metadata.Size.Height, 3840)
		options.Width = newWidth
		options.Height = newHeight
		options.Crop = false // Use thumbnail/resize, not crop
		options.Enlarge = false // Never upscale
	}

	// Decide output format based on alpha channel and transparency
	shouldConvertToJPEG := util.ShouldConvertToJPEG(originalContentType, hasAlpha && p.hasRealTransparency(data))

	fmt.Printf("🎨 Format decision: %s → %s (hasAlpha: %t, shouldConvert: %t)\n", 
		originalContentType, 
		map[bool]string{true: "JPEG", false: "PNG"}[shouldConvertToJPEG],
		hasAlpha, 
		shouldConvertToJPEG)

	var processedData []byte
	var outputContentType string

	if shouldConvertToJPEG || originalContentType == "image/jpeg" || originalContentType == "image/jpg" {
		// Convert to JPEG (or keep as JPEG)
		options.Type = bimg.JPEG
		processedData, err = bimg.NewImage(data).Process(options)
		if err != nil {
			return nil, fmt.Errorf("failed to process image as JPEG: %v", err)
		}
		outputContentType = "image/jpeg"
	} else {
		// Keep as PNG (only for PNGs with transparency)
		options.Type = bimg.PNG
		if p.pngStrip {
			options.StripMetadata = true
		}
		processedData, err = bimg.NewImage(data).Process(options)
		if err != nil {
			return nil, fmt.Errorf("failed to process image as PNG: %v", err)
		}
		outputContentType = "image/png"
		
		// Apply PNG optimization if available
		processedData, err = p.optimizePNG(processedData)
		if err != nil {
			// If PNG optimization fails, use original processed data
			fmt.Printf("PNG optimization failed: %v\n", err)
		}
	}

	// Get final dimensions
	finalMetadata, err := bimg.NewImage(processedData).Metadata()
	if err != nil {
		// Fallback to original dimensions if we can't read new metadata
		return &ProcessResult{
			Data:           processedData,
			ContentType:    outputContentType,
			Width:          metadata.Size.Width,
			Height:         metadata.Size.Height,
			HasAlpha:       hasAlpha,
			OriginalSize:   originalSize,
			CompressedSize: len(processedData),
		}, nil
	}

	return &ProcessResult{
		Data:           processedData,
		ContentType:    outputContentType,
		Width:          finalMetadata.Size.Width,
		Height:         finalMetadata.Size.Height,
		HasAlpha:       hasAlpha,
		OriginalSize:   originalSize,
		CompressedSize: len(processedData),
	}, nil
}

func calculateDimensionsWithMax(originalWidth, originalHeight, maxDimension int) (int, int) {
	// Calculate new dimensions maintaining aspect ratio with a maximum dimension
	aspectRatio := float64(originalWidth) / float64(originalHeight)
	
	var newWidth, newHeight int
	
	if originalWidth > originalHeight {
		// Landscape - limit by width
		newWidth = maxDimension
		newHeight = int(float64(newWidth) / aspectRatio)
	} else {
		// Portrait or square - limit by height  
		newHeight = maxDimension
		newWidth = int(float64(newHeight) * aspectRatio)
	}
	
	// Ensure we don't exceed original dimensions (no upscaling)
	if newWidth > originalWidth {
		newWidth = originalWidth
	}
	if newHeight > originalHeight {
		newHeight = originalHeight
	}
	
	return newWidth, newHeight
}



// hasRealTransparency checks if the image has any actually transparent pixels
func (p *Processor) hasRealTransparency(data []byte) bool {
	// This is a simplified check - in a production system you might want
	// to implement a more sophisticated transparency detection
	image := bimg.NewImage(data)
	metadata, err := image.Metadata()
	if err != nil || !metadata.Alpha {
		return false
	}
	
	// For now, assume that if alpha channel exists, there's transparency
	// A more sophisticated implementation would sample pixels to check
	// if any are actually transparent (alpha < 255)
	return true
}

// optimizePNG attempts to optimize PNG files using external tools
func (p *Processor) optimizePNG(data []byte) ([]byte, error) {
	// This would call external tools like oxipng
	// For now, return the original data
	// In a production system, you'd implement calls to:
	// - oxipng for lossless compression
	// - libimagequant for palette optimization
	return data, nil
}
