package fakeapi

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

// This file reads the contract on its own, with none of the generator's code:
// the fake server is the check on what the SDKs send and read, and it would
// check nothing if it shared the generator's mistakes.

const componentPrefix = "#/components/schemas/"

type schemas struct {
	components map[string]any
}

// resolve follows a $ref to a named schema. A schema that is a reference to
// itself (the contract writes a list that way) is returned as it is, with
// self set to the schema's name.
func (s *schemas) resolve(node any) (resolved map[string]any, self string) {
	m, _ := node.(map[string]any)
	for depth := 0; m != nil && depth < 32; depth++ {
		ref, _ := m["$ref"].(string)
		if ref == "" {
			return m, ""
		}
		name := strings.TrimPrefix(ref, componentPrefix)
		target, _ := s.components[name].(map[string]any)
		if target == nil {
			return nil, ""
		}
		if next, _ := target["$ref"].(string); next == ref {
			return target, name
		}
		m = target
	}
	return m, ""
}

// kinds are the JSON types a schema allows, and whether null is one of them.
func kinds(m map[string]any) (types []string, nullable bool) {
	switch t := m["type"].(type) {
	case string:
		types = []string{t}
	case []any:
		for _, each := range t {
			if text, _ := each.(string); text != "" {
				types = append(types, text)
			}
		}
	}
	var kept []string
	for _, t := range types {
		if t == "null" {
			nullable = true
			continue
		}
		kept = append(kept, t)
	}
	if n, _ := m["nullable"].(bool); n {
		nullable = true
	}
	return kept, nullable
}

var dateTime = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})$`)

// issue is one thing wrong with a value, written the way the API's own
// validation writes it.
type issue struct {
	Code    string `json:"code"`
	Path    []any  `json:"path"`
	Message string `json:"message"`
}

// validate checks a decoded JSON value against a schema. Unknown properties of
// an object are refused: an SDK must send only what the contract documents.
func (s *schemas) validate(node any, value any, path []any, out *[]issue) {
	m, self := s.resolve(node)
	if m == nil || self != "" {
		return
	}
	if anyOf, ok := m["anyOf"].([]any); ok {
		for _, option := range anyOf {
			var trial []issue
			s.validate(option, value, path, &trial)
			if len(trial) == 0 {
				return
			}
		}
		*out = append(*out, issue{Code: "invalid_union", Path: append([]any(nil), path...), Message: "Invalid input"})
		return
	}
	types, nullable := kinds(m)
	if value == nil {
		if !nullable && len(types) > 0 {
			*out = append(*out, issue{Code: "invalid_type", Path: append([]any(nil), path...), Message: "Expected " + types[0] + ", received null"})
		}
		return
	}
	if len(types) > 0 && !matches(types, value) {
		*out = append(*out, issue{Code: "invalid_type", Path: append([]any(nil), path...), Message: fmt.Sprintf("Expected %s, received %s", strings.Join(types, " or "), jsonType(value))})
		return
	}
	if enum, ok := m["enum"].([]any); ok {
		found := false
		for _, each := range enum {
			if each == value {
				found = true
			}
		}
		if !found {
			*out = append(*out, issue{Code: "invalid_enum_value", Path: append([]any(nil), path...), Message: fmt.Sprintf("Invalid enum value, received %v", value)})
			return
		}
	}
	switch v := value.(type) {
	case string:
		if format, _ := m["format"].(string); format == "date-time" {
			if _, err := time.Parse(time.RFC3339Nano, v); err != nil || !dateTime.MatchString(v) {
				*out = append(*out, issue{Code: "invalid_string", Path: append([]any(nil), path...), Message: "Invalid datetime"})
			}
		}
	case []any:
		if items, ok := m["items"]; ok {
			for i, each := range v {
				s.validate(items, each, append(append([]any(nil), path...), i), out)
			}
		}
	case map[string]any:
		properties, _ := m["properties"].(map[string]any)
		required, _ := m["required"].([]any)
		for _, name := range required {
			text, _ := name.(string)
			if _, there := v[text]; !there {
				*out = append(*out, issue{Code: "invalid_type", Path: append(append([]any(nil), path...), text), Message: "Required"})
			}
		}
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			property, known := properties[key]
			if known {
				s.validate(property, v[key], append(append([]any(nil), path...), key), out)
				continue
			}
			switch extra := m["additionalProperties"].(type) {
			case map[string]any:
				s.validate(extra, v[key], append(append([]any(nil), path...), key), out)
			case bool:
				if extra {
					continue
				}
				*out = append(*out, issue{Code: "unrecognized_keys", Path: append([]any(nil), path...), Message: fmt.Sprintf("Unrecognized key(s) in object: '%s'", key)})
			default:
				// The contract documents an object by its properties: a key
				// outside them is a typo in the SDK, not an extension.
				if len(properties) > 0 {
					*out = append(*out, issue{Code: "unrecognized_keys", Path: append([]any(nil), path...), Message: fmt.Sprintf("Unrecognized key(s) in object: '%s'", key)})
				}
			}
		}
	}
}

func matches(types []string, value any) bool {
	for _, t := range types {
		switch t {
		case "string":
			if _, ok := value.(string); ok {
				return true
			}
		case "boolean":
			if _, ok := value.(bool); ok {
				return true
			}
		case "number":
			if _, ok := value.(float64); ok {
				return true
			}
		case "integer":
			if f, ok := value.(float64); ok && f == float64(int64(f)) {
				return true
			}
		case "object":
			if _, ok := value.(map[string]any); ok {
				return true
			}
		case "array":
			if _, ok := value.([]any); ok {
				return true
			}
		}
	}
	return false
}

func jsonType(value any) string {
	switch value.(type) {
	case string:
		return "string"
	case bool:
		return "boolean"
	case float64:
		return "number"
	case map[string]any:
		return "object"
	case []any:
		return "array"
	}
	return "null"
}

// sample builds a value that satisfies a schema, the same every time: every
// property is present and not null, a list has one item and a map one entry,
// an enumeration answers its first value. name is the property it is the value
// of, which decides the text of a string (an id looks like one).
func (s *schemas) sample(node any, name string) any {
	m, self := s.resolve(node)
	if m == nil {
		return nil
	}
	if self != "" {
		// A list the contract writes as a schema that refers to itself: the list
		// of the schema without "List".
		item := strings.TrimSuffix(self, "List")
		return []any{s.sample(map[string]any{"$ref": componentPrefix + item}, name)}
	}
	if anyOf, ok := m["anyOf"].([]any); ok {
		for _, option := range anyOf {
			if o, _ := s.resolve(option); o != nil {
				if types, _ := kinds(o); len(types) > 0 {
					return s.sample(option, name)
				}
			}
		}
		return nil
	}
	if enum, ok := m["enum"].([]any); ok && len(enum) > 0 {
		return enum[0]
	}
	types, _ := kinds(m)
	kind := ""
	if len(types) > 0 {
		kind = types[0]
	}
	if kind == "" && m["properties"] != nil {
		kind = "object"
	}
	switch kind {
	case "string":
		return sampleString(m, name)
	case "integer":
		return float64(1)
	case "number":
		return 0.5
	case "boolean":
		return true
	case "array":
		if items, ok := m["items"]; ok {
			return []any{s.sample(items, name)}
		}
		return []any{}
	case "object":
		out := map[string]any{}
		if properties, ok := m["properties"].(map[string]any); ok {
			for key, property := range properties {
				out[key] = s.sample(property, key)
			}
			return out
		}
		if extra, ok := m["additionalProperties"].(map[string]any); ok && len(extra) > 0 {
			out["key"] = s.sample(extra, "key")
		}
		return out
	}
	return map[string]any{}
}

func sampleString(m map[string]any, name string) string {
	if format, _ := m["format"].(string); format == "date-time" {
		return "2026-11-20T12:00:00.000Z"
	}
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, "at") || strings.HasSuffix(lower, "time") || lower == "date":
		return "2026-11-20T12:00:00.000Z"
	case lower == "email" || lower == "from" || lower == "to" || strings.Contains(lower, "email") && !strings.HasSuffix(lower, "id"):
		return "sample@example.com"
	case lower == "id" || strings.HasSuffix(lower, "id"):
		return "sample_" + lower
	}
	return "sample"
}
