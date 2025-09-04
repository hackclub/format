package util

import (
	"mime"
	"net/http"
)

// DetectContentType detects the MIME type of the given data
func DetectContentType(data []byte) string {
	return http.DetectContentType(data)
}

// IsImageMIME checks if the MIME type is a supported image format
func IsImageMIME(contentType string) bool {
	switch contentType {
	case "image/jpeg", "image/jpg", "image/png", "image/webp", "image/gif", "image/tiff", "image/heif", "image/avif":
		return true
	default:
		return false
	}
}

// GetImageExtension returns the file extension for a given MIME type
func GetImageExtension(contentType string) string {
	switch contentType {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "image/tiff":
		return ".tiff"
	case "image/heif":
		return ".heif"
	case "image/avif":
		return ".avif"
	default:
		return ".jpg" // Default fallback
	}
}

// GetMIMEFromExtension returns the MIME type for a file extension
func GetMIMEFromExtension(ext string) string {
	return mime.TypeByExtension(ext)
}

// ShouldConvertToJPEG determines if an image should be converted to JPEG
func ShouldConvertToJPEG(contentType string, hasTransparency bool) bool {
	// Keep PNG if it has transparency
	if hasTransparency {
		return false
	}
	
	// Convert large formats to JPEG for better compression
	switch contentType {
	case "image/png", "image/tiff", "image/webp":
		return true
	case "image/jpeg", "image/jpg":
		return false // Already JPEG
	default:
		return true // Convert unknown/other formats to JPEG
	}
}
