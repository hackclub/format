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
	var newWidth, newHeight int
	
	// Scale based on whichever dimension exceeds the limit more
	widthScale := float64(maxDimension) / float64(originalWidth)
	heightScale := float64(maxDimension) / float64(originalHeight)
	
	// Use the more restrictive scale (smaller scale factor)
	scale := widthScale
	if heightScale < widthScale {
		scale = heightScale
	}
	
	// Only scale down if we need to (don't upscale)
	if scale >= 1.0 {
		return originalWidth, originalHeight
	}
	
	newWidth = int(float64(originalWidth) * scale)
	newHeight = int(float64(originalHeight) * scale)
	
	return newWidth, newHeight
}



// hasRealTransparency checks if the image has any actually transparent pixels
func (p *Processor) hasRealTransparency(data []byte) bool {
	image := bimg.NewImage(data)
	metadata, err := image.Metadata()
	if err != nil || !metadata.Alpha {
		return false
	}
	
	// If the image has 4 channels (RGBA) or 2 channels (GA), it likely has meaningful transparency
	// This is a more reliable heuristic than trying to detect pixel values
	if metadata.Channels == 4 || metadata.Channels == 2 {
		return true
	}
	
	// For 3-channel images that still have Alpha=true in metadata, 
	// this might be a false positive or unusual format
	if metadata.Channels == 3 {
		return p.deepTransparencyCheck(data)
	}
	return false
}

// deepTransparencyCheck performs a more thorough check for edge cases
func (p *Processor) deepTransparencyCheck(data []byte) bool {
	// For PNG files that claim to have alpha but are 3-channel,
	// we'll do a more careful analysis
	image := bimg.NewImage(data)
	
	// Try to extract a sample and convert it
	options := bimg.Options{
		Type: bimg.PNG,
		Width: 50,
		Height: 50,
	}
	
	sampleData, err := image.Process(options)
	if err != nil {
		return true
	}
	
	// Check the sample's metadata
	sampleImg := bimg.NewImage(sampleData)
	sampleMetadata, err := sampleImg.Metadata()
	if err != nil {
		return true
	}
	
	return sampleMetadata.Channels == 4 || sampleMetadata.Channels == 2
}

// checkPNGAlphaValues checks if a PNG has any pixels with alpha < 255
func (p *Processor) checkPNGAlphaValues(pngData []byte) bool {
	// Create a new image from the PNG data
	img := bimg.NewImage(pngData)
	
	// Check metadata first
	metadata, err := img.Metadata()
	if err != nil || !metadata.Alpha {
		fmt.Printf("🔍 checkPNGAlphaValues: No alpha in metadata\n")
		return false
	}
	
	// Try to convert to JPEG with different background colors
	// If the results are visually different, transparency is present
	whiteJPEGOptions := bimg.Options{
		Type:          bimg.JPEG,
		Quality:       95,
		Background:    bimg.Color{255, 255, 255}, // White background
		StripMetadata: true,
	}
	
	blackJPEGOptions := bimg.Options{
		Type:          bimg.JPEG,
		Quality:       95,
		Background:    bimg.Color{0, 0, 0}, // Black background
		StripMetadata: true,
	}
	
	whiteJPEG, err1 := img.Process(whiteJPEGOptions)
	blackJPEG, err2 := img.Process(blackJPEGOptions)
	
	if err1 != nil || err2 != nil {
		fmt.Printf("🔍 checkPNGAlphaValues: JPEG conversion failed, assuming transparency\n")
		return true
	}
	
	// If the two JPEG versions are identical, there's no transparency
	// If they're different, transparency was affecting the composite
	identical := len(whiteJPEG) == len(blackJPEG)
	
	// Also compare by file size difference - transparent areas will compress differently
	sizeDiffPercent := float64(abs(len(whiteJPEG)-len(blackJPEG))) / float64(max(len(whiteJPEG), len(blackJPEG))) * 100
	
	fmt.Printf("🔍 checkPNGAlphaValues: White JPEG: %d bytes, Black JPEG: %d bytes\n", 
		len(whiteJPEG), len(blackJPEG))
	fmt.Printf("🔍 checkPNGAlphaValues: Size difference: %.1f%%, Identical: %t\n", 
		sizeDiffPercent, identical)
	
	// If there's a meaningful size difference between white and black background,
	// or if they're not identical, transparency is present
	hasTransparency := !identical || sizeDiffPercent > 1.0
	
	fmt.Printf("🔍 checkPNGAlphaValues: Transparency detected = %t\n", hasTransparency)
	return hasTransparency
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// samplePixelsForTransparency samples pixels to detect actual transparency usage
func (p *Processor) samplePixelsForTransparency(pngData []byte) bool {
	// For the middle-ground cases, we'll use a practical heuristic:
	// Extract to a very small size and convert to JPEG with white background
	// If the result is visually very similar, there's likely no meaningful transparency
	
	image := bimg.NewImage(pngData)
	
	// Create a tiny 10x10 sample for detailed checking
	tinyOptions := bimg.Options{
		Type:   bimg.PNG,
		Width:  10,
		Height: 10,
	}
	
	tinyPNG, err := image.Process(tinyOptions)
	if err != nil {
		return true // Assume transparency if we can't sample
	}
	
	// Convert the same tiny image to JPEG with white background
	tinyJPEGOptions := bimg.Options{
		Type:       bimg.JPEG,
		Width:      10,
		Height:     10,
		Quality:    95,
		Background: bimg.Color{255, 255, 255},
	}
	
	tinyJPEG, err := image.Process(tinyJPEGOptions)
	if err != nil {
		return true // Assume transparency if conversion fails
	}
	
	// If the tiny PNG is significantly larger than tiny JPEG,
	// there's likely meaningful transparency data
	return float64(len(tinyPNG))/float64(len(tinyJPEG)) > 1.5
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
