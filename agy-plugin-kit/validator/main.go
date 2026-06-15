// agy-plugin-validator — agy プラグインソースを静的検査する決定的バリデータ。
//
// agy プラグイン開発で繰り返し踏んできた落とし穴（リポジトリ LESSONS.md / Issue #390）を
// install 前に機械的に検出する。agy 非依存で単体実行・テスト可能。
//
// モード:
//   validator <plugin-dir>        検査結果を表示。[ERROR] が1件でもあれば exit 1（/validate 用）。
//   validator --hook              stdin の hook JSON({"file_path":...}) からプラグインルートを導出し
//                                 マニフェスト編集時だけ検査。常に exit 0（助言的・非ブロッキング）。
//   validator --fix-paths <dir>   #390 ワークアラウンド: plugin.json 形式の mcp_config.json に残る
//                                 ${extensionPath} を <dir> の絶対パスへ書き換える。
//
// stdout/stderr 方針: 検査結果は stdout（--hook 時は stderr）に人間可読で出す。MCP の NDJSON は流さない。
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type severity int

const (
	sevWarn severity = iota
	sevError
)

func (s severity) tag() string {
	if s == sevError {
		return "[ERROR]"
	}
	return "[WARN]"
}

type finding struct {
	sev  severity
	code string // C1..C9
	msg  string
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: validator <plugin-dir> | --hook | --fix-paths <plugin-dir>")
		os.Exit(2)
	}

	switch args[0] {
	case "--hook":
		runHook()
	case "--fix-paths":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: validator --fix-paths <plugin-dir>")
			os.Exit(2)
		}
		runFixPaths(args[1])
	default:
		runCLI(args[0])
	}
}

// runCLI は <plugin-dir> を検査し、結果を stdout に出して ERROR があれば exit 1。
func runCLI(dir string) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "validator: bad path:", err)
		os.Exit(2)
	}
	if fi, err := os.Stat(abs); err != nil || !fi.IsDir() {
		fmt.Fprintln(os.Stderr, "validator: not a directory:", abs)
		os.Exit(2)
	}

	findings := validate(abs)
	printFindings(os.Stdout, abs, findings)

	for _, f := range findings {
		if f.sev == sevError {
			os.Exit(1)
		}
	}
}

// runHook は PostToolUse stdin を読み、編集されたファイルがマニフェスト系のときだけ検査する。
func runHook() {
	var payload struct {
		FilePath string `json:"file_path"`
		ToolName string `json:"tool_name"`
	}
	_ = json.NewDecoder(os.Stdin).Decode(&payload) // 失敗しても黙って 0 終了

	fp := payload.FilePath
	if fp == "" {
		return
	}
	base := strings.ToLower(filepath.Base(fp))
	manifests := map[string]bool{
		"plugin.json": true, "gemini-extension.json": true,
		"mcp_config.json": true, "hooks.json": true,
	}
	if !manifests[base] {
		return // マニフェスト以外の編集には反応しない（無音）
	}
	dir := filepath.Dir(fp)
	findings := validate(dir)
	if len(findings) > 0 {
		fmt.Fprintf(os.Stderr, "agy-plugin-kit: %s を検査しました:\n", filepath.Base(fp))
		printFindings(os.Stderr, dir, findings)
	}
	// hook は常に 0 終了（ブロックしない）
}

func printFindings(w *os.File, dir string, fs []finding) {
	if len(fs) == 0 {
		fmt.Fprintf(w, "OK: %s — 問題は検出されませんでした\n", dir)
		return
	}
	sort.SliceStable(fs, func(i, j int) bool { return fs[i].code < fs[j].code })
	var nErr int
	for _, f := range fs {
		if f.sev == sevError {
			nErr++
		}
		fmt.Fprintf(w, "%s %s %s\n", f.sev.tag(), f.code, f.msg)
	}
	fmt.Fprintf(w, "-- %d 件（ERROR %d）\n", len(fs), nErr)
}

// validate は全チェックを実行して findings を返す（純粋関数寄り・テスト容易）。
func validate(dir string) []finding {
	var out []finding
	has := func(name string) bool {
		_, err := os.Stat(filepath.Join(dir, name))
		return err == nil
	}
	read := func(name string) string {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return ""
		}
		return string(b)
	}

	hasPluginJSON := has("plugin.json")
	hasGeminiExt := has("gemini-extension.json")
	mcpRaw := read("mcp_config.json")
	hooksRaw := read("hooks.json")

	// C5: 識別ファイルが無い
	if !hasPluginJSON && !hasGeminiExt {
		out = append(out, finding{sevError, "C5",
			"plugin.json（または gemini-extension.json）が無く、agy がプラグインと認識できません。"})
	}
	for _, mf := range []string{"plugin.json", "gemini-extension.json"} {
		if has(mf) {
			if raw := read(mf); raw != "" {
				var v map[string]any
				if json.Unmarshal([]byte(raw), &v) != nil {
					out = append(out, finding{sevError, "C5", mf + " が不正な JSON です。"})
				}
			}
		}
	}

	// C1: 二重マニフェスト
	if hasPluginJSON && hasGeminiExt {
		out = append(out, finding{sevWarn, "C1",
			"plugin.json と gemini-extension.json が両方あります。plugin.json があると install は copy-only になり ${extensionPath} は解決されません（LESSONS #1）。"})
	}

	// C2: plugin.json 形式なのに mcp_config.json で ${extensionPath} を使用（Issue #390 で未解決のまま残る）
	if hasPluginJSON && !hasGeminiExt && mcpRaw != "" {
		if strings.Contains(mcpRaw, "${extensionPath}") || strings.Contains(mcpRaw, "${/}") {
			out = append(out, finding{sevError, "C2",
				"plugin.json 形式の mcp_config.json で ${extensionPath}/${/} を使用しています。native 形式では解決されず（Issue #390）パスが literal のまま壊れます。gemini-extension.json 形式にするか、/agy-plugin-kit:doctor の --fix-paths で絶対パス化してください。"})
		}
	}

	// command 文字列を収集（MCP と hook を分けて扱う）。C3/C6/C7 は MCP command 固有のルール
	// （agy の LookPath 補完・MCP spawn 前提）。hook command はシェル実行で .exe が要るため対象外。
	mcpCmds := collectMCPCommands(mcpRaw, read("gemini-extension.json"))
	hookCmds := collectHookCommands(hooksRaw)

	for _, c := range mcpCmds {
		low := strings.ToLower(c)
		// C3: .sh/.cmd/.bat を MCP command に直接指定（Windows で spawn 不可）
		for _, ext := range []string{".sh", ".cmd", ".bat"} {
			if strings.HasSuffix(strings.TrimSpace(low), ext) || strings.Contains(low, ext+" ") {
				out = append(out, finding{sevError, "C3",
					fmt.Sprintf("MCP command が %s を直接指定しています（Windows で spawn 不可）。拡張子なしの Go .exe ラッパーにしてください（LESSONS #10/#14）: %q", ext, c)})
				break
			}
		}
		// C6: ${extensionPath} 形式の MCP command に .exe 拡張子（拡張子なしにすべき）
		if strings.Contains(c, "${extensionPath}") && strings.Contains(low, ".exe") {
			out = append(out, finding{sevWarn, "C6",
				fmt.Sprintf("${extensionPath} 形式の MCP command に .exe を付けています。agy が補完するため拡張子なしフルパス推奨（LESSONS #10）: %q", c)})
		}
		// C7: トークン専用 MCP サーバーを wrapper 無しで直叩き（heuristic）
		if strings.Contains(low, "github-mcp-server") && !strings.Contains(low, "wrapper") {
			out = append(out, finding{sevWarn, "C7",
				"github-mcp-server を wrapper 無しで起動しています。GITHUB_PERSONAL_ACCESS_TOKEN 未設定だと即終了します。トークン解決ラッパー経由を推奨（LESSONS #5/#11）。(heuristic)"})
		}
	}

	// C8: ${CLAUDE_PLUGIN_ROOT}（agy に存在しないトークン）は MCP・hook どちらでも無効
	for _, c := range append(append([]string{}, mcpCmds...), hookCmds...) {
		if strings.Contains(c, "${CLAUDE_PLUGIN_ROOT}") {
			out = append(out, finding{sevError, "C8",
				fmt.Sprintf("command に ${CLAUDE_PLUGIN_ROOT} を使用。これは Claude Code 専用で agy では解決されません: %q", c)})
		}
	}

	cmds := append(append([]string{}, mcpCmds...), hookCmds...) // C4 用（参照ファイルは両方見る）

	// C4: command が参照するローカルバイナリが .gitignore で除外されている（URL install で消える）。
	// git check-ignore には絶対パスより dir 相対パスを渡す方がクロスプラットフォームで確実。
	for _, p := range referencedLocalFiles(dir, cmds) {
		if gitIgnored(dir, rel(dir, p)) {
			out = append(out, finding{sevError, "C4",
				fmt.Sprintf("マニフェストが参照する %s が .gitignore で除外されています。URL install で clone されず起動不能になります。明示的にコミットしてください（LESSONS #11）。", rel(dir, p))})
		}
	}

	// C9: 同梱 Go ラッパーが「自分で」stdout に書く（NDJSON 破壊・best-effort）。
	// `cmd.Stdout = os.Stdout` の素通しは正しいパターンなので除外し、自前の stdout 書き込み
	// （fmt.Print* / os.Stdout.Write / 組み込み println）だけを MCP 文脈で検出する。
	// mcpServers を宣言しないプラグイン（=この kit 自身など）は MCP ラッパーが無いので C9 をスキップ。
	hasMCP := strings.Contains(mcpRaw, "mcpServers") || strings.Contains(read("gemini-extension.json"), "mcpServers")
	for _, gofile := range goFiles(dir) {
		if !hasMCP {
			break
		}
		src := readFile(gofile)
		mcpCtx := strings.Contains(src, "stdio") || strings.Contains(src, "mcpServers") ||
			strings.Contains(src, "github-mcp-server")
		// fmt.Print* は stdout 書き込み（fmt.Fprint* は writer 引数を取るので "fmt.Print" には非該当）。
		// 組み込み println は "Fprintln" を部分一致で誤検出するため使わない。
		writesStdout := strings.Contains(src, "fmt.Print") || strings.Contains(src, "os.Stdout.Write")
		if mcpCtx && writesStdout {
			out = append(out, finding{sevWarn, "C9",
				fmt.Sprintf("%s が自前で stdout に書いています（MCP は stdout で NDJSON を流すため厳禁）。診断は stderr へ（LESSONS #2/#16）。(heuristic)", rel(dir, gofile))})
		}
	}

	return out
}

// collectMCPCommands は mcp_config.json と gemini-extension.json の mcpServers から command(+args) を抽出する。
func collectMCPCommands(raws ...string) []string {
	var cmds []string
	for _, raw := range raws {
		if raw == "" {
			continue
		}
		var m struct {
			McpServers map[string]struct {
				Command string   `json:"command"`
				Args    []string `json:"args"`
			} `json:"mcpServers"`
		}
		if json.Unmarshal([]byte(raw), &m) == nil {
			for _, s := range m.McpServers {
				if s.Command != "" {
					cmds = append(cmds, strings.Join(append([]string{s.Command}, s.Args...), " "))
				}
			}
		}
	}
	return cmds
}

// collectHookCommands は hooks.json の各 hook command を抽出する。
func collectHookCommands(hooksRaw string) []string {
	var cmds []string
	if hooksRaw == "" {
		return cmds
	}
	var h struct {
		Hooks map[string][]struct {
			Hooks []struct {
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if json.Unmarshal([]byte(hooksRaw), &h) == nil {
		for _, group := range h.Hooks {
			for _, entry := range group {
				for _, hk := range entry.Hooks {
					if hk.Command != "" {
						cmds = append(cmds, hk.Command)
					}
				}
			}
		}
	}
	return cmds
}

// referencedLocalFiles は command 中の ${extensionPath}${/}foo / ./foo を実ファイルに解決する。
func referencedLocalFiles(dir string, cmds []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, c := range cmds {
		// 先頭トークン（実行ファイル部分）だけを見る
		tok := c
		if i := strings.IndexAny(tok, " "); i >= 0 {
			tok = tok[:i]
		}
		// ${extensionPath}${/} を剥がしてプラグインルート相対にする
		t := strings.ReplaceAll(tok, "${extensionPath}", "")
		t = strings.ReplaceAll(t, "${/}", string(filepath.Separator))
		t = strings.TrimLeft(t, `/\`)
		t = strings.TrimPrefix(t, "."+string(filepath.Separator))
		if t == "" || strings.Contains(t, "$") {
			continue
		}
		// 拡張子なしフルパス指定なら .exe も候補にする
		candidates := []string{t}
		if filepath.Ext(t) == "" {
			candidates = append(candidates, t+".exe")
		}
		for _, cand := range candidates {
			p := filepath.Join(dir, cand)
			if _, err := os.Stat(p); err == nil && !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	return out
}

// gitIgnored は git check-ignore にシェルアウトして正確に判定する（gitignore を自前実装しない）。
func gitIgnored(dir, path string) bool {
	cmd := exec.Command("git", "-C", dir, "check-ignore", "-q", path)
	err := cmd.Run()
	if err == nil {
		return true // exit 0 = ignored
	}
	// exit 1 = not ignored, それ以外（git 無し等）は判定不能として false
	return false
}

func runFixPaths(dir string) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "validator: bad path:", err)
		os.Exit(2)
	}
	mcpPath := filepath.Join(abs, "mcp_config.json")
	raw, err := os.ReadFile(mcpPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "validator --fix-paths: mcp_config.json が読めません:", err)
		os.Exit(2)
	}
	sep := string(filepath.Separator)
	// Windows の絶対パス（C:\Users\...）はバックスラッシュを含むため、JSON 文字列値に埋め込む前に
	// \ を \\ にエスケープする。しないと \U 等が不正な JSON エスケープになり mcp_config.json が壊れる。
	absEsc := strings.ReplaceAll(abs, `\`, `\\`)
	sepEsc := strings.ReplaceAll(sep, `\`, `\\`)
	fixed := strings.ReplaceAll(string(raw), "${extensionPath}${/}", absEsc+sepEsc)
	fixed = strings.ReplaceAll(fixed, "${extensionPath}", absEsc)
	fixed = strings.ReplaceAll(fixed, "${/}", sepEsc)
	if fixed == string(raw) {
		fmt.Println("変更なし（${extensionPath} は見つかりませんでした）:", mcpPath)
		return
	}
	if err := os.WriteFile(mcpPath, []byte(fixed), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "validator --fix-paths: 書き込み失敗:", err)
		os.Exit(2)
	}
	fmt.Printf("Issue #390 ワークアラウンド適用: %s の ${extensionPath} を %q に置換しました。\n", mcpPath, abs)
}

// --- small helpers ---

func readFile(p string) string {
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return string(b)
}

func rel(dir, p string) string {
	if r, err := filepath.Rel(dir, p); err == nil {
		return r
	}
	return p
}

func goFiles(dir string) []string {
	var out []string
	_ = filepath.Walk(dir, func(p string, fi os.FileInfo, err error) error {
		if err == nil && !fi.IsDir() && strings.HasSuffix(p, ".go") {
			out = append(out, p)
		}
		return nil
	})
	return out
}
