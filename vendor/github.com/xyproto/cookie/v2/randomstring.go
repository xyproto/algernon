package cookie

// This file exists only for backwards compatibility reasons

import (
	"github.com/xyproto/randomstring"
)

// Seed the random number generator. One of many possible ways.
func Seed() {
	randomstring.Seed()
}

// RandomString generates a random string of a given length.
func RandomString(length int) string {
	return randomstring.String(length)
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
	return randomstring.HumanFriendlyString(length)
}

// RandomCookieFriendlyString generates a random, but cookie-friendly, string of
// the given length.
func RandomCookieFriendlyString(length int) string {
	return randomstring.CookieFriendlyString(length)
}
