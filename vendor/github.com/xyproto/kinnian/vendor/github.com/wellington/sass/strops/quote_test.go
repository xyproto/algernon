package strops

import "testing"

type s struct {
	a string
	e string
}

func TestQuote_unescape(t *testing.T) {
	tests := []struct {
		s string
		e string
	}{
		{`\9797`, "\u9797"},
		{`\n\9797`, "n\u9797"},
		{`foo`, "foo"},
		{`\201C`, "\u201C"},
		{`\201D`, "\u201D"},
		{`\2018`, "\u2018"},
	}

	for _, tst := range tests {
		if s := unescape(tst.s); s != tst.e {
			t.Errorf("got: %#v wanted: %#v", s, tst.e)
		}
	}
}
