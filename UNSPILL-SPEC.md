# Unspill — Product and Technical Specification

> Working name, not a cleared trademark or an existing implementation.
> Specification date: 18 October 2026.
> Status: product proposal with an experimental implementation of a subset.
> Implementation update (21 October 2026): CLI and Go API with synthetic-tested
> Claude Code, Copilot CLI, VS Code export, Codex, and generic-format parsers.
> No real-history or exact client-version certification has been performed.

## 1. Summary

**Unspill is an offline, read-only CLI that finds potential credentials in locally saved AI coding-agent histories and explains where they appeared.**

The first release helps an individual developer answer:

> “Did a credential end up in a saved agent conversation or tool output?”

It does not require an LLM, GPU, paid API, cloud account, or hosted backend.
Development and testing should use the owner's personal Intel i7 laptop, synthetic credentials, and authorized test data.

### Positioning

“A local secret scanner that understands AI coding-agent histories.”

The intended differentiator is not a new regular-expression engine. It is reliable discovery and parsing of agent histories, attribution to conversation events, and safe reporting.

### What this is not

- Proof that a credential was transmitted to a model provider.
- Proof that another person accessed a credential.
- A guarantee that all credentials have been found.
- A replacement for credential rotation or endpoint security.
- A prompt-injection detector, malware scanner, or compliance certification.
- A claim to be the first French or first worldwide implementation.

## 2. Goals and Constraints

### Goals

1. Find actionable potential secrets in supported agent histories.
2. Explain whether a finding came from a user message, assistant message, tool input, or tool output, when the format supports that distinction.
3. Never disclose the secret through normal output or exports.
4. Preserve all scanned files unchanged.
5. Work without network access or local model inference.
6. Be straightforward to install, inspect, and uninstall.

### Constraints

- Zero mandatory spending on compute, APIs, databases, or hosting.
- An ordinary Intel i7 laptop is the development and runtime target; RAM and storage still need to be established.
- No employer equipment, credentials, repositories, or data without explicit authorization.
- No scanning this work computer or its agent sessions as part of implementing the project.
- Free project infrastructure may be used subject to its current terms and quotas, but the scanner must not depend on it at runtime.
- “Zero budget” excludes the existing laptop, electricity, internet connection, and the developer's time.

## 3. Target Users and Problems

| User | Problem | Initial value |
|---|---|---|
| Developer using coding agents | Does not know what sensitive material was retained locally | A local, inspectable inventory |
| Open-source contributor | Wants to share a session for debugging without sharing credentials | A warning before sharing |
| Security engineer | Needs to understand credential persistence in developer tools | Structured findings and format documentation |
| Small engineering team | Uses several agent products with different history formats | Consistent local scanning |

Recurring enterprise demand is a hypothesis, not established evidence.
The individual CLI can be useful even if it never becomes a subscription business.

## 4. Evidence and Open Questions

### Existing relevant projects

- [Gitleaks](https://github.com/gitleaks/gitleaks): established secret detection; candidate engine or source of licensed rules.
- [TruffleHog](https://github.com/trufflesecurity/trufflehog): established secret detection; study overlap and licensing, and avoid credential verification/network behavior.
- [Mistral Vibe](https://github.com/mistralai/mistral-vibe): public agent implementation with documented session continuation and exports.

These projects demonstrate relevant components and adjacent functionality.
They do not establish that Unspill is novel, that any particular agent retains credentials insecurely, or that developers will pay.

### Research still required

- Check current scanners and endpoint-security products for equivalent history support.
- Inspect each agent's documented and implemented storage formats using permitted sources and synthetic fixtures.
- Establish supported versions, platforms, and exact discovery paths.
- Confirm whether existing redaction already handles the proposed test cases.
- Check the chosen detection engine's license, embedding API, configuration, and output behavior.
- Check availability of the working name across GitHub, package registries, domains, and trademarks.

Do not turn previous exploratory statements about storage formats or market gaps into product claims without verification.

## 5. MVP Scope

### Included

- One fully verified agent adapter initially.
- Read-only discovery within explicitly supported history directories.
- An explicit-path mode for an individual file or fixture directory.
- Parsing of the documented supported formats.
- Offline detection using established rules.
- Masked terminal findings and a sanitized JSON report.
- File location, rule identifier, severity, and confidence.
- Session and event attribution when available.
- Clear errors, partial-scan reporting, and resource limits.
- Local tests using synthetic credentials and synthetic histories.

### Excluded

- Automatic cleanup, deletion, rewriting, or retention enforcement.
- Credential validity checks, provider API calls, or automatic rotation.
- Background watching, startup services, or scheduled scans.
- MCP integration, editor extensions, web dashboards, and organization accounts.
- Scanning environment files, keychains, browser stores, or arbitrary home-directory content by default.
- Model inference, fine-tuning, or an AI explanation service.
- Archives, remote storage, and proprietary databases unless explicitly supported later.

### Adapter sequence

Candidate adapters are Mistral Vibe, Claude Code, and Codex CLI.
Do not advertise support for all three at launch unless their tests pass.
Choose the first adapter after a short feasibility investigation, based on accessible documentation and safely reproducible fixtures—not branding alone.

## 6. CLI Design

All commands below describe the intended interface, not currently working software.

```text
unspill scan
unspill scan --agent <supported-agent>
unspill scan --path ./synthetic-fixtures
unspill scan --format json --output report.json
unspill scan --fail-on high
unspill version
unspill --help
```

### Default behavior

1. Print which supported history roots will be scanned.
2. Scan only the current user's documented supported locations.
3. Do not follow symbolic links.
4. Parse histories and run detection in process.
5. Print sanitized findings, coverage, errors, and a summary.
6. Do not modify source files or upload anything.

An explicit path does not grant permission to access other users' data.
Discovery must not silently expand into a recursive home-directory scan.

### Example terminal output

```text
Unspill — local, read-only scan

Example Agent: 18 history files scanned
2 potential secrets found

HIGH    github-token-pattern
        Session: synthetic-session-01
        Source: tool output
        File: <home>/example-agent/history/session-01.jsonl:42
        Value: [REDACTED]
        Confidence: high

MEDIUM  generic-password-assignment
        Session: synthetic-session-02
        Source: user message
        Value: [REDACTED]
        Confidence: low

Coverage: complete for the selected files
Network requests: disabled by design
Source files modified: no
```

Paths and identifiers must be sanitized and terminal control sequences escaped.
Do not print surrounding content or partial credential characters by default.
“Confidence” concerns detection, not whether the credential works.

### Exit codes

| Code | Meaning |
|---|---|
| `0` | Complete scan; no findings at or above the selected failure threshold |
| `1` | Complete scan; findings meet the failure threshold |
| `2` | Operational error, invalid configuration, no supported input, or incomplete scan |

Default failure threshold: high.
Lower-severity findings still appear in reports.
Operational failure takes precedence over findings, so an incomplete scan cannot look successful.
An entirely empty input must be reported distinctly, not described as “clean.”

## 7. Findings and JSON Contract

### Severity and confidence

- Severity describes the potential impact if the value is a real credential.
- Confidence describes the evidence that the text is a credential rather than an example or ordinary string.
- Neither implies that the credential is active, externally exposed, or compromised.
- Start with high-specificity credential formats; label generic assignment rules conservatively.
- A clean result means “no matches under this scan's coverage and rules,” not “safe.”

### Sanitized report example

```json
{
  "schema_version": "1",
  "tool_version": "0.1.0",
  "status": "complete",
  "coverage": {
    "files_scanned": 18,
    "files_skipped": 0,
    "files_failed": 0
  },
  "findings": [
    {
      "id": "finding-1",
      "agent": "example-agent",
      "rule_id": "github-token-pattern",
      "severity": "high",
      "confidence": "high",
      "file": "<home>/example-agent/history/session-01.jsonl",
      "line": 42,
      "session_id": "synthetic-session-01",
      "event_type": "tool_output",
      "value": "[REDACTED]"
    }
  ],
  "errors": []
}
```

Required contract decisions:

- Use `null` for unavailable attribution; never fabricate a line number or session identity.
- Record original physical file lines, not lines in a reconstructed text buffer.
- Report unsupported formats, read failures, malformed records, and resource-limit exclusions in `errors`.
- Sanitize exception messages and metadata before serialization.
- Omit raw matched text, snippets, prompts, and command contents entirely.
- Do not include unhashed or hashed secrets as persistent finding identifiers.
- Use per-report identifiers; persistent suppression requires a separate privacy-reviewed design.
- Treat even sanitized reports as sensitive because paths and session metadata can reveal information.
- Create report files with owner-only permissions where supported; document platform limitations.
- Refuse to overwrite an existing report unless the user explicitly authorizes it.

## 8. Architecture

```text
CLI and validated options
          |
          v
Agent-specific discovery
          |
          v
Bounded, streaming parsers
          |
          v
Normalized events with source locations
          |
          v
Offline detection engine
          |
          v
Sanitization and finding attribution
          |
          v
Terminal output / local JSON report
```

### Recommended implementation

**Go is the initial recommendation**, primarily for distributing a single binary without requiring a Python or Node environment.
This is a recommendation, not a prerequisite; finalize it after checking the detection engine's supported integration.

Prefer embedding an established detector if its API and license permit it.
Avoid a subprocess that emits raw secrets into stdout, temporary files, or error logs.
If a subprocess is necessary, prove its redaction and offline behavior before using it.

Do not build a new detection engine, plugin system, database, daemon, or remote service for the MVP.
Begin with one adapter and extract a shared adapter interface only when a second adapter demonstrates the need.

### Adapter responsibilities

- Discover only supported paths for that agent and platform.
- Identify supported schema versions.
- Stream messages and tool events with reliable source attribution.
- Retain no unnecessary raw content after processing.
- Return explicit errors instead of guessing unknown structures.

### Proposed repository layout

```text
cmd/unspill/
internal/scan/
internal/agents/
internal/detect/
internal/report/
testdata/
docs/
```

This is a proposed implementation layout, not a request to create empty directories now.

## 9. Privacy and Threat Model

### Assets

- Credential strings present in local histories.
- Proprietary code, personal information, and conversation contents.
- Agent histories required for resumption or debugging.

### Relevant risks

- A history is shared accidentally with a credential inside it.
- Local malware or an unauthorized local reader accesses saved credentials.
- Unspill itself leaks data through output, crash reporting, temporary files, or a dependency.
- Malformed input causes excessive memory use, a crash, or misleading output.

### Boundaries

Unspill detects matching text already present in authorized readable files.
It does not stop malware, establish who accessed a file, prove remote transmission, or recover deleted data.
It cannot reliably detect arbitrary custom secrets or secrets split across multiple events in the MVP.

### Required safeguards

- No telemetry, analytics, update checks, uploads, or model calls.
- No credential verification or requests to credential providers.
- Never execute code, commands, embedded links, or instructions found in input.
- Never deserialize input using executable object formats.
- Treat history content as untrusted data, including apparent instructions to the scanner.
- Use bounded buffers and safe regular-expression behavior.
- Limit file size, event size, and parser depth where applicable; report every exclusion.
- Ignore symbolic links and explicitly document filesystem race limitations.
- Escape terminal control characters and sanitize metadata.
- Do not auto-load executable project configuration or untrusted detector plugins.
- Do not run as root or request administrator access.
- No writes to input files, temporary copies of raw histories, or persistent secret caches.

Offline operation does not mean forensic zero-impact operation: the OS can update access metadata or swap process memory.
Do not promise secure memory erasure, complete containment on a compromised host, or deletion from backups.

## 10. Detection Strategy and Accuracy

### First rules

Prioritize a limited set of recognizable credential families, such as GitHub tokens, cloud access-key formats, and private-key blocks.
Add provider-specific rules only after verifying current formats.
Recognizing an access-key identifier alone does not prove a complete usable credential pair.

Generic password assignments can be useful but should not dominate the results.
Classify obvious example values and placeholders conservatively, using tests rather than broad allowlists.

### Preserve attribution

For each detected value, preserve its original event type and physical file location when possible.
Handle JSON escaping before detection without losing the mapping back to the original record.
Test multiline private keys and credentials in nested tool results.

### Deduplication

Keep enough information to show distinct occurrences of the same credential without flooding users.
If temporary in-memory matching keys are used, do not persist them or include them in logs.
Do not silently collapse occurrences from different sources.

### Failure behavior

A malformed record must produce a sanitized error and mark coverage incomplete.
The scanner may continue with independent parseable records when boundaries are trustworthy.
Never silently switch to an unstructured whole-file scan and claim event-level attribution.

## 11. Test and Validation Plan

All fixtures must be synthetic and clearly documented as nonfunctional.
Use published detector test fixtures only when their licenses permit reuse.
Never commit real credentials or copied personal/employer histories.

### Required test groups

| Area | Cases |
|---|---|
| Detection | Recognizable fake tokens, private-key fixtures, generic assignments |
| False positives | UUIDs, checksums, example strings, placeholders, ordinary code |
| Agent parsing | Each supported schema, nested events, multiline values, escaping |
| Attribution | Correct file, physical line, session, and event type |
| Reporting | No secret or surrounding context in terminal, JSON, or errors |
| Coverage | Empty inputs, malformed records, unknown formats, unreadable files |
| Filesystem | Symlinks, disappearing files, concurrent appends, large records |
| Output safety | Terminal escapes, hostile filenames, sensitive metadata |
| Exit codes | Clean, threshold findings, incomplete scans, invalid options |
| Non-modification | Input hashes unchanged before and after scanning |
| Offline behavior | No outbound connection attempts in an isolated test environment |

### Initial acceptance targets

These are engineering targets, not measured claims:

- Detect every seeded positive case in the published supported-format fixture suite.
- Report zero raw seeded secrets or source snippets across every output path.
- Preserve every scanned file's bytes.
- Produce deterministic findings for unchanged fixtures and configuration.
- Correctly signal all intentionally incomplete scan cases.
- Publish precision and recall with counts on a separate held-out synthetic set.
- Measure throughput and peak memory on the actual personal laptop.

Passing synthetic tests does not establish production detection accuracy.
Do not promise a runtime or memory threshold before measuring the target machine.

## 12. Documentation and Release Requirements

Before calling the project usable:

- Installation and uninstall instructions.
- Exact supported agents, versions, operating systems, and file formats.
- A synthetic demo that takes only a few minutes to reproduce.
- Explanation of severity, confidence, coverage, and exit codes.
- Clear offline and privacy guarantees, with their limitations.
- A security policy with private reporting instructions.
- A contribution guide prohibiting real histories and credentials in issues.
- License notices for reused code and detection rules.
- Reproducible release instructions, checksums, and dependency pinning.
- No compatibility or security claims that tests do not support.

Choose the project license after reviewing dependency obligations.
Apache 2.0 is a candidate, not a settled decision.

## 13. Delivery Roadmap

Timing depends on available personal time; milestones matter more than dates.

### Phase 0 — Feasibility and differentiation

- Verify the personal laptop's OS, RAM, free storage, and available development tools.
- Compare existing scanners against representative synthetic agent fixtures.
- Pick one agent whose format can be verified without paid access.
- Confirm that useful event attribution or discovery is missing from existing workflows.

**Stop or pivot if:** the proposed experience is already adequately available, or the parser cannot be tested without unauthorized data.

### Phase 1 — Read-only vertical slice

- One adapter, a small rule set, masked terminal output.
- Tests for source preservation, malformed inputs, and raw-secret leakage.
- No cleanup, UI, background process, or remote component.

**Done when:** a fresh user can reproduce the synthetic demo offline.

### Phase 2 — Reliability and structured reporting

- JSON schema, explicit coverage, stable exit codes, resource limits.
- A second verified adapter.
- Performance measurements on the personal laptop.
- Security review before inviting external testers.

**Done when:** the acceptance tests pass and support limitations are documented.

### Phase 3 — Small public pilot

- Invite roughly 5–10 developers to test locally.
- Never request their raw histories or credential values.
- Collect sanitized feedback about usefulness, false positives, and unsupported formats.
- Add a third adapter only when users demonstrate demand.

**Done when:** users independently reproduce useful behavior and identify concrete missing support.

## 14. Launch and Marketing

### Honest positioning

“An offline scanner for potential secrets in supported AI coding-agent histories.”

Avoid:

- “Your agent leaked your keys” without proof of transmission or disclosure.
- “Finds every secret.”
- “Makes your agent secure.”
- “First in France” or “world first” without defensible evidence.
- Implying that intentional local credential configuration is automatically a vulnerability.

### Launch assets

1. A short screen recording with fake credentials.
2. A readable README with one-command scanning.
3. Supported-format and limitation tables.
4. Reproducible fixture tests and measured runtime.
5. A clear request for testers or adapter contributors.

### Potential communities

- [r/ClaudeAI](https://www.reddit.com/r/ClaudeAI/): only when Claude Code support exists.
- [r/cursor](https://www.reddit.com/r/cursor/): only when Cursor support exists.
- [r/opensource](https://www.reddit.com/r/opensource/): source, design, and contributions.
- [r/cybersecurity](https://www.reddit.com/r/cybersecurity/): technical evidence and limitations.
- [r/devops](https://www.reddit.com/r/devops/): developer-machine operational workflows.
- [r/developpeurs](https://www.reddit.com/r/developpeurs/): French-language feedback.

Their current promotion rules have not been verified.
Read the rules, disclose authorship, and ask moderators when needed.
Do not cross-post identical promotional text or contact users unsolicited.

### Example post

> I built a local-only scanner for potential secrets in AI coding-agent histories.
> It understands [verified supported agent] sessions and identifies whether a
> match appeared in a message or tool output. It uses established detection
> rules, does not upload data, and never modifies histories. Here is a demo
> using fake credentials, the source, and the known limitations. I would
> appreciate feedback on format compatibility and false positives. Please do
> not send me real histories or credentials.

Marketing ease is unproven. A clear demo can earn attention, but trust and actual usefulness determine adoption.

## 15. Potential Business Model

### Free core

- Offline individual scanning.
- Agent adapters.
- Sanitized reports.
- Published rules and reproducible tests.

### Possible paid needs, only after validation

- Organization policy and deployment.
- Centralized coverage reporting without collecting histories or raw findings.
- Enterprise support and maintained compatibility.
- Audit-friendly retention and handling workflows.

Do not build hosted infrastructure before a credible customer asks for a pilot.
Companies may already have adequate endpoint tools or expect agent vendors to provide this feature.
A one-time utility may not generate recurring revenue.

## 16. Success, Stop, and Pivot Criteria

### Early positive signals

- Developers actually run the scanner, not just star the repository.
- Findings reveal useful persistence not adequately surfaced by existing tools.
- Users ask for additional agent support or workflow integrations.
- Security engineers can describe a concrete internal use case.
- Agent maintainers accept a useful compatibility or redaction contribution.

### Weak signals

- Upvotes without installations or issue reports.
- Praise without willingness to test.
- Synthetic demo performance without real workflow validation.
- Downloads without repeat use.

### Stop or narrow the project if

- Existing tools solve the same workflow with little effort.
- Findings are mostly intentional configuration or low-confidence noise.
- Agent format maintenance overwhelms the value.
- Users do not trust or need an additional scanner.

### Sensible pivots

- Contribute the adapters upstream to an established scanner.
- Build a safe session-sharing preflight tool if that workflow has stronger demand.
- Focus on one well-supported agent rather than unreliable broad coverage.

## 17. Future Features — Separate Decisions

### Cleanup

Requires explicit user selection, format-aware redaction, safe backup handling, active-session checks, atomic writes, and resumption tests.
Backups themselves contain sensitive material.
Cleanup does not revoke credentials or erase provider-side copies.
Do not promise this feature until it can be validated safely.

### Editor integration

Potentially useful after the CLI: show sanitized results without exposing raw secret values.

### MCP

Not required for the product.
If introduced later, expose only explicit sanitized scan results with narrowly scoped permissions.
Never give an agent unrestricted history access or raw findings.

### Watch mode

Requires append-safe parsing, throttling, lifecycle management, and careful notification handling.
Not part of the MVP.

## 18. First Concrete Deliverable

**A read-only CLI supporting one verified history format, with an offline synthetic demo and tests showing that input files remain unchanged and output contains no raw secrets.**

Everything else—extra agents, cleanup, dashboard, MCP, company features, and fame—is optional and must earn its place through evidence.
