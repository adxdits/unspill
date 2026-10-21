package scanner

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

func (report Report) WriteJSON(writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return errors.New("could not write JSON report")
	}
	return nil
}

func (report Report) SaveJSON(path string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return errors.New("could not create report; existing files are never overwritten")
	}
	writeErr := report.WriteJSON(file)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return errors.New("could not close report file")
	}
	return nil
}

func (report Report) WriteText(writer io.Writer) error {
	var output bytes.Buffer
	fmt.Fprintln(&output, "Unspill - local, read-only scan")
	for _, root := range report.Roots {
		fmt.Fprintln(&output, "Root: "+root)
	}
	fmt.Fprintf(&output, "\n%d potential secrets found\n", len(report.Findings))
	for _, finding := range report.Findings {
		fmt.Fprintf(&output, "\n%s  %s (%s)\n", strings.ToUpper(finding.Severity), finding.RuleID, finding.Agent)
		fmt.Fprintf(&output, "  File: %s", finding.File)
		if finding.Line != nil {
			fmt.Fprintf(&output, ":%d", *finding.Line)
		}
		fmt.Fprintln(&output)
		if finding.SessionID != nil {
			fmt.Fprintln(&output, "  Session: "+*finding.SessionID)
		}
		if finding.EventType != nil {
			fmt.Fprintln(&output, "  Source: "+*finding.EventType)
		}
		fmt.Fprintln(&output, "  Value: [REDACTED]")
		fmt.Fprintln(&output, "  Confidence: "+finding.Confidence)
	}
	for _, issue := range report.Errors {
		fmt.Fprintf(&output, "\nERROR %s: %s", issue.Code, issue.Message)
		if issue.File != nil {
			fmt.Fprint(&output, " ("+*issue.File+")")
		}
		if issue.Line != nil {
			fmt.Fprintf(&output, " line %d", *issue.Line)
		}
		fmt.Fprintln(&output)
	}
	fmt.Fprintf(&output, "\nCoverage: %s for selected files and documented text fields\n", report.Status)
	fmt.Fprintf(&output, "Files: %d scanned, %d skipped, %d failed\n", report.Coverage.FilesScanned, report.Coverage.FilesSkipped, report.Coverage.FilesFailed)
	fmt.Fprintln(&output, "Network requests: disabled by design\nSource files modified: no")
	if len(report.Findings) == 0 {
		fmt.Fprintln(&output, "No matches under this scan's coverage and rules; this is not a safety guarantee.")
	}
	if _, err := writer.Write(output.Bytes()); err != nil {
		return errors.New("could not write terminal report")
	}
	return nil
}
