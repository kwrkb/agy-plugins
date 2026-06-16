package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// astGrepBinary is the ast-grep CLI name. The full name is used instead of the
// `sg` alias because on Linux `sg` resolves to the setgroups command, which
// would make every invocation fail before reaching ast-grep.
// See https://ast-grep.github.io/guide/quick-start.html
const astGrepBinary = "ast-grep"

// requiredString extracts a non-empty string argument, returning an error
// suitable for surfacing to the MCP client when it is missing or blank.
func requiredString(m map[string]any, key string) (string, error) {
	v, ok := m[key].(string)
	if !ok || v == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return v, nil
}

// optionalString returns the string value for key, or "" when absent.
func optionalString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// buildSearchArgs validates the search arguments and constructs the ast-grep
// command line for a structural search.
func buildSearchArgs(m map[string]any) ([]string, error) {
	pattern, err := requiredString(m, "pattern")
	if err != nil {
		return nil, err
	}
	language, err := requiredString(m, "language")
	if err != nil {
		return nil, err
	}
	args := []string{"run", "-p", pattern, "-l", language, "--json"}
	if dir := optionalString(m, "dir"); dir != "" {
		args = append(args, dir)
	}
	return args, nil
}

// buildReplaceArgs validates the replace arguments and constructs the ast-grep
// command line for an in-place rewrite. The rewrite must be present and a
// string, but an empty rewrite is allowed: it deletes the matched node.
func buildReplaceArgs(m map[string]any) ([]string, error) {
	pattern, err := requiredString(m, "pattern")
	if err != nil {
		return nil, err
	}
	rewrite, ok := m["rewrite"].(string)
	if !ok {
		return nil, fmt.Errorf("rewrite is required")
	}
	language, err := requiredString(m, "language")
	if err != nil {
		return nil, err
	}
	args := []string{"run", "-p", pattern, "-r", rewrite, "-l", language, "--update-all"}
	if dir := optionalString(m, "dir"); dir != "" {
		args = append(args, dir)
	}
	return args, nil
}

func runSgCommand(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, astGrepBinary, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		// The binary could not be started at all (e.g. ast-grep not on PATH).
		// This must always be reported; otherwise it is indistinguishable from
		// a search that found nothing.
		var execErr *exec.Error
		if errors.As(err, &execErr) {
			return "", fmt.Errorf("%s command failed: %w", astGrepBinary, err)
		}
		// ast-grep ran but exited non-zero. `ast-grep run` exits 0 even when it
		// finds no matches, so a non-zero exit carrying stderr diagnostics is a
		// real error (e.g. a pattern parse error).
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%s command failed: %v\nstderr: %s", astGrepBinary, err, stderr.String())
		}
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
		args, err := buildSearchArgs(argsMap)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
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
		args, err := buildReplaceArgs(argsMap)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
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
