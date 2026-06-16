package main

import (
	"context"
	"os/exec"
	"reflect"
	"testing"
)

func TestBuildSearchArgs(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]any
		want    []string
		wantErr string
	}{
		{
			name:  "pattern and language",
			input: map[string]any{"pattern": "fmt.Println($A)", "language": "go"},
			want:  []string{"run", "-p", "fmt.Println($A)", "-l", "go", "--json"},
		},
		{
			name:  "with dir",
			input: map[string]any{"pattern": "$A", "language": "go", "dir": "./pkg"},
			want:  []string{"run", "-p", "$A", "-l", "go", "--json", "./pkg"},
		},
		{
			name:  "empty dir is omitted",
			input: map[string]any{"pattern": "$A", "language": "go", "dir": ""},
			want:  []string{"run", "-p", "$A", "-l", "go", "--json"},
		},
		{
			name:    "missing pattern",
			input:   map[string]any{"language": "go"},
			wantErr: "pattern is required",
		},
		{
			name:    "empty pattern",
			input:   map[string]any{"pattern": "", "language": "go"},
			wantErr: "pattern is required",
		},
		{
			name:    "missing language",
			input:   map[string]any{"pattern": "$A"},
			wantErr: "language is required",
		},
		{
			name:    "non-string pattern",
			input:   map[string]any{"pattern": 42, "language": "go"},
			wantErr: "pattern is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildSearchArgs(tt.input)
			assertArgs(t, got, err, tt.want, tt.wantErr)
		})
	}
}

func TestBuildReplaceArgs(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]any
		want    []string
		wantErr string
	}{
		{
			name:  "pattern rewrite language",
			input: map[string]any{"pattern": "a", "rewrite": "b", "language": "go"},
			want:  []string{"run", "-p", "a", "-r", "b", "-l", "go", "--update-all"},
		},
		{
			name:  "empty rewrite is allowed (deletion)",
			input: map[string]any{"pattern": "a", "rewrite": "", "language": "go"},
			want:  []string{"run", "-p", "a", "-r", "", "-l", "go", "--update-all"},
		},
		{
			name:  "with dir",
			input: map[string]any{"pattern": "a", "rewrite": "b", "language": "go", "dir": "src"},
			want:  []string{"run", "-p", "a", "-r", "b", "-l", "go", "--update-all", "src"},
		},
		{
			name:    "missing pattern",
			input:   map[string]any{"rewrite": "b", "language": "go"},
			wantErr: "pattern is required",
		},
		{
			name:    "missing rewrite key",
			input:   map[string]any{"pattern": "a", "language": "go"},
			wantErr: "rewrite is required",
		},
		{
			name:    "non-string rewrite",
			input:   map[string]any{"pattern": "a", "rewrite": 1, "language": "go"},
			wantErr: "rewrite is required",
		},
		{
			name:    "missing language",
			input:   map[string]any{"pattern": "a", "rewrite": "b"},
			wantErr: "language is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildReplaceArgs(tt.input)
			assertArgs(t, got, err, tt.want, tt.wantErr)
		})
	}
}

func assertArgs(t *testing.T, got []string, err error, want []string, wantErr string) {
	t.Helper()
	if wantErr != "" {
		if err == nil {
			t.Fatalf("expected error %q, got nil (args=%v)", wantErr, got)
		}
		if err.Error() != wantErr {
			t.Fatalf("expected error %q, got %q", wantErr, err.Error())
		}
		return
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args mismatch:\n got: %v\nwant: %v", got, want)
	}
}

// TestRunSgCommandNotFound verifies that a missing ast-grep binary surfaces an
// error instead of being silently treated as "no matches".
func TestRunSgCommandNotFound(t *testing.T) {
	if _, err := exec.LookPath(astGrepBinary); err == nil {
		t.Skipf("%s is installed; skipping not-found check", astGrepBinary)
	}
	if _, err := runSgCommand(context.Background(), "run", "-p", "$A", "-l", "go"); err == nil {
		t.Fatalf("expected error when %s is not on PATH, got nil", astGrepBinary)
	}
}
