package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	token := os.Getenv("GITHUB_PERSONAL_ACCESS_TOKEN")
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}
	if token == "" {
		out, err := exec.Command("gh", "auth", "token").Output()
		if err != nil {
			fmt.Fprintln(os.Stderr, "github-mcp-wrapper: failed to resolve token.\nSet GITHUB_PERSONAL_ACCESS_TOKEN or run: gh auth login")
			os.Exit(1)
		}
		token = strings.TrimSpace(string(out))
	}

	self, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "github-mcp-wrapper: cannot determine executable path:", err)
		os.Exit(1)
	}

	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	bin := filepath.Join(filepath.Dir(self), "github-mcp-server"+ext)

	env := os.Environ()
	found := false
	for i, e := range env {
		if strings.HasPrefix(e, "GITHUB_PERSONAL_ACCESS_TOKEN=") {
			env[i] = "GITHUB_PERSONAL_ACCESS_TOKEN=" + token
			found = true
			break
		}
	}
	if !found {
		env = append(env, "GITHUB_PERSONAL_ACCESS_TOKEN="+token)
	}

	args := append([]string{"stdio"}, os.Args[1:]...)
	cmd := exec.Command(bin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}
