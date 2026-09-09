package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestParseWorktreePorcelain(t *testing.T) {
	input := `worktree /path/to/main
HEAD 8c56d782989b0d3ee78b7e289bf4e321ad8383cf
branch refs/heads/master

worktree /path/to/feature
HEAD 1234567890abcdef1234567890abcdef12345678
branch refs/heads/feature-x
locked reasons why locked

worktree /path/to/detached
HEAD abcdef1234567890abcdef1234567890abcdef12
detached
prunable gitdir file points to non-existent location
`

	results := parseWorktreePorcelain(input)
	if len(results) != 3 {
		t.Fatalf("expected 3 worktrees, got %d", len(results))
	}

	// 1st: main
	if !results[0].IsMain {
		t.Errorf("expected 1st worktree to be main")
	}
	if results[0].Path != "/path/to/main" {
		t.Errorf("expected path /path/to/main, got %s", results[0].Path)
	}
	if results[0].Branch != "master" {
		t.Errorf("expected branch master, got %s", results[0].Branch)
	}
	if results[0].HEAD != "8c56d782989b0d3ee78b7e289bf4e321ad8383cf" {
		t.Errorf("unexpected HEAD: %s", results[0].HEAD)
	}

	// 2nd: feature
	if results[1].IsMain {
		t.Errorf("expected 2nd worktree to not be main")
	}
	if results[1].Branch != "feature-x" {
		t.Errorf("expected branch feature-x, got %s", results[1].Branch)
	}
	if results[1].Locked != "reasons why locked" {
		t.Errorf("expected locked reason, got %s", results[1].Locked)
	}

	// 3rd: detached & prunable
	if !results[2].Detached {
		t.Errorf("expected 3rd worktree to be detached")
	}
	if results[2].Prunable != "gitdir file points to non-existent location" {
		t.Errorf("expected prunable reason, got %s", results[2].Prunable)
	}
}

func TestPathsEqual(t *testing.T) {
	tempDir := t.TempDir()
	p1 := filepath.Join(tempDir, "foo")
	p2 := filepath.Join(tempDir, "foo", "..", "foo")

	if !pathsEqual(p1, p2) {
		t.Errorf("expected %s and %s to be equal", p1, p2)
	}

	p3 := filepath.Join(tempDir, "bar")
	if pathsEqual(p1, p3) {
		t.Errorf("expected %s and %s to NOT be equal", p1, p3)
	}

	// Dot and trailing slash variations
	if !pathsEqual(tempDir, filepath.Join(tempDir, ".")) {
		t.Errorf("expected %s and %s to be equal", tempDir, filepath.Join(tempDir, "."))
	}
	if !pathsEqual(tempDir, tempDir+string(filepath.Separator)) {
		t.Errorf("expected %s and %s to be equal", tempDir, tempDir+string(filepath.Separator))
	}
}

func TestGitWorktreeOperations(t *testing.T) {
	tempDir := t.TempDir()
	ctx := context.Background()

	// Initialize git repo in tempDir
	if _, err := runGitCommand(ctx, tempDir, "init"); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	// Configure user name & email for commit
	runGitCommand(ctx, tempDir, "config", "user.name", "TestUser")
	runGitCommand(ctx, tempDir, "config", "user.email", "test@example.com")

	// Create initial commit
	dummyFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(dummyFile, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	if _, err := runGitCommand(ctx, tempDir, "add", "test.txt"); err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	if _, err := runGitCommand(ctx, tempDir, "commit", "-m", "initial commit"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// 1. List worktree (should be 1 main worktree)
	out, err := runGitCommand(ctx, tempDir, "worktree", "list", "--porcelain")
	if err != nil {
		t.Fatalf("worktree list failed: %v", err)
	}
	trees := parseWorktreePorcelain(out)
	if len(trees) != 1 || !trees[0].IsMain {
		t.Fatalf("expected 1 main worktree, got %v", trees)
	}

	// 2. Add a new worktree
	wtPath := filepath.Join(tempDir, "wt-feature")
	if _, err := runGitCommand(ctx, tempDir, "worktree", "add", "-b", "feature-branch", wtPath); err != nil {
		t.Fatalf("worktree add failed: %v", err)
	}

	// List again
	out, err = runGitCommand(ctx, tempDir, "worktree", "list", "--porcelain")
	if err != nil {
		t.Fatalf("worktree list after add failed: %v", err)
	}
	trees = parseWorktreePorcelain(out)
	if len(trees) != 2 {
		t.Fatalf("expected 2 worktrees, got %d", len(trees))
	}

	// 3. Verify main worktree removal is rejected
	mainPath, err := getMainWorktreePath(ctx, tempDir)
	if err != nil {
		t.Fatalf("failed to get main worktree path: %v", err)
	}
	if !pathsEqual(mainPath, tempDir) {
		t.Errorf("expected mainPath %s to equal tempDir %s", mainPath, tempDir)
	}

	// 4. Remove linked worktree
	if _, err := runGitCommand(ctx, tempDir, "worktree", "remove", wtPath); err != nil {
		t.Fatalf("worktree remove failed: %v", err)
	}

	// List again
	out, err = runGitCommand(ctx, tempDir, "worktree", "list", "--porcelain")
	if err != nil {
		t.Fatalf("worktree list after remove failed: %v", err)
	}
	trees = parseWorktreePorcelain(out)
	if len(trees) != 1 {
		t.Fatalf("expected 1 worktree after removal, got %d", len(trees))
	}

	// 5. Prune
	pruneOut, err := runGitCommand(ctx, tempDir, "worktree", "prune", "-v")
	if err != nil {
		t.Fatalf("worktree prune failed: %v", err)
	}
	t.Logf("prune output: %s", pruneOut)
}
