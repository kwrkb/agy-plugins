package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const retainedLogLimit = 1 << 20
const returnedLogLimit = 64 << 10
const eventLimit = 1 << 20
const recordLimit = 100000

type Counts struct {
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}
type Failure struct {
	Test  string  `json:"test,omitempty"`
	Log   string  `json:"log,omitempty"`
	Rerun Options `json:"rerun"`
}
type PackageResult struct {
	Package  string    `json:"package"`
	Status   string    `json:"status"`
	Elapsed  float64   `json:"elapsed_seconds"`
	Counts   Counts    `json:"tests"`
	Failures []Failure `json:"failures,omitempty"`
	Log      string    `json:"log,omitempty"`
}
type BuildFailure struct {
	Package string `json:"package"`
	Log     string `json:"log,omitempty"`
}
type Result struct {
	Status        string          `json:"status"`
	ExitCode      *int            `json:"exit_code"`
	Elapsed       float64         `json:"elapsed_seconds"`
	Counts        Counts          `json:"tests"`
	Packages      []PackageResult `json:"packages,omitempty"`
	BuildFailures []BuildFailure  `json:"build_failures,omitempty"`
	Stderr        string          `json:"stderr,omitempty"`
	Error         string          `json:"error,omitempty"`
	LogsTruncated bool            `json:"logs_truncated"`
	Incomplete    bool            `json:"incomplete"`
}
type event struct {
	Action      string
	Package     string
	Test        string
	Elapsed     float64
	Output      string
	ImportPath  string
	FailedBuild string
}
type testState struct {
	status string
	log    []byte
}
type packageState struct {
	status  string
	elapsed float64
	tests   map[string]*testState
	log     []byte
}
type eventStream struct {
	pending     []byte
	err         error
	packages    map[string]*packageState
	buildLogs   map[string][]byte
	buildFailed map[string]bool
	retained    int
	records     int
	truncated   bool
}

func newEventStream() *eventStream {
	return &eventStream{packages: map[string]*packageState{}, buildLogs: map[string][]byte{}, buildFailed: map[string]bool{}}
}

// Write parses complete lines without retaining the full stdout. On a malformed
// event, continue draining so that a child cannot deadlock on a full pipe.
func (s *eventStream) Write(p []byte) (int, error) {
	n := len(p)
	for len(p) > 0 && s.err == nil {
		i := bytes.IndexByte(p, '\n')
		if i < 0 {
			i = len(p)
		}
		if len(s.pending)+i > eventLimit {
			s.err = fmt.Errorf("JSON event exceeds %d bytes", eventLimit)
			s.pending = nil
			break
		}
		s.pending = append(s.pending, p[:i]...)
		p = p[i:]
		if len(p) == 0 {
			break
		}
		s.consume(s.pending)
		s.pending = s.pending[:0]
		p = p[1:]
	}
	return n, nil
}
func (s *eventStream) finish() {
	if s.err == nil && len(bytes.TrimSpace(s.pending)) > 0 {
		s.consume(s.pending)
	}
	s.pending = nil
}
func (s *eventStream) appendLog(dst []byte, v string) []byte {
	// Reserve room for the independently drained stderr buffer.
	n := min(len(v), retainedLogLimit-returnedLogLimit-s.retained)
	s.retained += n
	s.truncated = s.truncated || n < len(v)
	return append(dst, v[:n]...)
}
func (s *eventStream) addRecord() bool {
	s.records++
	if s.records > recordLimit {
		s.err = fmt.Errorf("more than %d result records", recordLimit)
		return false
	}
	return true
}
func (s *eventStream) markBuildFailure(name string) {
	if !s.buildFailed[name] && s.addRecord() {
		s.buildFailed[name] = true
	}
}
func (s *eventStream) consume(line []byte) {
	if len(bytes.TrimSpace(line)) == 0 {
		return
	}
	var e event
	if err := json.Unmarshal(line, &e); err != nil || e.Action == "" {
		s.err = fmt.Errorf("invalid go test JSON event")
		return
	}
	if e.Action == "build-output" || e.Action == "build-fail" {
		if e.ImportPath == "" {
			s.err = fmt.Errorf("build event missing ImportPath")
			return
		}
		if _, ok := s.buildLogs[e.ImportPath]; !ok {
			if !s.addRecord() {
				return
			}
			s.buildLogs[e.ImportPath] = nil
		}
		s.buildLogs[e.ImportPath] = s.appendLog(s.buildLogs[e.ImportPath], e.Output)
		if e.Action == "build-fail" {
			s.markBuildFailure(e.ImportPath)
		}
		return
	}
	if e.Package == "" {
		s.err = fmt.Errorf("test event missing Package")
		return
	}
	p := s.packages[e.Package]
	if p == nil {
		if !s.addRecord() {
			return
		}
		p = &packageState{status: "incomplete", tests: map[string]*testState{}}
		s.packages[e.Package] = p
	}
	if e.FailedBuild != "" {
		s.markBuildFailure(e.FailedBuild)
	}
	if e.Test == "" {
		if e.Action == "output" {
			p.log = s.appendLog(p.log, e.Output)
		}
		if terminal(e.Action) {
			p.status, p.elapsed = e.Action, e.Elapsed
		}
		return
	}
	t := p.tests[e.Test]
	if t == nil {
		if !s.addRecord() {
			return
		}
		t = &testState{status: "incomplete"}
		p.tests[e.Test] = t
	}
	if e.Action == "output" {
		t.log = s.appendLog(t.log, e.Output)
	}
	if terminal(e.Action) {
		t.status = e.Action
		if e.Action != "fail" {
			s.retained -= len(t.log)
			t.log = nil
		}
	}
}
func terminal(a string) bool { return a == "pass" || a == "fail" || a == "skip" }

// splitRun mirrors testing.splitRegexp: go test cuts a -run expression into
// elements at slashes and alternations that sit outside brackets, groups and
// escapes, then compiles each element on its own. The escape case is load
// bearing: without it an escaped separator such as `a\/b` splits into `a\`,
// which does not compile, so a valid expression would be rejected.
func splitRun(expr string) []string {
	var elements []string
	brackets, groups := 0, 0
	for i := 0; i < len(expr); {
		switch expr[i] {
		case '[':
			brackets++
		case ']':
			if brackets--; brackets < 0 { // An unmatched ']' is legal.
				brackets = 0
			}
		case '(':
			if brackets == 0 {
				groups++
			}
		case ')':
			if brackets == 0 {
				groups--
			}
		case '\\':
			i++
		case '/', '|':
			if brackets == 0 && groups == 0 {
				elements = append(elements, expr[:i])
				expr = expr[i+1:]
				i = 0
				continue
			}
		}
		i++
	}
	return append(elements, expr)
}

// validateRun rejects what the test binary would reject at startup. Left to go
// test, an invalid caller expression fails every package with no failing test,
// which reads as a project test failure rather than an input error.
func validateRun(expr string) error {
	if expr == "" {
		return nil
	}
	for i, element := range splitRun(expr) {
		// go test verifies each element after rewriting whitespace to '_', so
		// mirror that substitution. Its other rewrite, escaping non-printable
		// runes, cannot change whether an element compiles.
		normalized := strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return '_'
			}
			return r
		}, element)
		if _, err := regexp.Compile(normalized); err != nil {
			return fmt.Errorf("run element %d (%q) is not a valid expression: %w", i, element, err)
		}
	}
	return nil
}
func exactRun(name string) string {
	parts := strings.Split(name, "/")
	for i, p := range parts {
		parts[i] = "^" + regexp.QuoteMeta(p) + "$"
	}
	return strings.Join(parts, "/")
}

// Apply the byte budget after replacing invalid UTF-8, so JSON encoding cannot
// expand malformed stderr past the documented log-text limit.
func boundedLog(b []byte, limit int) (string, bool) {
	v := strings.ToValidUTF8(string(b), "�")
	n := min(len(v), limit)
	for n > 0 && !utf8.ValidString(v[:n]) {
		n--
	}
	return v[:n], n < len(v)
}

func (s *eventStream) result(o Options, stderr *cappedBuffer) Result {
	r := Result{Status: "passed", LogsTruncated: s.truncated || stderr.truncated}
	remaining := returnedLogLimit
	log := func(b []byte) string {
		v, truncated := boundedLog(b, remaining)
		remaining -= len(v)
		r.LogsTruncated = r.LogsTruncated || truncated
		return v
	}
	// Failures have priority over package framing and diagnostic stderr.
	for _, name := range sortedKeys(s.packages) {
		p := s.packages[name]
		pr := PackageResult{Package: name, Status: p.status, Elapsed: p.elapsed}
		for _, test := range sortedKeys(p.tests) {
			t := p.tests[test]
			switch t.status {
			case "pass":
				pr.Counts.Passed++
			case "skip":
				pr.Counts.Skipped++
			case "fail":
				pr.Counts.Failed++
				rerun := o
				rerun.Packages, rerun.Run = []string{name}, exactRun(test)
				pr.Failures = append(pr.Failures, Failure{Test: test, Log: log(t.log), Rerun: rerun})
			default:
				r.Incomplete = true
			}
		}
		if p.status == "incomplete" {
			r.Incomplete = true
		}
		if p.status == "fail" || pr.Counts.Failed > 0 {
			r.Status = "failed"
		}
		r.Counts.Passed += pr.Counts.Passed
		r.Counts.Failed += pr.Counts.Failed
		r.Counts.Skipped += pr.Counts.Skipped
		r.Packages = append(r.Packages, pr)
	}
	for _, name := range sortedKeys(s.buildFailed) {
		r.BuildFailures = append(r.BuildFailures, BuildFailure{Package: name, Log: log(s.buildLogs[name])})
		r.Status = "failed"
	}
	for i := range r.Packages {
		p := &r.Packages[i]
		// A panic or cancellation can leave test-scoped output without a fail
		// event. Keep this evidence without inventing a failed-test count.
		state := s.packages[p.Package]
		for _, name := range sortedKeys(state.tests) {
			t := state.tests[name]
			if t.status == "incomplete" && len(t.log) > 0 {
				p.Log += log([]byte(name+":\n")) + log(t.log)
			}
		}
		if p.Status != "pass" && p.Status != "skip" {
			p.Log += log(state.log)
		}
	}
	r.Stderr = log(stderr.data)
	if r.Status == "passed" && r.Counts == (Counts{}) {
		r.Status = "no_tests"
	}
	if s.err != nil {
		r.Status, r.Error, r.Incomplete = "error", s.err.Error(), true
	}
	return r
}
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
