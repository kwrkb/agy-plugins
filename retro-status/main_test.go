package main

import (
	"strings"
	"testing"
)

func TestStringWidth(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"hello", 5},
		{"こんにちは", 10},
		{"helloこんにちは", 15},
		{"║", 2},
		{"STATUS: TEST", 12},
	}

	for _, test := range tests {
		actual := stringWidth(test.input)
		if actual != test.expected {
			t.Errorf("stringWidth(%q) = %d; expected %d", test.input, actual, test.expected)
		}
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		input    string
		width    int
		expected string
	}{
		{"hello", 10, "hello     "},
		{"こんにちは", 12, "こんにちは  "},
		{"too long string", 5, "too long string"},
	}

	for _, test := range tests {
		actual := padRight(test.input, test.width)
		if actual != test.expected {
			t.Errorf("padRight(%q, %d) = %q; expected %q", test.input, test.width, actual, test.expected)
		}
	}
}

func TestBuildRPGStats(t *testing.T) {
	scan := &ScanResult{
		RepoName:      "test-repo",
		AuthorName:    "Tester",
		TotalLoc:      1520,
		FileCount:     25,
		TodoCount:     7,
		TestLoc:       300,
		LangLoc:       map[string]int{".go": 1200, ".yml": 320},
		HasDocker:     true,
		HasGitHubCI:   true,
		HasGitLabCI:   false,
		HasLinter:     true,
		CommitCount:   150,
		RecentCommits: 45,
	}

	stats := buildRPGStats(scan, ".")

	if stats.RepoName != "test-repo" {
		t.Errorf("expected repo name 'test-repo', got %q", stats.RepoName)
	}
	if stats.Level != 16 { // (150/10) + (1520/1000) = 15 + 1 = 16
		t.Errorf("expected level 16, got %d", stats.Level)
	}
	if stats.Class != "ゴーレム (Golem)" {
		t.Errorf("expected Class 'ゴーレム (Golem)', got %q", stats.Class)
	}
	if stats.Monsters != 7 {
		t.Errorf("expected Monsters 7, got %d", stats.Monsters)
	}
	if stats.MaxHP != 130 { // 100 + 1520/50 = 100 + 30 = 130
		t.Errorf("expected MaxHP 130, got %d", stats.MaxHP)
	}
	if stats.MaxMP != 120 { // 5 (default) + 3 (Docker) + 4 (GitHubCI) = 12 * 10 = 120
		t.Errorf("expected MaxMP 120, got %d", stats.MaxMP)
	}
	if stats.Gold != 1500 {
		t.Errorf("expected Gold 1500, got %d", stats.Gold)
	}
	if stats.Weapon != "Goの鋭いメス (Go Scalpel)" {
		t.Errorf("expected Weapon 'Goの鋭いメス (Go Scalpel)', got %q", stats.Weapon)
	}
}

// buildRPGStats は scan.Tools を持ち物（Inventory）へ順序どおり写すこと。
func TestBuildRPGStatsInventory(t *testing.T) {
	scan := &ScanResult{
		LangLoc: map[string]int{".go": 100},
		Tools: []devTool{
			{"rg", "鷹の目 (rg)", "高速索敵"},
			{"jq", "賢者の宝珠 (jq)", "JSON錬成"},
		},
	}
	stats := buildRPGStats(scan, ".")
	if len(stats.Inventory) != 2 {
		t.Fatalf("expected 2 inventory items, got %d", len(stats.Inventory))
	}
	if stats.Inventory[0].Name != "鷹の目 (rg)" || stats.Inventory[0].Effect != "高速索敵" {
		t.Errorf("unexpected first item: %+v", stats.Inventory[0])
	}
	if stats.Inventory[1].Name != "賢者の宝珠 (jq)" {
		t.Errorf("unexpected second item: %+v", stats.Inventory[1])
	}
}

// 持ち物が空でも renderAA が落ちず「手ぶら」を出すこと。
func TestRenderAAEmptyInventory(t *testing.T) {
	stats := buildRPGStats(&ScanResult{LangLoc: map[string]int{}}, ".")
	out := renderAA(stats)
	if !strings.Contains(out, "[INVENTORY / 開発者の秘宝]") {
		t.Error("INVENTORY section missing")
	}
	if !strings.Contains(out, "なし (手ぶら)") {
		t.Error("empty inventory should render 手ぶら")
	}
}

// detectTools は curated list に無いバイナリを返さない（PATH 依存なので存在判定はしない）。
func TestDetectToolsSubset(t *testing.T) {
	known := make(map[string]bool, len(detectableTools))
	for _, t := range detectableTools {
		known[t.bin] = true
	}
	for _, got := range detectTools() {
		if !known[got.bin] {
			t.Errorf("detectTools returned unknown bin %q", got.bin)
		}
	}
}
