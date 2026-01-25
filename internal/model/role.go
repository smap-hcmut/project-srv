package model

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
)

var (
	ErrInvalidRole = errors.New("invalid role")
)

// EncryptRole encrypts a role string using SHA256 and base64 encoding
// This prevents direct string comparison and obfuscates the role value
func EncryptRole(role string) (string, error) {
	// Validate role
	if role != RoleUser && role != RoleAdmin {
		return "", ErrInvalidRole
	}

	// Hash the role with a salt
	hash := sha256.Sum256([]byte(role + ":smap:role"))
	encrypted := base64.StdEncoding.EncodeToString(hash[:])

	return encrypted, nil
}

// VerifyRole verifies if a roleHash matches the given plaintext role
func VerifyRole(roleHash, plainRole string) bool {
	// Validate plainRole
	if plainRole != RoleUser && plainRole != RoleAdmin {
		return false
	}

	// Encrypt the plainRole and compare
	encrypted, err := EncryptRole(plainRole)
	if err != nil {
		return false
	}

	return roleHash == encrypted
}
