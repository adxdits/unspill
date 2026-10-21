# Unspill

An offline, read-only CLI for finding potential credentials in supported AI
coding-agent histories.

## Current Status

Initial Go scaffold, not a working scanner. Help and version commands work.
`scan` returns exit code `2` without accessing any histories. No agents,
formats, or discovery locations are supported yet. There are no published
binaries or download links yet.

The scope and privacy requirements are in [UNSPILL-SPEC.md](UNSPILL-SPEC.md).

## Build and Run

Install Go 1.24 or later from <https://go.dev/dl/>, then run from this directory:

```sh
go build -trimpath -o bin/unspill ./cmd/unspill
./bin/unspill --help
./bin/unspill version
./bin/unspill scan --help
```

The build currently has no third-party dependencies. The local module name
`unspill` can be changed once a public repository URL is chosen.

To remove a local build, delete `bin/unspill`. No services, configuration,
or global installation are created by these commands.

## Development

```sh
go test ./...
go vet ./...
gofmt -w cmd internal
```

CI runs formatting, tests, vet, and a build on Linux, macOS, and Windows.
This checks the scaffold, not agent compatibility or secret-detection accuracy.

Use only synthetic credentials and synthetic histories for development.
Do not scan this work computer or its agent sessions. Never submit real
credentials, private histories, or proprietary content in issues or pull requests.

## Next Milestone

Verify one agent format against permitted public sources, review an established
offline detector's embedding API and license, and implement an explicit-path
synthetic demo. Before enabling scanning, test source preservation, attribution,
resource limits, sanitized output, and operational exit codes.

No detector has been selected and no project license has been chosen yet.
There is no telemetry, update check, networking, or history access in this scaffold.
Do not treat this build as a security assessment of your machine.