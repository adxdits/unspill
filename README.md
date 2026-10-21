# Unspill

An offline, read-only CLI and Go package for finding potential credentials in
supported locally saved AI coding-agent histories.

## Current Status

Experimental implementation, validated with synthetic fixtures only. It is not
an "all agents" scanner and is not production-certified. There are no published
release binaries yet. Compatibility means the documented record shapes, not
every version or feature of an agent.

| Adapter | Accepted Input | Discovery |
| --- | --- | --- |
| `claude-code` | JSONL user/assistant text, tool use/results, selected metadata | `~/.claude/projects` |
| `copilot-cli` | JSONL session schema 1, selected message/tool events | `~/.copilot/session-state` |
| `copilot-vscode` | VS Code JSON chat exports: request text and markdown responses | Explicit path only |
| `codex` | JSONL rollout metadata, selected response items and events | `~/.codex/sessions` |
| `generic` | Explicit JSON/JSONL objects with a `messages` array | Explicit path only |

Copilot CLI is different from Copilot in VS Code. Native VS Code incremental
storage, Cursor/Windsurf databases, Claude's website, browser chats, Mistral Vibe,
and other undocumented formats are not supported. See
[docs/formats.md](docs/formats.md) for exact coverage and public schema sources.

## Quick Start

Install Go 1.24 or later from <https://go.dev/dl/>, then run from this directory:

```sh
go mod download
go build -trimpath -o bin/unspill ./cmd/unspill
./bin/unspill --help
./bin/unspill agents
./bin/unspill scan --path ./testdata/demo
```

The demo uses invented, nonfunctional values and should report four occurrences
with exit code `1`. It needs no account, installed agent, model, or network
connection after the executable has been built.

```sh
./bin/unspill scan --agent claude-code --path ./testdata/demo/claude-code.jsonl
./bin/unspill scan --agent copilot-cli --path ./testdata/demo/copilot-cli.jsonl
./bin/unspill scan --agent copilot-vscode --path ./testdata/demo/copilot-vscode.json
./bin/unspill scan --agent codex --path ./testdata/demo/codex.jsonl
./bin/unspill scan --agent generic --path ./testdata/generic.json
./bin/unspill scan --path ./testdata/demo --format json --output report.json
./bin/unspill scan --path ./testdata/demo --fail-on none
```

Repeat `--path` to select several inputs. Auto-detection recognizes only native
documented envelopes; generic exports require `--agent generic`. Exports are
never auto-converted from an unknown agent format.

On your own authorized machine, `unspill scan` discovers only the three history
roots in the table, printing them before scanning. Missing roots are tolerated;
entirely missing or empty input is not described as clean. Home and filesystem
roots are never substituted for missing history locations. Default discovery
selects Claude `.jsonl` files, Copilot `events.jsonl`, and Codex `rollout-*.jsonl`;
adjacent notes and indexes are not selected. Unsupported files in explicitly
selected directories and unknown history records make coverage incomplete. Environment
overrides and custom agent storage locations require an explicit `--path`.

To remove a local build, delete `bin/unspill`. No services, configuration,
or global installation are created by these commands.

## Go API

The public package is `github.com/adxdits/unspill/scanner`. It defaults to
explicit-path mode and does not discover histories merely because it is imported.

```go
package main

import (
		"context"
		"os"

		"github.com/adxdits/unspill/scanner"
)

func main() {
		report := scanner.Scan(context.Background(), scanner.Options{
				Paths: []string{"./testdata/demo"},
				Agent: "auto",
		})
		if err := report.WriteJSON(os.Stdout); err != nil {
				os.Exit(2)
		}
		os.Exit(report.ExitCode("high"))
}
```

The import path is prepared for the next published commit; these changes are not
available from the remote repository until published. Library callers should
pass a context with a deadline. `Discover: true` opts into documented home roots
and cannot be combined with `Paths`. Operational failures are returned in the
report, not as raw filesystem or parser exceptions.

## Reports and Exit Codes

| Code | Meaning |
| --- | --- |
| `0` | Complete for selected files and documented fields; no threshold findings |
| `1` | Complete; findings meet `--fail-on` |
| `2` | Invalid options, incomplete coverage, unsupported/empty input, or output failure |

`--fail-on` accepts `high` (default), `medium`, `low`, and `none`. It changes the
exit code, not which findings appear. Operational failure always wins.

Reports contain rule IDs, severity, confidence, sanitized paths, physical file
lines, and available session/event attribution. Unknown attribution is `null`.
Occurrences in different events are retained. Identifiers are per-report, never
secret hashes. Values are always `[REDACTED]`; prompts, snippets, commands, and
raw detector findings are not exported.

JSON files use exclusive creation and mode `0600` on Unix. Existing files are
never overwritten, including source files and symlinks. Windows uses platform
ACL inheritance; Unix permission bits do not establish owner-only Windows ACLs.
Even masked reports can reveal sensitive paths and session metadata.

Provider-specific rule matches are labelled high/high; the generic API-key rule
is medium/low. AWS access-key identifiers are medium/high because an identifier
alone is not a complete credential pair. These labels are policies, not measured
accuracy or evidence of active credentials, transmission, or compromise.

## Safety and Limits

- Embedded Gitleaks v8.24.3 rules and engine; no detector subprocess or local config loading.
- No telemetry, update checks, provider validation, model calls, or uploads.
- Gitleaks logging is disabled, including trace output that could contain raw matches.
- Read-only inputs; no history rewriting, temporary raw copies, or persistent secret cache.
- Symbolic links are excluded. macOS/Linux opens use no-follow and nonblocking flags;
	Windows opens reject reparse points. Ancestor-directory replacement races remain
	possible; do not treat this as a hardened forensic reader on a hostile filesystem.
- Changed size, modification time, or identity marks a file incomplete. Same-size
	concurrent rewrites with restored timestamps are not reliably detected.
- Default limits: 64 MiB/file, 1 MiB/JSONL record or whole JSON export, 10,000 files,
	10,000 findings, 256 MiB total input, depth 64 JSON, depth 16 directories,
	50,000 JSON nodes/record, 50,000 directory entries, and 1,000 reported errors.
- The CLI deadline defaults to 30 seconds. Limits/deadlines are checked between
	bounded operations, not a hard CPU/memory containment guarantee.
- `--max-file-bytes`, `--max-event-bytes`, `--max-files`, `--max-findings`, and
	`--timeout` are configurable. Hard caps are 64 MiB/file, 16 MiB/event,
	10,000 files and 10,000 findings. Exceeding a limit returns incomplete coverage.
- No cross-event secret reconstruction, image/OCR, encrypted-history decoding,
	archive extraction, database reading, or arbitrary base64 decoding.
- The CLI and library refuse root. OS access metadata and swap may still change;
	secure memory erasure and containment on a compromised host are not promised.

The embedded detector's package-global logger is disabled at import time. A Go
host also embedding Gitleaks must not re-enable it while processing histories.

## Development and Verification

```sh
gofmt -w cmd internal scanner
go test ./...
go test -race ./...
go vet ./...
GOPROXY=off GOTOOLCHAIN=local go test ./...
```

Dependency installation/builds may use the network; scanning does not. The last
command uses already cached dependencies and is not by itself a network-isolation
test. CI covers Linux, macOS, and Windows; cross-compilation alone is not runtime
certification. Synthetic tests do not establish real-world precision or recall.

Use only synthetic credentials and synthetic histories for development.
Synthetic-only builds/tests on this laptop were explicitly authorized for this
implementation. No authorization was given to scan its real agent sessions or
employer data. Never submit real credentials, private histories, or proprietary
content in issues or pull requests.

## Release Status

Not ready for a stable public release: actual client-version certification,
held-out accuracy counts, personal-laptop performance measurements, independent
security review, private vulnerability-reporting setup, and complete third-party
redistribution notices remain release gates. No vulnerability-reporting address
has been configured; do not post sensitive evidence in public issues.

Dependencies are pinned in `go.mod` and `go.sum`. Gitleaks is MIT-licensed; see
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md). The project license remains
undecided. The product roadmap and remaining requirements are in
[UNSPILL-SPEC.md](UNSPILL-SPEC.md).