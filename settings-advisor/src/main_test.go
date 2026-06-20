package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestScanWorkspace(t *testing.T) {
	root := t.TempDir()

	writeFile := func(rel, body string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	writeFile("main.go", "package main\nfunc main() {}\n")        // go
	writeFile("app.ts", "export const x = 1\n")                  // ts
	writeFile(".env", "SECRET=1\n")                              // → HasEnv
	writeFile(".github/workflows/ci.yml", "on: push\n")         // → HasCI
	writeFile("config.production.json", "{}\n")                  // → HasProdConfig（トークン一致）
	writeFile("product.json", "{}\n")                            // 誤検知してはいけない
	writeFile("reproduce.yaml", "k: v\n")                        // 誤検知してはいけない
	writeFile("node_modules/pkg/index.js", "skip me\n")         // SkipDir 対象

	m, err := scanWorkspace(root)
	if err != nil {
		t.Fatalf("scanWorkspace error: %v", err)
	}

	if !m.HasEnv {
		t.Error("expected HasEnv=true (.env)")
	}
	if !m.HasCI {
		t.Error("expected HasCI=true (.github/workflows)")
	}
	if !m.HasProdConfig {
		t.Error("expected HasProdConfig=true (config.production.json)")
	}
	// node_modules はスキップされるので js は言語に含まれない。決定論的にソート済み。
	if want := []string{"go", "ts"}; !reflect.DeepEqual(m.Languages, want) {
		t.Errorf("Languages = %v, want %v (sorted, node_modules excluded)", m.Languages, want)
	}
}

func TestScanWorkspaceProdFalsePositive(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"product.json", "reproduce.yaml", "productivity.toml"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m, err := scanWorkspace(root)
	if err != nil {
		t.Fatal(err)
	}
	if m.HasProdConfig {
		t.Error("expected HasProdConfig=false for product/reproduce/productivity files")
	}
}

func TestGenerateRecommendations(t *testing.T) {
	// テスト用モデル定義
	modelCfg := ModelConfig{
		Models: []Model{
			{"gemini-3.5-flash-low", "Gemini 3.5 Flash (Low)", "light", []string{"fast", "cheap"}},
			{"gemini-3.5-flash-medium", "Gemini 3.5 Flash (Medium)", "light", []string{"fast", "balanced"}},
			{"gemini-3.5-flash-high", "Gemini 3.5 Flash (High)", "mid", []string{"accurate", "cost-effective"}},
			{"gemini-3.1-pro-low", "Gemini 3.1 Pro (Low)", "mid", []string{"accurate", "context-long"}},
			{"gemini-3.1-pro-high", "Gemini 3.1 Pro (High)", "heavy", []string{"most-accurate", "context-long"}},
			{"claude-sonnet-4.6", "Claude Sonnet 4.6 (Thinking)", "mid", []string{"instruction-following", "format-strict"}},
			{"claude-opus-4.6", "Claude Opus 4.6 (Thinking)", "heavy", []string{"deep-reasoning", "multi-perspective"}},
			{"gpt-oss-120b", "GPT-OSS 120B (Medium)", "mid", []string{"open-source", "quota-independent"}},
		},
	}

	// テストケース 1: 軽量コードベース
	t.Run("Light workspace", func(t *testing.T) {
		metrics := WorkspaceMetrics{
			TotalLines: 1000,
			Languages:  []string{"go"},
		}
		rec := generateRecommendations(metrics, modelCfg, "")

		if rec.ModelTier != "light" {
			t.Errorf("Expected tier 'light', got '%s'", rec.ModelTier)
		}
		if len(rec.SuggestedModels) == 0 {
			t.Fatal("Expected suggested models, got none")
		}
		found := false
		for _, m := range rec.SuggestedModels {
			if m == "Gemini 3.5 Flash (Medium)" || m == "Gemini 3.5 Flash (Low)" {
				found = true
			}
		}
		if !found {
			t.Errorf("Expected light models in suggestion, got %v", rec.SuggestedModels)
		}
	})

	// テストケース 2: 大規模コードベースまたは複雑なタスク
	t.Run("Heavy workspace or task", func(t *testing.T) {
		metrics := WorkspaceMetrics{
			TotalLines: 40000,
			Languages:  []string{"go", "typescript", "python"},
		}
		rec := generateRecommendations(metrics, modelCfg, "Refactoring architecture")

		if rec.ModelTier != "heavy" {
			t.Errorf("Expected tier 'heavy', got '%s'", rec.ModelTier)
		}
		foundPro := false
		foundOpus := false
		for _, m := range rec.SuggestedModels {
			if m == "Gemini 3.1 Pro (High)" {
				foundPro = true
			}
			if m == "Claude Opus 4.6 (Thinking)" {
				foundOpus = true
			}
		}
		if !foundPro && !foundOpus {
			t.Errorf("Expected heavy models in suggestion, got %v", rec.SuggestedModels)
		}
	})

	// テストケース 2b: 大規模・単一言語（task_hint なし）→ heavy
	// 旧 `||` ロジックでは len(Languages)<=2 が真のため誤って mid に落ちていた回帰防止。
	t.Run("Large monolingual workspace reaches heavy", func(t *testing.T) {
		metrics := WorkspaceMetrics{
			TotalLines: 100000,
			Languages:  []string{"go"},
		}
		rec := generateRecommendations(metrics, modelCfg, "")
		if rec.ModelTier != "heavy" {
			t.Errorf("Expected tier 'heavy' for large monolingual repo, got '%s'", rec.ModelTier)
		}
	})

	// テストケース 2c: 本番設定と CI が共存 → strict を優先
	t.Run("Prod config takes precedence over CI", func(t *testing.T) {
		metrics := WorkspaceMetrics{
			TotalLines:    1000,
			Languages:     []string{"go"},
			HasCI:         true,
			HasProdConfig: true,
		}
		rec := generateRecommendations(metrics, modelCfg, "")
		var perm string
		for _, s := range rec.Settings {
			if s.Key == "toolPermission" {
				perm, _ = s.Suggested.(string)
			}
		}
		if perm != "strict" {
			t.Errorf("Expected toolPermission='strict' when prod config present, got '%s'", perm)
		}
	})

	// テストケース 3: 特化キーワード (Sonnet)
	t.Run("Specialized task - Sonnet", func(t *testing.T) {
		metrics := WorkspaceMetrics{
			TotalLines: 10000,
			Languages:  []string{"go"},
		}
		rec := generateRecommendations(metrics, modelCfg, "手順に従って厳密にドキュメントを更新する")

		if rec.SuggestedModels[0] != "Claude Sonnet 4.6 (Thinking)" {
			t.Errorf("Expected Claude Sonnet 4.6 (Thinking) as first suggestion, got '%s'", rec.SuggestedModels[0])
		}
	})

	// テストケース 4: セキュリティ推奨
	t.Run("Security recommendations", func(t *testing.T) {
		metrics := WorkspaceMetrics{
			TotalLines: 1000,
			Languages:  []string{"go"},
			HasEnv:     true,
			HasCI:      true,
		}
		rec := generateRecommendations(metrics, modelCfg, "")

		hasSandbox := false
		hasPermission := false
		for _, s := range rec.Settings {
			if s.Key == "enableTerminalSandbox" && s.Suggested == true {
				hasSandbox = true
			}
			if s.Key == "toolPermission" && s.Suggested == "proceed-in-sandbox" {
				hasPermission = true
			}
		}

		if !hasSandbox {
			t.Error("Expected enableTerminalSandbox=true recommendation")
		}
		if !hasPermission {
			t.Error("Expected toolPermission=proceed-in-sandbox recommendation")
		}
	})
}
