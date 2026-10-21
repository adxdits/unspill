# Experimental History Formats

This is a structural compatibility contract backed by invented fixtures, not a
certification of installed agent releases. No local real histories were inspected.
Public sources below were consulted during implementation on 21 October 2026;
links to moving branches are evidence sources, not a promise of future compatibility.

## Claude Code

Source: [Anthropic Agent SDK transcript reader](https://github.com/anthropics/claude-agent-sdk-python/blob/main/src/claude_agent_sdk/_internal/sessions.py).

One JSON object per physical line. User and assistant entries have `type`,
optional `sessionId`, and `message.content` as a string or a content-block array.
Text/input_text/output_text, thinking, tool_use.input, and nested tool_result.content
are parsed. Selected system/progress/attachment/summary/file-history/queue records
are scanned as metadata with unavailable event attribution (`null`). Auto mode
needs a recognizable message before metadata-only records can be interpreted;
use `--agent claude-code` for known Claude files starting with metadata.

Unknown records and nontext/binary/redacted-thinking blocks mark coverage
incomplete. Sessions are scanned as stored, including historical branches, not
reconstructed to represent only the active conversation.

Discovery path evidence: the same public SDK reader uses the Claude projects
directory. Only the default `~/.claude/projects` layout is currently considered.

## Copilot CLI

Source: [Copilot SDK generated session-event definitions](https://github.com/github/copilot-sdk/blob/main/nodejs/src/generated/session-events.ts).

JSONL envelopes use `type` and `data`. Supported start headers explicitly have
`data.version: 1`; other declared versions fail. Message-only excerpts can have
unknown session identity. This does not certify a particular Copilot CLI release.

- user.message / assistant.message: content, transformedContent, attachments,
  reasoningText, and toolRequests.arguments.
- tool.execution_start / tool.user_requested: arguments as tool input.
- tool.execution_complete: result and error as tool output.
- assistant.reasoning: content as assistant text.
- system.message / system.notification: system content.
- Selected lifecycle/error/context/compaction records: metadata, attribution null.

Unknown event kinds are not guessed. Binary assets and streamed deltas are not
reassembled. External resource paths and links in records are never opened.
Default discovery is restricted to `~/.copilot/session-state`; only JSON/JSONL
inputs can be parsed. Default discovery selects only `events.jsonl`; adjacent
notes/configuration are not inputs. Explicit directory scans report unrelated
selected files as exclusions.

Session-state path reference:
[Copilot CLI session documentation](https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/manage-sessions).
Custom or changed locations must be supplied explicitly. Discovery paths have
not been checked against a real installation.

## Copilot in VS Code

Source: [VS Code 1.105.0 export/serialized chat interfaces](https://github.com/microsoft/vscode/blob/1.105.0/src/vs/workbench/contrib/chat/common/chatModel.ts).

Explicit JSON export only. Root `requests` is an array; `sessionId` is optional.
An absent version is accepted for export envelopes; serialized versions 1, 2,
and 3 are recognized structurally. Request messages are strings or objects with
`text`. `variableData` is request context. Responses must be arrays of exported
markdown objects with a string `value` and no `kind` discriminator.

Other response parts, including tool invocations, fail explicitly rather than
being misattributed. A VS Code export can omit original tool inputs/results; a
complete result covers only the text actually present in the export. This does
not support native `.jsonl` patch storage, SQLite state, VS Code Insiders,
remote-host discovery, or full agent-mode history coverage.

## Codex

Sources: [persisted protocol types](https://github.com/openai/codex/blob/main/codex-rs/protocol/src/protocol.rs)
and [response/content definitions](https://github.com/openai/codex/blob/main/codex-rs/protocol/src/models.rs).

JSONL envelopes have `type` and object `payload`. Supported shapes:

- session_meta: session identity from payload.id; metadata scanned without event attribution.
- response_item: role-bearing text messages, function_call.arguments,
  custom_tool_call.input, function/custom call output, local_shell_call.action,
  and unencrypted reasoning summary/content.
- event_msg: user/agent messages, reasoning, exec command begin/end, MCP tool
  begin/end, selected task/token/turn metadata.
- turn_context and compacted: scanned as metadata with attribution null.

Unknown variants, encrypted reasoning, and external history-base references
fail. Referenced files are not followed. Duplicate representations in event_msg
and response_item are retained as distinct stored occurrences, not silently
collapsed. Default discovery uses `~/.codex/sessions`; custom CODEX_HOME and
archived histories need explicit paths. No exact CLI-version compatibility claim
is made for these unversioned record shapes.

## Generic Exports

This is an Unspill input contract, not native support for another product.
Select `--agent generic` explicitly. JSON objects have a `messages` array and
optional `session_id`. Each message has role user, assistant, system, developer,
tool, or function. Content is a string or supported text/tool blocks; OpenAI-style
tool_calls[].function.arguments are recognized. JSONL may contain one such
object per line. No automatic conversion or arbitrary raw-text fallback occurs.

## Shared Constraints

UTF-8 only. Duplicate JSON object fields and structures deeper than 64 are
rejected. Encoded JSON in textual payloads is decoded at most twice; original
physical source lines are retained rather than invented from decoded newlines.
Multiline keys inside a JSON string still belong to its original physical line.

Unknown records make the scan incomplete, even when later independent JSONL
records can be scanned. A malformed whole JSON export is rejected as one record.
Complete coverage means selected files and these documented textual fields, not
all credentials retained anywhere by an agent.