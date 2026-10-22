# Unspill

Scan local coding-agent histories for credentials using embedded Gitleaks rules.
Available as a Go library and CLI. Scanning runs offline, leaves inputs unchanged,
and redacts credential values in reports.

## Install

Requires Go 1.24 or newer. Install the current source revision:

```sh
go install github.com/adxdits/unspill/cmd/unspill@main
```

The executable is installed in `GOBIN`, or `$(go env GOPATH)/bin` by default;
that directory must be on `PATH`. Release archives target macOS, Linux, and
Windows on AMD64 and ARM64, with SHA-256 checksums. The `v0.2.0` tag has not yet
been published; use `@main` until a tagged release is available.

## CLI

```sh
unspill scan --path ./session.jsonl --format json --output report.json
unspill scan --agent generic --path ./export.json
unspill scan --help
```

For a demo with four fake credentials, run from a checkout:

```sh
git clone https://github.com/adxdits/unspill.git
cd unspill
go run ./cmd/unspill scan --path ./testdata/demo --fail-on none
```

Without `--path`, the CLI searches known Claude Code, Copilot CLI, and Codex directories.
Only scan files you are authorized to read; the CLI refuses to run as root.

Exit codes: **0** complete below the failure threshold, **1** complete with
threshold findings, **2** invalid options or incomplete coverage. The default
threshold is `high`; `--fail-on none` disables findings-based failure.

## Go Library

Add the library to your Go module:

```sh
go get github.com/adxdits/unspill/scanner@main
```

Run this example from the repository root to use the included fixtures. In your
application, replace `testdata/demo` with the histories you intend to scan.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/adxdits/unspill/scanner"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	report := scanner.Scan(ctx, scanner.Options{Paths: []string{"testdata/demo"}})
	if err := report.Err(); err != nil {
		log.Printf("incomplete scan: %v", err)
	}
	for _, finding := range report.Findings {
		fmt.Printf("%s: %s (%s)\n", finding.File, finding.RuleID, finding.Severity)
	}
}
```

`Scan` returns partial findings when individual records or blocks fail; findings
alone are not Go errors. Use `errors.As` with `scanner.ScanError` for operational
errors, or `errors.Is` with `context.Canceled` and `context.DeadlineExceeded`.
`WriteJSON`, `WriteText`, and `SaveJSON` export reports; `SaveJSON` never
overwrites an existing file. `Options` accepts resource limits with bounded
defaults. Explicit paths are required unless `Discover` is true. Independent
scans may run concurrently. Cancellation is checked between records and events,
not during an individual detection rule.

## Coverage And Safety

Supported inputs: Claude Code JSONL, Copilot CLI schema-1 JSONL, VS Code JSON chat
exports, Codex JSONL, and explicit generic message exports. See
[format constraints](docs/formats.md). Compatibility is tested with synthetic
fixtures, not certified against every client version.

Unknown records, empty inputs, and resource exclusions make coverage incomplete.
Symbolic links are excluded. Reports contain paths and session metadata, so treat
them as sensitive. No matches is not a safety guarantee; a match is not proof of
compromise. There are no uploads, telemetry, or credential-verification requests.

## Development And Release

```sh
go test -race ./...
go vet ./...
go build -trimpath -o bin/unspill ./cmd/unspill
```

Use synthetic test data only. For a release, update `scanner.Version`, commit,
and create a matching `vMAJOR.MINOR.PATCH` tag. Tagged releases require native
tests and CLI smoke checks on all three operating systems using Go 1.24 and
stable, parser fuzzing, and reachable-vulnerability checks for all six targets.
Release binaries use stable Go and include checksums, dependency licenses, and
HCL source. The workflow creates a **draft** GitHub release for review before
publication.

## License

[MIT](LICENSE). See [third-party notices](THIRD_PARTY_NOTICES.md) for dependencies.