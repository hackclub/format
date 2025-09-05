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

// Base32Key creates a secure key from hash using base32
// Uses 26 characters (130 bits) to prevent brute force attacks
func Base32Key(data []byte, ext string) string {
	hash := sha256.Sum256(data)
	encoder := base32.StdEncoding.WithPadding(base32.NoPadding)
	encoded := encoder.EncodeToString(hash[:])
	
	// Take 26 chars for 130 bits of entropy (collision-resistant and brute-force proof)
	key := strings.ToLower(encoded)[:26]
	
	// 2-char sharding for directory structure  
	return fmt.Sprintf("%s/%s%s", key[:2], key[2:], ext)
}
