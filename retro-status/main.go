package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// 各言語ごとの職業マッピング
var languageToClass = map[string]string{
	".go":   "ゴーレム (Golem)",
	".ts":   "時空の魔術師 (TS Mage)",
	".js":   "魔術師 (JS Mage)",
	".py":   "召喚士 (Py Summoner)",
	".rs":   "錬金術師 (Rust Alchemist)",
	".rb":   "吟遊詩人 (Ruby Bard)",
	".java": "聖騎士 (Java Paladin)",
	".kt":   "聖騎士 (Kotlin Paladin)",
	".cs":   "神官 (C# Cleric)",
	".cpp":  "戦士 (C++ Warrior)",
	".c":    "戦士 (C Warrior)",
	".html": "踊り子 (HTML Dancer)",
	".css":  "踊り子 (CSS Dancer)",
	".sh":   "シーフ (Shell Thief)",
}

// 拡張子ごとの武器マッピング
var extToWeapon = map[string]string{
	".go":   "Goの鋭いメス (Go Scalpel)",
	".ts":   "TSの魔導書 (TS Grimoire)",
	".js":   "JSの魔杖 (JS Wand)",
	".py":   "Pythonの鞭 (Python Whip)",
	".rs":   "Rustの鍛造ハンマー (Rust Hammer)",
	".rb":   "Rubyのバラッド (Ruby Ballad)",
	".java": "Javaの大盾 (Java Greatshield)",
	".cs":   "C#のルーンブレード (C# Rune Blade)",
	".cpp":  "C++の大剣 (C++ Greatsword)",
	".c":    "Cの直剣 (C Straight Sword)",
}

// devTool は「開発者が入れていそうな」CLI ツールと、RPG 風の表示名・効果。
type devTool struct {
	bin    string // exec.LookPath で探すバイナリ名
	name   string // 表示名（日本語 + 英名）
	effect string // 効果説明
}

// detectableTools はモダン開発 CLI の curated list（PATH に存在すれば持ち物に並ぶ）。
var detectableTools = []devTool{
	{"rg", "鷹の目 (rg)", "高速索敵"},
	{"fd", "韋駄天の靴 (fd)", "探索速度+"},
	{"bat", "灯火の書 (bat)", "視認性+"},
	{"eza", "千里眼の地図 (eza)", "全体把握"},
	{"jq", "賢者の宝珠 (jq)", "JSON錬成"},
	{"fzf", "選定の羅針盤 (fzf)", "瞬間選択"},
	{"gh", "ハブの紋章 (gh)", "遠隔交信"},
	{"delta", "差分の眼鏡 (delta)", "diff看破"},
	{"zoxide", "瞬間移動の靴 (zoxide)", "ワープ移動"},
	{"tmux", "多重影分身 (tmux)", "分身展開"},
	{"docker", "次元の箱舟 (docker)", "召喚"},
	{"kubectl", "艦隊の指揮杖 (kubectl)", "艦隊指揮"},
	{"nvim", "達人の刃 (nvim)", "高速編集"},
}

// detectTools は detectableTools のうち実行マシンの PATH 上に存在するものを返す。
// 対象はリポジトリではなく「サーバーを動かしているマシンの環境」である点に注意（演出用）。
func detectTools() []devTool {
	found := make([]devTool, 0, len(detectableTools))
	for _, t := range detectableTools {
		if _, err := exec.LookPath(t.bin); err == nil {
			found = append(found, t)
		}
	}
	return found
}

// stringWidth returns the visible width of a string.
// Assuming ASCII is 1 and others (Japanese, borders) are 2.
func stringWidth(s string) int {
	w := 0
	for _, r := range s {
		if r <= 127 {
			w += 1
		} else {
			w += 2
		}
	}
	return w
}

func padRight(s string, width int) string {
	sw := stringWidth(s)
	if sw >= width {
		return s
	}
	return s + strings.Repeat(" ", width-sw)
}

func drawLine(content string, width int) string {
	return "║ " + padRight(content, width-4) + " ║\n"
}

// ScanResult holds the raw scanned data.
type ScanResult struct {
	RepoName      string
	AuthorName    string
	TotalLoc      int
	FileCount     int
	TodoCount     int
	TestLoc       int
	LangLoc       map[string]int
	HasDocker     bool
	HasGitHubCI   bool
	HasGitLabCI   bool
	HasLinter     bool
	CommitCount   int
	RecentCommits int
	Tools         []devTool
}

// RPGStats holds computed RPG-themed values.
type RPGStats struct {
	RepoName   string   `json:"repo_name"`
	AuthorName string   `json:"author_name"`
	Class      string   `json:"class"`
	Level      int      `json:"level"`
	Depth      string   `json:"depth"`
	Monsters   int      `json:"monsters"`
	HP         int      `json:"hp"`
	MaxHP      int      `json:"max_hp"`
	MP         int      `json:"mp"`
	MaxMP      int      `json:"max_mp"`
	ATK        int      `json:"atk"`
	DEF        int      `json:"def"`
	SPD        int      `json:"spd"`
	Gold       int      `json:"gold"`
	Weapon     string   `json:"weapon"`
	Shield     string   `json:"shield"`
	Armor      string   `json:"armor"`
	Helm       string   `json:"helm"`
	Accessory  string          `json:"accessory"`
	Skills     []string        `json:"skills"`
	Inventory  []InventoryItem `json:"inventory"`
}

// InventoryItem は検出した開発 CLI を持ち物として表す（JSON 出力にも乗る）。
type InventoryItem struct {
	Name   string `json:"name"`
	Effect string `json:"effect"`
}

// Helper to run commands
func runCommand(ctx context.Context, dir string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

// scanRepository performs the scanning logic.
func scanRepository(ctx context.Context, repoPath string) (*ScanResult, error) {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, err
	}

	result := &ScanResult{
		LangLoc: make(map[string]int),
	}

	// 1. Git Info
	repoName, err := runCommand(ctx, absPath, "git", "config", "--get", "remote.origin.url")
	if err == nil && repoName != "" {
		parts := strings.Split(repoName, "/")
		if len(parts) >= 2 {
			result.RepoName = strings.TrimSuffix(parts[len(parts)-2]+"/"+parts[len(parts)-1], ".git")
		} else {
			result.RepoName = repoName
		}
	} else {
		result.RepoName = filepath.Base(absPath)
	}

	author, err := runCommand(ctx, absPath, "git", "log", "-1", "--format=%an")
	if err == nil && author != "" {
		result.AuthorName = author
	} else {
		result.AuthorName = "冒険者 (Hero)"
	}

	commitCountStr, err := runCommand(ctx, absPath, "git", "rev-list", "--count", "HEAD")
	if err == nil {
		fmt.Sscanf(commitCountStr, "%d", &result.CommitCount)
	}

	recentCommitStr, err := runCommand(ctx, absPath, "git", "log", "--since=30 days ago", "--oneline")
	if err == nil && recentCommitStr != "" {
		result.RecentCommits = len(strings.Split(strings.TrimSpace(recentCommitStr), "\n"))
	}

	// 2. 開発 CLI の検出（持ち物表示）。rg があれば TODO 数え方にも流用する。
	result.Tools = detectTools()
	hasRg := false
	for _, t := range result.Tools {
		if t.bin == "rg" {
			hasRg = true
			break
		}
	}

	if hasRg {
		todoStr, err := runCommand(ctx, absPath, "rg", "-i", `\btodo\b`, "--count")
		if err == nil {
			lines := strings.Split(todoStr, "\n")
			for _, line := range lines {
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					var count int
					if _, err := fmt.Sscanf(parts[len(parts)-1], "%d", &count); err == nil {
						result.TodoCount += count
					}
				}
			}
		}
	}

	// 3. File System Scan
	err = filepath.WalkDir(absPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" || name == "dist" || name == "build" || name == ".gemini" {
				return filepath.SkipDir
			}
			return nil
		}

		filename := d.Name()
		if filename == "Dockerfile" {
			result.HasDocker = true
		}
		if filename == ".gitlab-ci.yml" {
			result.HasGitLabCI = true
		}
		if strings.Contains(filename, "eslint") || strings.Contains(filename, "golangci-lint") {
			result.HasLinter = true
		}

		if strings.Contains(path, "/.github/workflows/") && (strings.HasSuffix(filename, ".yml") || strings.HasSuffix(filename, ".yaml")) {
			result.HasGitHubCI = true
		}

		ext := filepath.Ext(filename)
		isText := false
		switch ext {
		case ".go", ".ts", ".js", ".py", ".rs", ".rb", ".java", ".cpp", ".c", ".h", ".cs", ".html", ".css", ".sh", ".yml", ".yaml", ".json", ".md":
			isText = true
		}

		if isText {
			result.FileCount++
			file, err := os.Open(path)
			if err != nil {
				return nil
			}
			defer file.Close()

			loc := 0
			isTestFile := strings.Contains(filename, "_test.go") ||
				strings.Contains(filename, ".test.") ||
				strings.Contains(filename, "test_") ||
				strings.Contains(filename, ".spec.")

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				loc++
				line := scanner.Text()
				if !hasRg {
					if strings.Contains(strings.ToUpper(line), "TODO") {
						result.TodoCount++
					}
				}
			}

			result.TotalLoc += loc
			result.LangLoc[ext] += loc
			if isTestFile {
				result.TestLoc += loc
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// buildRPGStats calculates the RPG statistics.
func buildRPGStats(scan *ScanResult, repoPath string) *RPGStats {
	lv := (scan.CommitCount / 10) + (scan.TotalLoc / 1000)
	if lv < 1 {
		lv = 1
	}
	if lv > 99 {
		lv = 99
	}

	dominantLang := ""
	maxLoc := -1
	for ext, loc := range scan.LangLoc {
		if loc > maxLoc {
			maxLoc = loc
			dominantLang = ext
		}
	}

	class := "冒険者 (Adventurer)"
	if c, ok := languageToClass[dominantLang]; ok {
		class = c
	}

	depthF := scan.TotalLoc / 100
	depthSteps := scan.TotalLoc % 100
	depth := fmt.Sprintf("地下 %d 階 (B%dF) ＋ %d 歩", depthF, depthF, depthSteps)

	maxHP := 100 + (scan.TotalLoc / 50)
	depCount := 5
	if scan.HasDocker {
		depCount += 3
	}
	if scan.HasGitHubCI || scan.HasGitLabCI {
		depCount += 4
	}
	maxMP := depCount * 10

	atk := scan.RecentCommits * 3 + 10
	if atk > 999 {
		atk = 999
	}

	def := (scan.TestLoc / 50) + 10
	if scan.HasGitHubCI || scan.HasGitLabCI {
		def += 30
	}
	if scan.HasLinter {
		def += 15
	}
	if def > 999 {
		def = 999
	}

	spd := 100 - (scan.FileCount / 10)
	if spd < 5 {
		spd = 5
	}

	gold := scan.CommitCount * 10

	weapon := "素手 (Bare Hands)"
	if w, ok := extToWeapon[dominantLang]; ok {
		weapon = w
	}

	shield := "皮の盾 (Leather Shield)"
	if _, err := os.Stat(filepath.Join(repoPath, "go.sum")); err == nil {
		shield = "go.sumの円盾 (Module Buckler)"
	} else if _, err := os.Stat(filepath.Join(repoPath, "package-lock.json")); err == nil {
		shield = "npmの守護盾 (Lock Shield)"
	} else if _, err := os.Stat(filepath.Join(repoPath, "yarn.lock")); err == nil {
		shield = "yarnの糸盾 (Yarn Shield)"
	} else if _, err := os.Stat(filepath.Join(repoPath, "Cargo.lock")); err == nil {
		shield = "Cargoの鉄盾 (Cargo Shield)"
	}

	armor := "旅人の服 (Traveler's Garb)"
	if scan.HasLinter {
		armor = "静的解析の魔導鎧 (Linter Plate)"
	} else if scan.TestLoc > 0 {
		armor = "テストコードの鎖帷子 (Test Chainmail)"
	}

	helm := "なべのふた (Pot Lid)"
	if scan.HasGitHubCI {
		helm = "Actionsの王冠 (Actions Crown)"
	} else if scan.HasGitLabCI {
		helm = "GitLab-CIの星兜 (GitLab Star Helm)"
	}

	accessory := "なし (None)"
	if scan.HasDocker {
		accessory = "コンテナの指輪 (Docker Ring)"
	}

	var skills []string
	if scan.TestLoc > 0 {
		skills = append(skills, "* ベホマ (Auto-Test): 自己修復能力")
	}
	if scan.HasGitHubCI || scan.HasGitLabCI {
		skills = append(skills, "* ルーラ (Auto-Deploy): 自動飛行転送")
	}
	if scan.HasDocker {
		skills = append(skills, "* バシルーラ (Containerize): 敵を別次元に隔離")
	}
	if scan.RecentCommits > 30 {
		skills = append(skills, "* バイキルト (Hyper-Commit): 怒涛の連続攻撃")
	}
	if len(skills) == 0 {
		skills = append(skills, "* なし (No special skill)")
	}

	inventory := make([]InventoryItem, 0, len(scan.Tools))
	for _, t := range scan.Tools {
		inventory = append(inventory, InventoryItem{Name: t.name, Effect: t.effect})
	}

	return &RPGStats{
		RepoName:   scan.RepoName,
		AuthorName: scan.AuthorName,
		Class:      class,
		Level:      lv,
		Depth:      depth,
		Monsters:   scan.TodoCount,
		HP:         maxHP,
		MaxHP:      maxHP,
		MP:         maxMP,
		MaxMP:      maxMP,
		ATK:        atk,
		DEF:        def,
		SPD:        spd,
		Gold:       gold,
		Weapon:     weapon,
		Shield:     shield,
		Armor:      armor,
		Helm:       helm,
		Accessory:  accessory,
		Skills:     skills,
		Inventory:  inventory,
	}
}

// renderAA outputs the status screen as Famicom style ASCII art.
func renderAA(stats *RPGStats) string {
	const boxWidth = 54
	var buf bytes.Buffer
	buf.WriteString("╔" + strings.Repeat("═", boxWidth-2) + "╗\n")
	buf.WriteString(drawLine("STATUS: "+strings.ToUpper(stats.RepoName), boxWidth))
	buf.WriteString("╠" + strings.Repeat("═", boxWidth-2) + "╣\n")
	buf.WriteString(drawLine(fmt.Sprintf("NAME  : %s", stats.AuthorName), boxWidth))
	buf.WriteString(drawLine(fmt.Sprintf("CLASS : %s", stats.Class), boxWidth))
	buf.WriteString(drawLine(fmt.Sprintf("LEVEL : %d (LV%d)", stats.Level, stats.Level), boxWidth))
	buf.WriteString(drawLine(fmt.Sprintf("DEPTH : %s", stats.Depth), boxWidth))
	buf.WriteString(drawLine(fmt.Sprintf("MONSTERS: %d (残りTODO数)", stats.Monsters), boxWidth))
	buf.WriteString("╟" + strings.Repeat("─", boxWidth-2) + "╢\n")
	buf.WriteString(drawLine(fmt.Sprintf("HP    : %5d / %5d    MP : %5d / %5d", stats.HP, stats.MaxHP, stats.MP, stats.MaxMP), boxWidth))
	buf.WriteString(drawLine(fmt.Sprintf("ATK   : %5d           DEF: %5d", stats.ATK, stats.DEF), boxWidth))
	buf.WriteString(drawLine(fmt.Sprintf("SPD   : %5d           GOLD: %d G", stats.SPD, stats.Gold), boxWidth))
	buf.WriteString("╠" + strings.Repeat("═", boxWidth-2) + "╣\n")
	buf.WriteString(drawLine("[EQUIPMENT]", boxWidth))
	buf.WriteString(drawLine(fmt.Sprintf("WEAPON: %s", stats.Weapon), boxWidth))
	buf.WriteString(drawLine(fmt.Sprintf("SHIELD: %s", stats.Shield), boxWidth))
	buf.WriteString(drawLine(fmt.Sprintf("ARMOR : %s", stats.Armor), boxWidth))
	buf.WriteString(drawLine(fmt.Sprintf("HELM  : %s", stats.Helm), boxWidth))
	buf.WriteString(drawLine(fmt.Sprintf("ACCESS: %s", stats.Accessory), boxWidth))
	buf.WriteString("╠" + strings.Repeat("═", boxWidth-2) + "╣\n")
	buf.WriteString(drawLine("[INVENTORY / 開発者の秘宝]", boxWidth))
	if len(stats.Inventory) == 0 {
		buf.WriteString(drawLine("なし (手ぶら)", boxWidth))
	} else {
		// 名前幅をそろえてコロンを縦に整列させる
		nameW := 0
		for _, it := range stats.Inventory {
			if w := stringWidth(it.Name); w > nameW {
				nameW = w
			}
		}
		for _, it := range stats.Inventory {
			buf.WriteString(drawLine(fmt.Sprintf("%s : %s", padRight(it.Name, nameW), it.Effect), boxWidth))
		}
	}
	buf.WriteString("╠" + strings.Repeat("═", boxWidth-2) + "╣\n")
	buf.WriteString(drawLine("[SPELLS & SKILLS]", boxWidth))
	for _, sk := range stats.Skills {
		buf.WriteString(drawLine(sk, boxWidth))
	}
	buf.WriteString("╚" + strings.Repeat("═", boxWidth-2) + "╝\n")

	return buf.String()
}

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--standalone" || os.Args[1] == "-s") {
		repoPath := "."
		if len(os.Args) > 2 {
			repoPath = os.Args[2]
		}
		ctx := context.Background()
		scan, err := scanRepository(ctx, repoPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error scanning repository: %v\n", err)
			os.Exit(1)
		}
		stats := buildRPGStats(scan, repoPath)
		fmt.Print(renderAA(stats))
		return
	}

	s := server.NewMCPServer(
		"retro-status",
		"1.0.0",
	)

	retroStatusTool := mcp.NewTool("retro_status",
		mcp.WithDescription("指定したパスのリポジトリをスキャンし、RPGのレトロステータス画面風のAA（アスキーアート）で出力します。"),
		mcp.WithString("path", mcp.Description("スキャンするリポジトリのパス。指定がない場合はカレントディレクトリ。")),
		mcp.WithString("format", mcp.Description("出力フォーマット。'text' (アスキーアート) または 'json' を選択できます。デフォルトは 'text'。")),
	)

	s.AddTool(retroStatusTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		argsMap, ok := request.Params.Arguments.(map[string]any)
		if !ok {
			return mcp.NewToolResultError("arguments must be a map"), nil
		}

		repoPath := "."
		if p, ok := argsMap["path"].(string); ok && p != "" {
			repoPath = p
		}

		format := "text"
		if f, ok := argsMap["format"].(string); ok && f != "" {
			format = f
		}

		scan, err := scanRepository(ctx, repoPath)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Scanning failed: %v", err)), nil
		}

		stats := buildRPGStats(scan, repoPath)

		if format == "json" {
			jsonData, err := json.MarshalIndent(stats, "", "  ")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal JSON: %v", err)), nil
			}
			return mcp.NewToolResultText(string(jsonData)), nil
		}

		// Text (ASCII Art) output
		return mcp.NewToolResultText(renderAA(stats)), nil
	})

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
