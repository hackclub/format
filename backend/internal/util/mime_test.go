package util

import (
	"testing"
)

func TestIsImageMIME(t *testing.T) {
	tests := []struct {
		mime     string
		expected bool
	}{
		{"image/jpeg", true},
		{"image/jpg", true},
		{"image/png", true},
		{"image/webp", true},
		{"image/gif", true},
		{"text/plain", false},
		{"application/json", false},
		{"", false},
	}

	for _, test := range tests {
		result := IsImageMIME(test.mime)
		if result != test.expected {
			t.Errorf("IsImageMIME(%s) = %v, expected %v", test.mime, result, test.expected)
		}
	}
}

func TestGetImageExtension(t *testing.T) {
	tests := []struct {
		mime     string
		expected string
	}{
		{"image/jpeg", ".jpg"},
		{"image/jpg", ".jpg"},
		{"image/png", ".png"},
		{"image/webp", ".webp"},
		{"image/gif", ".gif"},
		{"unknown/type", ".jpg"}, // fallback
	}

	for _, test := range tests {
		result := GetImageExtension(test.mime)
		if result != test.expected {
			t.Errorf("GetImageExtension(%s) = %s, expected %s", test.mime, result, test.expected)
		}
	}
}

func TestShouldConvertToJPEG(t *testing.T) {
	tests := []struct {
		mime            string
		hasTransparency bool
		expected        bool
	}{
		{"image/png", false, true},   // PNG without transparency should convert
		{"image/png", true, false},   // PNG with transparency should not convert
		{"image/jpeg", false, false}, // JPEG should stay JPEG
		{"image/webp", false, true},  // WebP without transparency should convert
		{"image/tiff", false, true},  // TIFF should convert
	}

	for _, test := range tests {
		result := ShouldConvertToJPEG(test.mime, test.hasTransparency)
		if result != test.expected {
			t.Errorf("ShouldConvertToJPEG(%s, %v) = %v, expected %v", 
				test.mime, test.hasTransparency, result, test.expected)
		}
	}
}
