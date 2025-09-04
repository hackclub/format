package util

import (
	"testing"
)

func TestHashBytes(t *testing.T) {
	testData := []byte("test data")
	hash := HashBytes(testData)
	
	if hash == "" {
		t.Error("HashBytes returned empty string")
	}
	
	// Hash should be consistent
	hash2 := HashBytes(testData)
	if hash != hash2 {
		t.Error("HashBytes returned different hashes for same data")
	}
	
	// Different data should produce different hashes
	differentData := []byte("different data")
	differentHash := HashBytes(differentData)
	if hash == differentHash {
		t.Error("HashBytes returned same hash for different data")
	}
}

func TestBase32Key(t *testing.T) {
	testData := []byte("test image data")
	ext := ".jpg"
	
	key := Base32Key(testData, ext)
	
	if key == "" {
		t.Error("Base32Key returned empty string")
	}
	
	if key[2:3] != "/" {
		t.Error("Base32Key should have slash separator at position 2")
	}
	
	if key[len(key)-4:] != ext {
		t.Error("Base32Key should end with extension")
	}
	
	// Key should be consistent
	key2 := Base32Key(testData, ext)
	if key != key2 {
		t.Error("Base32Key returned different keys for same data")
	}
}
