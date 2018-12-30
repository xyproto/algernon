// Package randomstring can be used for generating different types of random strings
package randomstring

import (
	"math/rand"
	"time"
)

// Seed the random number generator in one of many possible ways.
func Seed() {
	rand.Seed(time.Now().UTC().UnixNano() + 1337)
}

// String generates a random string of a given length.
func String(length int) string {
	b := make([]byte, length)
	for i := 0; i < length; i++ {
		b[i] = byte(rand.Int63() & 0xff)
	}
	return string(b)
}

/*HumanFriendlyString generates a random, but human-friendly, string of
 * the given length. It should be possible to read out loud and send in an email
 * without problems. The string alternates between vowels and consontants.
 *
 * Google Translate believes the output is Samoan.
 *
 * Example output for length 7: rabunor
 */
func HumanFriendlyString(length int) string {
	const (
		someVowels     = "aeoiu"          // a selection of vowels. email+browsers didn't like "æøå" too much
		someConsonants = "bdfgklmnoprstv" // a selection of consonants
		moreLetters    = "chjqwxyz"       // the rest of the letters from a-z
	)
	vowelOffset := rand.Intn(2)
	vowelDistribution := 2
	b := make([]byte, length)
	for i := 0; i < length; i++ {
	again:
		if (i+vowelOffset)%vowelDistribution == 0 {
			b[i] = someVowels[rand.Intn(len(someVowels))]
		} else if rand.Intn(100) > 0 { // 99 of 100 times
			b[i] = someConsonants[rand.Intn(len(someConsonants))]
			// Don't repeat
			if i >= 1 && b[i] == b[i-1] {
				// Also use more vowels
				vowelDistribution = 1
				// Then try again
				goto again
			}
		} else {
			b[i] = moreLetters[rand.Intn(len(moreLetters))]
			// Don't repeat
			if i >= 1 && b[i] == b[i-1] {
				// Also use more vowels
				vowelDistribution = 1
				// Then try again
				goto again
			}
		}
		// Avoid three letters in a row
		if i >= 2 && b[i] == b[i-2] {
			// Then try again
			goto again
		}
	}
	return string(b)
}

// CookieFriendlyString generates a random, but cookie-friendly, string of
// the given length.
func CookieFriendlyString(length int) string {
	const allowed = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := 0; i < length; i++ {
		b[i] = allowed[rand.Intn(len(allowed))]
	}
	return string(b)
}
