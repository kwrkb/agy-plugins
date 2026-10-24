package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// glabCommand builds the child process for one glab invocation.
// Stdin is left nil (the null device), so glab never waits on input; glab's
// own non-interactive detection then fails fast when a required flag is
// missing. GLAB_NO_PROMPT=1 disables prompts explicitly as well. Appended last,
// it overrides a GLAB_NO_PROMPT=0 inherited from the user's environment.
func glabCommand(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "glab", args...)
	cmd.Env = append(os.Environ(), "GLAB_NO_PROMPT=1")
	return cmd
}

func runGlabCommand(ctx context.Context, args ...string) (string, error) {
	cmd := glabCommand(ctx, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("glab command failed: %v\nstderr: %s", err, stderr.String())
	}
	return stdout.String(), nil
}

func main() {
	s := server.NewMCPServer(
		"gitlab",
		"1.0.0",
	)

	glabCommandTool := mcp.NewTool("glab_command",
		mcp.WithDescription("Run an arbitrary glab CLI command. Pass the subcommands and arguments as an array of separate strings, e.g. [\"issue\", \"list\", \"-R\", \"group/project\"] or [\"mr\", \"create\", \"-R\", \"group/project\", \"--title\", \"My Title\"]. Keep each value that contains spaces (titles, descriptions, search queries) as a single array element."),
		mcp.WithArray("args",
			mcp.Description("The glab subcommands and arguments, one array element per token. Do NOT include the 'glab' executable itself. A value with spaces must be one element, e.g. [\"--title\", \"My Title\"]."),
			mcp.Items(map[string]any{"type": "string"}),
			mcp.Required(),
		),
	)
	s.AddTool(glabCommandTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := request.RequireStringSlice("args")
		if err != nil {
			return mcp.NewToolResultError("args is required and must be an array of strings"), nil
		}
		if len(args) == 0 {
			return mcp.NewToolResultError("args must contain at least one element"), nil
		}

		output, err := runGlabCommand(ctx, args...)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(output), nil
	})

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
