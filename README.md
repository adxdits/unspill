# Unspill

Find potential credentials in locally saved AI coding-agent histories.
Runs offline, masks findings, and never modifies your histories. Available as a
CLI and Go package, powered by Gitleaks.

**Experimental:** tested with synthetic fixtures, not certified for every agent
version. No release binaries are available yet.

## Quick Start

Install [Go 1.24+](https://go.dev/dl/), then run from this repository:

```sh
go build -trimpath -o bin/unspill ./cmd/unspill
./bin/unspill scan --path ./testdata/demo --fail-on none
```

The demo reports four masked findings using fake credentials. No installed agent
or account is needed. Building may download dependencies; scanning stays offline.

## Usage

Scan only histories you are authorized to read:

```sh
./bin/unspill scan
./bin/unspill scan --path ./saved-session.jsonl
./bin/unspill scan --path ./saved-session.jsonl --format json --output report.json
./bin/unspill scan --help
```

Without `--path`, discovery is limited to known Claude Code, Copilot CLI, and
Codex history directories. Use `--agent` to select a format when needed.

## Supported Formats

| Adapter | Input |
| --- | --- |
| `claude-code` | Claude Code JSONL histories |
| `copilot-cli` | Copilot CLI JSONL events, schema 1 |
| `copilot-vscode` | VS Code JSON chat exports, not native session storage |
| `codex` | Codex JSONL rollouts |
| `generic` | Message exports; requires `--agent generic --path ...` |

Support is limited to documented text fields. Browser chats, Cursor/Windsurf
databases, and other formats are unsupported. [Format details](docs/formats.md).

## Safety

No uploads, telemetry, credential verification, or history rewriting. Symbolic
links are excluded. Existing report files are never overwritten; treat reports
as sensitive because they contain paths and session metadata.

A match is not proof of compromise, and no matches is not a safety guarantee.
Unknown formats, empty inputs, and resource-limit exclusions return an error.

Exit codes: **0** complete without threshold findings; **1** complete with
threshold findings; **2** error or incomplete scan. The default threshold is
`high`; `--fail-on none` reports findings without a findings failure exit.

## Development

```sh
go test -race ./...
go vet ./...
```

Go package: `github.com/adxdits/unspill/scanner`. Use only synthetic test data;
never submit real credentials or private histories in issues.

[Specification](UNSPILL-SPEC.md) | [Third-party notices](THIRD_PARTY_NOTICES.md).
The project license has not yet been selected.