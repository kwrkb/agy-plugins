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

func runSgCommand(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "sg", args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// sg returns exit code 1 if no matches are found, which is not strictly an error.
		// However, for standard execution errors we want to return stderr.
		if stderr.Len() > 0 {
			return "", fmt.Errorf("sg command failed: %v\nstderr: %s", err, stderr.String())
		}
		// If stderr is empty and exit code is 1, it might just mean no matches.
	}
	return stdout.String(), nil
}

func main() {
	s := server.NewMCPServer(
		"ast-grep",
		"1.0.0",
	)

	astSearchTool := mcp.NewTool("ast_search",
		mcp.WithDescription("Search code structurally using ast-grep (sg). Returns matches in JSON format."),
		mcp.WithString("pattern", mcp.Required(), mcp.Description("The structural search pattern (e.g., 'fmt.Println($A)').")),
		mcp.WithString("language", mcp.Required(), mcp.Description("The language of the files to search (e.g., 'go', 'python', 'typescript').")),
		mcp.WithString("dir", mcp.Description("Optional directory to search in. Defaults to current directory.")),
	)

	s.AddTool(astSearchTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		argsMap, ok := request.Params.Arguments.(map[string]any)
		if !ok {
			return mcp.NewToolResultError("arguments must be a map"), nil
		}
		pattern, ok := argsMap["pattern"].(string)
		if !ok || pattern == "" {
			return mcp.NewToolResultError("pattern is required"), nil
		}
		language, ok := argsMap["language"].(string)
		if !ok || language == "" {
			return mcp.NewToolResultError("language is required"), nil
		}

		args := []string{"run", "-p", pattern, "-l", language, "--json"}
		if dir, ok := argsMap["dir"].(string); ok && dir != "" {
			args = append(args, dir)
		}

		output, err := runSgCommand(ctx, args...)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		if output == "" {
			return mcp.NewToolResultText("[]"), nil
		}

		return mcp.NewToolResultText(output), nil
	})

	astReplaceTool := mcp.NewTool("ast_replace",
		mcp.WithDescription("Rewrite code structurally using ast-grep (sg). Updates files in place."),
		mcp.WithString("pattern", mcp.Required(), mcp.Description("The structural search pattern (e.g., 'fmt.Println($A)').")),
		mcp.WithString("rewrite", mcp.Required(), mcp.Description("The rewrite pattern (e.g., 'log.Println($A)').")),
		mcp.WithString("language", mcp.Required(), mcp.Description("The language of the files to search (e.g., 'go', 'python', 'typescript').")),
		mcp.WithString("dir", mcp.Description("Optional directory to search in. Defaults to current directory.")),
	)

	s.AddTool(astReplaceTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		argsMap, ok := request.Params.Arguments.(map[string]any)
		if !ok {
			return mcp.NewToolResultError("arguments must be a map"), nil
		}
		pattern, ok := argsMap["pattern"].(string)
		if !ok || pattern == "" {
			return mcp.NewToolResultError("pattern is required"), nil
		}
		rewrite, ok := argsMap["rewrite"].(string)
		if !ok || rewrite == "" {
			return mcp.NewToolResultError("rewrite is required"), nil
		}
		language, ok := argsMap["language"].(string)
		if !ok || language == "" {
			return mcp.NewToolResultError("language is required"), nil
		}

		args := []string{"run", "-p", pattern, "-r", rewrite, "-l", language, "--update-all"}
		if dir, ok := argsMap["dir"].(string); ok && dir != "" {
			args = append(args, dir)
		}

		output, err := runSgCommand(ctx, args...)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Rewrite completed successfully.\n%s", output)), nil
	})

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
