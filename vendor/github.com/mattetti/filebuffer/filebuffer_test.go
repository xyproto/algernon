package filebuffer

import (
	"fmt"
	"io"
	"os"
	"sync"
	"testing"
)

func TestReader(t *testing.T) {
	r := New([]byte("0123456789"))
	tests := []struct {
		off     int64
		seek    int
		n       int
		want    string
		wantpos int64
		seekerr string
	}{
		{seek: os.SEEK_SET, off: 0, n: 20, want: "0123456789"},
		{seek: os.SEEK_SET, off: 1, n: 1, want: "1"},
		{seek: os.SEEK_CUR, off: 1, wantpos: 3, n: 2, want: "34"},
		{seek: os.SEEK_SET, off: -1, seekerr: "filebuffer.Seek: negative position"},
		{seek: os.SEEK_SET, off: 1 << 33, wantpos: 1 << 33},
		{seek: os.SEEK_CUR, off: 1, wantpos: 1<<33 + 1},
		{seek: os.SEEK_SET, n: 5, want: "01234"},
		{seek: os.SEEK_CUR, n: 5, want: "56789"},
		{seek: os.SEEK_END, off: -1, n: 1, wantpos: 9, want: "9"},
	}

	for i, tt := range tests {
		pos, err := r.Seek(tt.off, tt.seek)
		if err == nil && tt.seekerr != "" {
			t.Errorf("%d. want seek error %q", i, tt.seekerr)
			continue
		}
		if err != nil && err.Error() != tt.seekerr {
			t.Errorf("%d. seek error = %q; want %q", i, err.Error(), tt.seekerr)
			continue
		}
		if tt.wantpos != 0 && tt.wantpos != pos {
			t.Errorf("%d. pos = %d, want %d", i, pos, tt.wantpos)
		}
		buf := make([]byte, tt.n)
		n, err := r.Read(buf)
		if err != nil {
			t.Errorf("%d. read = %v", i, err)
			continue
		}
		got := string(buf[:n])
		if got != tt.want {
			t.Errorf("%d. got %q; want %q", i, got, tt.want)
		}
	}
}

func TestReadAfterBigSeek(t *testing.T) {
	r := New([]byte("0123456789"))
	if _, err := r.Seek(1<<31+5, os.SEEK_SET); err != nil {
		t.Fatal(err)
	}
	if n, err := r.Read(make([]byte, 10)); n != 0 || err != io.EOF {
		t.Errorf("Read = %d, %v; want 0, EOF", n, err)
	}
}

func TestReaderAt(t *testing.T) {
	r := New([]byte("0123456789"))
	tests := []struct {
		off     int64
		n       int
		want    string
		wanterr interface{}
	}{
		{0, 10, "0123456789", nil},
		{1, 10, "123456789", io.EOF},
		{1, 9, "123456789", nil},
		{11, 10, "", io.EOF},
		{0, 0, "", nil},
		{-1, 0, "", "filebuffer.ReadAt: negative offset"},
	}
	for i, tt := range tests {
		b := make([]byte, tt.n)
		rn, err := r.ReadAt(b, tt.off)
		got := string(b[:rn])
		if got != tt.want {
			t.Errorf("%d. got %q; want %q", i, got, tt.want)
		}
		if fmt.Sprintf("%v", err) != fmt.Sprintf("%v", tt.wanterr) {
			t.Errorf("%d. got error = %v; want %v", i, err, tt.wanterr)
		}
	}
}

func TestReaderAtConcurrent(t *testing.T) {
	// Test for the race detector, to verify ReadAt doesn't mutate
	// any state.
	r := New([]byte("0123456789"))
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			var buf [1]byte
			r.ReadAt(buf[:], int64(i))
		}(i)
	}
	wg.Wait()
}

func TestEmptyReaderConcurrent(t *testing.T) {
	// Test for the race detector, to verify a Read that doesn't yield any bytes
	// is okay to use from multiple goroutines.
	r := New([]byte(""))
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			var buf [1]byte
			r.Read(buf[:])
		}()
		go func() {
			defer wg.Done()
			r.Read(nil)
		}()
	}
	wg.Wait()
}

func TestClose(t *testing.T) {
	b := New(nil)
	err := b.Close()
	if err != nil {
		t.Fatalf("expected closing to not return an error but returned %v", err)
	}
	n, err := b.Write([]byte{0x42, 0x42, 0x42})
	if n != 0 {
		t.Fatalf("expected 0 bytes to be written but %d were reported written", n)
	}
}

func TestWriter(t *testing.T) {
	b := New(nil)
	n, err := b.Write([]byte("this is a test"))
	if n != 14 {
		t.Fatalf("expected 14 characters written, reported %d", n)
	}
	if err != nil {
		t.Fatal(err)
	}

	testString := `this is a test`

	dst := make([]byte, 14)
	_, err = b.Read(dst)
	if err != io.EOF {
		t.Fatalf("expected an EOF error but got %v", err)
	}
	b.Seek(0, 0)
	_, err = b.Read(dst)
	if string(dst) != testString {
		t.Fatalf("expected `%s` but got `%s`", testString, string(dst))
	}

	// testing Bytes()
	b.Seek(0, 0)
	content := string(b.Bytes())
	if content != testString {
		t.Fatalf("expected a rewinded buffer calling Bytes to output `%s` but got `%s`", testString, content)
	}

	// testing overwriting content
	b.Seek(0, 0)
	altTestString := `maybe, this is the real test`
	b.Write([]byte(altTestString))
	b.Seek(0, 0)
	_, err = b.Read(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(dst) != altTestString[:14] {
		t.Fatalf("expected overwriting the buffer content to read as `%s` but got `%s`", altTestString[:14], string(dst))
	}
	// reset the index to the end of the buffer
	b.Seek(0, 2)

	// testing appending
	_, err = b.Write([]byte(` or maybe it's not`))
	if err != nil {
		t.Fatal(err)
	}
	b.Seek(0, 0)
	content = string(b.Bytes())
	if content != altTestString+` or maybe it's not` {
		t.Fatalf("unexpected appended buffer, content: `%s`", content)
	}

}
