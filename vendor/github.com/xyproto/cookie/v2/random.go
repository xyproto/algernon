package cookie

// This file exists only for backwards compatibility reasons

import (
	"github.com/xyproto/randomstring"
)

// Seed the random number generator. One of many possible ways. This is a function.
var Seed = randomstring.Seed

// RandomString generates a random string of a given length. This is a function.
var RandomString = randomstring.String

/*RandomHumanFriendlyString generates a random, but human-friendly, string of
 * the given length. It should be possible to read out loud and send in an email
 * without problems. The string alternates between vowels and consontants.
 *
 * Google Translate believes the output is Samoan.
 *
 * Example output for length 10: ookeouvapu
 *
 * This is a function.
 */
var RandomHumanFriendlyString = randomstring.HumanFriendlyString

// RandomCookieFriendlyString generates a random, but cookie-friendly, string of
// the given length. This is a function.
var RandomCookieFriendlyString = randomstring.CookieFriendlyString

// RandomCookieFriendlyBytes is like RandomCookieFriendlyString, but returns a byte slice.
var RandomCookieFriendlyBytes = randomstring.CookieFriendlyBytes
