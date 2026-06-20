package main

import (
	"testing"
)

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
