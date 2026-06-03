package utils

import (
	"os"
	"testing"
)

func TestURL2filename(t *testing.T) {
	tests := []struct {
		dirname string
		urlpath string
		want    string
	}{
		{"/srv", "/index.html", "/srv" + Pathsep + "index.html"},
		{"/srv/", "/index.html", "/srv/index.html"},
		{"/srv", "/sub/page.md", "/srv" + Pathsep + "sub/page.md"},
		{"/srv", "relative.txt", "/srv/relative.txt"},
		{"/srv", "/../etc/passwd", "/srv" + Pathsep},
	}
	for _, tt := range tests {
		got := URL2filename(tt.dirname, tt.urlpath)
		if got != tt.want {
			t.Errorf("URL2filename(%q, %q) = %q, want %q", tt.dirname, tt.urlpath, got, tt.want)
		}
	}
}

func TestGetFilenames(t *testing.T) {
	dir, err := os.MkdirTemp("", "algernon-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Create some files
	for _, name := range []string{"a.txt", "b.md", "c.html"} {
		if err := os.WriteFile(dir+"/"+name, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	filenames := GetFilenames(dir)
	if len(filenames) != 3 {
		t.Errorf("GetFilenames returned %d files, want 3", len(filenames))
	}

	// Non-existent directory returns empty slice
	filenames = GetFilenames("/nonexistent_dir_xyz")
	if len(filenames) != 0 {
		t.Errorf("GetFilenames on nonexistent dir returned %d files, want 0", len(filenames))
	}
}

func TestDescribeBytes(t *testing.T) {
	tests := []struct {
		size int64
		want string
	}{
		{0, "0 KiB"},
		{1024, "1 KiB"},
		{512, "0 KiB"},
		{1048576, "1 MiB"},
		{2097152, "2 MiB"},
		{10240, "10 KiB"},
	}
	for _, tt := range tests {
		got := DescribeBytes(tt.size)
		if got != tt.want {
			t.Errorf("DescribeBytes(%d) = %q, want %q", tt.size, got, tt.want)
		}
	}
}
