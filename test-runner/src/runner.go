package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type listedPackage struct {
	Dir        string
	ImportPath string
	Module     *struct{ Dir string }
	Error      *struct{ Err string }
}

func samePath(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
func canonical(path string) (string, error) {
	p, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(p)
}
func validatePackages(data []byte, root string) ([]string, error) {
	d := json.NewDecoder(bytes.NewReader(data))
	var packages []string
	seen := map[string]bool{}
	for {
		var p listedPackage
		if err := d.Decode(&p); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("invalid go list output: %w", err)
		}
		if p.Module == nil || p.Dir == "" {
			if p.Error != nil {
				return nil, fmt.Errorf("package discovery: %s", p.Error.Err)
			}
			// go list synthesizes this import path for a .go file list, which has no
			// module framing and cannot be reproduced by a rerun. A module that
			// declares this path is reported with a module and stays valid.
			if p.ImportPath == "command-line-arguments" {
				return nil, errors.New(".go file lists are not supported; pass package patterns")
			}
			return nil, fmt.Errorf("package %q is not in the selected module", p.ImportPath)
		}
		moduleDir, err := canonical(p.Module.Dir)
		if err != nil || !samePath(moduleDir, root) {
			return nil, fmt.Errorf("package %q is outside the selected module", p.ImportPath)
		}
		dir, err := canonical(p.Dir)
		if err != nil {
			return nil, err
		}
		rel, err := filepath.Rel(root, dir)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
			return nil, fmt.Errorf("package %q resolves outside the module directory", p.ImportPath)
		}
		if p.ImportPath == "" || strings.HasPrefix(p.ImportPath, "-") || strings.ContainsAny(p.ImportPath, "\x00\r\n") {
			return nil, fmt.Errorf("invalid resolved package name %q", p.ImportPath)
		}
		if !seen[p.ImportPath] {
			packages = append(packages, p.ImportPath)
			seen[p.ImportPath] = true
		}
	}
	return packages, nil
}

func runTests(parent context.Context, o Options) (r Result) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(parent, time.Duration(o.Timeout)*time.Second)
	defer cancel()
	defer func() {
		r.Elapsed = time.Since(start).Seconds()
		if ctx.Err() != nil {
			r.Status, r.Incomplete = "cancelled", true
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				r.Status = "timeout"
			}
			if r.Error == "" {
				r.Error = ctx.Err().Error()
			}
		}
	}()
	r.Status = "error"
	root, err := canonical(o.ModulePath)
	if err != nil {
		r.Error = err.Error()
		return
	}
	if st, err := os.Stat(filepath.Join(root, "go.mod")); err != nil || !st.Mode().IsRegular() {
		r.Error = "module_path must contain a regular go.mod file"
		return
	}
	o.ModulePath = root
	goBinary, err := exec.LookPath("go")
	if err != nil {
		r.Error = err.Error()
		return
	}
	// An absolute executable remains valid after cmd.Dir changes.
	goBinary, err = filepath.Abs(goBinary)
	if err != nil {
		r.Error = err.Error()
		return
	}
	args := append([]string{"list", "-e", "-json=Dir,ImportPath,Module,Error", "--"}, o.Packages...)
	cmd := exec.CommandContext(ctx, goBinary, args...)
	cmd.Dir = root
	listed := &cappedBuffer{limit: 8 << 20}
	stderr := &cappedBuffer{limit: returnedLogLimit}
	code, err := execute(cmd, listed, stderr)
	var truncated bool
	r.Stderr, truncated = boundedLog(stderr.data, returnedLogLimit)
	r.LogsTruncated = stderr.truncated || truncated
	if err != nil || listed.truncated {
		r.ExitCode = code
		r.Error = fmt.Sprintf("package discovery failed: %v", err)
		if listed.truncated {
			r.Error = "package discovery exceeds 8 MiB"
		}
		return
	}
	packages, err := validatePackages(listed.data, root)
	if err != nil {
		r.Error = err.Error()
		return
	}
	if len(packages) == 0 {
		r.Status, r.ExitCode = "no_tests", code
		return
	}
	args = []string{"test", "-json", "-count=1", fmt.Sprintf("-timeout=%ds", o.Timeout)}
	if o.Run != "" {
		args = append(args, "-run="+o.Run)
	}
	args = append(args, packages...)
	cmd = exec.CommandContext(ctx, goBinary, args...)
	cmd.Dir = root
	stream := newEventStream()
	code, err = execute(cmd, stream, stderr)
	stream.finish()
	r = stream.result(o, stderr)
	r.ExitCode = code
	if err != nil {
		var exit *exec.ExitError
		// Test/build failures are normal tool results. Infrastructure errors and
		// unexplained nonzero exits are tool errors, even with partial events.
		if !errors.As(err, &exit) || r.Status != "failed" {
			r.Status = "error"
		}
		if r.Status == "error" || ctx.Err() != nil {
			if r.Error == "" {
				r.Error = err.Error()
			} else {
				r.Error += "; " + err.Error()
			}
		}
	}
	if r.Incomplete && r.Status != "failed" && r.Error == "" {
		r.Status, r.Error = "error", "test output ended before terminal events"
	}
	if len(stream.packages) == 0 && r.Status == "no_tests" {
		r.Status, r.Error, r.Incomplete = "error", "go test returned no package events", true
	}
	return
}
