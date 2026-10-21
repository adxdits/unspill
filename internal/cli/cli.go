package cli

import (
	"fmt"
	"io"
)

const help = `Unspill - local, read-only credential scanner

Usage:
  unspill --help
  unspill version
  unspill scan --help

Commands:
  version  Print the build version
  scan     Reserved for the scanner; not implemented yet

Development scaffold: no agent formats are supported yet.
This build does not discover, read, or modify agent histories.
`

const scanHelp = `Usage: unspill scan

Scanning is not implemented yet. No files will be read or modified.
A verified adapter and offline detector are required before scanning is enabled.
Running scan currently returns exit code 2 (operational error).
`

func Run(args []string, stdout, stderr io.Writer, version string) int {
	if len(args) == 0 {
		return writeOutput(stdout, stderr, help)
	}

	switch args[0] {
	case "help", "--help", "-h":
		if len(args) != 1 {
			return fail(stderr, "invalid help arguments; use unspill --help")
		}
		return writeOutput(stdout, stderr, help)
	case "version", "--version":
		if len(args) != 1 {
			return fail(stderr, "version accepts no arguments")
		}
		return writeOutput(stdout, stderr, "unspill "+version+"\n")
	case "scan":
		if len(args) == 2 && (args[1] == "--help" || args[1] == "-h") {
			return writeOutput(stdout, stderr, scanHelp)
		}
		return fail(stderr, "scanning is not implemented; no files were scanned")
	default:
		return fail(stderr, "unknown command; use unspill --help")
	}
}

func writeOutput(stdout, stderr io.Writer, text string) int {
	if _, err := io.WriteString(stdout, text); err != nil {
		return fail(stderr, "could not write output")
	}
	return 0
}

func fail(stderr io.Writer, message string) int {
	fmt.Fprintln(stderr, "unspill: "+message)
	return 2
}