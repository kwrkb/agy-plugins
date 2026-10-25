package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ghCommand builds the child process for one gh invocation.
func ghCommand(ctx context.Context, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "gh", args...)
}

func runGhCommand(ctx context.Context, timeout time.Duration, args ...string) (string, error) {
	return run(ctx, timeout, "gh", func(ctx context.Context) *exec.Cmd { return ghCommand(ctx, args...) })
}

func main() {
	s := server.NewMCPServer(
		"github",
		"1.0.0",
	)

	ghCommandTool := mcp.NewTool("gh_command",
		mcp.WithDescription("Run an arbitrary gh CLI command. Pass the subcommands and arguments as an array of separate strings, e.g. [\"issue\", \"list\", \"--limit\", \"10\"] or [\"pr\", \"create\", \"--title\", \"My Title\"]. Keep each value that contains spaces (titles, bodies, search queries) as a single array element. stdout is cut at 64 KiB and stderr at 16 KiB; a cut ends with an \"[output truncated: ...]\" line. The command is stopped after timeout_seconds (default 300)."),
		mcp.WithArray("args",
			mcp.Description("The gh subcommands and arguments, one array element per token. Do NOT include the 'gh' executable itself. A value with spaces must be one element, e.g. [\"--title\", \"My Title\"]."),
			mcp.Items(map[string]any{"type": "string"}),
			mcp.Required(),
		),
		mcp.WithInteger("timeout_seconds",
			mcp.Min(1), mcp.Max(maxTimeout), mcp.DefaultNumber(defaultTimeout),
			mcp.Description("Whole execution timeout in integer seconds, 1–3600, default 300. Raise it for commands that stream until a job ends, such as [\"run\", \"watch\"]; on timeout the output so far is returned with the error."),
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

		timeout, err := parseTimeout(request.GetArguments())
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		output, err := runGhCommand(ctx, timeout, args...)
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
