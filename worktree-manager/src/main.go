package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// WorktreeInfo represents a single git worktree entry.
type WorktreeInfo struct {
	Path     string `json:"path"`
	HEAD     string `json:"head"`
	Branch   string `json:"branch,omitempty"`
	Bare     bool   `json:"bare,omitempty"`
	Detached bool   `json:"detached,omitempty"`
	Locked   string `json:"locked,omitempty"`
	Prunable string `json:"prunable,omitempty"`
	IsMain   bool   `json:"is_main"`
}

func runGitCommand(ctx context.Context, repoPath string, args ...string) (string, error) {
	cmdArgs := make([]string, 0, len(args)+2)
	if repoPath != "" && repoPath != "." {
		cmdArgs = append(cmdArgs, "-C", repoPath)
	}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.CommandContext(ctx, "git", cmdArgs...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errStr := strings.TrimSpace(stderr.String())
		if errStr != "" {
			return "", fmt.Errorf("%s", errStr)
		}
		return "", err
	}

	outStr := strings.TrimSpace(stdout.String())
	errStr := strings.TrimSpace(stderr.String())
	if outStr != "" && errStr != "" {
		return outStr + "\n" + errStr, nil
	} else if outStr != "" {
		return outStr, nil
	}
	return errStr, nil
}

// parseWorktreePorcelain parses the output of `git worktree list --porcelain` (including -z).
func parseWorktreePorcelain(output string) []WorktreeInfo {
	var lines []string
	if strings.Contains(output, "\x00") {
		lines = strings.Split(output, "\x00")
	} else {
		lines = strings.Split(output, "\n")
	}
	var results []WorktreeInfo
	var current *WorktreeInfo
	isFirst := true

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if current != nil {
				results = append(results, *current)
				current = nil
			}
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			if current != nil {
				results = append(results, *current)
			}
			path := strings.TrimPrefix(line, "worktree ")
			current = &WorktreeInfo{
				Path:   path,
				IsMain: isFirst,
			}
			isFirst = false
			continue
		}

		if current == nil {
			continue
		}

		switch {
		case strings.HasPrefix(line, "HEAD "):
			current.HEAD = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			ref := strings.TrimPrefix(line, "branch ")
			current.Branch = strings.TrimPrefix(ref, "refs/heads/")
		case line == "bare":
			current.Bare = true
		case line == "detached":
			current.Detached = true
		case strings.HasPrefix(line, "locked"):
			reason := strings.TrimPrefix(line, "locked")
			reason = strings.TrimSpace(reason)
			if reason == "" {
				current.Locked = "locked"
			} else {
				current.Locked = reason
			}
		case strings.HasPrefix(line, "prunable"):
			reason := strings.TrimPrefix(line, "prunable")
			reason = strings.TrimSpace(reason)
			if reason == "" {
				current.Prunable = "prunable"
			} else {
				current.Prunable = reason
			}
		}
	}

	if current != nil {
		results = append(results, *current)
	}
	return results
}

func getMainWorktreePath(ctx context.Context, repoPath string) (string, error) {
	out, err := runGitCommand(ctx, repoPath, "worktree", "list", "--porcelain", "-z")
	if err != nil {
		return "", err
	}
	trees := parseWorktreePorcelain(out)
	if len(trees) == 0 {
		return "", fmt.Errorf("no worktrees found")
	}
	return trees[0].Path, nil
}

func pathsEqual(p1, p2 string) bool {
	abs1, err1 := filepath.Abs(p1)
	abs2, err2 := filepath.Abs(p2)
	if err1 != nil || err2 != nil {
		abs1 = filepath.Clean(p1)
		abs2 = filepath.Clean(p2)
	} else {
		// Evaluate symlinks if possible
		if real1, err := filepath.EvalSymlinks(abs1); err == nil {
			abs1 = real1
		}
		if real2, err := filepath.EvalSymlinks(abs2); err == nil {
			abs2 = real2
		}
		abs1 = filepath.Clean(abs1)
		abs2 = filepath.Clean(abs2)
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(abs1, abs2)
	}
	return abs1 == abs2
}

func main() {
	s := server.NewMCPServer("worktree-manager", "1.0.0")

	// 1. worktree_list
	listTool := mcp.NewTool("worktree_list",
		mcp.WithDescription("List all git worktrees in the repository with details including branch, HEAD commit, lock status, and whether it is the main worktree."),
		mcp.WithString("repo_path",
			mcp.Description("Optional path to the git repository or working directory. Defaults to current directory."),
		),
	)
	s.AddTool(listTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		repoPath := request.GetString("repo_path", ".")
		cmdCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		output, err := runGitCommand(cmdCtx, repoPath, "worktree", "list", "--porcelain", "-z")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list worktrees: %v", err)), nil
		}

		worktrees := parseWorktreePorcelain(output)
		data, err := json.MarshalIndent(worktrees, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to encode JSON: %v", err)), nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d worktree(s):\n\n", len(worktrees)))
		for i, wt := range worktrees {
			kind := "Linked"
			if wt.IsMain {
				kind = "Main"
			}
			branchInfo := wt.Branch
			if wt.Detached {
				branchInfo = "(detached HEAD)"
			} else if wt.Bare {
				branchInfo = "(bare)"
			}
			sb.WriteString(fmt.Sprintf("[%d] %s (%s)\n    Path:   %s\n    HEAD:   %s\n", i+1, branchInfo, kind, wt.Path, wt.HEAD))
			if wt.Locked != "" {
				sb.WriteString(fmt.Sprintf("    Locked: %s\n", wt.Locked))
			}
			if wt.Prunable != "" {
				sb.WriteString(fmt.Sprintf("    Prunable: %s\n", wt.Prunable))
			}
		}
		sb.WriteString("\nJSON data:\n")
		sb.WriteString(string(data))

		return mcp.NewToolResultText(sb.String()), nil
	})

	// 2. worktree_add
	addTool := mcp.NewTool("worktree_add",
		mcp.WithDescription("Create a new git worktree for parallel development or isolated subagent tasks."),
		mcp.WithString("path",
			mcp.Description("Path where the new worktree should be created (relative or absolute)."),
			mcp.Required(),
		),
		mcp.WithString("branch",
			mcp.Description("Existing branch to check out in the new worktree."),
		),
		mcp.WithString("new_branch",
			mcp.Description("New branch name to create and check out in the new worktree (-b option)."),
		),
		mcp.WithString("commit",
			mcp.Description("Base commit, branch, or tag to start the new branch/worktree from."),
		),
		mcp.WithString("repo_path",
			mcp.Description("Optional path to the git repository. Defaults to current directory."),
		),
	)
	s.AddTool(addTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path, err := request.RequireString("path")
		if err != nil || strings.TrimSpace(path) == "" {
			return mcp.NewToolResultError("path is required and cannot be empty"), nil
		}
		path = strings.TrimSpace(path)

		branch := strings.TrimSpace(request.GetString("branch", ""))
		newBranch := strings.TrimSpace(request.GetString("new_branch", ""))
		commit := strings.TrimSpace(request.GetString("commit", ""))
		repoPath := strings.TrimSpace(request.GetString("repo_path", "."))

		if branch != "" && newBranch != "" {
			return mcp.NewToolResultError("Cannot specify both 'branch' and 'new_branch'. Choose one."), nil
		}

		args := []string{"worktree", "add"}
		if newBranch != "" {
			args = append(args, "-b", newBranch)
		}
		args = append(args, "--", path)

		if branch != "" {
			args = append(args, branch)
		} else if commit != "" {
			args = append(args, commit)
		}

		cmdCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		out, err := runGitCommand(cmdCtx, repoPath, args...)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to add worktree: %v", err)), nil
		}

		msg := fmt.Sprintf("Successfully added worktree at %s\n%s", path, strings.TrimSpace(out))
		return mcp.NewToolResultText(msg), nil
	})

	// 3. worktree_remove
	removeTool := mcp.NewTool("worktree_remove",
		mcp.WithDescription("Safely remove a git worktree. Rejects removal of the main working tree."),
		mcp.WithString("path",
			mcp.Description("Path of the worktree to remove."),
			mcp.Required(),
		),
		mcp.WithBoolean("force",
			mcp.Description("Force removal even if the worktree contains untracked or uncommitted changes or is locked. Defaults to false."),
		),
		mcp.WithString("repo_path",
			mcp.Description("Optional path to the git repository. Defaults to current directory."),
		),
	)
	s.AddTool(removeTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path, err := request.RequireString("path")
		if err != nil || strings.TrimSpace(path) == "" {
			return mcp.NewToolResultError("path is required and cannot be empty"), nil
		}
		path = strings.TrimSpace(path)
		force := request.GetBool("force", false)
		repoPath := strings.TrimSpace(request.GetString("repo_path", "."))

		cmdCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		// Guard: protect main worktree
		targetPath := path
		if !filepath.IsAbs(targetPath) && repoPath != "" {
			targetPath = filepath.Join(repoPath, targetPath)
		}
		mainPath, err := getMainWorktreePath(cmdCtx, repoPath)
		if err == nil && pathsEqual(mainPath, targetPath) {
			return mcp.NewToolResultError(fmt.Sprintf("Cannot remove the main worktree (%s). Only linked worktrees can be removed.", mainPath)), nil
		}

		args := []string{"worktree", "remove"}
		if force {
			args = append(args, "--force", "--force")
		}
		args = append(args, "--", path)

		out, err := runGitCommand(cmdCtx, repoPath, args...)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to remove worktree: %v", err)), nil
		}

		msg := fmt.Sprintf("Successfully removed worktree at %s\n%s", path, strings.TrimSpace(out))
		return mcp.NewToolResultText(msg), nil
	})

	// 4. worktree_prune
	pruneTool := mcp.NewTool("worktree_prune",
		mcp.WithDescription("Prune git worktree administrative files for deleted or moved worktrees."),
		mcp.WithBoolean("dry_run",
			mcp.Description("Only report what worktrees would be pruned without actually removing them. Defaults to false."),
		),
		mcp.WithString("repo_path",
			mcp.Description("Optional path to the git repository. Defaults to current directory."),
		),
	)
	s.AddTool(pruneTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		dryRun := request.GetBool("dry_run", false)
		repoPath := strings.TrimSpace(request.GetString("repo_path", "."))

		args := []string{"worktree", "prune", "-v"}
		if dryRun {
			args = append(args, "--dry-run")
		}

		cmdCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		out, err := runGitCommand(cmdCtx, repoPath, args...)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to prune worktrees: %v", err)), nil
		}

		msg := strings.TrimSpace(out)
		if msg == "" {
			msg = "No worktrees needed pruning."
		}
		return mcp.NewToolResultText(msg), nil
	})

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
