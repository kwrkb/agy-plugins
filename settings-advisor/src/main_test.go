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

	writeFile("main.go", "package main\nfunc main() {}\n") // go
	writeFile("app.ts", "export const x = 1\n")            // ts
	writeFile("lib.dart", "void main() {}\n")              // dart
	writeFile("App.swift", "import Foundation\n")          // swift
	writeFile(".env", "SECRET=1\n")                        // → HasEnv
	writeFile(".github/workflows/ci.yml", "on: push\n")    // → HasCI
	writeFile("config.production.json", "{}\n")            // → HasProdConfig（トークン一致）
	writeFile("product.json", "{}\n")                      // 誤検知してはいけない
	writeFile("reproduce.yaml", "k: v\n")                  // 誤検知してはいけない
	writeFile("node_modules/pkg/index.js", "skip me\n")    // SkipDir 対象

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
	if want := []string{"dart", "go", "swift", "ts"}; !reflect.DeepEqual(m.Languages, want) {
		t.Errorf("Languages = %v, want %v (sorted, node_modules excluded)", m.Languages, want)
	}
}

func TestScanWorkspaceCIAndProdDetection(t *testing.T) {
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

	writeFile(".gitlab-ci.yml", "stages:\n  - test\n")
	writeFile("config/prod/settings.json", "{}\n") // ディレクトリ名による prod 検出

	m, err := scanWorkspace(root)
	if err != nil {
		t.Fatalf("scanWorkspace error: %v", err)
	}

	if !m.HasCI {
		t.Error("expected HasCI=true (.gitlab-ci.yml)")
	}
	if !m.HasProdConfig {
		t.Error("expected HasProdConfig=true (config/prod/settings.json)")
	}
}

func TestScanWorkspaceEnvFilter(t *testing.T) {
	t.Run("Env templates excluded", func(t *testing.T) {
		root := t.TempDir()
		for _, name := range []string{".env.example", ".env.sample", ".env.template", ".env.dist"} {
			if err := os.WriteFile(filepath.Join(root, name), []byte("A=B\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		m, err := scanWorkspace(root)
		if err != nil {
			t.Fatal(err)
		}
		if m.HasEnv {
			t.Error("expected HasEnv=false when only .env template/sample files exist")
		}
	})

	t.Run("Actual env included", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, ".env.production"), []byte("SECRET=1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		m, err := scanWorkspace(root)
		if err != nil {
			t.Fatal(err)
		}
		if !m.HasEnv {
			t.Error("expected HasEnv=true for .env.production")
		}
	})
}

func TestScanWorkspaceSkipDirs(t *testing.T) {
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

	writeFile("main.go", "package main\n")
	writeFile("target/debug/app.rs", "fn main() {}\n")
	writeFile(".next/static/app.js", "console.log(1)\n")
	writeFile(".dart_tool/package_config.json", "{}\n")
	writeFile("Pods/SomeLib/lib.swift", "import UIKit\n")

	m, err := scanWorkspace(root)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"go"}; !reflect.DeepEqual(m.Languages, want) {
		t.Errorf("Languages = %v, want %v (build artifact directories should be skipped)", m.Languages, want)
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
			{"gemini-3.8-flash-low", "Gemini 3.8 Flash (Low)", "light", []string{"fast", "cheap"}},
			{"gemini-3.8-flash-medium", "Gemini 3.8 Flash (Medium)", "light", []string{"fast", "balanced"}},
			{"gemini-3.8-flash-high", "Gemini 3.8 Flash (High)", "mid", []string{"accurate", "cost-effective"}},
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
			if m == "Gemini 3.8 Flash (Medium)" || m == "Gemini 3.8 Flash (Low)" {
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

	// テストケース 3: 特化キーワード (instruction-following trait -> Sonnet)
	t.Run("Specialized task - Sonnet via trait", func(t *testing.T) {
		metrics := WorkspaceMetrics{
			TotalLines: 10000,
			Languages:  []string{"go"},
		}
		rec := generateRecommendations(metrics, modelCfg, "手順に従って厳密にドキュメントを更新する")

		if rec.SuggestedModels[0] != "Claude Sonnet 4.6 (Thinking)" {
			t.Errorf("Expected Claude Sonnet 4.6 (Thinking) as first suggestion, got '%s'", rec.SuggestedModels[0])
		}
	})

	// テストケース 3b: 特化キーワード (quota-independent trait -> GPT-OSS)
	t.Run("Specialized task - GPT-OSS via trait", func(t *testing.T) {
		metrics := WorkspaceMetrics{
			TotalLines: 10000,
			Languages:  []string{"go"},
		}
		rec := generateRecommendations(metrics, modelCfg, "quota枯渇時のフォールバックとして実行")

		if rec.SuggestedModels[0] != "GPT-OSS 120B (Medium)" {
			t.Errorf("Expected GPT-OSS 120B (Medium) as first suggestion, got '%s'", rec.SuggestedModels[0])
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

// TestScanWorkspaceRootNamedSkipDir は、利用者が明示指定したルート自身が
// skipDirs に一致する名前（target / out / Pods 等）でも走査されることを確認する。
func TestScanWorkspaceRootNamedSkipDir(t *testing.T) {
	for _, name := range []string{"target", "out", "Pods", "build", "dist"} {
		t.Run(name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), name)
			if err := os.MkdirAll(root, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
				t.Fatal(err)
			}

			m, err := scanWorkspace(root)
			if err != nil {
				t.Fatalf("scanWorkspace error: %v", err)
			}
			if want := []string{"go"}; !reflect.DeepEqual(m.Languages, want) {
				t.Errorf("Languages = %v, want %v (explicitly requested root must not be skipped)", m.Languages, want)
			}
			if m.TotalLines == 0 {
				t.Error("TotalLines = 0, want > 0 (explicitly requested root must not be skipped)")
			}
		})
	}
}

// TestScanWorkspaceCINearMissDirs は、CI ディレクトリの近似名が部分文字列一致で
// HasCI を立てないことを確認する（パス成分単位の比較）。
func TestScanWorkspaceCINearMissDirs(t *testing.T) {
	root := t.TempDir()
	writeFile := func(rel, body string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	writeFile(".circleci-disabled/job.yml", "jobs: {}\n")
	writeFile(".gitlab/circle/config.yml", "x: 1\n")
	writeFile(".github/workflows-archive/old.yml", "on: push\n")
	writeFile(".gitlab/ci-templates/base.yml", "x: 1\n")
	writeFile("docs/circleci.md", "notes\n")

	m, err := scanWorkspace(root)
	if err != nil {
		t.Fatalf("scanWorkspace error: %v", err)
	}
	if m.HasCI {
		t.Error("expected HasCI=false for directories that only resemble CI directory names")
	}
}

// TestScanWorkspaceCIComponentMatch は、成分単位比較にしても本来の CI ディレクトリを
// 取りこぼさないことを確認する（ネストした配置を含む）。
func TestScanWorkspaceCIComponentMatch(t *testing.T) {
	cases := map[string]string{
		".github/workflows/ci.yml":        "on: push\n",
		".gitlab/ci/build.yml":            "x: 1\n",
		".circleci/config.yml":            "version: 2.1\n",
		"sub/.github/workflows/build.yml": "on: push\n",
		"sub/.circleci/config.yml":        "version: 2.1\n",
	}
	for rel, body := range cases {
		t.Run(rel, func(t *testing.T) {
			root := t.TempDir()
			p := filepath.Join(root, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}

			m, err := scanWorkspace(root)
			if err != nil {
				t.Fatalf("scanWorkspace error: %v", err)
			}
			if !m.HasCI {
				t.Errorf("expected HasCI=true for %s", rel)
			}
		})
	}
}

// TestScanWorkspaceAncestorOutsideRoot は、スキャンルートより上位（ワークスペース外）の
// ディレクトリ名が prod / CI 判定に混入しないことを確認する。
func TestScanWorkspaceAncestorOutsideRoot(t *testing.T) {
	for _, ancestor := range []string{"production", "prod", ".circleci"} {
		t.Run(ancestor, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), ancestor, "repos", "app")
			if err := os.MkdirAll(root, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "settings.json"), []byte("{}\n"), 0o644); err != nil {
				t.Fatal(err)
			}

			m, err := scanWorkspace(root)
			if err != nil {
				t.Fatalf("scanWorkspace error: %v", err)
			}
			if m.HasProdConfig {
				t.Errorf("expected HasProdConfig=false: %q ancestor is outside the scan root", ancestor)
			}
			if m.HasCI {
				t.Errorf("expected HasCI=false: %q ancestor is outside the scan root", ancestor)
			}
		})
	}
}

// TestIsEnvFileQualifiedTemplates は修飾付きテンプレート（.env.production.example 等）を
// 除外しつつ、実値を持つ .env.production / .env.test.local を env ファイルとして残すことを確認する。
func TestIsEnvFileQualifiedTemplates(t *testing.T) {
	cases := map[string]bool{
		".env":                    true,
		".env.production":         true,
		".env.local":              true,
		".env.test.local":         true,
		".env.production.local":   true,
		".env.example":            false,
		".env.sample":             false,
		".env.template":           false,
		".env.dist":               false,
		".env.test":               false,
		".env.defaults":           false,
		".env.schema":             false,
		".env.production.example": false,
		".env.local.sample":       false,
		".env.staging.template":   false,
		".env.production.dist":    false,
		"env.production":          false, // 先頭ドット無しは対象外
		"main.go":                 false,
	}
	for name, want := range cases {
		if got := isEnvFile(name); got != want {
			t.Errorf("isEnvFile(%q) = %v, want %v", name, got, want)
		}
	}
}

// TestScanWorkspaceQualifiedEnvTemplates は修飾付きテンプレートだけのワークスペースで
// HasEnv が立たない（＝不要な sandbox 推奨が出ない）ことを確認する。
func TestScanWorkspaceQualifiedEnvTemplates(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{".env.production.example", ".env.local.sample", ".env.staging.template"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("A=B\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m, err := scanWorkspace(root)
	if err != nil {
		t.Fatal(err)
	}
	if m.HasEnv {
		t.Error("expected HasEnv=false for qualified env templates (.env.<name>.example etc.)")
	}
}
