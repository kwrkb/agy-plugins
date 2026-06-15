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

func runGhCommand(args ...string) (string, error) {
	cmd := exec.Command("gh", args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gh command failed: %v\nstderr: %s", err, stderr.String())
	}
	return stdout.String(), nil
}

func main() {
	s := server.NewMCPServer(
		"github",
		"1.0.0",
	)

	ghCommandTool := mcp.NewTool("gh_command",
		mcp.WithDescription("Run an arbitrary gh CLI command. Pass the subcommands and arguments as an array of separate strings, e.g. [\"issue\", \"list\", \"--limit\", \"10\"] or [\"pr\", \"create\", \"--title\", \"My Title\"]. Keep each value that contains spaces (titles, bodies, search queries) as a single array element."),
		mcp.WithArray("args",
			mcp.Description("The gh subcommands and arguments, one array element per token. Do NOT include the 'gh' executable itself. A value with spaces must be one element, e.g. [\"--title\", \"My Title\"]."),
			mcp.Items(map[string]any{"type": "string"}),
			mcp.Required(),
		),
	)
	s.AddTool(ghCommandTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := request.RequireStringSlice("args")
		if err != nil {
			return mcp.NewToolResultError("args is required and must be an array of strings"), nil
		}
		if len(args) == 0 {
			return mcp.NewToolResultError("args must contain at least one element"), nil
		}

		output, err := runGhCommand(args...)
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
