package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Options struct {
	ModulePath string   `json:"module_path"`
	Packages   []string `json:"packages"`
	Run        string   `json:"run,omitempty"`
	Timeout    int      `json:"timeout_seconds"`
}

func parseOptions(args map[string]any) (Options, error) {
	o := Options{Packages: []string{"./..."}, Timeout: 60}
	for k := range args {
		if k != "module_path" && k != "packages" && k != "run" && k != "timeout_seconds" {
			return o, fmt.Errorf("unknown argument %q", k)
		}
	}
	var ok bool
	o.ModulePath, ok = args["module_path"].(string)
	if !ok || strings.TrimSpace(o.ModulePath) == "" {
		return o, fmt.Errorf("module_path must be a nonempty string")
	}
	if v, exists := args["packages"]; exists {
		// Marshal/unmarshal accepts both []string and JSON-decoded []any, but
		// rejects mixed types. Do not use the SDK's coercing argument getters.
		b, err := json.Marshal(v)
		if err != nil || json.Unmarshal(b, &o.Packages) != nil || len(o.Packages) == 0 {
			return o, fmt.Errorf("packages must be a nonempty string array")
		}
	}
	for _, p := range o.Packages {
		if p == "" || strings.HasPrefix(p, "-") || strings.ContainsAny(p, "\x00\r\n") || strings.HasSuffix(p, ".go") {
			return o, fmt.Errorf("invalid package pattern %q (file lists and flags are not supported)", p)
		}
	}
	if v, exists := args["run"]; exists {
		o.Run, ok = v.(string)
		if !ok || strings.ContainsRune(o.Run, 0) {
			return o, fmt.Errorf("run must be a string without NUL bytes")
		}
	}
	if v, exists := args["timeout_seconds"]; exists {
		b, err := json.Marshal(v)
		var n float64
		if err != nil || json.Unmarshal(b, &n) != nil || math.IsNaN(n) || n < 1 || n > 300 || math.Trunc(n) != n {
			return o, fmt.Errorf("timeout_seconds must be an integer from 1 to 300")
		}
		o.Timeout = int(n)
	}
	return o, nil
}

func newServer() *server.MCPServer {
	s := server.NewMCPServer("test-runner", "1.0.0")
	s.AddTool(mcp.NewTool("go_test",
		mcp.WithDescription("Run Go tests in one module, summarizing failures and rerun arguments. Executes project code with inherited permissions/environment; tests and Go may write files, use caches, or access the network. No automatic retries. Timeout includes package discovery and compilation."),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("module_path", mcp.Required(), mcp.Description("Directory containing go.mod. Prefer an absolute path; relative paths use the MCP server's working directory.")),
		mcp.WithArray("packages", mcp.Items(map[string]any{"type": "string"}), mcp.MinItems(1), mcp.DefaultArray([]string{"./..."}), mcp.Description("Package patterns within this module, default [\"./...\"]. No flags or .go file lists.")),
		mcp.WithString("run", mcp.Description("Optional Go -run expression, including slash-separated subtest expressions.")),
		mcp.WithInteger("timeout_seconds", mcp.Min(1), mcp.Max(300), mcp.DefaultNumber(60), mcp.Description("Whole execution timeout in integer seconds, 1–300, default 60.")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		o, err := parseOptions(req.GetArguments())
		if err != nil {
			return resultTool(Result{Status: "error", Error: err.Error()}), nil
		}
		return resultTool(runTests(ctx, o)), nil
	})
	return s
}

func resultTool(r Result) *mcp.CallToolResult {
	b, err := json.Marshal(r)
	if err != nil {
		return mcp.NewToolResultError(err.Error())
	}
	res := mcp.NewToolResultText(string(b))
	res.IsError = r.Status == "error" || r.Status == "timeout" || r.Status == "cancelled"
	return res
}

func main() {
	if err := server.ServeStdio(newServer()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
