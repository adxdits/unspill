package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/adxdits/unspill/scanner"
)

const help = `Unspill - local, read-only credential scanner

Usage:
  unspill --help
  unspill version
	unspill agents
	unspill scan --path ./synthetic-history.jsonl
  unspill scan --help

Commands:
  version  Print the build version
	agents   List experimental history adapters
	scan     Scan supported local histories without modifying them

Offline detection uses embedded Gitleaks rules. No credential verification.
Only scan histories you are authorized to read. Not all agents are supported.
`

const scanHelp = `Usage: unspill scan [options]

	--path PATH           Explicit file or directory; repeat for multiple paths
	--agent NAME          auto (default), claude-code, copilot-cli, copilot-vscode,
												codex, generic
	--format FORMAT       text (default) or json
	--output PATH         Create an owner-only JSON report; never overwrite
	--fail-on LEVEL       high (default), medium, low, none
	--max-file-bytes N    Maximum file size (default 67108864)
	--max-event-bytes N   Maximum JSONL record or JSON export size (default 1048576)
	--max-files N         Maximum history files (default 10000)
	--max-findings N      Maximum findings (default 10000)
	--timeout DURATION    Scan deadline (default 30s)

Without --path, discovery is restricted to the current user's known Claude Code,
Copilot CLI, and Codex history roots. VS Code and generic exports need --path.
Unknown records, exclusions, and incomplete scans return exit code 2.
Complete scans return 1 at the failure threshold, otherwise 0.
Compatibility is experimental and limited to documented text fields.
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
	case "agents":
		if len(args) != 1 {
			return fail(stderr, "agents accepts no arguments")
		}
		return writeOutput(stdout, stderr, "Experimental adapters (not all client versions):\n"+strings.Join(scanner.SupportedAgents(), "\n")+"\n\nCopilot CLI logs and VS Code JSON exports are separate formats.\nVS Code incremental storage, Cursor databases, and browser chats are unsupported.\n")
	case "scan":
		if len(args) == 2 && (args[1] == "--help" || args[1] == "-h") {
			return writeOutput(stdout, stderr, scanHelp)
		}
		return runScan(args[1:], stdout, stderr, version)
	default:
		return fail(stderr, "unknown command; use unspill --help")
	}
}

type pathsFlag []string

func (paths *pathsFlag) String() string { return "" }

func (paths *pathsFlag) Set(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("empty path")
	}
	*paths = append(*paths, path)
	return nil
}

func runScan(args []string, stdout, stderr io.Writer, version string) int {
	flags := flag.NewFlagSet("scan", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var paths pathsFlag
	var options scanner.Options
	flags.Var(&paths, "path", "history file or directory")
	flags.StringVar(&options.Agent, "agent", "auto", "history adapter")
	format := flags.String("format", "text", "report format")
	output := flags.String("output", "", "JSON report destination")
	threshold := flags.String("fail-on", "high", "failure threshold")
	flags.Int64Var(&options.MaxFileBytes, "max-file-bytes", 64<<20, "file byte limit")
	flags.IntVar(&options.MaxEventBytes, "max-event-bytes", 1<<20, "event byte limit")
	flags.IntVar(&options.MaxFiles, "max-files", 10000, "file count limit")
	flags.IntVar(&options.MaxFindings, "max-findings", 10000, "finding count limit")
	timeout := flags.Duration("timeout", 30*time.Second, "scan deadline")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		return fail(stderr, "invalid scan options; use unspill scan --help")
	}
	validAgent := options.Agent == "auto" || slices.Contains(scanner.SupportedAgents(), options.Agent)
	validFormat := slices.Contains([]string{"text", "json"}, *format)
	validThreshold := slices.Contains([]string{"high", "medium", "low", "none"}, *threshold)
	if !validAgent || !validFormat || !validThreshold || (*output != "" && *format != "json") || *timeout <= 0 || options.MaxFileBytes <= 0 || options.MaxEventBytes <= 0 || options.MaxFiles <= 0 || options.MaxFindings <= 0 {
		return fail(stderr, "invalid scan options; use unspill scan --help")
	}
	if os.Geteuid() == 0 {
		return fail(stderr, "do not run Unspill as root")
	}
	options.Paths = paths
	options.Discover = len(paths) == 0
	if err := discloseRoots(options, stderr); err != nil {
		return fail(stderr, err.Error())
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	report := scanner.Scan(ctx, options)
	report.ToolVersion = version
	var err error
	if *output != "" {
		err = report.SaveJSON(*output)
	} else if *format == "json" {
		err = report.WriteJSON(stdout)
	} else {
		err = report.WriteText(stdout)
	}
	if err != nil {
		return fail(stderr, "could not write report; existing files are never overwritten")
	}
	return report.ExitCode(*threshold)
}

func discloseRoots(options scanner.Options, stderr io.Writer) error {
	if !options.Discover {
		return nil
	}
	if options.Agent == "generic" || options.Agent == "copilot-vscode" {
		return errors.New("this export adapter requires an explicit --path")
	}
	roots := []string{}
	for _, root := range []struct{ agent, path string }{
		{"claude-code", "<home>/.claude/projects"}, {"copilot-cli", "<home>/.copilot/session-state"}, {"codex", "<home>/.codex/sessions"},
	} {
		if options.Agent == "auto" || options.Agent == root.agent {
			roots = append(roots, root.path)
		}
	}
	if _, err := fmt.Fprintln(stderr, "Selected history roots: "+strings.Join(roots, ", ")); err != nil {
		return errors.New("could not write selected history roots")
	}
	return nil
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
