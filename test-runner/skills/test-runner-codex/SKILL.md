---
name: test-runner-codex
description: Use the go_test MCP tool to run Go tests in one module, inspect failures, and rerun selected tests after a fix. Use when the test-runner MCP plugin is available.
---

# Go test runner

Call `go_test` with the absolute `module_path` containing `go.mod`. The MCP server's working directory may differ from the repository. For a multi-module repository, call each intended module separately; the repository root may not be a module.

- Default `packages` is `["./..."]`. `run` follows Go's slash-separated subtest regex syntax. `timeout_seconds` defaults to 60 and accepts integers 1–300, including compilation time.
- Read `status`, `incomplete`, `build_failures`, package results, and `logs_truncated`. Test counts include both parent tests and subtests. `no_tests` is not evidence that relevant tests passed.
- A test/build failure is a normal MCP result with `status: "failed"`. A tool error, timeout, or cancellation must not be reported as a clean test run. If no named test failed, inspect the package log and stderr for panic, setup, or `TestMain` errors.
- After a fix, pass a selected failure's `rerun` object back to `go_test`. Parent/child failures overlap: choose the appropriate level instead of rerunning both automatically. Broaden testing only when the change warrants it. The plugin does not automatically retry or store history.
- Truncated logs require a narrower package/test selection to recover useful diagnostics. Persistent execution failures or timeouts should be reported with their actual validation limits rather than retried indefinitely.

Tests execute project code with the host's permissions and may write files or access the network. Go environment settings are inherited. This tool provides no sandbox or additional permission; use it within the current task's authorized scope.
