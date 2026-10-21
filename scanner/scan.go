package scanner

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/adxdits/unspill/internal/agents"
	"github.com/adxdits/unspill/internal/detect"
)

const jsonlExtension = ".jsonl"

type run struct {
	options    Options
	report     Report
	engine     *detect.Engine
	home       string
	seen       map[string]bool
	entries    int
	totalBytes int64
	events     int
	stopped    bool
}

func Scan(ctx context.Context, options Options) Report {
	state := &run{options: options, report: Report{
		SchemaVersion: "1", ToolVersion: Version, Status: "complete",
		Roots: []string{}, Findings: []Finding{}, Errors: []ScanError{},
	}, seen: make(map[string]bool)}
	if !state.setup() {
		return state.report
	}
	paths := state.selectedPaths()
	if state.report.Status != "complete" {
		return state.report
	}
	sort.Strings(paths)
	for _, path := range paths {
		state.visitRoot(ctx, path)
	}
	if state.report.Coverage.FilesScanned == 0 && state.report.Coverage.FilesFailed == 0 {
		state.problem("no_supported_input", "no supported history files were scanned; input is empty or excluded", "", 0)
	}
	if state.events == 0 && state.report.Status == "complete" {
		state.problem("empty_input", "selected files contain no scannable text events", "", 0)
	}
	return state.report
}

func (state *run) setup() bool {
	state.options = withDefaults(state.options)
	if !agents.Valid(state.options.Agent) || (state.options.Discover && len(state.options.Paths) != 0) {
		state.problem("invalid_options", "invalid agent or conflicting scan paths", "", 0)
		return false
	}
	if os.Geteuid() == 0 {
		state.problem("permission_error", "do not run Unspill as root", "", 0)
		return false
	}
	if !validLimits(state.options) {
		state.problem("invalid_options", "resource limits are outside supported bounds", "", 0)
		return false
	}
	state.home, _ = os.UserHomeDir()
	engine, err := detect.New()
	if err != nil {
		state.problem("detector_error", "could not initialize offline detector", "", 0)
		return false
	}
	state.engine = engine
	return true
}

func validLimits(options Options) bool {
	return options.MaxFileBytes > 0 && options.MaxFileBytes <= 64<<20 && options.MaxEventBytes > 0 && options.MaxEventBytes <= 16<<20 && options.MaxFiles > 0 && options.MaxFiles <= 10000 && options.MaxFindings > 0 && options.MaxFindings <= 10000
}

func withDefaults(options Options) Options {
	if options.Agent == "" {
		options.Agent = "auto"
	}
	if options.MaxFileBytes == 0 {
		options.MaxFileBytes = 64 << 20
	}
	if options.MaxEventBytes == 0 {
		options.MaxEventBytes = 1 << 20
	}
	if options.MaxFiles == 0 {
		options.MaxFiles = 10000
	}
	if options.MaxFindings == 0 {
		options.MaxFindings = 10000
	}
	return options
}

func (state *run) selectedPaths() []string {
	if !state.options.Discover {
		return append([]string(nil), state.options.Paths...)
	}
	if state.home == "" {
		state.problem("discovery_error", "could not determine the current user's home directory", "", 0)
		return nil
	}
	var paths []string
	for _, root := range []struct{ agent, directory string }{
		{"claude-code", ".claude/projects"},
		{"copilot-cli", ".copilot/session-state"},
		{"codex", ".codex/sessions"},
	} {
		if state.options.Agent == "auto" || state.options.Agent == root.agent {
			paths = append(paths, filepath.Join(state.home, filepath.FromSlash(root.directory)))
		}
	}
	return paths
}

func (state *run) visitRoot(ctx context.Context, path string) {
	state.report.Roots = append(state.report.Roots, state.safePath(path))
	if state.options.Discover {
		if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
			return
		}
	}
	if err := noSymlinkComponents(path); err != nil {
		state.report.Coverage.FilesSkipped++
		state.problem("unsafe_path", "input path is inaccessible or contains a symbolic link", path, 0)
		return
	}
	state.walk(ctx, path, 0)
}

func noSymlinkComponents(path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	for current := absolute; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("symbolic link")
		}
		if filepath.Dir(current) == current {
			return nil
		}
	}
}

func (state *run) walk(ctx context.Context, path string, depth int) {
	if state.stopped {
		return
	}
	if ctx.Err() != nil {
		state.problem("cancelled", "scan cancelled; coverage is incomplete", path, 0)
		state.stopped = true
		return
	}
	state.entries++
	if state.entries > 50000 || depth > 16 {
		state.problem("resource_limit", "directory traversal limit exceeded", path, 0)
		state.stopped = true
		return
	}
	info, err := os.Lstat(path)
	if err != nil {
		state.report.Coverage.FilesFailed++
		state.problem("read_error", "could not inspect input", path, 0)
		return
	}
	if info.Mode()&os.ModeSymlink != 0 {
		state.report.Coverage.FilesSkipped++
		state.problem("symlink_skipped", "symbolic link excluded", path, 0)
		return
	}
	if !info.IsDir() && state.options.Discover && !state.historyCandidate(path) {
		return
	}
	if info.IsDir() {
		state.walkDirectory(ctx, path, depth)
		return
	}
	state.selectFile(ctx, path, info)
}

func (state *run) walkDirectory(ctx context.Context, path string, depth int) {
	directory, err := os.Open(path)
	if err != nil {
		state.problem("read_error", "could not read directory", path, 0)
		return
	}
	defer directory.Close()
	var names []string
	for {
		batch, err := directory.Readdirnames(128)
		names = append(names, batch...)
		if len(names)+state.entries > 50000 {
			state.problem("resource_limit", "directory entry limit exceeded", path, 0)
			state.stopped = true
			return
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			state.problem("read_error", "could not enumerate directory", path, 0)
			return
		}
	}
	sort.Strings(names)
	for _, name := range names {
		state.walk(ctx, filepath.Join(path, name), depth+1)
	}
}

func (state *run) selectFile(ctx context.Context, path string, info os.FileInfo) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		state.problem("read_error", "could not resolve input path", path, 0)
		return
	}
	if state.seen[absolute] {
		return
	}
	state.seen[absolute] = true
	if state.report.Coverage.FilesScanned+state.report.Coverage.FilesFailed >= state.options.MaxFiles {
		state.report.Coverage.FilesSkipped++
		state.problem("resource_limit", "file count limit exceeded", path, 0)
		state.stopped = true
		return
	}
	extension := strings.ToLower(filepath.Ext(path))
	if !info.Mode().IsRegular() || (extension != jsonlExtension && extension != ".json") {
		state.report.Coverage.FilesSkipped++
		state.problem("unsupported_input", "only regular JSON and JSONL history files are supported", path, 0)
		return
	}
	if info.Size() > state.options.MaxFileBytes || state.totalBytes+info.Size() > 256<<20 {
		state.report.Coverage.FilesSkipped++
		state.problem("resource_limit", "input exceeds file or total byte limit", path, 0)
		return
	}
	state.scanFile(ctx, path, info, extension)
}

func (state *run) historyCandidate(path string) bool {
	relative, err := filepath.Rel(state.home, path)
	if err != nil {
		return false
	}
	relative = filepath.ToSlash(relative)
	name := filepath.Base(path)
	switch {
	case strings.HasPrefix(relative, ".claude/projects/"):
		return strings.HasSuffix(name, jsonlExtension)
	case strings.HasPrefix(relative, ".copilot/session-state/"):
		return name == "events.jsonl"
	case strings.HasPrefix(relative, ".codex/sessions/"):
		return strings.HasPrefix(name, "rollout-") && strings.HasSuffix(name, jsonlExtension)
	default:
		return false
	}
}

func (state *run) scanFile(ctx context.Context, path string, original os.FileInfo, extension string) {
	errorCount := len(state.report.Errors)
	file, err := openInput(path)
	if err != nil {
		state.report.Coverage.FilesFailed++
		state.problem("read_error", "could not open input file", path, 0)
		return
	}
	defer file.Close()
	before, err := file.Stat()
	if err != nil || !before.Mode().IsRegular() || !os.SameFile(original, before) || before.Size() != original.Size() {
		state.report.Coverage.FilesFailed++
		state.problem("input_changed", "input changed before reading", path, 0)
		return
	}
	state.totalBytes += before.Size()
	reader := io.LimitReader(file, before.Size()+1)
	parser := agents.Parser{Agent: state.options.Agent}
	if extension == jsonlExtension {
		state.readJSONL(ctx, &parser, reader, path)
	} else {
		state.readJSON(ctx, &parser, reader, path)
	}
	if inputChanged(file, path, before) {
		state.problem("input_changed", "input changed during scan; coverage is incomplete", path, 0)
	}
	if state.stopped || len(state.report.Errors) != errorCount {
		state.report.Coverage.FilesFailed++
	} else {
		state.report.Coverage.FilesScanned++
	}
}

func inputChanged(file *os.File, path string, before os.FileInfo) bool {
	after, statErr := file.Stat()
	current, pathErr := os.Lstat(path)
	return statErr != nil || pathErr != nil || current.Mode()&os.ModeSymlink != 0 || !os.SameFile(before, current) || after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime())
}

func (state *run) readJSONL(ctx context.Context, parser *agents.Parser, reader io.Reader, path string) {
	lines := bufio.NewScanner(reader)
	lines.Buffer(make([]byte, 4096), state.options.MaxEventBytes+1)
	line := 0
	for lines.Scan() {
		line++
		if len(bytes.TrimSpace(lines.Bytes())) == 0 {
			continue
		}
		state.record(ctx, parser, lines.Bytes(), path, line)
		if state.stopped {
			break
		}
	}
	if err := lines.Err(); err != nil {
		state.problem("read_or_event_limit", "could not read record or record exceeds event byte limit", path, line+1)
	}
}

func (state *run) readJSON(ctx context.Context, parser *agents.Parser, reader io.Reader, path string) {
	data, err := io.ReadAll(io.LimitReader(reader, int64(state.options.MaxEventBytes)+1))
	if err != nil {
		state.problem("read_error", "could not read JSON export", path, 0)
	} else if len(data) > state.options.MaxEventBytes {
		state.problem("resource_limit", "JSON export exceeds event byte limit", path, 0)
	} else if len(bytes.TrimSpace(data)) == 0 {
		state.problem("empty_input", "input file is empty", path, 0)
	} else {
		state.record(ctx, parser, data, path, 1)
	}
}

func (state *run) record(ctx context.Context, parser *agents.Parser, data []byte, path string, line int) {
	if ctx.Err() != nil {
		state.problem("cancelled", "scan cancelled; coverage is incomplete", path, line)
		state.stopped = true
		return
	}
	if len(data) > state.options.MaxEventBytes {
		state.problem("resource_limit", "record exceeds event byte limit", path, line)
		return
	}
	events, err := parser.Parse(data, line)
	if err != nil {
		state.problem("parse_error", err.Error(), path, line)
		return
	}
	for _, event := range events {
		state.events++
		for _, match := range state.engine.Scan(event.Text) {
			if len(state.report.Findings) >= state.options.MaxFindings {
				state.problem("resource_limit", "finding count limit exceeded", path, event.Line)
				state.stopped = true
				return
			}
			state.report.Findings = append(state.report.Findings, Finding{
				ID:    fmt.Sprintf("finding-%d", len(state.report.Findings)+1),
				Agent: event.Agent, RuleID: match.RuleID, Severity: match.Severity, Confidence: match.Confidence,
				File: state.safePath(path), Line: pointer(event.Line),
				SessionID: optional(state.safe(event.SessionID)), EventType: optional(event.Type), Value: "[REDACTED]",
			})
		}
	}
}

func pointer[value any](item value) *value { return &item }

func optional(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func (state *run) safe(value string) string {
	if len(value) > 2048 {
		return "[OMITTED]"
	}
	if state.engine != nil && len(state.engine.Scan(value)) != 0 {
		return "[REDACTED]"
	}
	quoted := strconv.QuoteToASCII(value)
	return quoted[1 : len(quoted)-1]
}

func (state *run) safePath(path string) string {
	if state.home != "" && (path == state.home || strings.HasPrefix(path, state.home+string(filepath.Separator))) {
		path = "<home>" + strings.TrimPrefix(path, state.home)
	}
	return state.safe(filepath.ToSlash(path))
}

func (state *run) problem(code, message, path string, line int) {
	state.report.Status = "incomplete"
	if len(state.report.Errors) >= 1000 {
		state.stopped = true
		return
	}
	var location *int
	if line > 0 {
		location = pointer(line)
	}
	state.report.Errors = append(state.report.Errors, ScanError{Code: code, Message: message, File: optional(state.safePath(path)), Line: location})
}
