package main

import (
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
