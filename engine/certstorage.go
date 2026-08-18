package engine

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/xyproto/env/v2"
)

const (
	// xdgDataHomeEnvVar is where user-specific data files are stored
	xdgDataHomeEnvVar = "XDG_DATA_HOME"

	// legacyCertStorageEnvVar is the non-standard variable that earlier
	// versions of Algernon read
	legacyCertStorageEnvVar = "XDG_CONFIG_DIR"

	// certStorageName is the directory that CertMagic stores certificates in
	certStorageName = "certmagic"

	// systemCertStorageDir is used when there is no home directory,
	// like when running as a system service
	systemCertStorageDir = "/var/lib/algernon"
)

// certStorageDir returns the directory where Let's Encrypt certificates are kept.
// It must survive a reboot: fetching new certificates on every boot counts
// against the Let's Encrypt rate limits, and private keys do not belong in a
// world-writable directory like /tmp.
//
// In order of precedence: a directory that already holds certificates,
// $XDG_DATA_HOME, the home directory and then a system directory.
func certStorageDir() string {
	// Keep using a location that already holds certificates, so that
	// upgrading Algernon does not orphan them
	if dir := existingCertStorageDir(); dir != "" {
		return dir
	}
	if dir := env.Str(xdgDataHomeEnvVar); dir != "" {
		return filepath.Join(dir, certStorageName)
	}
	if home := env.Str("HOME"); home != "" {
		return filepath.Join(home, ".local", "share", certStorageName)
	}
	// A system service often has no home directory
	if runtime.GOOS == "windows" {
		if dir, err := os.UserConfigDir(); err == nil { // success
			return filepath.Join(dir, certStorageName)
		}
	}
	return systemCertStorageDir
}

// existingCertStorageDir returns the first location that already holds
// certificates, or an empty string. Earlier versions of Algernon pointed
// CertMagic at the legacy directory, or at the home directory itself.
func existingCertStorageDir() string {
	var candidates []string
	if dir := env.Str(xdgDataHomeEnvVar); dir != "" {
		candidates = append(candidates, filepath.Join(dir, certStorageName))
	}
	if home := env.Str("HOME"); home != "" {
		candidates = append(candidates, filepath.Join(home, ".local", "share", certStorageName), home)
	}
	if dir := env.Str(legacyCertStorageEnvVar); dir != "" {
		candidates = append(candidates, dir)
	}
	candidates = append(candidates, systemCertStorageDir)
	for _, dir := range candidates {
		if hasCertificates(dir) {
			return dir
		}
	}
	return ""
}

// hasCertificates checks if CertMagic has stored certificates in the given directory
func hasCertificates(dir string) bool {
	if dir == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(dir, "certificates"))
	return err == nil && info.IsDir()
}

// volatileCertStorage checks if the given directory is one that is likely to be
// cleared at boot, which would mean new certificates for every boot
func volatileCertStorage(dir string) bool {
	tempDir := filepath.Clean(env.Str("TMPDIR", os.TempDir()))
	for _, volatile := range []string{tempDir, "/tmp", "/var/tmp", "/run", "/dev/shm"} {
		if dir == volatile || strings.HasPrefix(dir, volatile+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}
