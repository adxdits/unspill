package agents

import (
	"encoding/json"
	"errors"
	"slices"
	"sort"
)

const (
	claudeCode            = "claude-code"
	copilotCLI            = "copilot-cli"
	copilotVSCode         = "copilot-vscode"
	assistantMessageEvent = "assistant.message"
)

type emitter func(*node, string)

type Event struct {
	Agent     string
	SessionID string
	Type      string
	Line      int
	Text      string
}

type Parser struct {
	Agent     string
	SessionID string
}

func Names() []string {
	return []string{claudeCode, copilotCLI, copilotVSCode, "codex", "generic"}
}

func Valid(name string) bool {
	if name == "auto" {
		return true
	}
	for _, supported := range Names() {
		if name == supported {
			return true
		}
	}
	return false
}

func (parser *Parser) Parse(data []byte, firstLine int) ([]Event, error) {
	root, err := parseJSON(data, firstLine)
	if err != nil {
		return nil, err
	}
	if root.fields == nil {
		return nil, errors.New("expected a history object")
	}
	if parser.Agent == "" || parser.Agent == "auto" {
		parser.Agent = detectAgent(root)
		if parser.Agent == "" {
			return nil, errors.New("unsupported history format; select a documented adapter")
		}
	}
	var events []Event
	emit := func(value *node, eventType string) {
		appendText(value, "", Event{Agent: parser.Agent, SessionID: parser.SessionID, Type: eventType}, &events)
	}
	handlers := map[string]func(*node, emitter) error{
		claudeCode: parser.claude, copilotCLI: parser.copilot,
		copilotVSCode: parser.vscode, "codex": parser.codex, "generic": parser.generic,
	}
	handler, supported := handlers[parser.Agent]
	if !supported {
		return nil, errors.New("unsupported agent")
	}
	err = handler(root, emit)
	return events, err
}

func detectAgent(root *node) string {
	kind := root.field("type").text()
	switch {
	case root.field("requests") != nil:
		return copilotVSCode
	case root.field("message").field("content") != nil && slices.Contains([]string{"user", "assistant"}, kind):
		return claudeCode
	case root.field("payload") != nil && slices.Contains([]string{"session_meta", "response_item", "event_msg"}, kind):
		return "codex"
	case root.field("data") != nil && slices.Contains([]string{"session.start", "user.message", assistantMessageEvent, "tool.execution_start", "tool.execution_complete"}, kind):
		return copilotCLI
	default:
		return ""
	}
}

func (parser *Parser) claude(root *node, emit emitter) error {
	if session := root.field("sessionId").text(); session != "" {
		parser.SessionID = session
	}
	switch root.field("type").text() {
	case "user", "assistant":
		return content(root.field("message").field("content"), root.field("type").text()+"_message", emit)
	case "system", "progress", "attachment", "summary", "file-history-snapshot", "queue-operation":
		emit(root, "")
		return nil
	default:
		return errors.New("unsupported Claude record type")
	}
}

func schemaVersion(value *node, versions ...string) bool {
	if value == nil {
		return false
	}
	number, ok := value.value.(json.Number)
	return ok && slices.Contains(versions, number.String())
}

func (parser *Parser) copilot(root *node, emit emitter) error {
	data := root.field("data")
	if data == nil || data.fields == nil {
		return errors.New("missing Copilot event data")
	}
	switch kind := root.field("type").text(); kind {
	case "session.start":
		if !schemaVersion(data.field("version"), "1") {
			return errors.New("unsupported Copilot session schema version")
		}
		parser.SessionID = data.field("sessionId").text()
		emit(data, "")
	case "user.message", assistantMessageEvent:
		return copilotMessage(data, kind, emit)
	case "tool.execution_start", "tool.user_requested":
		emit(data.field("arguments"), "tool_input")
	case "tool.execution_complete":
		emit(data.field("result"), "tool_output")
		emit(data.field("error"), "tool_output")
	case "assistant.reasoning":
		emit(data.field("content"), "assistant_message")
	case "system.message", "system.notification":
		emit(data, "system_message")
	case "session.resume", "session.shutdown", "session.info", "session.error", "session.warning", "session.model_change", "session.truncation", "session.compaction_start", "session.compaction_complete", "session.context_changed", "assistant.turn_start", "assistant.turn_end", "abort":
		emit(data, "")
	default:
		return errors.New("unsupported Copilot record type")
	}
	return nil
}

func copilotMessage(data *node, kind string, emit emitter) error {
	if !data.field("content").isString() {
		return errors.New("missing Copilot message content")
	}
	eventType := "user_message"
	if kind == assistantMessageEvent {
		eventType = "assistant_message"
	}
	for _, field := range []string{"content", "transformedContent", "attachments", "reasoningText"} {
		emit(data.field(field), eventType)
	}
	if err := toolRequests(data.field("toolRequests"), false, emit); err != nil {
		return err
	}
	for _, field := range []string{"encryptedContent", "reasoningOpaque", "reasoningBlocks", "serverTools"} {
		value := data.field(field)
		if value != nil && value.value != nil && (!value.isString() || value.text() != "") {
			return errors.New("opaque Copilot reasoning or server tool payload is unsupported")
		}
	}
	return nil
}

func toolRequests(requests *node, generic bool, emit emitter) error {
	if requests == nil || requests.value == nil {
		return nil
	}
	if requests.items == nil {
		return errors.New("unsupported tool request collection")
	}
	for _, request := range requests.items {
		if generic {
			request = request.field("function")
		}
		emit(request.field("arguments"), "tool_input")
	}
	return nil
}

func (parser *Parser) codex(root *node, emit emitter) error {
	payload := root.field("payload")
	if payload == nil || payload.fields == nil {
		return errors.New("missing Codex payload")
	}
	switch root.field("type").text() {
	case "session_meta":
		if base := payload.field("history_base"); base != nil && base.value != nil {
			return errors.New("external Codex history references are unsupported")
		}
		parser.SessionID = payload.field("id").text()
		emit(payload, "")
	case "response_item":
		return codexResponse(payload, emit)
	case "event_msg":
		return codexEvent(payload, emit)
	case "turn_context", "compacted":
		emit(payload, "")
	default:
		return errors.New("unsupported Codex record type")
	}
	return nil
}

func codexResponse(payload *node, emit emitter) error {
	switch payload.field("type").text() {
	case "message":
		kind, ok := roleType(payload.field("role").text())
		if !ok {
			return errors.New("unsupported Codex message role")
		}
		return content(payload.field("content"), kind, emit)
	case "function_call":
		if !payload.field("arguments").isString() {
			return errors.New("missing Codex function arguments")
		}
		emit(payload.field("arguments"), "tool_input")
	case "custom_tool_call":
		emit(payload.field("input"), "tool_input")
	case "function_call_output", "custom_tool_call_output":
		emit(payload.field("output"), "tool_output")
	case "local_shell_call":
		emit(payload.field("action"), "tool_input")
	case "reasoning":
		if payload.field("encrypted_content").text() != "" {
			return errors.New("encrypted Codex content is unsupported")
		}
		emit(payload.field("summary"), "assistant_message")
		emit(payload.field("content"), "assistant_message")
	default:
		return errors.New("unsupported Codex response item")
	}
	return nil
}

func codexEvent(payload *node, emit emitter) error {
	switch payload.field("type").text() {
	case "user_message":
		emit(payload.field("message"), "user_message")
	case "agent_message", "agent_reasoning":
		emit(payload.field("message"), "assistant_message")
		emit(payload.field("text"), "assistant_message")
	case "exec_command_begin", "mcp_tool_call_begin":
		emit(payload.field("command"), "tool_input")
		emit(payload.field("invocation").field("arguments"), "tool_input")
	case "exec_command_end", "mcp_tool_call_end":
		emit(payload.field("aggregated_output"), "tool_output")
		emit(payload.field("result"), "tool_output")
	case "token_count", "task_started", "task_complete", "turn_started", "turn_complete", "turn_aborted", "context_compacted":
		emit(payload, "")
	default:
		return errors.New("unsupported Codex event type")
	}
	return nil
}

func (parser *Parser) vscode(root *node, emit emitter) error {
	if version := root.field("version"); version != nil && !schemaVersion(version, "1", "2", "3") {
		return errors.New("unsupported VS Code export version")
	}
	requests := root.field("requests")
	if requests == nil || requests.items == nil {
		return errors.New("missing VS Code export requests")
	}
	parser.SessionID = root.field("sessionId").text()
	var parseErr error
	for _, request := range requests.items {
		if err := vscodeRequest(request, emit); err != nil && parseErr == nil {
			parseErr = err
		}
	}
	return parseErr
}

func vscodeRequest(request *node, emit emitter) error {
	message := request.field("message")
	if !message.isString() {
		message = message.field("text")
	}
	if !message.isString() {
		return errors.New("unsupported VS Code request message")
	}
	emit(message, "user_message")
	emit(request.field("variableData"), "user_message")
	for _, unsupported := range []string{"toolInvocations", "toolCalls"} {
		if request.field(unsupported) != nil {
			return errors.New("VS Code tool event export is unsupported")
		}
	}
	return vscodeResponse(request.field("response"), emit)
}

func vscodeResponse(response *node, emit emitter) error {
	if response == nil || response.value == nil {
		return nil
	}
	if response.items == nil {
		return errors.New("unsupported VS Code response")
	}
	var parseErr error
	for _, part := range response.items {
		if !part.field("value").isString() || part.field("kind") != nil {
			if parseErr == nil {
				parseErr = errors.New("unsupported VS Code response part; only exported markdown text is supported")
			}
			continue
		}
		emit(part.field("value"), "assistant_message")
	}
	return parseErr
}

func (parser *Parser) generic(root *node, emit emitter) error {
	messages := root.field("messages")
	if messages == nil || messages.items == nil {
		return errors.New("generic input requires a messages array")
	}
	parser.SessionID = root.field("session_id").text()
	var parseErr error
	for _, message := range messages.items {
		if err := genericMessage(message, emit); err != nil && parseErr == nil {
			parseErr = err
		}
	}
	return parseErr
}

func genericMessage(message *node, emit emitter) error {
	kind, ok := roleType(message.field("role").text())
	if !ok {
		return errors.New("unsupported generic message role")
	}
	value := message.field("content")
	if value != nil && value.value != nil {
		if err := content(value, kind, emit); err != nil {
			return err
		}
	} else if calls := message.field("tool_calls"); kind != "assistant_message" || calls == nil || len(calls.items) == 0 {
		return errors.New("missing generic message content")
	}
	return toolRequests(message.field("tool_calls"), true, emit)
}

func roleType(role string) (string, bool) {
	switch role {
	case "user":
		return "user_message", true
	case "assistant":
		return "assistant_message", true
	case "system", "developer":
		return "system_message", true
	case "tool", "function":
		return "tool_output", true
	default:
		return "", false
	}
}

func content(value *node, kind string, emit func(*node, string)) error {
	if value == nil {
		return errors.New("missing message content")
	}
	if value.isString() {
		emit(value, kind)
		return nil
	}
	if value.items == nil {
		return errors.New("unsupported message content")
	}
	var parseErr error
	for _, block := range value.items {
		if err := contentBlock(block, kind, emit); err != nil && parseErr == nil {
			parseErr = err
		}
	}
	return parseErr
}

func contentBlock(block *node, kind string, emit emitter) error {
	switch block.field("type").text() {
	case "text", "input_text", "output_text":
		if !block.field("text").isString() {
			return errors.New("missing text block value")
		}
		emit(block.field("text"), kind)
	case "thinking":
		emit(block.field("thinking"), "assistant_message")
	case "tool_use":
		emit(block.field("input"), "tool_input")
	case "tool_result":
		return content(block.field("content"), "tool_output", emit)
	default:
		return errors.New("unsupported content block; binary and encrypted content are not scanned")
	}
	return nil
}

func appendText(value *node, key string, event Event, events *[]Event) {
	appendNodeText(value, key, event, events, 0, 0)
}

func appendNodeText(value *node, key string, event Event, events *[]Event, decodedDepth, physicalLine int) {
	if value == nil {
		return
	}
	if value.isString() {
		event.Line, event.Text = value.line, value.text()
		if physicalLine > 0 {
			event.Line = physicalLine
		}
		if decoded := nestedJSON(event.Text, event.Line, decodedDepth); decoded != nil {
			appendNodeText(decoded, key, event, events, decodedDepth+1, event.Line)
			return
		}
		if key != "" {
			event.Text = key + " = " + event.Text
		}
		*events = append(*events, event)
	}
	keys := make([]string, 0, len(value.fields))
	for name := range value.fields {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	for _, name := range keys {
		appendNodeText(value.fields[name], name, event, events, decodedDepth, physicalLine)
	}
	for _, child := range value.items {
		appendNodeText(child, key, event, events, decodedDepth, physicalLine)
	}
}

func nestedJSON(text string, line, depth int) *node {
	if depth >= 2 || !json.Valid([]byte(text)) {
		return nil
	}
	decoded, err := parseJSON([]byte(text), line)
	if err != nil || (decoded.fields == nil && decoded.items == nil) {
		return nil
	}
	return decoded
}
