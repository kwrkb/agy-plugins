package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

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
		mcp.WithDescription("Run an arbitrary gh CLI command. Example command: 'issue list --limit 10' or 'pr view 123'."),
		mcp.WithString("command", mcp.Description("The subcommands and arguments for gh separated by spaces. Do NOT include the 'gh' executable itself."), mcp.Required()),
	)
	s.AddTool(ghCommandTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		argsMap, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("invalid arguments format"), nil
		}
		cmdStr, ok := argsMap["command"].(string)
		if !ok || cmdStr == "" {
			return mcp.NewToolResultError("command is required"), nil
		}
		
		args := strings.Fields(cmdStr)
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
