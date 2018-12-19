package cookie

// Functions for generating random strings

import (
	"math/rand"
	"time"
)

// Seed the random number generator. One of many possible ways.
func Seed() {
	rand.Seed(time.Now().UTC().UnixNano())
}

// RandomString generates a random string of a given length.
func RandomString(length int) string {
	b := make([]byte, length)
	for i := 0; i < length; i++ {
		b[i] = byte(rand.Int63() & 0xff)
	}
	return string(b)
}

/*RandomHumanFriendlyString generates a random, but human-friendly, string of
 * the given length. It should be possible to read out loud and send in an email
 * without problems. The string alternates between vowels and consontants.
 *
 * Google Translate believes the output is Samoan.
 *
 * Example output for length 10: ookeouvapu
 */
func RandomHumanFriendlyString(length int) string {
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
			if i >= 1 {
				if b[i] == b[i-1] {
					// Also use more vowels
					vowelDistribution = 1
					// Then try again
					goto again
				}
			}
		} else {
			b[i] = moreLetters[rand.Intn(len(moreLetters))]
			// Don't repeat
			if i >= 1 {
				if b[i] == b[i-1] {
					// Also use more vowels
					vowelDistribution = 1
					// Then try again
					goto again
				}
			}
		}
		// Avoid three letters in a row
		if i >= 2 {
			if b[i] == b[i-2] {
				goto again
			}
		}
	}
	return string(b)
}

// RandomCookieFriendlyString generates a random, but cookie-friendly, string of
// the given length.
func RandomCookieFriendlyString(length int) string {
	const allowed = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := 0; i < length; i++ {
		b[i] = allowed[rand.Intn(len(allowed))]
	}
	return string(b)
}
