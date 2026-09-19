# test-runner — Go Test Runner

A Go-only MCP plugin for agy. Runs one module at a time, summarizes test/build failures, and returns arguments for explicitly rerunning failed tests. Calls wait for completion; there is no background job, persistent history, or automatic retry.

## Requirements and installation

- Go 1.24 or later on `PATH`, with the toolchain/dependencies required by the target module available.
- Prebuilt servers: Linux amd64, macOS arm64, Windows amd64. Windows uses the OS-provided `taskkill.exe` to stop a process tree.

```sh
agy plugin install https://github.com/kwrkb/agy-plugins/test-runner
```

## Tool: `go_test`

| Argument | Meaning |
| :-- | :-- |
| `module_path` | Required directory containing `go.mod`. Prefer an absolute path: relative paths use the MCP server's working directory. |
| `packages` | Nonempty string array of Go package patterns; default `["./..."]`. Local paths and import paths are accepted only when all resolved packages belong to the selected module and are physically inside it. |
| `run` | Optional Go `-run` expression, including slash-separated subtest expressions. |
| `timeout_seconds` | Integer 1–300, default 60. Covers discovery, compilation, and tests. Process cleanup can take a few additional seconds. |

Example arguments:

```json
{"module_path":"/code/agy-plugins/test-runner/src","packages":["./..."],"timeout_seconds":60}
```

The server resolves packages with `go list -e -json`, then runs `go test -json -count=1 -timeout=…` using an argument array, without a shell. It always disables successful-test caching. It inherits the Go environment (including `go.work`, `GOFLAGS`, build settings, and toolchain selection); it does not recursively discover other modules. A workspace containing several modules can be tested through separate calls with explicit module directories.

No arbitrary command, extra flags, `.go` file list, race/coverage option, fuzzing session, or benchmark option is exposed. Standard `go test` behavior, including examples and fuzz seed cases, remains in effect. Tests **execute project code** with the server's permissions and may change files or access the network. Go may use/download caches, dependencies, or a toolchain according to the inherited environment. This plugin is not an execution sandbox; use the host's execution permissions.

## Results and retry

Each MCP result contains a JSON text object:

- `status`: `passed`, `failed`, `no_tests`, `timeout`, `cancelled`, or `error`. Test/build failures are ordinary results; execution errors, timeouts, and cancellation set MCP `isError`.
- `exit_code`: Go process exit code, or `null` when no process exit code is available. Discovery errors can report the discovery process's exit code.
- `elapsed_seconds`: Total elapsed time; `packages[].elapsed_seconds` is Go's per-package duration.
- `tests`: `passed`, `failed`, `skipped` terminal-event counts, **including parent tests and subtests**. Package events are not counted as tests. All-skipped tests are distinguished by these counts; `no_tests` means no test terminal events and no failures.
- `packages`: Sorted package results with Go status `pass`, `fail`, `skip`, or `incomplete`; failing tests include their names, logs, and `rerun` arguments. Package-level failures such as a panic or `TestMain` exit may have no named failure; inspect the package log.
- `build_failures`: Build package IDs and diagnostics, separate from failed test counts. IDs may include a test variant suffix.
- `stderr`, `error`: Diagnostics and execution errors, when present.
- `logs_truncated`: Some log bytes were omitted. Retained event logs and stderr together are bounded to 1 MiB; returned log text is bounded to 64 KiB, prioritizing named failures. JSON escaping can make the serialized result larger.
- `incomplete`: Partial output, interrupted execution, or missing terminal events. Never interpret this as complete validation.

Use a failure's `rerun` object as the next `go_test` arguments. It retains the absolute module path and timeout, selects one package, and anchors/escapes each subtest-name segment. Go still runs parent setup to discover a selected subtest. Parent failures can overlap child failures; choose the parent to rerun a group or the child to narrow the retry. There is no server-side retry state.

Malformed/oversized JSON events (over 1 MiB), over 100,000 result records, and discovery output over 8 MiB produce explicit execution errors. Output continues to drain after a parsing error to avoid blocking the child; the overall timeout still applies. Timeout/cancellation stops the Unix process group or the Windows process tree and bounds pipe shutdown. Programs that deliberately detach from the process tree are outside this mechanism.

## Development and validation

In `test-runner/src`, run `gofmt`, `go vet ./...`, and `go test -count=1 ./...`. Tests use temporary modules without third-party dependencies and verify the real Go JSON stream, exact subtest retries, child-process termination, and MCP stdio calls.

From the repository root, run `./build.sh test-runner` (or `./build.ps1 test-runner`). These use the pinned `.go-version`; do not replace distributed binaries with an unpinned build. CI runs tests on Linux, macOS, and Windows and checks vulnerability policy and deterministic artifacts on Linux.

The bundled agent guide is [test-runner-gemini](skills/test-runner-gemini/SKILL.md). This is an agy plugin package, not a Codex plugin manifest.
