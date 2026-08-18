package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/caddyserver/certmagic"
	"github.com/xyproto/env/v2"
)

// setCertEnv sets the environment variables that certStorageDir looks at and
// refreshes the cache in the env package
func setCertEnv(t *testing.T, kv map[string]string) {
	t.Helper()
	for _, name := range []string{xdgDataHomeEnvVar, legacyCertStorageEnvVar, "HOME"} {
		t.Setenv(name, kv[name])
	}
	env.Load()
	t.Cleanup(env.Load)
}

// storeCertificates makes a directory look like CertMagic has used it
func storeCertificates(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "certificates"), 0o700); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestCertStorageDir(t *testing.T) {
	home := t.TempDir()
	xdg := t.TempDir()

	tests := []struct {
		env  map[string]string
		name string
		want string
	}{
		{
			name: "XDG_DATA_HOME is used, in a subdirectory",
			env:  map[string]string{xdgDataHomeEnvVar: xdg, "HOME": home},
			want: filepath.Join(xdg, certStorageName),
		},
		{
			name: "the home directory is not used directly",
			env:  map[string]string{"HOME": home},
			want: filepath.Join(home, ".local", "share", certStorageName),
		},
		{
			name: "a system service without a home directory",
			env:  map[string]string{},
			want: systemCertStorageDir,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setCertEnv(t, tc.env)
			if got := certStorageDir(); got != tc.want {
				t.Errorf("certStorageDir() = %q, want %q", got, tc.want)
			}
		})
	}
}

// Upgrading must not orphan certificates that earlier versions placed directly
// in the home directory, since re-requesting them counts against the rate limits.
func TestCertStorageDirKeepsLegacyLocation(t *testing.T) {
	home := storeCertificates(t, t.TempDir())
	setCertEnv(t, map[string]string{"HOME": home})

	if got := certStorageDir(); got != home {
		t.Errorf("certStorageDir() = %q, want the legacy location %q", got, home)
	}
}

// The same goes for the non-standard XDG_CONFIG_DIR that earlier versions read.
func TestCertStorageDirKeepsLegacyConfigDir(t *testing.T) {
	legacy := storeCertificates(t, t.TempDir())
	setCertEnv(t, map[string]string{legacyCertStorageEnvVar: legacy, "HOME": t.TempDir()})

	if got := certStorageDir(); got != legacy {
		t.Errorf("certStorageDir() = %q, want the legacy location %q", got, legacy)
	}
}

// A home directory that has certificates in the current location must win over
// the same home directory used as a legacy location.
func TestCertStorageDirPrefersCurrentOverLegacy(t *testing.T) {
	home := t.TempDir()
	current := storeCertificates(t, filepath.Join(home, ".local", "share", certStorageName))
	storeCertificates(t, home)
	setCertEnv(t, map[string]string{"HOME": home})

	if got := certStorageDir(); got != current {
		t.Errorf("certStorageDir() = %q, want %q", got, current)
	}
}

// configureCertMagic must actually point CertMagic at the chosen directory.
func TestConfigureCertMagicUsesCertStorageDir(t *testing.T) {
	dir := t.TempDir()
	setCertEnv(t, map[string]string{xdgDataHomeEnvVar: dir})
	want := filepath.Join(dir, certStorageName)

	storage := certmagic.Default.Storage
	t.Cleanup(func() { certmagic.Default.Storage = storage })

	ac := &Config{}
	ac.configureCertMagic()

	fileStorage, ok := certmagic.Default.Storage.(*certmagic.FileStorage)
	if !ok {
		t.Fatalf("Storage is %T, want *certmagic.FileStorage", certmagic.Default.Storage)
	}
	if fileStorage.Path != want {
		t.Errorf("Storage path = %q, want %q", fileStorage.Path, want)
	}
}

func TestVolatileCertStorage(t *testing.T) {
	tests := []struct {
		dir  string
		want bool
	}{
		{"/tmp", true},
		{"/tmp/algernon", true},
		{"/var/tmp/certs", true},
		{"/run/algernon", true},
		{"/dev/shm/certs", true},
		{"/var/lib/algernon", false},
		{"/home/user/.local/share/certmagic", false},
		{"/tmpfoo", false}, // not below /tmp
	}
	for _, tc := range tests {
		if got := volatileCertStorage(tc.dir); got != tc.want {
			t.Errorf("volatileCertStorage(%q) = %v, want %v", tc.dir, got, tc.want)
		}
	}
}
