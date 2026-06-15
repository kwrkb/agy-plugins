// github-mcp-wrapper は github-unix/github-mcp-wrapper.sh の Windows 向け Go 移植。
//
// 公式 github-mcp-server は GITHUB_PERSONAL_ACCESS_TOKEN のみを読む。env に無ければ
// GITHUB_TOKEN / GH_TOKEN / `gh auth token` の順で解決して env に設定し、PATH 上の
// github-mcp-server を stdio で起動する。Windows は .sh を command に直接置けないため
// (.cmd/.bat も CreateProcess 直接実行不可)、このラッパーを .exe にビルドして使う。
//
// 注意:
//   - stdout には一切書かない (MCP は stdout で NDJSON を流すため)。診断は stderr へ。
//   - github-mcp-server は同梱せず PATH のユーザー導入バイナリを exec する (再配布なし)。
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// resolveToken は sh の `${A:-${B:-${C:-}}}` と同じ意味論でトークンを解決する。
// 空文字列は未設定扱いとし、いずれの env も空なら `gh auth token` にフォールバックする。
func resolveToken() string {
	for _, name := range []string{"GITHUB_PERSONAL_ACCESS_TOKEN", "GITHUB_TOKEN", "GH_TOKEN"} {
		if v := os.Getenv(name); v != "" {
			return v
		}
	}
	// gh auth token: stdout はバッファに取り込む (継承しない)。失敗しても空文字で続行する
	// (github-mcp-server が "token not set" を出して終了し、その終了コードを伝播させる)。
	out, err := exec.Command("gh", "auth", "token").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func main() {
	token := resolveToken()

	serverPath, err := exec.LookPath("github-mcp-server")
	if err != nil {
		fmt.Fprintln(os.Stderr, "github-mcp-wrapper: github-mcp-server not found in PATH:", err)
		fmt.Fprintln(os.Stderr, "  install it, e.g.: go install github.com/github/github-mcp-server/cmd/github-mcp-server@latest")
		os.Exit(127)
	}

	args := append([]string{"stdio"}, os.Args[1:]...)
	cmd := exec.Command(serverPath, args...)
	cmd.Env = append(os.Environ(), "GITHUB_PERSONAL_ACCESS_TOKEN="+token)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// 子の終了コードを伝播する。
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, "github-mcp-wrapper: failed to run github-mcp-server:", err)
		os.Exit(1)
	}
}
