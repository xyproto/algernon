package permissionsql

import (
	"crypto/sha256"
	"crypto/subtle"

	"golang.org/x/crypto/bcrypt"
	"io"
)

// Hash the password with sha256 (the username is needed for salting)
func hash_sha256(cookieSecret, username, password string) []byte {
	hasher := sha256.New()
	// Use the cookie secret as additional salt
	io.WriteString(hasher, password+cookieSecret+username)
	return hasher.Sum(nil)
}

// Hash the password with bcrypt
func hash_bcrypt(password string) []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic("Permissions: bcrypt password hashing unsuccessful")
	}
	return hash
}

// Check if a given password(+username) is correct, for a given sha256 hash
func correct_sha256(hash []byte, cookieSecret, username, password string) bool {
	comparisonHash := hash_sha256(cookieSecret, username, password)
	// check that the lengths are equal before calling ConstantTimeCompare
	if len(hash) != len(comparisonHash) {
		return false
	}
	// prevents timing attack
	return subtle.ConstantTimeCompare(hash, comparisonHash) == 1
}

// Check if a given password is correct, for a given bcrypt hash
func correct_bcrypt(hash []byte, password string) bool {
	// prevents timing attack
	return bcrypt.CompareHashAndPassword(hash, []byte(password)) == nil
}

// Check if the given hash is sha256 (when the alternative is only bcrypt)
func is_sha256(hash []byte) bool {
	return len(hash) == 32
}
