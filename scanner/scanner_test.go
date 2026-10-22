package scanner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/adxdits/unspill/internal/agents"
)

func fixture(t *testing.T, name, content string) string {
	t.Helper()
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func syntheticToken() string { return "ghp_" + "7Fj29kLmN4pQr8StU1vWx5YzA0bCd3EfG6hI" }

func TestEndToEndAdapters(t *testing.T) {
	for _, test := range []struct{ name, agent, content, kind string }{
		{"claude.jsonl", "auto", `{"type":"user","sessionId":"synthetic-session","message":{"content":"%s"}}`, "user_message"},
		{"copilot.jsonl", "auto", `{"type":"tool.execution_complete","data":{"result":{"content":"%s"}}}`, "tool_output"},
		{"codex.jsonl", "auto", `{"type":"response_item","payload":{"type":"function_call_output","output":"%s"}}`, "tool_output"},
		{"vscode.json", "auto", `{"version":3,"requests":[{"message":"ordinary request","response":[{"value":"%s"}]}]}`, "assistant_message"},
		{"generic.json", "generic", `{"messages":[{"role":"assistant","content":"%s"}]}`, "assistant_message"},
	} {
		t.Run(test.name, func(t *testing.T) {
			content := strings.Replace(test.content, "%s", syntheticToken(), 1)
			path := fixture(t, test.name, content)
			before := sha256.Sum256([]byte(content))
			report := Scan(context.Background(), Options{Paths: []string{path}, Agent: test.agent})
			if report.Status != "complete" || len(report.Findings) != 1 || report.ExitCode("high") != 1 || report.ExitCode("none") != 0 {
				t.Fatalf("unexpected coverage or findings: status %s, findings %d, errors %v", report.Status, len(report.Findings), report.Errors)
			}
			if report.Findings[0].EventType == nil || *report.Findings[0].EventType != test.kind || *report.Findings[0].Line != 1 {
				t.Fatal("incorrect source attribution")
			}
			assertSanitizedReport(t, report)
			assertPreservedAndDeterministic(t, path, test.agent, before, report)
		})
	}
}

func assertSanitizedReport(t *testing.T, report Report) {
	t.Helper()
	var terminal, structured bytes.Buffer
	if err := report.WriteText(&terminal); err != nil {
		t.Fatal(err)
	}
	if err := report.WriteJSON(&structured); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(terminal.String()+structured.String(), syntheticToken()) {
		t.Fatal("credential leaked")
	}
	if !json.Valid(structured.Bytes()) {
		t.Fatal("report is not valid JSON")
	}
}

func assertPreservedAndDeterministic(t *testing.T, path, agent string, before [32]byte, report Report) {
	t.Helper()
	after, err := os.ReadFile(path)
	if err != nil || sha256.Sum256(after) != before {
		t.Fatal("input changed")
	}
	if next := Scan(context.Background(), Options{Paths: []string{path}, Agent: agent}); !reflect.DeepEqual(report, next) {
		t.Fatal("scan is nondeterministic")
	}
}

func TestIncompleteCoverage(t *testing.T) {
	for _, test := range []struct {
		name, content string
		options       Options
	}{
		{"malformed", "{\n", Options{}},
		{"unknown", `{"type":"future","secret":"` + syntheticToken() + `"}`, Options{}},
		{"empty", "\n\n", Options{}},
		{"file limit", strings.Repeat("x", 100), Options{MaxFileBytes: 10}},
		{"event limit", strings.Repeat("x", 100), Options{MaxEventBytes: 10}},
		{"json depth", strings.Repeat("[", 70) + strings.Repeat("]", 70), Options{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := fixture(t, "synthetic.jsonl", test.content)
			test.options.Paths = []string{path}
			report := Scan(context.Background(), test.options)
			if report.ExitCode("none") != 2 || len(report.Errors) == 0 {
				t.Fatal("incomplete input must return operational failure")
			}
			var output bytes.Buffer
			report.WriteJSON(&output)
			if strings.Contains(output.String(), syntheticToken()) {
				t.Fatal("error leaked content")
			}
		})
	}
}

func TestMalformedLineContinuesAndPreservesPhysicalLocation(t *testing.T) {
	content := "{broken\n\n" + `{"type":"user","message":{"content":"` + syntheticToken() + `"}}` + "\n"
	report := Scan(context.Background(), Options{Paths: []string{fixture(t, "synthetic.jsonl", content)}})
	if report.ExitCode("high") != 2 || len(report.Findings) != 1 || *report.Findings[0].Line != 3 || report.Coverage.FilesFailed != 1 {
		t.Fatal("malformed line must not hide later independent findings")
	}
}

func TestJSONLRecordByteLimit(t *testing.T) {
	content := `{"type":"user","message":{"content":"ordinary"}}`
	for _, ending := range []string{"", "\n", "\r\n"} {
		t.Run("ending="+strconv.Quote(ending), func(t *testing.T) {
			path := fixture(t, "clean.jsonl", content+ending)
			report := Scan(context.Background(), Options{Paths: []string{path}, MaxEventBytes: len(content)})
			if err := report.Err(); err != nil {
				t.Fatalf("a record exactly at the byte limit must be accepted: %v", err)
			}
			report = Scan(context.Background(), Options{Paths: []string{path}, MaxEventBytes: len(content) - 1})
			if report.Err() == nil {
				t.Fatal("an oversized record must fail")
			}
		})
	}
	path := fixture(t, "blank.jsonl", "   \n"+content+"\n")
	report := Scan(context.Background(), Options{Paths: []string{path}, MaxEventBytes: 2})
	if len(report.Errors) == 0 || report.Errors[0].Code != "resource_limit" || report.Errors[0].Line == nil || *report.Errors[0].Line != 1 {
		t.Fatal("oversized whitespace records must not bypass the byte limit")
	}
}

func TestEmptyFileDoesNotHideBehindScannedFiles(t *testing.T) {
	clean := fixture(t, "clean.jsonl", `{"type":"user","message":{"content":"ordinary"}}`)
	blank := fixture(t, "blank.jsonl", "\n\n")
	report := Scan(context.Background(), Options{Paths: []string{clean, blank}})
	if report.Err() == nil || report.Coverage.FilesFailed != 1 || report.Coverage.FilesScanned != 1 {
		t.Fatal("an empty JSONL file must be reported even when another file contains events")
	}
}

func TestReportFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.json")
	report := Scan(context.Background(), Options{Paths: []string{fixture(t, "clean.jsonl", `{"type":"user","message":{"content":"ordinary code"}}`)}})
	if report.ExitCode("high") != 0 {
		t.Fatal("clean synthetic history should complete")
	}
	if err := report.SaveJSON(path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatal("report permissions must be owner-only")
	}
	before, _ := os.ReadFile(path)
	if report.SaveJSON(path) == nil {
		t.Fatal("existing report must not be overwritten")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("existing report was modified")
	}
}

func TestSymlinkAndHostileMetadata(t *testing.T) {
	token := syntheticToken()
	content := `{"type":"user","sessionId":"` + token + `","message":{"content":"` + token + `"}}`
	path := fixture(t, token+".jsonl", content)
	report := Scan(context.Background(), Options{Paths: []string{path}})
	var output bytes.Buffer
	report.WriteJSON(&output)
	if strings.Contains(output.String(), token) || len(report.Findings) != 1 {
		t.Fatal("metadata must be sanitized without losing finding")
	}
	link := filepath.Join(filepath.Dir(path), "link.jsonl")
	if err := os.Symlink(path, link); err != nil {
		t.Skip("symbolic links unavailable")
	}
	report = Scan(context.Background(), Options{Paths: []string{link}})
	if report.ExitCode("high") != 2 || len(report.Findings) != 0 {
		t.Fatal("symbolic link must not be scanned")
	}
}

func TestCancellationAndFindingLimit(t *testing.T) {
	path := fixture(t, "synthetic.jsonl", strings.Repeat(`{"type":"user","message":{"content":"`+syntheticToken()+`"}}`+"\n", 2))
	report := Scan(context.Background(), Options{Paths: []string{path}, MaxFindings: 1})
	if report.ExitCode("none") != 2 || len(report.Findings) != 1 {
		t.Fatal("finding limit must mark coverage incomplete")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if Scan(ctx, Options{Paths: []string{path}}).ExitCode("high") != 2 {
		t.Fatal("cancelled scan must not succeed")
	}
}

type cancellingReader struct {
	cancel context.CancelFunc
}

func (reader cancellingReader) Read(buffer []byte) (int, error) {
	reader.cancel()
	return copy(buffer, "\n\n\n"), io.EOF
}

func TestCancellationWhileReadingBlankRecords(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	state := &run{options: withDefaults(Options{}), report: Report{Status: "complete"}}
	state.readJSONL(ctx, &agents.Parser{}, cancellingReader{cancel: cancel}, "synthetic.jsonl")
	if !state.stopped || len(state.report.Errors) != 1 || state.report.Errors[0].Code != "cancelled" {
		t.Fatal("blank records must not bypass cancellation")
	}
}

func TestReportErr(t *testing.T) {
	for _, report := range []Report{{Status: "complete"}, {Status: "complete", Findings: []Finding{{Severity: "high"}}}} {
		if report.Err() != nil {
			t.Fatal("findings alone are not operational errors")
		}
	}
	if (Report{}).Err() == nil {
		t.Fatal("an uninitialized report must not indicate success")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	report := Scan(ctx, Options{})
	var issue ScanError
	if !errors.As(report.Err(), &issue) || issue.Code != "cancelled" || len(report.Errors) != 1 {
		t.Fatal("a cancelled scan must expose a typed operational error")
	}
	if report.ToolVersion != Version || report.SchemaVersion != "1" {
		t.Fatal("early cancellation must preserve report version metadata")
	}
}

func TestMultilinePrivateKeyAndOriginalLines(t *testing.T) {
	key := "-----BEGIN PRIVATE KEY-----\n" + strings.Repeat("U3ludGhldGljLW5vbmZ1bmN0aW9uYWw=\n", 4) + "-----END PRIVATE KEY-----"
	encoded, err := json.Marshal(map[string]any{"type": "assistant", "message": map[string]any{"content": []map[string]any{{"type": "tool_use", "input": map[string]any{"key": key}}}}})
	if err != nil {
		t.Fatal(err)
	}
	path := fixture(t, "private-key.jsonl", "\n"+string(encoded)+"\n")
	report := Scan(context.Background(), Options{Paths: []string{path}})
	if report.ExitCode("high") != 1 || len(report.Findings) != 1 || *report.Findings[0].Line != 2 || *report.Findings[0].EventType != "tool_input" {
		t.Fatal("multiline key attribution failed")
	}
	var output bytes.Buffer
	report.WriteJSON(&output)
	if strings.Contains(output.String(), "U3ludGhldGlj") || strings.Contains(output.String(), "BEGIN PRIVATE KEY") {
		t.Fatal("private-key content leaked")
	}
}

func TestTerminalControlsAndHostileFilename(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("control characters are invalid Windows filename components")
	}
	path := fixture(t, "synthetic\x1b[31m\nforged.jsonl", `{"type":"user","sessionId":"synthetic\u001b[31m\nforged","message":{"content":"`+syntheticToken()+`"}}`)
	report := Scan(context.Background(), Options{Paths: []string{path}})
	var output bytes.Buffer
	report.WriteText(&output)
	if strings.Contains(output.String(), "\x1b") || strings.Contains(output.String(), "\nforged") || len(report.Findings) != 1 {
		t.Fatal("terminal controls must be escaped without losing findings")
	}
}

func TestLimitsMissingPathsAndUnsupportedInput(t *testing.T) {
	path := fixture(t, "clean.jsonl", `{"type":"user","message":{"content":"ordinary"}}`)
	for _, options := range []Options{
		{Paths: []string{path}, MaxEventBytes: int(^uint(0) >> 1)},
		{Paths: []string{path}, MaxFiles: 10001},
		{Paths: []string{path}, MaxFindings: -1},
		{Paths: []string{path}, Discover: true},
		{Paths: []string{filepath.Join(filepath.Dir(path), "missing.jsonl")}},
		{Paths: []string{fixture(t, "unknown.txt", "ordinary")}},
		{Paths: []string{fixture(t, "invalid-utf8.jsonl", string([]byte{0xff}))}},
	} {
		if report := Scan(context.Background(), options); report.ExitCode("none") != 2 {
			t.Fatal("invalid, missing, or unsupported input must not succeed")
		}
	}
	second := filepath.Join(filepath.Dir(path), "second.jsonl")
	if err := os.WriteFile(second, []byte(`{"type":"user","message":{"content":"ordinary"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	report := Scan(context.Background(), Options{Paths: []string{filepath.Dir(path)}, MaxFiles: 1})
	if report.Coverage.FilesScanned != 1 || report.ExitCode("none") != 2 {
		t.Fatal("file limit must not silently produce complete coverage")
	}
}

func TestOverlappingPathsDoNotRepeatFiles(t *testing.T) {
	path := fixture(t, "synthetic.jsonl", `{"type":"user","message":{"content":"`+syntheticToken()+`"}}`)
	report := Scan(context.Background(), Options{Paths: []string{path, filepath.Dir(path)}})
	if report.Coverage.FilesScanned != 1 || len(report.Findings) != 1 {
		t.Fatal("overlapping selections must not scan the same path twice")
	}
}

func TestDiscoveryUsesOnlySyntheticHomeAndHistoryNames(t *testing.T) {
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	for _, test := range []struct{ path, content string }{
		{".claude/projects/synthetic/session.jsonl", `{"type":"user","message":{"content":"ordinary"}}`},
		{".claude/projects/synthetic/sessions-index.json", `{}`},
		{".copilot/session-state/synthetic/events.jsonl", `{"type":"user.message","data":{"content":"ordinary"}}`},
		{".copilot/session-state/synthetic/workspace.yaml", "synthetic: true"},
		{".codex/sessions/2026/rollout-synthetic.jsonl", `{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"ordinary"}]}}`},
		{".codex/sessions/2026/index.json", `{}`},
	} {
		path := filepath.Join(home, filepath.FromSlash(test.path))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(test.content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	report := Scan(context.Background(), Options{Discover: true})
	if report.ExitCode("high") != 0 || report.Coverage.FilesScanned != 3 || len(report.Roots) != 3 {
		t.Fatal("default discovery must select only documented history filenames under the synthetic home")
	}
}
