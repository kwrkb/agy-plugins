package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// resolveEditedFile は agy / Claude Code の別スキーマな PostToolUse payload から
// 編集ファイルを取り出せること、および非対象（非 write / toolCall:null / 解析不能）を
// 無音 return（ok=false）にできることを検証する。
func TestResolveEditedFile(t *testing.T) {
	cases := []struct {
		name     string
		json     string
		wantPath string
		wantOK   bool
	}{
		{
			name:     "agy 新規作成（write_to_file）",
			json:     `{"stepIdx":3,"toolCall":{"name":"write_to_file","args":{"TargetFile":"/abs/plugin.json","CodeContent":"{}","Overwrite":true}},"workspacePaths":["/abs"]}`,
			wantPath: "/abs/plugin.json",
			wantOK:   true,
		},
		{
			name:     "agy 既存編集（replace_file_content）",
			json:     `{"stepIdx":3,"toolCall":{"name":"replace_file_content","args":{"TargetFile":"/abs/mcp_config.json"}}}`,
			wantPath: "/abs/mcp_config.json",
			wantOK:   true,
		},
		{
			name:     "agy 非ファイルステップ（toolCall:null）",
			json:     `{"stepIdx":1,"toolCall":null,"error":""}`,
			wantPath: "",
			wantOK:   false,
		},
		{
			name:     "agy 非ファイルツール（TargetFile 無し）はガード",
			json:     `{"toolCall":{"name":"run_command","args":{}}}`,
			wantPath: "",
			wantOK:   false,
		},
		{
			name:     "Claude Code tool_input.file_path（後方互換）",
			json:     `{"tool_name":"Write","tool_input":{"file_path":"/abs/plugin.json"}}`,
			wantPath: "/abs/plugin.json",
			wantOK:   true,
		},
		{
			name:     "Claude Code トップレベル file_path",
			json:     `{"file_path":"/abs/mcp_config.json"}`,
			wantPath: "/abs/mcp_config.json",
			wantOK:   true,
		},
		{
			name:     "解析不能な stdin",
			json:     `not json`,
			wantPath: "",
			wantOK:   false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := resolveEditedFile(strings.NewReader(c.json))
			if got != c.wantPath || ok != c.wantOK {
				t.Errorf("resolveEditedFile(%s) = (%q, %v), want (%q, %v)", c.json, got, ok, c.wantPath, c.wantOK)
			}
		})
	}
}

// isManifestFile はマニフェスト系の basename だけ true を返すこと（パスや大小文字に依らず）。
func TestIsManifestFile(t *testing.T) {
	manifests := []string{
		"/ws/myplugin/plugin.json",
		"/ws/myplugin/gemini-extension.json",
		"/ws/myplugin/mcp_config.json",
		"/ws/myplugin/hooks.json",
		"PLUGIN.JSON", // 大小文字非依存
	}
	for _, p := range manifests {
		if !isManifestFile(p) {
			t.Errorf("isManifestFile(%q) = false, want true", p)
		}
	}
	nonManifests := []string{
		"/ws/myplugin/main.go",
		"/ws/myplugin/README.md",
		"/ws/notes.txt",
		"/ws/skills/x/SKILL.md",
	}
	for _, p := range nonManifests {
		if isManifestFile(p) {
			t.Errorf("isManifestFile(%q) = true, want false", p)
		}
	}
}

func TestFixPaths(t *testing.T) {
	// 1. 正常な mcp_config.json のパス置換
	t.Run("normal mcp_config.json", func(t *testing.T) {
		tmpDir := t.TempDir()
		mcpPath := filepath.Join(tmpDir, "mcp_config.json")
		content := `{"mcpServers":{"test":{"command":"${extensionPath}${/}bin${/}test"}}}`
		if err := os.WriteFile(mcpPath, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write mcp_config.json: %v", err)
		}

		if err := fixPaths(tmpDir); err != nil {
			t.Fatalf("fixPaths failed: %v", err)
		}

		fixed, err := os.ReadFile(mcpPath)
		if err != nil {
			t.Fatalf("failed to read fixed mcp_config.json: %v", err)
		}
		if strings.Contains(string(fixed), "${extensionPath}") || strings.Contains(string(fixed), "${/}") {
			t.Errorf("expected placeholders to be replaced, got: %s", string(fixed))
		}
	})

	// 2. プラグイン外を指す symlink のブロック (FD-KITVAL-001)
	t.Run("symlink escaping plugin directory", func(t *testing.T) {
		outsideDir := t.TempDir()
		outsideFile := filepath.Join(outsideDir, "target_config.json")
		if err := os.WriteFile(outsideFile, []byte(`{"outside":true}`), 0o644); err != nil {
			t.Fatalf("failed to write outside file: %v", err)
		}

		pluginDir := t.TempDir()
		symlinkPath := filepath.Join(pluginDir, "mcp_config.json")
		if err := os.Symlink(outsideFile, symlinkPath); err != nil {
			t.Skipf("symlinks not supported on this platform: %v", err)
		}

		err := fixPaths(pluginDir)
		if err == nil {
			t.Fatal("expected fixPaths to fail on escaping symlink, but it succeeded")
		}
		if !strings.Contains(err.Error(), "セキュリティエラー") {
			t.Errorf("expected security error message, got: %v", err)
		}
	})
}

func TestPluginRootFromManifest(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{filepath.Join("/ws", "p", "plugin.json"), filepath.Join("/ws", "p")},
		{filepath.Join("/ws", "p", ".antigravity-plugin", "plugin.json"), filepath.Join("/ws", "p")},
		{filepath.Join("/ws", "p", ".gemini-plugin", "plugin.json"), filepath.Join("/ws", "p")},
	}
	for _, c := range cases {
		if got := pluginRootFromManifest(c.in); got != c.want {
			t.Errorf("pluginRootFromManifest(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAgy142Rules(t *testing.T) {
	hasCode := func(fs []finding, code string) bool {
		for _, f := range fs {
			if f.code == code {
				return true
			}
		}
		return false
	}

	t.Run(".antigravity-plugin/plugin.json passes C5", func(t *testing.T) {
		dir := t.TempDir()
		sub := filepath.Join(dir, ".antigravity-plugin")
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatal(err)
		}
		mustWrite(t, filepath.Join(sub, "plugin.json"), `{"name":"sub"}`)
		fs := validate(dir)
		if hasCode(fs, "C5") {
			t.Errorf("expected no C5 error for .antigravity-plugin/plugin.json, got %+v", fs)
		}
	})

	t.Run("native mcp_config command with ${PLUGIN_ROOT} triggers C2", func(t *testing.T) {
		dir := t.TempDir()
		mustWrite(t, filepath.Join(dir, "plugin.json"), `{"name":"p"}`)
		mustWrite(t, filepath.Join(dir, "mcp_config.json"), `{"mcpServers":{"s":{"command":"${PLUGIN_ROOT}/bin/srv"}}}`)
		fs := validate(dir)
		if !hasCode(fs, "C2") {
			t.Errorf("expected C2 error when ${PLUGIN_ROOT} is used in command, got %+v", fs)
		}
	})

	t.Run("native mcp_config args/env with ${PLUGIN_ROOT} and ${PLUGIN_DATA} is allowed", func(t *testing.T) {
		dir := t.TempDir()
		mustWrite(t, filepath.Join(dir, "plugin.json"), `{"name":"p"}`)
		mustWrite(t, filepath.Join(dir, "mcp_config.json"), `{"mcpServers":{"s":{"command":"glab","args":["${PLUGIN_ROOT}/cfg","${PLUGIN_DATA}"],"env":{"D":"${PLUGIN_DATA}"}}}}`)
		fs := validate(dir)
		if hasCode(fs, "C2") {
			t.Errorf("expected no C2 when ${PLUGIN_ROOT}/${PLUGIN_DATA} is only in args/env, got %+v", fs)
		}
	})

	t.Run("hooks.json with ${PLUGIN_ROOT} triggers C10 and agy named hook schema detects C8", func(t *testing.T) {
		dir := t.TempDir()
		mustWrite(t, filepath.Join(dir, "plugin.json"), `{"name":"p"}`)
		mustWrite(t, filepath.Join(dir, "hooks.json"), `{"my-hook":{"PreInvocation":[{"type":"command","command":"${PLUGIN_ROOT}/h"},{"type":"command","command":"${CLAUDE_PLUGIN_ROOT}/c"}]}}`)
		fs := validate(dir)
		if !hasCode(fs, "C10") {
			t.Errorf("expected C10 warning for ${PLUGIN_ROOT} in hooks.json, got %+v", fs)
		}
		if !hasCode(fs, "C8") {
			t.Errorf("expected C8 error for ${CLAUDE_PLUGIN_ROOT} in agy named hooks.json, got %+v", fs)
		}
	})

	t.Run("gemini-extension.json with hooks.json macros also triggers C10", func(t *testing.T) {
		dir := t.TempDir()
		mustWrite(t, filepath.Join(dir, "gemini-extension.json"), `{"name":"p"}`)
		mustWrite(t, filepath.Join(dir, "hooks.json"), `{"my-hook":{"PreInvocation":[{"type":"command","command":"${extensionPath}${/}h"}]}}`)
		fs := validate(dir)
		if !hasCode(fs, "C10") {
			t.Errorf("expected C10 warning for hooks.json macros in gemini-extension.json plugin, got %+v", fs)
		}
	})

	t.Run("gemini-extension.json command with ${PLUGIN_ROOT} triggers C2", func(t *testing.T) {
		// install は ${PLUGIN_ROOT} を書き換えず生成先 mcp_config.json の command に literal で残す（agy 1.4.2 実測）
		dir := t.TempDir()
		mustWrite(t, filepath.Join(dir, "gemini-extension.json"), `{"name":"p","mcpServers":{"s":{"command":"${PLUGIN_ROOT}/bin/srv"}}}`)
		fs := validate(dir)
		if !hasCode(fs, "C2") {
			t.Errorf("expected C2 error for ${PLUGIN_ROOT} in gemini-extension.json command, got %+v", fs)
		}
	})

	t.Run("gemini-extension.json command with ${extensionPath}${/} does not trigger C2", func(t *testing.T) {
		dir := t.TempDir()
		mustWrite(t, filepath.Join(dir, "gemini-extension.json"), `{"name":"p","mcpServers":{"s":{"command":"${extensionPath}${/}bin${/}srv"}}}`)
		fs := validate(dir)
		if hasCode(fs, "C2") {
			t.Errorf("expected no C2 for ${extensionPath}${/} in gemini-extension.json command, got %+v", fs)
		}
	})
}

// useRelayDir は受け渡しファイルの置き場をテスト用の一時ディレクトリに差し替える。
func useRelayDir(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	orig := relayDir
	relayDir = func() string { return d }
	t.Cleanup(func() { relayDir = orig })
	return d
}

func TestRelayPath(t *testing.T) {
	useRelayDir(t)
	for _, id := range []string{"ec33ebf9-0cba-4100-8142-c61503f6c587", "abc_DEF-1"} {
		if _, ok := relayPath(id); !ok {
			t.Errorf("relayPath(%q) rejected a valid id", id)
		}
	}
	for _, id := range []string{"", "../x", "a/b", `a\b`, "a.b", strings.Repeat("a", 129)} {
		if _, ok := relayPath(id); ok {
			t.Errorf("relayPath(%q) accepted an unsafe id", id)
		}
	}
}

// --hook が残した検査結果を --inject が ephemeralMessage として 1 回だけ差し込むことを検証する。
func TestHookInjectRelay(t *testing.T) {
	useRelayDir(t)
	plugin := t.TempDir()
	mustWrite(t, filepath.Join(plugin, "plugin.json"), `{"name":"p"}`)
	mustWrite(t, filepath.Join(plugin, "mcp_config.json"), `{"mcpServers":{"s":{"command":"${PLUGIN_ROOT}/bin/srv"}}}`)
	const conv = "ec33ebf9-0cba-4100-8142-c61503f6c587"
	hookPayload := func(target string) []byte {
		b, _ := json.Marshal(map[string]any{
			"conversationId": conv,
			"toolCall":       map[string]any{"name": "write_to_file", "args": map[string]string{"TargetFile": target}},
		})
		return b
	}
	injectPayload := []byte(`{"conversationId":"` + conv + `","invocationNum":2}`)

	var log bytes.Buffer
	handleHook(hookPayload(filepath.Join(plugin, "plugin.json")), &log)
	if !strings.Contains(log.String(), "C2") {
		t.Fatalf("hook log should contain C2, got %q", log.String())
	}

	var got struct {
		InjectSteps []struct {
			EphemeralMessage string `json:"ephemeralMessage"`
		} `json:"injectSteps"`
	}
	if err := json.Unmarshal(injectOutput(injectPayload), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.InjectSteps) != 1 || !strings.Contains(got.InjectSteps[0].EphemeralMessage, "C2") {
		t.Fatalf("expected one ephemeralMessage containing C2, got %+v", got)
	}
	if out := string(injectOutput(injectPayload)); out != "{}" {
		t.Errorf("relay should be consumed once, second inject = %s", out)
	}

	t.Run("non-manifest edit leaves nothing to inject", func(t *testing.T) {
		handleHook(hookPayload(filepath.Join(plugin, "README.md")), &bytes.Buffer{})
		if out := string(injectOutput(injectPayload)); out != "{}" {
			t.Errorf("inject after non-manifest edit = %s, want {}", out)
		}
	})

	t.Run("Claude payload without conversationId does not relay", func(t *testing.T) {
		claude, _ := json.Marshal(map[string]string{"file_path": filepath.Join(plugin, "plugin.json")})
		handleHook(claude, &bytes.Buffer{})
		if out := string(injectOutput(injectPayload)); out != "{}" {
			t.Errorf("inject after Claude payload = %s, want {}", out)
		}
	})

	t.Run("unparsable inject payload yields {}", func(t *testing.T) {
		if out := string(injectOutput([]byte("not json"))); out != "{}" {
			t.Errorf("injectOutput(garbage) = %s, want {}", out)
		}
	})
}
