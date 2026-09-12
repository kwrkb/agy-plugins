package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Model は models.json で定義される各モデルの構造
type Model struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Tier   string   `json:"tier"`
	Traits []string `json:"traits"`
}

// ModelConfig は models.json のルート構造
type ModelConfig struct {
	Models []Model `json:"models"`
}

// WorkspaceMetrics はスキャン結果のメトリクス
type WorkspaceMetrics struct {
	TotalLines    int      `json:"total_lines"`
	Languages     []string `json:"languages"`
	HasEnv        bool     `json:"has_env"`
	HasCI         bool     `json:"has_ci"`
	HasProdConfig bool     `json:"has_prod_config"`
}

var targetExts = map[string]bool{
	".go":     true,
	".ts":     true,
	".js":     true,
	".py":     true,
	".rs":     true,
	".rb":     true,
	".java":   true,
	".kt":     true,
	".cs":     true,
	".cpp":    true,
	".c":      true,
	".html":   true,
	".css":    true,
	".sh":     true,
	".dart":   true,
	".swift":  true,
	".vue":    true,
	".svelte": true,
	".php":    true,
	".scala":  true,
	".zig":    true,
	".lua":    true,
	".sql":    true,
	".tf":     true,
}

var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	".venv":        true,
	"dist":         true,
	"build":        true,
	"target":       true,
	".next":        true,
	".nuxt":        true,
	"out":          true,
	".dart_tool":   true,
	"Pods":         true,
}

func main() {
	// コマンドライン引数（デバッグ用）
	standalone := flag.Bool("standalone", false, "Run in standalone mode for debugging")
	flag.Parse()

	if *standalone {
		runStandalone()
		return
	}

	// MCPサーバーの起動
	s := server.NewMCPServer(
		"settings-advisor",
		"1.0.0",
	)

	tool := mcp.NewTool("settings_advisor",
		mcp.WithDescription("Analyze workspace scale, languages, and task characteristics to recommend optimal agy settings (models, sandbox, permissions) in a non-intrusive way."),
		mcp.WithString("path", mcp.Description("Path to the repository to analyze (default: '.')")),
		mcp.WithString("task_hint", mcp.Description("Brief description of the planned task (e.g. 'large refactoring') to refine recommendations")),
		mcp.WithString("format", mcp.Description("Output format: 'text' or 'json' (default: 'text')")),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		argsMap, _ := request.Params.Arguments.(map[string]any)
		if argsMap == nil {
			argsMap = map[string]any{}
		}

		path := "."
		if p, ok := argsMap["path"].(string); ok && p != "" {
			path = p
		}

		taskHint := ""
		if h, ok := argsMap["task_hint"].(string); ok {
			taskHint = h
		}

		format := "text"
		if f, ok := argsMap["format"].(string); ok && f != "" {
			format = f
		}

		// 1. ワークスペースのスキャン
		metrics, err := scanWorkspace(path)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to scan workspace: %v", err)), nil
		}

		// 2. models.json のロード
		modelCfg, err := loadModelsConfig()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to load models.json: %v", err)), nil
		}

		// 3. 推奨事項の決定
		recommendations := generateRecommendations(metrics, modelCfg, taskHint)

		// 4. フォーマット出力
		if format == "json" {
			outBytes, err := json.MarshalIndent(recommendations, "", "  ")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to format JSON: %v", err)), nil
			}
			return mcp.NewToolResultText(string(outBytes)), nil
		}

		// textフォーマット（コンパクト2〜3行）
		responseText := formatTextOutput(metrics, recommendations)
		return mcp.NewToolResultText(responseText), nil
	})

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
		os.Exit(1)
	}
}

// scanWorkspace は指定されたパス配下のメトリクスを収集する
func scanWorkspace(root string) (WorkspaceMetrics, error) {
	var metrics WorkspaceMetrics
	langMap := make(map[string]bool)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if path == root {
				return err // ルートパス自体のエラー（存在しない/権限なし）は呼び出し元に伝播
			}
			return nil // 走査中の個別エラーは無視して続行
		}

		if info.IsDir() {
			// ルート自身は利用者が明示指定したパスなので、名前が skipDirs に
			// 一致しても走査する（例: "target" というリポジトリを直接指定）。
			if path != root && skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		fileName := strings.ToLower(info.Name())

		// .env 検知（テンプレート・サンプルは除外）
		if isEnvFile(fileName) {
			metrics.HasEnv = true
		}

		// パス由来の判定は root からの相対パスのみを見る。絶対パスを使うと
		// ワークスペース外の祖先ディレクトリ名（例: /mnt/production/repos/app,
		// ~/.circleci/repos/app）が誤検知の原因になる。
		dirParts, relOK := relDirComponents(root, path)

		// CI/CD 検知（GitHub Actions, GitLab CI, CircleCI, Bitbucket, Azure）
		if isCIPath(dirParts, relOK, fileName) {
			metrics.HasCI = true
		}

		ext := filepath.Ext(fileName)

		// 本番設定ファイル検知。"product.json" / "reproduce.yaml" 等の誤検知を避けるため、
		// 拡張子を除いたファイル名およびパス中のディレクトリ名を区切り（. - _）でトークン化し
		// "prod"/"production" 単独一致のみ採る。
		if ext == ".json" || ext == ".yaml" || ext == ".yml" || ext == ".toml" {
			base := strings.TrimSuffix(fileName, ext)
			if hasProdToken(base) {
				metrics.HasProdConfig = true
			} else if relOK {
				for _, part := range dirParts {
					if hasProdToken(strings.ToLower(part)) {
						metrics.HasProdConfig = true
						break
					}
				}
			}
		}

		// ターゲット言語の行数カウント
		if targetExts[ext] {
			langMap[ext] = true
			lines, err := countLines(path)
			if err == nil {
				metrics.TotalLines += lines
			}
		}

		return nil
	})

	for lang := range langMap {
		metrics.Languages = append(metrics.Languages, strings.TrimPrefix(lang, "."))
	}
	// map 由来の順序揺れを除き、出力・推奨理由を決定論的にする。
	sort.Strings(metrics.Languages)

	return metrics, err
}

// envTemplateMarkers は「中身がダミーのテンプレート」を示す末尾トークン。
var envTemplateMarkers = map[string]bool{
	"example":  true,
	"sample":   true,
	"template": true,
	"dist":     true,
	"test":     true,
	"defaults": true,
	"schema":   true,
}

// isEnvFile は実値が入った .env ファイルであるかを判定する（.env.example 等のテンプレートは除外）。
// 判定は接尾辞の完全一致ではなく**末尾トークン**で行う。`.env.production.example` や
// `.env.local.sample` のような修飾付きテンプレートを取りこぼさず、かつ `.env.production`
// や `.env.test.local`（実値を持つ）は env ファイルとして残すため。
func isEnvFile(fileName string) bool {
	if fileName == ".env" {
		return true
	}
	if strings.HasPrefix(fileName, ".env.") {
		suffix := strings.TrimPrefix(fileName, ".env.")
		parts := strings.Split(suffix, ".")
		return !envTemplateMarkers[parts[len(parts)-1]]
	}
	return false
}

// relDirComponents は root から path までの相対パスの「ディレクトリ部分」を
// パス区切りで分解して返す。path が root 配下でない、または相対化できない場合は
// ok=false を返し、呼び出し側はパス由来の判定を行わない。
func relDirComponents(root, path string) ([]string, bool) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return nil, false
	}
	slashRel := filepath.ToSlash(rel)
	if slashRel == ".." || strings.HasPrefix(slashRel, "../") {
		return nil, false
	}
	dir := filepath.ToSlash(filepath.Dir(slashRel))
	if dir == "." || dir == "" {
		return nil, true // root 直下のファイル（ディレクトリ成分なし）
	}
	return strings.Split(dir, "/"), true
}

// isCIPath は CI/CD 関連のパスまたは設定ファイルであるかを判定する。
// dirParts は root からの相対ディレクトリ成分。部分文字列ではなくパス成分単位で
// 比較し、".circleci-disabled/" や ".gitlab/circle/" のような近似名を弾く。
func isCIPath(dirParts []string, relOK bool, fileName string) bool {
	if fileName == ".gitlab-ci.yml" ||
		fileName == "bitbucket-pipelines.yml" ||
		fileName == "azure-pipelines.yml" {
		return true
	}
	if !relOK {
		return false
	}
	for i, part := range dirParts {
		if part == ".circleci" {
			return true
		}
		if i+1 < len(dirParts) {
			next := dirParts[i+1]
			if (part == ".github" && next == "workflows") ||
				(part == ".gitlab" && next == "ci") {
				return true
			}
		}
	}
	return false
}

// hasProdToken はトークン分割した名前に prod / production が含まれるかを判定する。
func hasProdToken(name string) bool {
	for _, tok := range strings.FieldsFunc(name, isNameSeparator) {
		if tok == "prod" || tok == "production" {
			return true
		}
	}
	return false
}

// isNameSeparator はファイル名・ディレクトリ名トークン分割に使う区切り文字を判定する。
func isNameSeparator(r rune) bool {
	return r == '.' || r == '-' || r == '_'
}

func countLines(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	// 1MBを超えるファイルは行数カウントをスキップ
	info, err := file.Stat()
	if err == nil && info.Size() > 1024*1024 {
		return 0, nil
	}

	// bufio.Scanner は既定 64KB の行長制限があり、ミニファイ JS/巨大 JSON 等の長い行で
	// ErrTooLong により途中停止する。ReadSlice はバッファ満杯を継続扱いにでき、行長制限なしで
	// 安全かつ低割り当てに行数をカウントできる。
	r := bufio.NewReader(file)
	count := 0
	for {
		line, err := r.ReadSlice('\n')
		if err == nil {
			count++
			continue
		}
		if err == bufio.ErrBufferFull {
			continue // 1行がバッファを超過。改行はまだなのでカウントせず継続
		}
		if err == io.EOF {
			// 末尾に改行が無い最終行を取りこぼさない（Scanner と同じ数え方）
			if len(line) > 0 {
				count++
			}
			break
		}
		return count, err
	}
	return count, nil
}

func loadModelsConfig() (ModelConfig, error) {
	var cfg ModelConfig

	// 実行バイナリからの相対パス解決を試みる
	exePath, err := os.Executable()
	var searchPaths []string
	if err == nil {
		// dispatcher 等の symlink 経由起動でも実体から相対解決できるよう実パスに正規化
		if evalPath, evalErr := filepath.EvalSymlinks(exePath); evalErr == nil {
			exePath = evalPath
		}
		// bin/settings-advisor の親の親に models.json がある想定
		pluginRoot := filepath.Dir(filepath.Dir(exePath))
		searchPaths = append(searchPaths, filepath.Join(pluginRoot, "models.json"))
	}
	// カレントディレクトリ、およびテスト実行時用
	searchPaths = append(searchPaths, "models.json", "../models.json", "../../models.json")

	var content []byte
	var loaded bool
	for _, p := range searchPaths {
		data, err := os.ReadFile(p)
		if err == nil {
			content = data
			loaded = true
			break
		}
	}

	if !loaded {
		return cfg, fmt.Errorf("could not find models.json in search paths")
	}

	err = json.Unmarshal(content, &cfg)
	return cfg, err
}

// RecommendedSetting は提案する設定の構造
type RecommendedSetting struct {
	Key       string      `json:"key"`
	Suggested interface{} `json:"suggested"`
	Reason    string      `json:"reason"`
	Command   string      `json:"command"`
}

// RecommendationResult は最終的な推奨結果
type RecommendationResult struct {
	Workspace       WorkspaceMetrics     `json:"workspace"`
	ModelTier       string               `json:"model_tier"`
	SuggestedModels []string             `json:"suggested_models"`
	Reason          string               `json:"model_reason"`
	Settings        []RecommendedSetting `json:"settings"`
}

func generateRecommendations(metrics WorkspaceMetrics, modelCfg ModelConfig, taskHint string) RecommendationResult {
	result := RecommendationResult{
		Workspace: metrics,
	}

	// 1. モデルのティア決定
	tier := "light"
	if metrics.TotalLines < 5000 && len(metrics.Languages) <= 1 {
		tier = "light"
	} else if metrics.TotalLines < 30000 && len(metrics.Languages) <= 2 {
		// mid = 小規模 かつ 少言語。これにより heavy = 大規模 または 多言語（>=3）となり、
		// 大規模な単一言語リポジトリも正しく heavy に分類される。
		tier = "mid"
	} else {
		tier = "heavy"
	}

	// task_hint によるオーバーライド
	taskHintLower := strings.ToLower(taskHint)
	heavyKeywords := []string{"リファクタ", "設計", "アーキ", "migrate", "refactor", "design", "architecture", "規約", "ライセンス", "legal"}
	midSonnetKeywords := []string{"手順", "フォーマット", "strict", "lint", "ドキュメント", "readme", "document"}
	midOSSKeywords := []string{"quota", "枯渇", "制限", "フォールバック", "代替", "別視点", "fallback"}

	isHeavy := false
	for _, kw := range heavyKeywords {
		if strings.Contains(taskHintLower, kw) {
			isHeavy = true
			break
		}
	}

	if isHeavy {
		tier = "heavy"
	}

	result.ModelTier = tier

	// モデル候補の選定
	var candidates []Model
	for _, m := range modelCfg.Models {
		if m.Tier == tier {
			candidates = append(candidates, m)
		}
	}

	// 特化モデルの優先（traits 駆動のマッチング）
	var targetTrait string
	for _, kw := range midSonnetKeywords {
		if strings.Contains(taskHintLower, kw) {
			targetTrait = "instruction-following"
			break
		}
	}
	if targetTrait == "" {
		for _, kw := range midOSSKeywords {
			if strings.Contains(taskHintLower, kw) {
				targetTrait = "quota-independent"
				break
			}
		}
	}

	preferModel := ""
	if targetTrait != "" {
		for _, m := range modelCfg.Models {
			for _, tr := range m.Traits {
				if tr == targetTrait {
					preferModel = m.Name
					break
				}
			}
			if preferModel != "" {
				break
			}
		}
	}

	var suggestedNames []string
	if preferModel != "" {
		suggestedNames = append(suggestedNames, preferModel)
	}

	// 候補モデルの名前をリストに追加
	for _, c := range candidates {
		if c.Name != preferModel {
			suggestedNames = append(suggestedNames, c.Name)
		}
	}

	// 上位2件に絞る
	if len(suggestedNames) > 2 {
		suggestedNames = suggestedNames[:2]
	}
	result.SuggestedModels = suggestedNames

	// モデル推奨理由
	var reasonParts []string
	reasonParts = append(reasonParts, fmt.Sprintf("%d行", metrics.TotalLines))
	if len(metrics.Languages) > 0 {
		reasonParts = append(reasonParts, strings.Join(metrics.Languages, "+"))
	}
	if taskHint != "" {
		reasonParts = append(reasonParts, fmt.Sprintf("タスク: %s", taskHint))
	}
	result.Reason = fmt.Sprintf("%sに基づく判定", strings.Join(reasonParts, " / "))

	// 2. セキュリティ設定の推奨
	// Sandbox
	if metrics.HasEnv {
		result.Settings = append(result.Settings, RecommendedSetting{
			Key:       "enableTerminalSandbox",
			Suggested: true,
			Reason:    ".env検出（機密漏洩防止）",
			Command:   "/settings",
		})
	}

	// Permissions（toolPermission は単一値のため、より厳格な strict を優先。
	// CI と本番設定が共存するリポジトリでも本番検出時は strict を提示する）
	if metrics.HasProdConfig {
		result.Settings = append(result.Settings, RecommendedSetting{
			Key:       "toolPermission",
			Suggested: "strict",
			Reason:    "本番設定ファイル検出",
			Command:   "/settings",
		})
	} else if metrics.HasCI {
		result.Settings = append(result.Settings, RecommendedSetting{
			Key:       "toolPermission",
			Suggested: "proceed-in-sandbox",
			Reason:    "CI/CD 設定ファイル変更の可能性",
			Command:   "/settings",
		})
	}

	return result
}

func formatTextOutput(metrics WorkspaceMetrics, rec RecommendationResult) string {
	var sb strings.Builder

	// メトリクス行
	var meta []string
	meta = append(meta, fmt.Sprintf("%d行", metrics.TotalLines))
	if len(metrics.Languages) > 0 {
		meta = append(meta, strings.Join(metrics.Languages, ","))
	}
	if metrics.HasEnv {
		meta = append(meta, ".env✓")
	}
	sb.WriteString(fmt.Sprintf("⚙️ Advisor [%s]\n", strings.Join(meta, " / ")))

	// モデル推奨
	if len(rec.SuggestedModels) > 0 {
		tierEmoji := "⚡"
		if rec.ModelTier == "mid" {
			tierEmoji = "🔥"
		} else if rec.ModelTier == "heavy" {
			tierEmoji = "🧠"
		}
		sb.WriteString(fmt.Sprintf("  💡 Model: %s %s 推奨 (/model)\n", tierEmoji, strings.Join(rec.SuggestedModels, " or ")))
	}

	// その他設定
	for _, s := range rec.Settings {
		var valStr string
		switch v := s.Suggested.(type) {
		case bool:
			if v {
				valStr = "ON"
			} else {
				valStr = "OFF"
			}
		case string:
			valStr = v
		}

		emoji := "⚙️"
		if s.Key == "enableTerminalSandbox" {
			emoji = "🛡️"
		} else if s.Key == "toolPermission" {
			emoji = "🔒"
		}
		sb.WriteString(fmt.Sprintf("  %s %s: %s 推奨 — %s (%s)\n", emoji, strings.TrimPrefix(s.Key, "enable"), valStr, s.Reason, s.Command))
	}

	return sb.String()
}

func runStandalone() {
	metrics, err := scanWorkspace(".")
	if err != nil {
		fmt.Printf("Error scanning workspace: %v\n", err)
		return
	}

	modelCfg, err := loadModelsConfig()
	if err != nil {
		fmt.Printf("Error loading models.json: %v\n", err)
		return
	}

	rec := generateRecommendations(metrics, modelCfg, "リファクタリング作業")
	fmt.Print(formatTextOutput(metrics, rec))
}
