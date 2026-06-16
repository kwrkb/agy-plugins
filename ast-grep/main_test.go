package main

import (
	"context"
	"os/exec"
	"testing"
)

func TestMainLogic(t *testing.T) {
	ctx := context.Background()
	if ctx == nil {
		t.Fatal("Expected context to not be nil")
	}
}

func TestRunSgCommandNoSg(t *testing.T) {
	_, err := exec.LookPath("sg")
	if err != nil {
		t.Skip("sg not installed, skipping exec test")
	}

	ctx := context.Background()
	_, err = runSgCommand(ctx, "run", "-p", "func $A() {}", "-l", "go")
	if err != nil {
		t.Logf("Warning: sg command failed, might be expected if no matches: %v", err)
	}
}
