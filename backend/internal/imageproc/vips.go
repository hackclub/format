package imageproc

import (
    "bytes"
    "fmt"
    "image"
    "os/exec"

    "github.com/gen2brain/jpegli"
    "github.com/h2non/bimg"
    "github.com/hackclub/format/internal/util"
)

type Processor struct {
    jpegQuality     int
    jpegProgressive bool
    pngStrip        bool
}

type ProcessResult struct {
    Data           []byte
    ContentType    string
    Width          int
    Height         int
    HasAlpha       bool
    OriginalSize   int
    CompressedSize int
}

func NewProcessor(jpegQuality int, jpegProgressive, pngStrip bool) *Processor {
    return &Processor{
        jpegQuality:     jpegQuality,
        jpegProgressive: jpegProgressive,
        pngStrip:        pngStrip,
    }
}

const oneMB = 1024 * 1024
const maxDimension = 3840

func (p *Processor) Process(data []byte, originalContentType string) (*ProcessResult, error) {
    originalSize := len(data)

    // 1. If the file is under 1MB, don't touch it.
    if originalSize <= oneMB {
        fmt.Printf("✅ Image size is %d bytes (<= 1MB), skipping processing.\n", originalSize)
        metadata, err := bimg.NewImage(data).Metadata()
        if err != nil {
            // Could fail on non-images, but that's ok. Return original data.
            return &ProcessResult{
                Data:           data,
                ContentType:    originalContentType,
                OriginalSize:   originalSize,
                CompressedSize: originalSize,
            }, nil
        }
        return &ProcessResult{
            Data:           data,
            ContentType:    originalContentType,
            Width:          metadata.Size.Width,
            Height:         metadata.Size.Height,
            HasAlpha:       metadata.Alpha,
            OriginalSize:   originalSize,
            CompressedSize: originalSize,
        }, nil
    }

    fmt.Printf("🚀 Image size is %d bytes (> 1MB), starting SOTA processing pipeline.\n", originalSize)

    // Validate input is a supported image format
    if !util.IsImageMIME(originalContentType) {
        detectedType := util.DetectContentType(data)
        if !util.IsImageMIME(detectedType) {
            return nil, fmt.Errorf("input is not a valid image format, detected: %s", detectedType)
        }
        originalContentType = detectedType
    }

    // 2. Get image metadata
    metadata, err := bimg.NewImage(data).Metadata()
    if err != nil {
        return nil, fmt.Errorf("failed to read image metadata: %v", err)
    }

    // 3. Resize if necessary
    imageToProcess := data
    needsResize := metadata.Size.Width > maxDimension || metadata.Size.Height > maxDimension
    if needsResize {
        fmt.Printf("🔄 Image resize triggered: %dx%d -> max %dpx\n", metadata.Size.Width, metadata.Size.Height, maxDimension)
        newWidth, newHeight := calculateDimensionsWithMax(metadata.Size.Width, metadata.Size.Height, maxDimension)

        // Resize using bimg with proper format output
        resizeOptions := bimg.Options{
            Width: newWidth,
            Height: newHeight,
            Type: bimg.PNG,  // Use PNG to preserve quality for next stage
            Quality: 100,
        }
        
        resizedData, err := bimg.NewImage(data).Process(resizeOptions)
        if err != nil {
            return nil, fmt.Errorf("failed to resize image: %v", err)
        }
        imageToProcess = resizedData
    }

    // 4. Decide format and apply SOTA compression
    var processedData []byte
    var outputContentType string

    // Use more accurate transparency detection - check if image actually uses transparency
    hasRealTransparency := hasActualTransparency(data, metadata)
    shouldConvertToJPEG := util.ShouldConvertToJPEG(originalContentType, hasRealTransparency)
    
    fmt.Printf("🔍 Transparency analysis: hasAlphaChannel=%t, hasRealTransparency=%t, shouldConvertToJPEG=%t\n", 
        metadata.Alpha, hasRealTransparency, shouldConvertToJPEG)

    if shouldConvertToJPEG {
        fmt.Println("✨ Compressing with state-of-the-art jpegli...")
        outputContentType = "image/jpeg"
        processedData, err = compressWithJpegli(imageToProcess)
        if err != nil {
            return nil, fmt.Errorf("jpegli compression failed: %w", err)
        }
    } else {
        fmt.Println("✨ Compressing with oxipng...")
        outputContentType = "image/png"
        // If we resized, the intermediate is a PNG. If not, it's the original PNG.
        // In either case, it's safe to run through oxipng.
        processedData, err = compressWithOxipng(imageToProcess)
        if err != nil {
            return nil, fmt.Errorf("oxipng compression failed: %w", err)
        }
    }

    // 5. Get final metadata and return
    finalMetadata, err := bimg.NewImage(processedData).Metadata()
    if err != nil {
        return nil, fmt.Errorf("failed to read final image metadata: %v", err)
    }

    return &ProcessResult{
        Data:           processedData,
        ContentType:    outputContentType,
        Width:          finalMetadata.Size.Width,
        Height:         finalMetadata.Size.Height,
        HasAlpha:       finalMetadata.Alpha,
        OriginalSize:   originalSize,
        CompressedSize: len(processedData),
    }, nil
}

// compressWithJpegli uses the Go jpegli library for state-of-the-art JPEG compression.
func compressWithJpegli(input []byte) ([]byte, error) {
    // Decode the input image data to Go image.Image
    var img image.Image
    var err error
    
    // Try to decode as various formats
    reader := bytes.NewReader(input)
    img, _, err = image.Decode(reader)
    if err != nil {
        // Fall back to bimg if standard decoders fail
        fmt.Printf("⚠️ Standard image decode failed, falling back to bimg. Error: %v\n", err)
        return fallbackJPEGCompression(input)
    }

    // Use jpegli to encode with optimal settings
    var buf bytes.Buffer
    
    // jpegli.EncodingOptions with high quality and optimal settings
    options := &jpegli.EncodingOptions{
        Quality:               95,    // High quality for minimal loss
        ProgressiveLevel:      2,     // Maximum progressive JPEG
        OptimizeCoding:        true,  // Huffman code optimization
        AdaptiveQuantization:  true,  // Better quality
        FancyDownsampling:     true,  // Better quality
        ChromaSubsampling:     image.YCbCrSubsampleRatio444, // No chroma subsampling for max quality
    }
    
    err = jpegli.Encode(&buf, img, options)
    if err != nil {
        // Fall back to bimg if jpegli fails
        fmt.Printf("⚠️ jpegli encoding failed, falling back to bimg. Error: %v\n", err)
        return fallbackJPEGCompression(input)
    }

    fmt.Printf("✅ jpegli compression successful: %d bytes -> %d bytes (%.1f%% reduction)\n", 
        len(input), buf.Len(), float64(len(input)-buf.Len())/float64(len(input))*100)
    
    return buf.Bytes(), nil
}

// fallbackJPEGCompression uses bimg as fallback when jpegli fails
func fallbackJPEGCompression(input []byte) ([]byte, error) {
    img := bimg.NewImage(input)
    jpegOptions := bimg.Options{
        Type: bimg.JPEG,
        Quality: 90,
        StripMetadata: true,
        Interpretation: bimg.InterpretationSRGB,
    }
    
    jpegData, err := img.Process(jpegOptions)
    if err != nil {
        fmt.Printf("⚠️ Fallback JPEG compression also failed, returning original data. Error: %v", err)
        return input, nil
    }
    
    fmt.Printf("✅ Fallback bimg compression: %d bytes -> %d bytes\n", len(input), len(jpegData))
    return jpegData, nil
}

// compressWithOxipng uses `oxipng` for lossless PNG optimization.
func compressWithOxipng(input []byte) ([]byte, error) {
    // Universal web-safe default: purely lossless, keeps display-critical metadata
    cmd := exec.Command("oxipng", "-o", "4", "--strip", "safe", "-i", "0", "-")

    var out, stderr bytes.Buffer
    cmd.Stdin = bytes.NewReader(input)
    cmd.Stdout = &out
    cmd.Stderr = &stderr

    if err := cmd.Run(); err != nil {
        // If oxipng fails (e.g., on a non-PNG passed to it), just return the input
        fmt.Printf("⚠️ oxipng compression failed, returning unoptimized data. Error: %v\nStderr: %s", err, stderr.String())
        return input, nil
    }

    // oxipng returns original if it can't improve it, which results in an empty stdout.
    if out.Len() == 0 {
        return input, nil
    }

    return out.Bytes(), nil
}

// calculateDimensionsWithMax maintains aspect ratio while ensuring neither width nor height exceeds a max value.
func calculateDimensionsWithMax(originalWidth, originalHeight, maxDimension int) (int, int) {
    if originalWidth <= maxDimension && originalHeight <= maxDimension {
        return originalWidth, originalHeight
    }

    ratio := float64(originalWidth) / float64(originalHeight)

    if originalWidth > originalHeight {
        return maxDimension, int(float64(maxDimension) / ratio)
    }
    return int(float64(maxDimension) * ratio), maxDimension
}

// hasActualTransparency checks if image actually uses transparency by sampling alpha values
func hasActualTransparency(data []byte, metadata bimg.ImageMetadata) bool {
    // If no alpha channel, definitely no transparency
    if !metadata.Alpha {
        return false
    }
    
    // Decode the image using Go's standard image decoder to access raw pixel data
    reader := bytes.NewReader(data)
    img, _, err := image.Decode(reader)
    if err != nil {
        fmt.Printf("🔍 Failed to decode image for alpha sampling, assuming transparency. Error: %v\n", err)
        return true // Conservative approach - assume transparency if we can't decode
    }
    
    bounds := img.Bounds()
    width := bounds.Dx()
    height := bounds.Dy()
    
    // Sample pixels to check for actual transparency (alpha < 255)
    // Use a grid sampling approach to check pixels across the entire image
    sampleStep := max(1, max(width/20, height/20)) // Sample roughly 400 pixels (20x20 grid)
    transparentPixels := 0
    totalSampled := 0
    
    for y := bounds.Min.Y; y < bounds.Max.Y; y += sampleStep {
        for x := bounds.Min.X; x < bounds.Max.X; x += sampleStep {
            color := img.At(x, y)
            
            // Check if this color has alpha information
            if alphaColor, hasAlpha := color.(interface{ RGBA() (r, g, b, a uint32) }); hasAlpha {
                _, _, _, alpha := alphaColor.RGBA()
                totalSampled++
                
                // Alpha values are 16-bit (0-65535), so 65535 = fully opaque
                if alpha < 65535 {
                    transparentPixels++
                }
            }
        }
    }
    
    // If we found any transparent pixels, the image uses transparency
    hasTransparency := transparentPixels > 0
    
    fmt.Printf("🔍 Alpha sampling: %d/%d pixels have transparency (%.1f%%), result=%t\n", 
        transparentPixels, totalSampled, float64(transparentPixels)/float64(totalSampled)*100, hasTransparency)
    
    return hasTransparency
}

// min returns the minimum of multiple integers
func min(values ...int) int {
    if len(values) == 0 {
        return 0
    }
    minVal := values[0]
    for _, v := range values[1:] {
        if v < minVal {
            minVal = v
        }
    }
    return minVal
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
