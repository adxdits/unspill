package agents

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"unicode/utf8"
)

type node struct {
	value  any
	fields map[string]*node
	items  []*node
	line   int
}

type jsonReader struct {
	decoder     *json.Decoder
	data        []byte
	lineOffsets []int
	firstLine   int
	nodes       int
}

func parseJSON(data []byte, firstLine int) (*node, error) {
	if !utf8.Valid(data) {
		return nil, errors.New("input is not valid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	lineOffsets := []int{0}
	for offset, character := range data {
		if character == '\n' {
			lineOffsets = append(lineOffsets, offset+1)
		}
	}
	reader := jsonReader{decoder: decoder, data: data, lineOffsets: lineOffsets, firstLine: firstLine}
	root, err := reader.read(0)
	if err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, errors.New("trailing JSON content")
	}
	return root, nil
}

func (reader *jsonReader) read(depth int) (*node, error) {
	reader.nodes++
	if depth > 64 || reader.nodes > 50000 {
		return nil, errors.New("JSON structure limit exceeded")
	}
	start := int(reader.decoder.InputOffset())
	for start < len(reader.data) && bytes.ContainsRune([]byte(" \t\r\n:,"), rune(reader.data[start])) {
		start++
	}
	line := reader.firstLine + sort.Search(len(reader.lineOffsets), func(index int) bool { return reader.lineOffsets[index] > start }) - 1
	token, err := reader.decoder.Token()
	if err != nil {
		return nil, errors.New("malformed JSON record")
	}
	result := &node{value: token, line: line}
	delimiter, composite := token.(json.Delim)
	if !composite {
		return result, nil
	}
	ending := json.Delim('}')
	switch delimiter {
	case '{':
		err = reader.object(result, depth)
	case '[':
		ending = json.Delim(']')
		err = reader.array(result, depth)
	default:
		return nil, errors.New("malformed JSON record")
	}
	if err != nil {
		return nil, err
	}
	closing, err := reader.decoder.Token()
	if err != nil || closing != ending {
		return nil, errors.New("malformed JSON record")
	}
	return result, nil
}

func (reader *jsonReader) object(result *node, depth int) error {
	result.fields = make(map[string]*node)
	for reader.decoder.More() {
		keyToken, err := reader.decoder.Token()
		if err != nil {
			return errors.New("malformed JSON record")
		}
		key, ok := keyToken.(string)
		if !ok {
			return errors.New("malformed JSON record")
		}
		if _, exists := result.fields[key]; exists {
			return errors.New("duplicate JSON field")
		}
		child, err := reader.read(depth + 1)
		if err != nil {
			return err
		}
		result.fields[key] = child
	}
	return nil
}

func (reader *jsonReader) array(result *node, depth int) error {
	result.items = make([]*node, 0)
	for reader.decoder.More() {
		child, err := reader.read(depth + 1)
		if err != nil {
			return err
		}
		result.items = append(result.items, child)
	}
	return nil
}

func (value *node) field(name string) *node {
	if value == nil {
		return nil
	}
	return value.fields[name]
}

func (value *node) text() string {
	if value == nil {
		return ""
	}
	text, _ := value.value.(string)
	return text
}

func (value *node) isString() bool {
	if value == nil {
		return false
	}
	_, ok := value.value.(string)
	return ok
}
