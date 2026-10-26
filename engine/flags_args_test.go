package engine

import "testing"

func TestClassifyPositionalArgsMissingServerFile(t *testing.T) {
	for _, arg := range []string{"missing.lua", "missing.md", "missing.zip"} {
		got := classifyPositionalArgs([]string{arg}, func(string) bool { return false })
		if !got.ServerDirSet || got.ServerDirOrFilename != arg {
			t.Errorf("%s: expected it to be the server file, got %+v", arg, got)
		}
		if len(got.Remaining) != 0 {
			t.Errorf("%s: should not be treated as a certificate, got %v", arg, got.Remaining)
		}
	}
}

func TestClassifyPositionalArgsMissingCertStillRemaining(t *testing.T) {
	got := classifyPositionalArgs([]string{"site", "cert.pem", "key.pem"}, func(p string) bool { return p == "site" })
	if got.ServerDirOrFilename != "site" || len(got.Remaining) != 2 {
		t.Errorf("unexpected classification: %+v", got)
	}
}
