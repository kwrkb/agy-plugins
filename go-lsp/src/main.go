package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// LSPClient manages the life cycle of a gopls server and communicates with it via JSON-RPC.
type LSPClient struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	idGen   int64
	mu      sync.Mutex
	pending map[int64]chan []byte

	rootPath    string
	initialized bool
}

func NewLSPClient(rootPath string) (*LSPClient, error) {
	goplsPath, err := exec.LookPath("gopls")
	if err != nil {
		return nil, fmt.Errorf("gopls not found in PATH: %w", err)
	}

	cmd := exec.Command(goplsPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	client := &LSPClient{
		cmd:      cmd,
		stdin:    stdin,
		stdout:   stdout,
		stderr:   stderr,
		pending:  make(map[int64]chan []byte),
		rootPath: rootPath,
	}

	// Output gopls stderr to MCP server's stderr for debugging.
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Fprintf(os.Stderr, "[gopls stderr] %s\n", scanner.Text())
		}
	}()

	go client.readLoop()

	return client, nil
}

func (c *LSPClient) readLoop() {
	reader := bufio.NewReader(c.stdout)
	for {
		contentLength := 0
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					fmt.Fprintf(os.Stderr, "error reading header: %v\n", err)
				}
				return
			}
			line = strings.TrimSpace(line)
			if line == "" {
				break
			}
			if strings.HasPrefix(line, "Content-Length:") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &contentLength)
				}
			}
		}

		if contentLength == 0 {
			continue
		}

		body := make([]byte, contentLength)
		_, err := io.ReadFull(reader, body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading body: %v\n", err)
			return
		}

		var msg struct {
			ID *int64 `json:"id"`
		}
		if err := json.Unmarshal(body, &msg); err != nil {
			fmt.Fprintf(os.Stderr, "error unmarshaling incoming message: %v\n", err)
			continue
		}

		if msg.ID != nil {
			c.mu.Lock()
			ch, ok := c.pending[*msg.ID]
			if ok {
				delete(c.pending, *msg.ID)
			}
			c.mu.Unlock()
			if ok {
				ch <- body
			}
		}
	}
}

type Request struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type Notification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

func (c *LSPClient) Call(ctx context.Context, method string, params any) ([]byte, error) {
	id := atomic.AddInt64(&c.idGen, 1)
	req := Request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	ch := make(chan []byte, 1)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	if _, err := io.WriteString(c.stdin, header); err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, err
	}
	if _, err := c.stdin.Write(body); err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, err
	}

	select {
	case res := <-ch:
		return res, nil
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, ctx.Err()
	}
}

func (c *LSPClient) Notify(method string, params any) error {
	req := Notification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	if _, err := io.WriteString(c.stdin, header); err != nil {
		return err
	}
	if _, err := c.stdin.Write(body); err != nil {
		return err
	}
	return nil
}

// escapeDriveLetter prefixes a Windows drive-letter path (e.g. "C:/Users/...")
// with a leading slash so net/url doesn't mistake the drive letter for a
// host when building a file:// URI: "file:///C:/...", not "file://C:/...".
func escapeDriveLetter(slashed string) string {
	if len(slashed) > 1 && slashed[1] == ':' {
		return "/" + slashed
	}
	return slashed
}

func pathToURI(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		abs = p
	}
	u := url.URL{
		Scheme: "file",
		Path:   escapeDriveLetter(filepath.ToSlash(abs)),
	}
	return u.String()
}

func uriToPath(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return u
	}
	if parsed.Scheme != "file" {
		return u
	}
	path := parsed.Path
	if os.PathSeparator == '\\' && len(path) > 3 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}
	return filepath.FromSlash(path)
}

func (c *LSPClient) Initialize(ctx context.Context) error {
	if c.initialized {
		return nil
	}

	rootURI := pathToURI(c.rootPath)

	params := map[string]any{
		"processId": os.Getpid(),
		"rootPath":  c.rootPath,
		"rootUri":   rootURI,
		"capabilities": map[string]any{
			"textDocument": map[string]any{
				"definition": map[string]any{
					"dynamicRegistration": true,
				},
				"references": map[string]any{
					"dynamicRegistration": true,
				},
				"hover": map[string]any{
					"dynamicRegistration": true,
				},
			},
		},
	}

	res, err := c.Call(ctx, "initialize", params)
	if err != nil {
		return fmt.Errorf("LSP initialize failed: %w", err)
	}

	var rpcRes struct {
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(res, &rpcRes); err != nil {
		return err
	}
	if rpcRes.Error != nil {
		return fmt.Errorf("LSP initialize response error (code %d): %s", rpcRes.Error.Code, rpcRes.Error.Message)
	}

	if err := c.Notify("initialized", map[string]any{}); err != nil {
		return fmt.Errorf("LSP initialized notification failed: %w", err)
	}

	c.initialized = true
	return nil
}

func (c *LSPClient) Close() {
	if c.stdin != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, _ = c.Call(ctx, "shutdown", nil)
		_ = c.Notify("exit", nil)
		c.stdin.Close()
	}
	if c.cmd != nil {
		_ = c.cmd.Wait()
	}
}

var (
	// clients holds one LSPClient per resolved workspace root, so requests
	// against different `dir` values don't silently reuse an unrelated root.
	clients   = make(map[string]*LSPClient)
	clientsMu sync.Mutex
)

func getLSPClient(ctx context.Context, dir string) (*LSPClient, error) {
	rootPath := dir
	if rootPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current working directory: %w", err)
		}
		rootPath = cwd
	}
	if abs, err := filepath.Abs(rootPath); err == nil {
		rootPath = abs
	}

	clientsMu.Lock()
	defer clientsMu.Unlock()

	if c, ok := clients[rootPath]; ok {
		return c, nil
	}

	c, err := NewLSPClient(rootPath)
	if err != nil {
		return nil, err
	}

	if err := c.Initialize(ctx); err != nil {
		c.Close()
		return nil, err
	}

	clients[rootPath] = c
	return c, nil
}

func closeAllLSPClients() {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	for _, c := range clients {
		c.Close()
	}
	clients = make(map[string]*LSPClient)
}

func requiredString(m map[string]any, key string) (string, error) {
	v, ok := m[key].(string)
	if !ok || v == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return v, nil
}

func requiredInt(m map[string]any, key string) (int, error) {
	v, ok := m[key].(float64)
	if !ok {
		vi, ok := m[key].(int)
		if !ok {
			return 0, fmt.Errorf("%s is required as a number", key)
		}
		return vi, nil
	}
	return int(v), nil
}

func optionalString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

type Position struct {
	Line      int `json:"line"`      // 0-indexed
	Character int `json:"character"` // 0-indexed
}

type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type ReferenceParams struct {
	TextDocumentPositionParams
	Context ReferenceContext `json:"context"`
}

type ReferenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

type HoverResult struct {
	Contents json.RawMessage `json:"contents"`
}

type MarkupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

func main() {
	s := server.NewMCPServer("go-lsp", "1.0.0")

	goDefinitionTool := mcp.NewTool("go_definition",
		mcp.WithDescription("Get the definition of the Go symbol at the specified position. line and character are 1-indexed."),
		mcp.WithString("file_path", mcp.Required(), mcp.Description("The absolute file path of the Go source file.")),
		mcp.WithNumber("line", mcp.Required(), mcp.Description("The 1-indexed line number.")),
		mcp.WithNumber("character", mcp.Required(), mcp.Description("The 1-indexed character offset.")),
		mcp.WithString("dir", mcp.Description("Optional workspace root directory. Defaults to current directory.")),
	)

	s.AddTool(goDefinitionTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		argsMap, ok := request.Params.Arguments.(map[string]any)
		if !ok {
			return mcp.NewToolResultError("arguments must be a map"), nil
		}

		filePath, err := requiredString(argsMap, "file_path")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		line, err := requiredInt(argsMap, "line")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		character, err := requiredInt(argsMap, "character")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		dir := optionalString(argsMap, "dir")

		lsp, err := getLSPClient(ctx, dir)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to initialize LSP client: %v", err)), nil
		}

		params := TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{
				URI: pathToURI(filePath),
			},
			Position: Position{
				Line:      line - 1,
				Character: character - 1,
			},
		}

		resBytes, err := lsp.Call(ctx, "textDocument/definition", params)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		var rpcRes struct {
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(resBytes, &rpcRes); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to unmarshal JSON-RPC response: %v", err)), nil
		}
		if rpcRes.Error != nil {
			return mcp.NewToolResultError(rpcRes.Error.Message), nil
		}

		if len(rpcRes.Result) == 0 || string(rpcRes.Result) == "null" {
			return mcp.NewToolResultText("Definition not found"), nil
		}

		var locs []Location
		if rpcRes.Result[0] == '[' {
			if err := json.Unmarshal(rpcRes.Result, &locs); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to parse locations array: %v", err)), nil
			}
		} else {
			var loc Location
			if err := json.Unmarshal(rpcRes.Result, &loc); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to parse single location: %v", err)), nil
			}
			locs = append(locs, loc)
		}

		var lines []string
		for _, loc := range locs {
			p := uriToPath(loc.URI)
			lines = append(lines, fmt.Sprintf("%s: line %d, char %d to line %d, char %d",
				p,
				loc.Range.Start.Line+1,
				loc.Range.Start.Character+1,
				loc.Range.End.Line+1,
				loc.Range.End.Character+1,
			))
		}

		return mcp.NewToolResultText(strings.Join(lines, "\n")), nil
	})

	goReferencesTool := mcp.NewTool("go_references",
		mcp.WithDescription("Find all references to the Go symbol at the specified position. line and character are 1-indexed."),
		mcp.WithString("file_path", mcp.Required(), mcp.Description("The absolute file path of the Go source file.")),
		mcp.WithNumber("line", mcp.Required(), mcp.Description("The 1-indexed line number.")),
		mcp.WithNumber("character", mcp.Required(), mcp.Description("The 1-indexed character offset.")),
		mcp.WithString("dir", mcp.Description("Optional workspace root directory. Defaults to current directory.")),
	)

	s.AddTool(goReferencesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		argsMap, ok := request.Params.Arguments.(map[string]any)
		if !ok {
			return mcp.NewToolResultError("arguments must be a map"), nil
		}

		filePath, err := requiredString(argsMap, "file_path")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		line, err := requiredInt(argsMap, "line")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		character, err := requiredInt(argsMap, "character")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		dir := optionalString(argsMap, "dir")

		lsp, err := getLSPClient(ctx, dir)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to initialize LSP client: %v", err)), nil
		}

		params := ReferenceParams{
			TextDocumentPositionParams: TextDocumentPositionParams{
				TextDocument: TextDocumentIdentifier{
					URI: pathToURI(filePath),
				},
				Position: Position{
					Line:      line - 1,
					Character: character - 1,
				},
			},
			Context: ReferenceContext{
				IncludeDeclaration: true,
			},
		}

		resBytes, err := lsp.Call(ctx, "textDocument/references", params)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		var rpcRes struct {
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(resBytes, &rpcRes); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to unmarshal JSON-RPC response: %v", err)), nil
		}
		if rpcRes.Error != nil {
			return mcp.NewToolResultError(rpcRes.Error.Message), nil
		}

		if len(rpcRes.Result) == 0 || string(rpcRes.Result) == "null" {
			return mcp.NewToolResultText("References not found"), nil
		}

		var locs []Location
		if err := json.Unmarshal(rpcRes.Result, &locs); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse references array: %v", err)), nil
		}

		var lines []string
		for _, loc := range locs {
			p := uriToPath(loc.URI)
			lines = append(lines, fmt.Sprintf("%s: line %d, char %d to line %d, char %d",
				p,
				loc.Range.Start.Line+1,
				loc.Range.Start.Character+1,
				loc.Range.End.Line+1,
				loc.Range.End.Character+1,
			))
		}

		return mcp.NewToolResultText(strings.Join(lines, "\n")), nil
	})

	goHoverTool := mcp.NewTool("go_hover",
		mcp.WithDescription("Get documentation or type information for the Go symbol at the specified position. line and character are 1-indexed."),
		mcp.WithString("file_path", mcp.Required(), mcp.Description("The absolute file path of the Go source file.")),
		mcp.WithNumber("line", mcp.Required(), mcp.Description("The 1-indexed line number.")),
		mcp.WithNumber("character", mcp.Required(), mcp.Description("The 1-indexed character offset.")),
		mcp.WithString("dir", mcp.Description("Optional workspace root directory. Defaults to current directory.")),
	)

	s.AddTool(goHoverTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		argsMap, ok := request.Params.Arguments.(map[string]any)
		if !ok {
			return mcp.NewToolResultError("arguments must be a map"), nil
		}

		filePath, err := requiredString(argsMap, "file_path")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		line, err := requiredInt(argsMap, "line")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		character, err := requiredInt(argsMap, "character")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		dir := optionalString(argsMap, "dir")

		lsp, err := getLSPClient(ctx, dir)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to initialize LSP client: %v", err)), nil
		}

		params := TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{
				URI: pathToURI(filePath),
			},
			Position: Position{
				Line:      line - 1,
				Character: character - 1,
			},
		}

		resBytes, err := lsp.Call(ctx, "textDocument/hover", params)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		var rpcRes struct {
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(resBytes, &rpcRes); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to unmarshal JSON-RPC response: %v", err)), nil
		}
		if rpcRes.Error != nil {
			return mcp.NewToolResultError(rpcRes.Error.Message), nil
		}

		if len(rpcRes.Result) == 0 || string(rpcRes.Result) == "null" {
			return mcp.NewToolResultText("No hover information available"), nil
		}

		var hover HoverResult
		if err := json.Unmarshal(rpcRes.Result, &hover); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse hover result: %v", err)), nil
		}

		// Try parsing markup content first.
		var markup MarkupContent
		if err := json.Unmarshal(hover.Contents, &markup); err == nil && markup.Value != "" {
			return mcp.NewToolResultText(markup.Value), nil
		}

		// Fallback to string.
		var str string
		if err := json.Unmarshal(hover.Contents, &str); err == nil && str != "" {
			return mcp.NewToolResultText(str), nil
		}

		// Fallback to array of strings.
		var arr []string
		if err := json.Unmarshal(hover.Contents, &arr); err == nil && len(arr) > 0 {
			return mcp.NewToolResultText(strings.Join(arr, "\n")), nil
		}

		return mcp.NewToolResultText(string(hover.Contents)), nil
	})

	defer closeAllLSPClients()

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
