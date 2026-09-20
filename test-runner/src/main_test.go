package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestOptions(t *testing.T) {
	for _, args := range []map[string]any{
		{}, {"module_path": 1}, {"module_path": " "},
		{"module_path": ".", "packages": []any{".", 2}},
		{"module_path": ".", "packages": []string{}},
		{"module_path": ".", "packages": []string{"-exec=evil"}},
		{"module_path": ".", "packages": nil},
		{"module_path": ".", "run": false},
		{"module_path": ".", "run": "["},
		{"module_path": ".", "run": "A/("},
		{"module_path": ".", "timeout_seconds": "60"},
		{"module_path": ".", "timeout_seconds": 0},
		{"module_path": ".", "timeout_seconds": 301},
		{"module_path": ".", "timeout_seconds": 1.5},
		{"module_path": ".", "timeout_seconds": nil},
		{"module_path": ".", "args": []string{"-race"}},
	} {
		if _, err := parseOptions(args); err == nil {
			t.Errorf("accepted invalid arguments: %#v", args)
		}
	}
	for _, timeout := range []int{1, 300} {
		o, err := parseOptions(map[string]any{"module_path": "日本語 space", "timeout_seconds": timeout})
		if err != nil || o.Timeout != timeout || o.Packages[0] != "./..." {
			t.Fatalf("%+v %v", o, err)
		}
	}
	// A directory or import path may end in ".go"; only discovery can tell such a
	// package from a file list, so the pattern itself is accepted here.
	o, err := parseOptions(map[string]any{"module_path": ".", "packages": []string{"./pkg.go", "example.com/m/gen.go"}})
	if err != nil || len(o.Packages) != 2 {
		t.Fatalf("%+v %v", o, err)
	}
	// Expressions go test accepts, verified against a real "go test -run" run.
	for _, run := range []string{"TestA", "(a/b)", "a[/]b", "A|B", "^A$/^B$", `^TestParent$/^a\+b\[1\]$`, "A//B", "A|", "(?P<a b>x)", "a b", `a\ b`, `a\/b`, `a\|b`, "", "\u088f", "A/\u088f",
		// testing.rewrite escapes non-printable runes before compiling, so a
		// backslash before one becomes an escaped backslash rather than an
		// invalid escape, and "\a" stays the bell class it already was.
		"\\\u200b", "\u0007", "a\u0007b", "\u00a0", "A/\\\u200b"} {
		if o, err := parseOptions(map[string]any{"module_path": ".", "run": run}); err != nil || o.Run != run {
			t.Fatalf("rejected valid run %q: %v", run, err)
		}
	}
	// Expressions the test binary rejects at startup, which would otherwise fail
	// every package and read as a project test failure.
	for _, run := range []string{"[", "(a", "A/[", "a|(", "A/b)", `a\`, "((A)/B", "[a-", "A/*",
		// The same rewrite turns a bare non-printable rune into "\u200b", which
		// the regexp parser rejects; go test then fails every package instead.
		"\u200b", "A/\u200b", "A|\u200b", "(\u200b)"} {
		if _, err := parseOptions(map[string]any{"module_path": ".", "run": run}); err == nil {
			t.Errorf("accepted invalid run %q", run)
		}
	}
}

// A failing package whose import path ends in ".go" must produce a rerun the
// tool can accept again; the pattern check used to reject its own output.
// The rewrite go test applies to every -run element decides whether the
// element compiles, so rewriteRun must reproduce it rune for rune. Each
// expectation below was read off a real "go test -run" run: feeding it "[" plus
// the rune makes the element invalid, and the resulting error echoes the
// rewritten form back. Note the space set is testing's hand-written switch, not
// the Unicode Z class -- U+200A rewrites to "_" while U+200B, one past the end
// of that range, becomes the escape "\u200b" the regexp parser rejects.
// validateRun compiles one regexp per top-level element before runTests builds
// the timeout context, so the expression is bounded by size rather than left to
// spend that time outside the timeout and outside elapsed_seconds.
func TestRunSizeIsBounded(t *testing.T) {
	if _, err := parseOptions(map[string]any{"module_path": ".", "run": strings.Repeat("A", runLimit)}); err != nil {
		t.Fatalf("rejected run at the limit: %v", err)
	}
	// Worst case for validation cost: every byte pair is its own element.
	dense := strings.Repeat("A|", runLimit/2)
	start := time.Now()
	o, err := parseOptions(map[string]any{"module_path": ".", "run": dense})
	if err != nil || o.Run != dense {
		t.Fatalf("rejected dense run at the limit: %v", err)
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("validating a run at the limit took %s", d)
	}
	for _, run := range []string{strings.Repeat("A", runLimit+1), strings.Repeat("A|", runLimit)} {
		if _, err := parseOptions(map[string]any{"module_path": ".", "run": run}); err == nil {
			t.Errorf("accepted run of %d bytes", len(run))
		}
	}
}

func TestRewriteRunMatchesGoTest(t *testing.T) {
	for in, want := range map[string]string{
		"\t": "_", " ": "_", "\u0085": "_", "\u00a0": "_", "\u1680": "_",
		"\u2000": "_", "\u2005": "_", "\u200a": "_", "\u2028": "_",
		"\u202f": "_", "\u205f": "_", "\u3000": "_",
		"\u200b": `\u200b`, "\a": `\a`, "\ufeff": `\ufeff`, "\ue000": `\ue000`,
		// Unassigned under the pinned toolchain and printable under a newer one,
		// so it must survive raw: "go test -run" accepts it on go1.27.1 and this
		// server would otherwise reject a rerun it built itself.
		"\u088f":    "\u088f",
		"TestA":     "TestA",
		"a b":       "a_b",
		"a\\\u200b": `a\\u200b`,
	} {
		if got := rewriteRun(in); got != want {
			t.Errorf("rewriteRun(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRerunRoundTripsThroughOptions(t *testing.T) {
	s := newEventStream()
	emit(s,
		event{Action: "run", Package: "example.com/m/gen.go", Test: "TestParent/a+b[1]"},
		event{Action: "fail", Package: "example.com/m/gen.go", Test: "TestParent/a+b[1]"},
		event{Action: "fail", Package: "example.com/m/gen.go"},
	)
	s.finish()
	r := s.result(Options{ModulePath: ".", Packages: []string{"./..."}, Timeout: 60}, &cappedBuffer{})
	if r.Status != "failed" || len(r.Packages) != 1 || len(r.Packages[0].Failures) != 1 {
		t.Fatalf("%+v", r)
	}
	rerun := r.Packages[0].Failures[0].Rerun
	b, err := json.Marshal(rerun)
	if err != nil {
		t.Fatal(err)
	}
	var args map[string]any
	if err := json.Unmarshal(b, &args); err != nil {
		t.Fatal(err)
	}
	o, err := parseOptions(args)
	if err != nil {
		t.Fatalf("rerun rejected: %v (%+v)", err, rerun)
	}
	if o.Packages[0] != "example.com/m/gen.go" || o.Run != rerun.Run {
		t.Fatalf("%+v", o)
	}
	// go test spells non-printable runes in test names with their escape
	// sequence, so exactRun must quote the backslash for the rerun to survive
	// validateRun, which mirrors the same rewrite.
	for _, name := range []string{`TestZero/a\u200bb`, "TestSpace/a b", "TestParent/a+b[1]"} {
		if err := validateRun(exactRun(name)); err != nil {
			t.Fatalf("rerun for %q rejected: %v", name, err)
		}
	}
}

func emit(s *eventStream, es ...event) {
	for _, e := range es {
		b, _ := json.Marshal(e)
		_, _ = s.Write(append(b, '\n'))
	}
}
func TestInterleavedEventsAndRerun(t *testing.T) {
	s := newEventStream()
	emit(s,
		event{Action: "start", Package: "example.com/a"},
		event{Action: "run", Package: "example.com/a", Test: "TestParent"},
		event{Action: "run", Package: "example.com/a", Test: "TestParent/a+b[1]"},
		event{Action: "pause", Package: "example.com/a", Test: "TestParent/a+b[1]"},
		event{Action: "start", Package: "example.com/b"},
		event{Action: "skip", Package: "example.com/b"},
		event{Action: "cont", Package: "example.com/a", Test: "TestParent/a+b[1]"},
		event{Action: "output", Package: "example.com/a", Test: "TestParent/a+b[1]", Output: "failure reason\n"},
		event{Action: "fail", Package: "example.com/a", Test: "TestParent/a+b[1]"},
		event{Action: "fail", Package: "example.com/a", Test: "TestParent"},
		event{Action: "skip", Package: "example.com/a", Test: "TestSkip"},
		event{Action: "fail", Package: "example.com/a", Elapsed: 0.1},
	)
	s.finish()
	r := s.result(Options{ModulePath: "/module", Timeout: 60}, &cappedBuffer{})
	if r.Status != "failed" || r.Incomplete || r.Counts != (Counts{Failed: 2, Skipped: 1}) {
		t.Fatalf("%+v", r)
	}
	f := r.Packages[0].Failures[1]
	if f.Log != "failure reason\n" || f.Rerun.Run != `^TestParent$/^a\+b\[1\]$` || f.Rerun.Packages[0] != "example.com/a" {
		t.Fatalf("%+v", f)
	}
}

func TestBuildEventsAndMalformedOutput(t *testing.T) {
	s := newEventStream()
	emit(s, event{Action: "build-output", ImportPath: "example.com/a [example.com/a.test]", Output: "compile error"},
		event{Action: "build-fail", ImportPath: "example.com/a [example.com/a.test]"},
		event{Action: "fail", Package: "example.com/a", FailedBuild: "example.com/a [example.com/a.test]"})
	r := s.result(Options{}, &cappedBuffer{})
	if r.Status != "failed" || len(r.BuildFailures) != 1 || r.BuildFailures[0].Log != "compile error" || r.Counts.Failed != 0 {
		t.Fatalf("%+v", r)
	}
	for _, bad := range []string{"not json\n", "{}\n", `{"Action":"pass"}`, `{"Action":"start","Package":`, strings.Repeat("x", eventLimit+1)} {
		s = newEventStream()
		_, _ = s.Write([]byte(bad))
		s.finish()
		if s.result(Options{}, &cappedBuffer{}).Status != "error" {
			t.Fatalf("accepted malformed output %.50q", bad)
		}
	}
	// Complete JSON without a trailing newline is valid, including chunk splits.
	s = newEventStream()
	for _, chunk := range []string{`{"Action":"pa`, `ss","Package":"p"}`} {
		_, _ = s.Write([]byte(chunk))
	}
	s.finish()
	if s.result(Options{}, &cappedBuffer{}).Status != "no_tests" {
		t.Fatal(s.err)
	}
}

func TestInterruptedTestKeepsDiagnostic(t *testing.T) {
	s := newEventStream()
	emit(s, event{Action: "run", Package: "p", Test: "TestInterrupted"},
		event{Action: "output", Package: "p", Test: "TestInterrupted", Output: "last diagnostic before interruption\n"})
	s.finish()
	r := s.result(Options{}, &cappedBuffer{})
	if !r.Incomplete || r.Counts != (Counts{}) || !strings.Contains(r.Packages[0].Log, "last diagnostic") {
		t.Fatalf("lost partial diagnostic: %+v", r)
	}
}

func TestLogsAreBoundedWithoutLosingResults(t *testing.T) {
	s := newEventStream()
	for i := 0; i < 40; i++ {
		emit(s, event{Action: "output", Package: "p", Test: "TestFail", Output: strings.Repeat("x", 65536)})
	}
	emit(s, event{Action: "fail", Package: "p", Test: "TestFail"}, event{Action: "pass", Package: "p", Test: "TestPass"}, event{Action: "fail", Package: "p"})
	r := s.result(Options{}, &cappedBuffer{})
	if !r.LogsTruncated || r.Counts != (Counts{Passed: 1, Failed: 1}) || len(r.Packages[0].Failures[0].Log) != returnedLogLimit || s.retained > retainedLogLimit {
		t.Fatalf("%+v retained=%d", r.Counts, s.retained)
	}
	// >64 KiB JSON lines are accepted; do not regress to Scanner's default limit.
	s = newEventStream()
	emit(s, event{Action: "output", Package: "p", Output: strings.Repeat("x", 128<<10)}, event{Action: "pass", Package: "p"})
	if s.err != nil {
		t.Fatal(s.err)
	}
}

func TestLogBudgetAfterUTF8Replacement(t *testing.T) {
	text, truncated := boundedLog([]byte(strings.Repeat("\xffx", returnedLogLimit)), returnedLogLimit)
	if len(text) > returnedLogLimit || !utf8.ValidString(text) || !truncated {
		t.Fatal("invalid UTF-8 exceeded log budget")
	}
	text, truncated = boundedLog([]byte("あい"), 4)
	if text != "あ" || !truncated {
		t.Fatalf("split multibyte character: %q", text)
	}
}

func writeFile(t *testing.T, name, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
}
func moduleFixture(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "日本語 module space")
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/fixture\n\ngo 1.24.0\n")
	writeFile(t, filepath.Join(dir, "fixture_test.go"), `package fixture
import "testing"
func TestPass(t *testing.T) { t.Parallel() }
func TestSkip(t *testing.T) { t.Skip("skip") }
func TestFail(t *testing.T) {
 t.Run("a+b[1]", func(t *testing.T) { t.Fatal("expected failure") })
 t.Run("unrelated", func(t *testing.T) { t.Log("must not run in child retry") })
}
func TestPanic(t *testing.T) { panic("panic marker") }
`)
	return dir
}
func offlineGo(t *testing.T) {
	t.Helper()
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOSUMDB", "off")
	t.Setenv("GOTOOLCHAIN", "local")
	t.Setenv("GOWORK", "off")
	t.Setenv("GOFLAGS", "")
}
func TestRealGoResultsAndRetry(t *testing.T) {
	offlineGo(t)
	dir := moduleFixture(t)
	o := Options{ModulePath: dir, Packages: []string{"./..."}, Timeout: 60}
	for _, tt := range []struct {
		run, status             string
		passed, failed, skipped int
	}{
		{"^TestPass$", "passed", 1, 0, 0},
		{"^TestSkip$", "passed", 0, 0, 1},
		{"^DoesNotExist$", "no_tests", 0, 0, 0},
		{"^TestFail$", "failed", 1, 2, 0},
		{"^TestPanic$", "failed", 0, 0, 0},
	} {
		t.Run(tt.run, func(t *testing.T) {
			o.Run = tt.run
			r := runTests(t.Context(), o)
			if r.Status != tt.status {
				t.Fatalf("%+v", r)
			}
			if tt.run == "^TestPanic$" {
				b, _ := json.Marshal(r)
				if !strings.Contains(string(b), "panic marker") {
					t.Fatalf("lost panic diagnostic: %s", b)
				}
			}
			if tt.run != "^TestPanic$" && r.Counts != (Counts{tt.passed, tt.failed, tt.skipped}) {
				t.Fatalf("%+v", r)
			}
			if tt.run == "^TestFail$" {
				f := r.Packages[0].Failures[1]
				rr := runTests(t.Context(), f.Rerun)
				if rr.Status != "failed" || rr.Counts.Passed != 0 || rr.Counts.Failed != 2 || !strings.Contains(rr.Packages[0].Failures[1].Log, "expected failure") {
					t.Fatalf("retry: %+v", rr)
				}
			}
		})
	}
	writeFile(t, filepath.Join(dir, "broken.go"), "package fixture\nvar x = undefinedSymbol\n")
	r := runTests(t.Context(), o)
	if r.Status != "failed" || len(r.BuildFailures) == 0 || r.Counts.Failed != 0 {
		t.Fatalf("build failure: %+v", r)
	}
}

func TestModuleBoundariesAndEmptyModule(t *testing.T) {
	offlineGo(t)
	dir := moduleFixture(t)
	o := Options{ModulePath: dir, Packages: []string{"fmt"}, Timeout: 60}
	if r := runTests(t.Context(), o); r.Status != "error" {
		t.Fatalf("external package allowed: %+v", r)
	}
	o.Packages = []string{"./does-not-exist"}
	if r := runTests(t.Context(), o); r.Status != "error" {
		t.Fatalf("missing package: %+v", r)
	}
	empty := t.TempDir()
	writeFile(t, filepath.Join(empty, "go.mod"), "module example.com/empty\n\ngo 1.24.0\n")
	o.ModulePath, o.Packages = empty, []string{"./..."}
	if r := runTests(t.Context(), o); r.Status != "no_tests" {
		t.Fatalf("empty module: %+v", r)
	}
	writeFile(t, filepath.Join(empty, "empty.go"), "package empty\n")
	if r := runTests(t.Context(), o); r.Status != "no_tests" || len(r.Packages) != 1 {
		t.Fatalf("package without tests: %+v", r)
	}
	writeFile(t, filepath.Join(empty, "empty_test.go"), "package empty\nimport (\"os\";\"testing\")\nfunc TestMain(m *testing.M) { os.Exit(2) }\n")
	if r := runTests(t.Context(), o); r.Status != "failed" || r.Counts != (Counts{}) {
		t.Fatalf("TestMain failure: %+v", r)
	}
	workspace := filepath.Join(t.TempDir(), "go.work")
	writeFile(t, workspace, fmt.Sprintf("go 1.24.0\nuse (\n%q\n%q\n)\n", dir, empty))
	t.Setenv("GOWORK", workspace)
	o.ModulePath, o.Packages = dir, []string{"example.com/empty"}
	if r := runTests(t.Context(), o); r.Status != "error" || !strings.Contains(r.Error, "outside") {
		t.Fatalf("other workspace module allowed: %+v", r)
	}
	t.Setenv("GOWORK", "off")
	o.ModulePath = t.TempDir()
	if r := runTests(t.Context(), o); r.Status != "error" {
		t.Fatal(r)
	}
	o.ModulePath = dir
	t.Setenv("PATH", t.TempDir())
	if r := runTests(t.Context(), o); r.Status != "error" || r.ExitCode != nil {
		t.Fatalf("missing go: %+v", r)
	}
}

func TestPackageValidation(t *testing.T) {
	root, _ := canonical(t.TempDir())
	outside, _ := canonical(t.TempDir())
	for _, p := range []listedPackage{
		{Dir: outside, ImportPath: "p", Module: &struct{ Dir string }{root}},
		{Dir: root, ImportPath: "p", Module: &struct{ Dir string }{outside}},
		{Dir: root, ImportPath: "-bad", Module: &struct{ Dir string }{root}},
		// go list reports a .go file list under this synthesized import path and
		// without module framing.
		{Dir: root, ImportPath: "command-line-arguments"},
	} {
		b, _ := json.Marshal(p)
		if _, err := validatePackages(b, root); err == nil {
			t.Fatalf("accepted %+v", p)
		}
	}
	// A module may legitimately declare "command-line-arguments" as its path; go
	// list then reports it with a module, and "go test" on it succeeds.
	b, _ := json.Marshal(listedPackage{Dir: root, ImportPath: "command-line-arguments", Module: &struct{ Dir string }{root}})
	if got, err := validatePackages(b, root); err != nil || len(got) != 1 || got[0] != "command-line-arguments" {
		t.Fatalf("rejected module named command-line-arguments: %v %v", got, err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(outside, link); err == nil {
		p := listedPackage{Dir: link, ImportPath: "p", Module: &struct{ Dir string }{root}}
		b, _ := json.Marshal(p)
		if _, err := validatePackages(b, root); err == nil {
			t.Fatal("symlink escaped module")
		}
	}
}

// The helper's child appends a heartbeat, allowing cancellation tests to verify
// descendants are gone on every supported OS without platform-specific probes.
func TestProcessHelper(t *testing.T) {
	mode := os.Getenv("TEST_RUNNER_HELPER")
	if mode == "" {
		return
	}
	if mode == "stderr" {
		fmt.Fprint(os.Stderr, "setup failed")
		os.Exit(2)
	}
	if mode == "child" {
		for {
			f, err := os.OpenFile(os.Getenv("TEST_RUNNER_HEARTBEAT"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				os.Exit(3)
			}
			_, _ = f.WriteString("x")
			_ = f.Close()
			time.Sleep(20 * time.Millisecond)
		}
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestProcessHelper$")
	cmd.Env = append(os.Environ(), "TEST_RUNNER_HELPER=child")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		os.Exit(4)
	}
	os.Exit(0)
}
func TestCancellationStopsDescendants(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		t.Run(fmt.Sprint(deadline), func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			if deadline {
				ctx, cancel = context.WithTimeout(t.Context(), 2*time.Second)
			}
			defer cancel()
			path := filepath.Join(t.TempDir(), "heartbeat")
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestProcessHelper$")
			cmd.Env = append(os.Environ(), "TEST_RUNNER_HELPER=parent", "TEST_RUNNER_HEARTBEAT="+path)
			done := make(chan error, 1)
			go func() { _, err := execute(cmd, io.Discard, io.Discard); done <- err }()
			until := time.Now().Add(5 * time.Second)
			for {
				if st, err := os.Stat(path); err == nil && st.Size() > 0 {
					break
				}
				if time.Now().After(until) {
					t.Fatal("no child heartbeat")
				}
				time.Sleep(20 * time.Millisecond)
			}
			if !deadline {
				cancel()
			}
			select {
			case err := <-done:
				if err == nil {
					t.Fatal("expected cancellation")
				}
			case <-time.After(8 * time.Second):
				t.Fatal("process did not stop")
			}
			before, _ := os.ReadFile(path)
			time.Sleep(150 * time.Millisecond)
			after, _ := os.ReadFile(path)
			if len(before) != len(after) {
				t.Fatal("child survived cancellation")
			}
		})
	}
}
func TestStderrAndCancelledContext(t *testing.T) {
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestProcessHelper$")
	cmd.Env = append(os.Environ(), "TEST_RUNNER_HELPER=stderr")
	errout := &cappedBuffer{limit: 100}
	code, err := execute(cmd, io.Discard, errout)
	if err == nil || code == nil || *code != 2 || string(errout.data) != "setup failed" {
		t.Fatalf("%v %v %s", code, err, errout.data)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	r := runTests(ctx, Options{ModulePath: t.TempDir(), Timeout: 60})
	if r.Status != "cancelled" || !r.Incomplete {
		t.Fatal(r)
	}
}

func TestMCPServerProcess(t *testing.T) {
	if os.Getenv("TEST_RUNNER_MCP") != "1" {
		return
	}
	main()
	os.Exit(0)
}
func TestMCPStdio(t *testing.T) {
	offlineGo(t)
	dir := moduleFixture(t)
	binary, args := os.Args[0], []string{"-test.run=^TestMCPServerProcess$"}
	if distributed := os.Getenv("TEST_RUNNER_BINARY"); distributed != "" {
		binary, args = distributed, nil
	}
	// Installation smoke tests may supply the copied dispatcher/native binary.
	// All variants start outside both the plugin and the module under test.
	t.Chdir(t.TempDir())
	c, err := client.NewStdioMCPClient(binary, append(os.Environ(), "TEST_RUNNER_MCP=1"), args...)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	init := mcp.InitializeRequest{}
	init.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	init.Params.ClientInfo = mcp.Implementation{Name: "test", Version: "1"}
	if _, err = c.Initialize(ctx, init); err != nil {
		t.Fatal(err)
	}
	list, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil || len(list.Tools) != 1 || list.Tools[0].Name != "go_test" {
		t.Fatalf("%+v %v", list, err)
	}
	for _, tt := range []struct {
		args    map[string]any
		status  string
		isError bool
	}{
		{map[string]any{"module_path": dir, "run": "^TestPass$"}, "passed", false},
		{map[string]any{"module_path": dir, "run": "^TestFail$"}, "failed", false},
		{map[string]any{"module_path": dir, "timeout_seconds": "60"}, "error", true},
	} {
		req := mcp.CallToolRequest{}
		req.Params.Name = "go_test"
		req.Params.Arguments = tt.args
		res, err := c.CallTool(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		var r Result
		if err = json.Unmarshal([]byte(res.Content[0].(mcp.TextContent).Text), &r); err != nil {
			t.Fatal(err)
		}
		if r.Status != tt.status || res.IsError != tt.isError {
			t.Fatalf("%+v isError=%v", r, res.IsError)
		}
	}
	marker := filepath.Join(t.TempDir(), "test-started")
	writeFile(t, filepath.Join(dir, "slow_test.go"), fmt.Sprintf(`package fixture
import ("os"; "testing"; "time")
func TestSlow(t *testing.T) { if err := os.WriteFile(%q, []byte("started"), 0600); err != nil { t.Fatal(err) }; time.Sleep(time.Minute) }
`, marker))
	// Use an explicit request ID so the test can send the real MCP cancellation
	// notification while the tool is running, not just cancel its local waiter.
	done := make(chan *transport.JSONRPCResponse, 1)
	errors := make(chan error, 1)
	go func() {
		res, err := c.GetTransport().SendRequest(ctx, transport.JSONRPCRequest{
			JSONRPC: "2.0", ID: mcp.NewRequestId(int64(777)), Method: "tools/call",
			Params: map[string]any{"name": "go_test", "arguments": map[string]any{"module_path": dir, "run": "^TestSlow$"}},
		})
		if err != nil {
			errors <- err
			return
		}
		done <- res
	}()
	until := time.Now().Add(30 * time.Second)
	for {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		if time.Now().After(until) {
			t.Fatal("MCP test did not start")
		}
		time.Sleep(20 * time.Millisecond)
	}
	err = c.GetTransport().SendNotification(ctx, mcp.JSONRPCNotification{
		JSONRPC: "2.0", Notification: mcp.Notification{Method: "notifications/cancelled",
			Params: mcp.NotificationParams{AdditionalFields: map[string]any{"requestId": 777}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case raw := <-done:
		res, err := mcp.ParseCallToolResult(&raw.Result)
		if err != nil {
			t.Fatal(err)
		}
		var r Result
		if err := json.Unmarshal([]byte(res.Content[0].(mcp.TextContent).Text), &r); err != nil {
			t.Fatal(err)
		}
		if r.Status != "cancelled" || !r.Incomplete || !res.IsError {
			t.Fatalf("cancellation: %+v", r)
		}
	case err := <-errors:
		t.Fatal(err)
	case <-time.After(10 * time.Second):
		t.Fatal("MCP cancellation did not reach the running test")
	}
	r := runTests(ctx, Options{ModulePath: dir, Packages: []string{"."}, Run: "^TestSlow$", Timeout: 1})
	if r.Status != "timeout" || !r.Incomplete {
		t.Fatalf("timeout: %+v", r)
	}
}
