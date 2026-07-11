package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestEscapeDriveLetter(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"C:/Users/foo/bar.go", "/C:/Users/foo/bar.go"},
		{"/home/foo/bar.go", "/home/foo/bar.go"}, // POSIX paths are untouched
	}
	for _, c := range cases {
		if got := escapeDriveLetter(c.in); got != c.want {
			t.Errorf("escapeDriveLetter(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestPathToURIWindowsDriveLetter(t *testing.T) {
	// url.URL with Path "C:/Users/..." (no leading slash) renders "C:" as a
	// host, producing an invalid file URI. Verify the escaped form avoids it.
	got := "file://" + escapeDriveLetter("C:/Users/foo/bar.go")

	want := "file:///C:/Users/foo/bar.go"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if strings.HasPrefix(got, "file://C:") {
		t.Fatalf("URI %q would parse %q as a host, not a path", got, "C:")
	}
}

func TestPathToURIRoundTrip(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "sub", "main.go")

	uri := pathToURI(file)
	if !strings.HasPrefix(uri, "file://") {
		t.Fatalf("pathToURI(%q) = %q, want file:// scheme", file, uri)
	}

	back := uriToPath(uri)
	abs, err := filepath.Abs(file)
	if err != nil {
		t.Fatal(err)
	}
	if back != abs {
		t.Fatalf("uriToPath(pathToURI(%q)) = %q, want %q", file, back, abs)
	}
}

func TestUriToPathNonFileScheme(t *testing.T) {
	u := "https://example.com/foo"
	if got := uriToPath(u); got != u {
		t.Fatalf("uriToPath(%q) = %q, want unchanged", u, got)
	}
}

func TestRequiredString(t *testing.T) {
	m := map[string]any{"file_path": "/tmp/a.go", "empty": ""}

	if v, err := requiredString(m, "file_path"); err != nil || v != "/tmp/a.go" {
		t.Fatalf("requiredString(file_path) = %q, %v", v, err)
	}
	if _, err := requiredString(m, "empty"); err == nil {
		t.Fatal("requiredString(empty) should error on empty string")
	}
	if _, err := requiredString(m, "missing"); err == nil {
		t.Fatal("requiredString(missing) should error on missing key")
	}
}

func TestRequiredInt(t *testing.T) {
	// JSON-decoded arguments always surface numbers as float64.
	m := map[string]any{"line": float64(42)}

	v, err := requiredInt(m, "line")
	if err != nil || v != 42 {
		t.Fatalf("requiredInt(line) = %d, %v", v, err)
	}
	if _, err := requiredInt(m, "missing"); err == nil {
		t.Fatal("requiredInt(missing) should error")
	}
}

func TestOptionalString(t *testing.T) {
	m := map[string]any{"dir": "/tmp"}
	if got := optionalString(m, "dir"); got != "/tmp" {
		t.Fatalf("optionalString(dir) = %q", got)
	}
	if got := optionalString(m, "missing"); got != "" {
		t.Fatalf("optionalString(missing) = %q, want empty", got)
	}
}
