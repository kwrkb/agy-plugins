package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// syncBuffer wraps bytes.Buffer with a lock so it's safe as a concurrent
// io.Writer target. It does NOT serialize a writeFrame's header+body pair
// as a unit — that's writeFrame's own job, and what these tests verify.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (w *syncBuffer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}

func (w *syncBuffer) Close() error { return nil }

// readFrames parses the Content-Length-delimited LSP frames out of r, the
// same wire format LSPClient.readLoop consumes.
func readFrames(t *testing.T, r io.Reader) [][]byte {
	t.Helper()
	reader := bufio.NewReader(r)
	var frames [][]byte
	for {
		contentLength := 0
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return frames
			}
			line = strings.TrimSpace(line)
			if line == "" {
				break
			}
			if rest, ok := strings.CutPrefix(line, "Content-Length:"); ok {
				contentLength, _ = strconv.Atoi(strings.TrimSpace(rest))
			}
		}
		body := make([]byte, contentLength)
		if _, err := io.ReadFull(reader, body); err != nil {
			t.Fatalf("truncated frame (corrupted by interleaved writes?): %v", err)
		}
		frames = append(frames, body)
	}
}

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

// TestWriteFrameConcurrentDoesNotInterleave guards against regressing the
// missing write lock: concurrent Call/Notify used to write their
// Content-Length header and body as two separate stdin writes, so two
// goroutines could interleave and corrupt the LSP stream.
func TestWriteFrameConcurrentDoesNotInterleave(t *testing.T) {
	w := &syncBuffer{}
	c := &LSPClient{stdin: w}

	const n = 100
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			body, err := json.Marshal(map[string]any{"n": i})
			if err != nil {
				t.Error(err)
				return
			}
			if err := c.writeFrame(body); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()

	frames := readFrames(t, bytes.NewReader(w.buf.Bytes()))
	if len(frames) != n {
		t.Fatalf("got %d frames, want %d (interleaved writes corrupt framing)", len(frames), n)
	}

	seen := make(map[int]bool)
	for _, f := range frames {
		var m struct {
			N int `json:"n"`
		}
		if err := json.Unmarshal(f, &m); err != nil {
			t.Fatalf("frame %q is not valid JSON (interleaved writes corrupt framing): %v", f, err)
		}
		seen[m.N] = true
	}
	if len(seen) != n {
		t.Fatalf("got %d distinct payloads, want %d", len(seen), n)
	}
}

// TestRespondToServerRequest guards against regressing unanswered
// server-initiated requests: gopls can block waiting for a reply to
// requests like workspace/configuration if the client never responds.
func TestRespondToServerRequest(t *testing.T) {
	w := &syncBuffer{}
	c := &LSPClient{stdin: w}

	params, err := json.Marshal(map[string]any{
		"items": []any{map[string]any{"section": "gopls"}, map[string]any{"section": "go"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	c.respondToServerRequest(7, "workspace/configuration", params)

	frames := readFrames(t, bytes.NewReader(w.buf.Bytes()))
	if len(frames) != 1 {
		t.Fatalf("got %d frames, want 1", len(frames))
	}

	var resp struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int64  `json:"id"`
		Result  []any  `json:"result"`
	}
	if err := json.Unmarshal(frames[0], &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if resp.ID != 7 {
		t.Fatalf("response id = %d, want 7 (must echo the request id)", resp.ID)
	}
	if len(resp.Result) != 2 {
		t.Fatalf("workspace/configuration result has %d entries, want 2 (one per requested item)", len(resp.Result))
	}
}

// TestReadLoopClosesPendingChannelsOnExit guards against Call() hanging
// forever when gopls exits or the pipe breaks: readLoop must close every
// outstanding pending channel (and clear the map) once its read loop ends.
func TestReadLoopClosesPendingChannelsOnExit(t *testing.T) {
	c := &LSPClient{
		stdout:  io.NopCloser(strings.NewReader("")),
		pending: make(map[int64]chan []byte),
	}
	ch := make(chan []byte, 1)
	c.pending[1] = ch

	c.readLoop()

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("pending channel received a value, want it closed with no value")
		}
	default:
		t.Fatal("pending channel not closed after readLoop returned")
	}
	if len(c.pending) != 0 {
		t.Fatalf("pending map has %d entries after readLoop returned, want 0", len(c.pending))
	}
}

// TestCallReturnsErrorWhenConnectionCloses guards against Call() unmarshaling
// a nil response (and hanging until the context deadline) when the
// connection closes while a call is in flight: it must observe the closed
// channel and return an error immediately instead.
func TestCallReturnsErrorWhenConnectionCloses(t *testing.T) {
	pr, pw := io.Pipe()
	c := &LSPClient{
		stdin:   &syncBuffer{},
		stdout:  io.NopCloser(pr),
		pending: make(map[int64]chan []byte),
	}
	go c.readLoop()

	result := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := c.Call(ctx, "textDocument/definition", nil)
		result <- err
	}()

	// Wait until Call has registered its pending entry before closing the
	// pipe, so the test deterministically exercises the close path instead
	// of racing it against the ctx timeout.
	deadline := time.Now().Add(2 * time.Second)
	for {
		c.mu.Lock()
		n := len(c.pending)
		c.mu.Unlock()
		if n > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("Call did not register a pending entry in time")
		}
		time.Sleep(time.Millisecond)
	}
	pw.Close()

	select {
	case err := <-result:
		if err == nil || !strings.Contains(err.Error(), "LSP connection closed") {
			t.Fatalf("Call() error = %v, want an \"LSP connection closed\" error", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Call did not return after the gopls connection closed")
	}
}

// findPosition returns the 0-indexed LSP line/character of needle's first
// occurrence in src.
func findPosition(t *testing.T, src, needle string) (line, character int) {
	t.Helper()
	idx := strings.Index(src, needle)
	if idx < 0 {
		t.Fatalf("needle %q not found in source", needle)
	}
	line = strings.Count(src[:idx], "\n")
	character = idx - strings.LastIndex(src[:idx], "\n") - 1
	return line, character
}

// TestLiveGoplsDefinition starts a real gopls process and asks for the
// definition of a symbol. Skipped where gopls isn't installed. This is the
// only test that actually drives the initialize -> workspace/configuration
// -> textDocument/definition handshake with a live LSP server, which is
// what the respondToServerRequest fix (unanswered server requests could
// block gopls indefinitely) needs to prove.
func TestLiveGoplsDefinition(t *testing.T) {
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not found in PATH")
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/livetest\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := "package main\n\nfunc greet() string {\n\treturn \"hi\"\n}\n\nfunc main() {\n\t_ = greet()\n}\n"
	mainFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(mainFile, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	line, character := findPosition(t, src, "greet()")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := NewLSPClient(dir)
	if err != nil {
		t.Fatalf("NewLSPClient: %v", err)
	}
	defer client.Close()

	if err := client.Initialize(ctx); err != nil {
		t.Fatalf("Initialize: %v (would hang instead of erroring if gopls is blocked waiting on an unanswered server request)", err)
	}

	params := TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: pathToURI(mainFile)},
		Position:     Position{Line: line, Character: character},
	}
	resBytes, err := client.Call(ctx, "textDocument/definition", params)
	if err != nil {
		t.Fatalf("textDocument/definition: %v", err)
	}

	var rpcRes struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(resBytes, &rpcRes); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if rpcRes.Error != nil {
		t.Fatalf("gopls returned error: %s", rpcRes.Error.Message)
	}
	if len(rpcRes.Result) == 0 || string(rpcRes.Result) == "null" {
		t.Fatal("expected a definition location for greet(), got none")
	}
}
