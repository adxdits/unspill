# History Formats

Support is structural and tested with synthetic fixtures. Changed or unknown
records make coverage incomplete rather than being silently skipped.

| Adapter | Input | Default discovery |
| --- | --- | --- |
| `claude-code` | JSONL messages and selected metadata | `~/.claude/projects` |
| `copilot-cli` | JSONL events, declared schema version 1 | `~/.copilot/session-state/**/events.jsonl` |
| `copilot-vscode` | JSON chat exports | None |
| `codex` | JSONL rollouts | `~/.codex/sessions/**/rollout-*.jsonl` |
| `generic` | JSON/JSONL message exports | None; requires explicit adapter |

## Claude Code

Reads `message.content` strings or supported content blocks from user/assistant
entries, including thinking, tool inputs, and nested tool results. Selected
metadata records are scanned without event attribution. Discovery selects the
adapter from the known history root. For explicit metadata-first files, choose
`claude-code`; content auto-detection requires a recognizable message first.
Binary and redacted-thinking blocks are unsupported. Stored branches are
scanned as-is, not reconstructed into an active conversation.

## Copilot CLI

Reads `type`/`data` envelopes: user and assistant messages, reasoning, tool
arguments/results/errors, system messages, and selected lifecycle metadata.
Declared session versions other than 1 fail. Message-only excerpts may lack
session identity. Opaque reasoning and provider-specific server-tool payloads
make coverage incomplete. Binary assets and streamed deltas are not reassembled.
Referenced files and links are never opened.

## Copilot In VS Code

Reads JSON exports with a `requests` array, string or `text` request messages,
`variableData`, and markdown response objects with a string `value` and no `kind`.
Absent versions and serialized versions 1, 2, and 3 are accepted structurally.
Tool invocations and other response parts are unsupported. Coverage includes only
text actually present in the export, not omitted tool activity. Native JSONL
patch storage, SQLite state, Insiders, and remote discovery are unsupported.

## Codex

Reads `type`/`payload` envelopes: session metadata, text messages, function/custom
tool calls and outputs, shell actions, reasoning, selected event messages, turn
context, and compaction metadata. Encrypted reasoning and external history-base
references fail. Duplicate stored representations remain separate occurrences.
Custom `CODEX_HOME` and archived sessions require explicit paths.

## Generic

Requires `--agent generic` or `Options{Agent: "generic"}`. Each JSON object has
`messages` and optional `session_id`. Roles: user, assistant, system, developer,
tool, function. Content is a string or supported text/tool blocks; OpenAI-style
`tool_calls[].function.arguments` is recognized. JSONL may contain one export
per line. Arbitrary raw text is unsupported.

## Limits

UTF-8 only. Duplicate JSON fields and nesting deeper than 64 are rejected.
Encoded JSON text is decoded at most twice. Lines refer to physical source lines.
A malformed JSON export fails as a whole; independent JSONL records can continue
after an error. For valid JSON with unsupported blocks, independent supported
text before and after the error is scanned while coverage remains incomplete.
Defaults: 64 MiB per file, 1 MiB per record/export, 10,000 files,
and 10,000 findings. Event limits can be raised to 16 MiB. Scans also have fixed
256 MiB total input and bounded directory traversal limits.