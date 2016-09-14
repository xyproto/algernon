package cookie

// Functions for generating random strings

import (
	"math/rand"
)

// Generate a random string of the given length.
func RandomString(length int) string {
	b := make([]byte, length)
	for i := 0; i < length; i++ {
		b[i] = byte(rand.Int63() & 0xff)
	}
	return string(b)
}

// Generate a random, but human-friendly, string of the given length.
// Should be possible to read out loud and send in an email without problems.
func RandomHumanFriendlyString(length int) string {
	const (
		vowels     = "aeiouy" // email+browsers didn't like "æøå" too much
		consonants = "bcdfghjklmnpqrstvwxz"
	)
	b := make([]byte, length)
	for i := 0; i < length; i++ {
		if i%2 == 0 {
			b[i] = vowels[rand.Intn(len(vowels))]
		} else {
			b[i] = consonants[rand.Intn(len(consonants))]
		}
	}
	return string(b)
}

// Generate a random, but cookie-friendly, string of the given length.
func RandomCookieFriendlyString(length int) string {
	const allowed = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := 0; i < length; i++ {
		b[i] = allowed[rand.Intn(len(allowed))]
	}
	return string(b)
}
