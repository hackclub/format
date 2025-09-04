package util

import (
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"strings"
)

// HashBytes computes SHA256 hash of the given bytes
func HashBytes(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// Base32Key creates a shorter key from hash using base32
func Base32Key(data []byte, ext string) string {
	hash := sha256.Sum256(data)
	encoder := base32.StdEncoding.WithPadding(base32.NoPadding)
	encoded := encoder.EncodeToString(hash[:])
	
	// Take first 9 chars and make lowercase for consistency
	key := strings.ToLower(encoded)[:9]
	
	// Shard by first two chars
	return fmt.Sprintf("%s/%s%s", key[:2], key[2:], ext)
}
