package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{name: "no arguments", wantStdout: help},
		{name: "long help", args: []string{"--help"}, wantStdout: help},
		{name: "short help", args: []string{"-h"}, wantStdout: help},
		{name: "help command", args: []string{"help"}, wantStdout: help},
		{name: "version", args: []string{"version"}, wantStdout: "unspill test-version\n"},
		{name: "version flag", args: []string{"--version"}, wantStdout: "unspill test-version\n"},
		{name: "scan help", args: []string{"scan", "--help"}, wantStdout: scanHelp},
		{name: "scan short help", args: []string{"scan", "-h"}, wantStdout: scanHelp},
		{name: "invalid scan agent", args: []string{"scan", "--agent", "unsupported"}, wantCode: 2, wantStderr: "unspill: invalid scan options; use unspill scan --help\n"},
		{name: "invalid scan option", args: []string{"scan", "--unknown"}, wantCode: 2, wantStderr: "unspill: invalid scan options; use unspill scan --help\n"},
		{name: "unknown command", args: []string{"unknown"}, wantCode: 2, wantStderr: "unspill: unknown command; use unspill --help\n"},
		{name: "extra help arguments", args: []string{"--help", "extra"}, wantCode: 2, wantStderr: "unspill: invalid help arguments; use unspill --help\n"},
		{name: "extra version arguments", args: []string{"version", "extra"}, wantCode: 2, wantStderr: "unspill: version accepts no arguments\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(test.args, &stdout, &stderr, "test-version")
			if code != test.wantCode {
				t.Errorf("exit code = %d, want %d", code, test.wantCode)
			}
			if stdout.String() != test.wantStdout {
				t.Error("unexpected standard output")
			}
			if stderr.String() != test.wantStderr {
				t.Error("unexpected standard error")
			}
		})
	}
}

func TestArgumentsAreNeverEchoed(t *testing.T) {
	untrusted := "synthetic-private-value\x1b[31m\nforged output"
	for _, args := range [][]string{
		{untrusted},
		{"version", untrusted},
		{"--help", untrusted},
		{"scan", "--unknown", untrusted},
	} {
		var stdout, stderr bytes.Buffer
		if code := Run(args, &stdout, &stderr, "test-version"); code != 2 {
			t.Fatal("invalid or unavailable commands must return exit code 2")
		}
		output := stdout.String() + stderr.String()
		for _, forbidden := range []string{"synthetic-private-value", "\x1b", "forged output"} {
			if strings.Contains(output, forbidden) {
				t.Fatal("output contains untrusted argument content")
			}
		}
	}
}

type failingWriter struct{}

func (failingWriter) Write(content []byte) (int, error) {
	return 0, errors.New("synthetic-private-error")
}

func TestOutputFailure(t *testing.T) {
	var stderr bytes.Buffer
	if code := Run([]string{"version"}, failingWriter{}, &stderr, "test-version"); code != 2 {
		t.Fatal("output failure must return exit code 2")
	}
	if stderr.String() != "unspill: could not write output\n" {
		t.Fatal("output failure must use a static sanitized error")
	}
}

func TestScanSyntheticDemo(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("CLI refuses root")
	}
	var stdout, stderr bytes.Buffer
	code := Run([]string{"scan", "--path", "../../testdata/demo", "--format", "json"}, &stdout, &stderr, "test-version")
	var report struct {
		Status   string `json:"status"`
		Findings []struct {
			Value string `json:"value"`
		} `json:"findings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal("expected a JSON report without progress on stdout")
	}
	if code != 1 || report.Status != "complete" || len(report.Findings) != 4 || stderr.Len() != 0 {
		t.Fatalf("demo failed: code %d, status %s, findings %d", code, report.Status, len(report.Findings))
	}
	for _, finding := range report.Findings {
		if finding.Value != "[REDACTED]" {
			t.Fatal("finding value must always be masked")
		}
	}
	if strings.Contains(stdout.String(), "7Fj29kLmN4pQ") {
		t.Fatal("raw fixture value leaked")
	}
}

func TestScanReportDestination(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("CLI refuses root")
	}
	path := filepath.Join(t.TempDir(), "report.json")
	args := []string{"scan", "--agent", "generic", "--path", "../../testdata/generic.json", "--format", "json", "--output", path, "--fail-on", "none"}
	var stdout, stderr bytes.Buffer
	if code := Run(args, &stdout, &stderr, "test-version"); code != 0 || stdout.Len() != 0 {
		t.Fatal("sanitized report export failed")
	}
	if code := Run(args, &stdout, &stderr, "test-version"); code != 2 {
		t.Fatal("existing report must not be overwritten")
	}
}
