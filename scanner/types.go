// Package scanner finds potential credentials in local coding-agent histories.
// Scans are offline and read-only; finding values are always redacted.
package scanner

import (
	"errors"

	"github.com/adxdits/unspill/internal/agents"
)

// Version is the library release version.
const Version = "0.2.0"

// Options selects inputs and bounds scan resources. Zero limits use defaults.
// Paths must be explicit unless Discover is true; the two modes cannot be combined.
type Options struct {
	Paths         []string
	Agent         string
	Discover      bool
	MaxFileBytes  int64 // Default and maximum: 64 MiB.
	MaxEventBytes int   // Default: 1 MiB; maximum: 16 MiB, excluding JSONL line endings.
	MaxFiles      int   // Default and maximum: 10000.
	MaxFindings   int   // Default and maximum: 10000.
}

// Coverage counts successfully scanned, excluded, and failed files.
type Coverage struct {
	FilesScanned int `json:"files_scanned"`
	FilesSkipped int `json:"files_skipped"`
	FilesFailed  int `json:"files_failed"`
}

// Finding identifies a potential credential without retaining its value or context.
// Line is a physical source line; nil attribution fields indicate unavailable data.
type Finding struct {
	ID         string  `json:"id"`
	Agent      string  `json:"agent"`
	RuleID     string  `json:"rule_id"`
	Severity   string  `json:"severity"`
	Confidence string  `json:"confidence"`
	File       string  `json:"file"`
	Line       *int    `json:"line"`
	SessionID  *string `json:"session_id"`
	EventType  *string `json:"event_type"`
	Value      string  `json:"value"`
}

// ScanError describes incomplete coverage using sanitized source metadata.
type ScanError struct {
	Code    string  `json:"code"`
	Message string  `json:"message"`
	File    *string `json:"file"`
	Line    *int    `json:"line"`
}

// Error returns the error code and message without source metadata.
func (issue ScanError) Error() string {
	return issue.Code + ": " + issue.Message
}

// Report contains findings and coverage for one scan. Status is complete or incomplete.
type Report struct {
	SchemaVersion string      `json:"schema_version"`
	ToolVersion   string      `json:"tool_version"`
	Status        string      `json:"status"`
	Roots         []string    `json:"roots"`
	Coverage      Coverage    `json:"coverage"`
	Findings      []Finding   `json:"findings"`
	Errors        []ScanError `json:"errors"`
}

// SupportedAgents returns adapter names. The auto selector is not included.
func SupportedAgents() []string {
	return agents.Names()
}

// Err joins operational errors, or returns nil for a complete scan.
// Findings alone are not errors. Use errors.As to extract a ScanError.
func (report Report) Err() error {
	issues := make([]error, 0, len(report.Errors))
	for _, issue := range report.Errors {
		issues = append(issues, issue)
	}
	if len(issues) == 0 && report.Status != "complete" {
		return errors.New("scan is incomplete")
	}
	return errors.Join(issues...)
}

// ExitCode returns 0 for success, 1 for threshold findings, or 2 for incomplete
// coverage or an invalid threshold. Threshold is high, medium, low, or none.
func (report Report) ExitCode(threshold string) int {
	levels := map[string]int{"none": 4, "high": 3, "medium": 2, "low": 1}
	level, valid := levels[threshold]
	if !valid || report.Status != "complete" || len(report.Errors) != 0 {
		return 2
	}
	for _, finding := range report.Findings {
		if levels[finding.Severity] >= level {
			return 1
		}
	}
	return 0
}
