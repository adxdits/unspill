package agents

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAdapters(t *testing.T) {
	tests := []struct {
		name, input, agent, kind, session string
		line                              int
	}{
		{"Claude", `{"type":"user","sessionId":"synthetic-claude","message":{"content":"needle"}}`, "claude-code", "user_message", "synthetic-claude", 7},
		{"Claude tool input", `{"type":"assistant","message":{"content":[{"type":"tool_use","input":{"command":"needle"}}]}}`, "claude-code", "tool_input", "", 7},
		{"Claude tool output", `{"type":"user","message":{"content":[{"type":"tool_result","content":[{"type":"text","text":"needle"}]}]}}`, "claude-code", "tool_output", "", 7},
		{"Copilot CLI", `{"type":"tool.execution_complete","data":{"result":{"content":"needle"}}}`, "copilot-cli", "tool_output", "", 7},
		{"Codex", `{"type":"response_item","payload":{"type":"function_call","arguments":"needle"}}`, "codex", "tool_input", "", 7},
		{"VS Code export", "{\n\"version\":3,\"sessionId\":\"synthetic-vscode\",\"requests\":[{\n\"message\":{\"text\":\"needle\"},\"response\":[]}]}\n", "copilot-vscode", "user_message", "synthetic-vscode", 9},
		{"Generic export", `{"messages":[{"role":"tool","content":"needle"}]}`, "generic", "tool_output", "", 7},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parser := Parser{Agent: test.agent}
			events, err := parser.Parse([]byte(test.input), 7)
			if err != nil {
				t.Fatal(err)
			}
			if len(events) != 1 || !strings.Contains(events[0].Text, "needle") || events[0].Type != test.kind || events[0].SessionID != test.session || events[0].Line != test.line {
				t.Fatal("incorrect event attribution")
			}
		})
	}
}

func TestRejectMalformedAndUnknown(t *testing.T) {
	for _, input := range []string{`{`, `null`, `{"type":"future"}`, `{"type":"user","type":"assistant"}`, `{"messages":[]}`, strings.Repeat("[", 70) + strings.Repeat("]", 70)} {
		parser := Parser{Agent: "auto"}
		if _, err := parser.Parse([]byte(input), 1); err == nil {
			t.Fatal("unknown or malformed input must not look supported")
		}
	}
}

func TestNestedJSONEscapingKeepsPhysicalLine(t *testing.T) {
	arguments := "{\n\"command\":\"\\u0067hp_synthetic\"\n}"
	encoded, err := json.Marshal(map[string]any{
		"type":    "response_item",
		"payload": map[string]any{"type": "function_call", "arguments": arguments},
	})
	if err != nil {
		t.Fatal(err)
	}
	parser := Parser{Agent: "auto"}
	events, err := parser.Parse(encoded, 42)
	if err != nil || len(events) != 1 || events[0].Line != 42 || !strings.Contains(events[0].Text, "ghp_synthetic") {
		t.Fatal("decoded function arguments must retain their original physical record line")
	}
}

func TestVSCodeToolPartsAreNotMisattributed(t *testing.T) {
	parser := Parser{Agent: "copilot-vscode"}
	if _, err := parser.Parse([]byte(`{"requests":[{"message":"hello","response":[{"kind":"toolInvocation","value":"sensitive"}]}]}`), 1); err == nil {
		t.Fatal("tool parts must not be attributed as assistant text")
	}
}

func TestSessionHeaderAndAutoDetection(t *testing.T) {
	parser := Parser{Agent: "auto"}
	if _, err := parser.Parse([]byte(`{"type":"session.start","data":{"version":1,"sessionId":"synthetic-session"}}`), 1); err != nil {
		t.Fatal(err)
	}
	events, err := parser.Parse([]byte(`{"type":"user.message","data":{"content":"hello"}}`), 2)
	if err != nil || len(events) != 1 || events[0].Agent != "copilot-cli" || events[0].SessionID != "synthetic-session" {
		t.Fatal("session header attribution was not preserved")
	}
	for _, version := range []string{"99", `"sensitive-schema-value"`, "null"} {
		if _, err := parser.Parse([]byte(`{"type":"session.start","data":{"version":`+version+`}}`), 3); err == nil {
			t.Fatal("unknown schema versions must be rejected")
		}
	}
}

func FuzzParse(fuzz *testing.F) {
	for _, seed := range []struct{ agent, input string }{
		{"auto", `{"type":"user","message":{"content":"ordinary"}}`},
		{"claude-code", `{"type":"assistant","message":{"content":[{"type":"tool_use","input":{"command":"ordinary"}}]}}`},
		{"copilot-cli", `{"type":"session.start","data":{"version":1,"sessionId":"synthetic"}}`},
		{"copilot-vscode", `{"requests":[{"message":{"text":"ordinary"},"response":[{"value":"ordinary"}]}]}`},
		{"codex", `{"type":"response_item","payload":{"type":"function_call","arguments":"{\"command\":\"ordinary\"}"}}`},
		{"generic", `{"messages":[{"role":"assistant","content":null,"tool_calls":[{"function":{"arguments":"ordinary"}}]}]}`},
		{"auto", `{"type":"user","type":"assistant"}`},
		{"auto", "\xff"},
	} {
		fuzz.Add(seed.agent, []byte(seed.input))
	}
	fuzz.Fuzz(func(t *testing.T, agent string, data []byte) {
		if !Valid(agent) || len(data) > 64<<10 {
			t.Skip()
		}
		parser := Parser{Agent: agent}
		events, _ := parser.Parse(data, 7)
		for _, event := range events {
			if !Valid(event.Agent) || event.Agent == "auto" || event.Line < 7 {
				t.Fatal("parsed text must retain a valid adapter and physical source line")
			}
		}
	})
}
