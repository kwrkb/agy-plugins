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

func runGlabCommand(ctx context.Context, timeout time.Duration, args ...string) (string, error) {
	return run(ctx, timeout, "glab", func(ctx context.Context) *exec.Cmd { return glabCommand(ctx, args...) })
}

func main() {
	s := server.NewMCPServer(
		"gitlab",
		"1.0.0",
	)

	glabCommandTool := mcp.NewTool("glab_command",
		mcp.WithDescription("Run an arbitrary glab CLI command. Pass the subcommands and arguments as an array of separate strings, e.g. [\"issue\", \"list\", \"-R\", \"group/project\"] or [\"mr\", \"create\", \"-R\", \"group/project\", \"--title\", \"My Title\"]. Keep each value that contains spaces (titles, descriptions, search queries) as a single array element. stdout is cut at 64 KiB and stderr at 16 KiB; a cut ends with an \"[output truncated: ...]\" line. The command is stopped after timeout_seconds (default 300)."),
		mcp.WithArray("args",
			mcp.Description("The glab subcommands and arguments, one array element per token. Do NOT include the 'glab' executable itself. A value with spaces must be one element, e.g. [\"--title\", \"My Title\"]."),
			mcp.Items(map[string]any{"type": "string"}),
			mcp.Required(),
		),
		mcp.WithInteger("timeout_seconds",
			mcp.Min(1), mcp.Max(maxTimeout), mcp.DefaultNumber(defaultTimeout),
			mcp.Description("Whole execution timeout in integer seconds, 1–3600, default 300. Raise it for commands that stream until a job ends, such as [\"ci\", \"trace\"]; on timeout the output so far is returned with the error."),
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

		timeout, err := parseTimeout(request.GetArguments())
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		output, err := runGlabCommand(ctx, timeout, args...)
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
