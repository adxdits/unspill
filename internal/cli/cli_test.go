package cli

import (
	"bytes"
	"errors"
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
		{name: "scan unavailable", args: []string{"scan"}, wantCode: 2, wantStderr: "unspill: scanning is not implemented; no files were scanned\n"},
		{name: "explicit path unavailable", args: []string{"scan", "--path", "/not-a-real-history"}, wantCode: 2, wantStderr: "unspill: scanning is not implemented; no files were scanned\n"},
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
		{"scan", "--path", untrusted},
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