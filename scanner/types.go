package scanner

import "github.com/adxdits/unspill/internal/agents"

const Version = "0.2.0-dev"

type Options struct {
	Paths         []string
	Agent         string
	Discover      bool
	MaxFileBytes  int64
	MaxEventBytes int
	MaxFiles      int
	MaxFindings   int
}

type Coverage struct {
	FilesScanned int `json:"files_scanned"`
	FilesSkipped int `json:"files_skipped"`
	FilesFailed  int `json:"files_failed"`
}

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

type ScanError struct {
	Code    string  `json:"code"`
	Message string  `json:"message"`
	File    *string `json:"file"`
	Line    *int    `json:"line"`
}

type Report struct {
	SchemaVersion string      `json:"schema_version"`
	ToolVersion   string      `json:"tool_version"`
	Status        string      `json:"status"`
	Roots         []string    `json:"roots"`
	Coverage      Coverage    `json:"coverage"`
	Findings      []Finding   `json:"findings"`
	Errors        []ScanError `json:"errors"`
}

func SupportedAgents() []string {
	return agents.Names()
}

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
