package webauthncose

import "sync/atomic"

var allowBERIntegers atomic.Bool

// SetExperimentalInsecureAllowBERIntegers allows credentials which have BER integer encoding for their signatures
// which do not conform to the specification. This is an experimental option that may be removed without any notice
// and could potentially lead to zero-day exploits due to the ambiguity of encoding practices. This is not a recommended
// option.
func SetExperimentalInsecureAllowBERIntegers(value bool) {
	allowBERIntegers.Store(value)
}
