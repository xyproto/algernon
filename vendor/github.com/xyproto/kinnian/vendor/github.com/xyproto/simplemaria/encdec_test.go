package simplemaria

import (
	"log"
	"math/rand"
	"testing"
	"time"
)

func TestEncodeDecode(t *testing.T) {
	hello := "hello"
	original := hello
	Encode(&hello)
	Decode(&hello)
	if hello != original {
		t.Error("Unable to encode and decode: " + original)
	}
}

func TestEncodeDecodeWithNewline(t *testing.T) {
	newlinedrop := "\n!''' DROP TABLES EVERYWHERE"
	original := newlinedrop
	Encode(&newlinedrop)
	Decode(&newlinedrop)
	if newlinedrop != original {
		t.Error("Unable to encode and decode: " + original)
	}
}

func TestEncodeDecodeWithEOB(t *testing.T) {
	weirdness := "\xbd\xb2\x3d\x17\xbc\x20\xe2\x8c\x98"
	original := weirdness
	Encode(&weirdness)
	Decode(&weirdness)
	if weirdness != original {
		t.Error("Unable to encode and decode: " + original)
	}
}

// Generate a random string of the given length.
func randomString(length int) string {
	b := make([]byte, length)
	for i := 0; i < length; i++ {
		b[i] = byte(rand.Int63() & 0xff)
	}
	return string(b)
}

func TestRandom(t *testing.T) {
	// Generate 10 random strings and check if they encode and decode correctly
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 10; i++ {
		log.Printf("Random string %d\n", i)
		s1 := randomString(100)
		s2 := s1
		Encode(&s2)
		Decode(&s2)
		if s1 != s2 {
			t.Error(s1, "is different from", s2)
		}
	}
}
