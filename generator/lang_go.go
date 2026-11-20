package generator

import (
	"go/format"
	"strings"
	"text/template"
)

// goKeywords are names a Go parameter cannot take.
var goKeywords = map[string]bool{"type": true, "func": true, "range": true, "map": true, "default": true, "interface": true, "select": true, "package": true}

func init() {
	register(Language{Name: "go", Funcs: template.FuncMap{
		// quote writes a string literal, escaped the way go reads it.
		"quote":  goQuote,
		"goType": goType,
		"goBase": goBase,
		"goParam": func(name string) string {
			camel := Camel(name)
			if goKeywords[camel] {
				return camel + "ID"
			}
			return camel
		},
		"goName": goName,
		"goEnum": enumPascal,
		// goQuery is the Go type of a query filter.
		"goQuery": func(p *Param) string {
			switch p.Kind {
			case "integer":
				return "int"
			case "stringlist":
				return "[]string"
			}
			return "string"
		},
		"goTag": func(f *Field, input bool) string {
			if !f.Required {
				return f.Wire + ",omitempty"
			}
			return f.Wire
		},
		"goComment": goComment,
	}})
}

// goType is a field's Go type: a pointer when the API may answer null, or
// when a request may leave the field out.
func goType(t *TypeRef, f *Field, input bool) string {
	base := goBase(t)
	if f == nil {
		return base
	}
	pointer := f.Nullable || input && !f.Required || t.Kind == "model" && !f.Required
	if pointer && t.Kind != "array" && t.Kind != "map" && t.Kind != "any" {
		if t.Kind == "model" {
			return "*" + base
		}
		return "*" + base
	}
	return base
}

func goBase(t *TypeRef) string {
	switch t.Kind {
	case "string":
		return "string"
	case "integer":
		return "int64"
	case "number":
		return "float64"
	case "boolean":
		return "bool"
	case "model":
		return t.Model
	case "array":
		if t.OrScalar {
			return "StringList"
		}
		return "[]" + goBase(t.Items)
	case "map":
		return "map[string]" + goBase(t.Items)
	case "any":
		return "any"
	}
	return "any"
}

// goComment writes a description as a Go comment, or nothing.
func goComment(prefix, text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	return prefix + "// " + strings.ReplaceAll(text, "\n", "\n"+prefix+"// ") + "\n"
}

// FormatGo gofmt's a generated Go file.
func FormatGo(source []byte) ([]byte, error) { return format.Source(source) }

// words splits a name written in camelCase, snake_case or with hyphens into
// its words: contactBookId is contact, Book, Id.
func words(name string) []string {
	var out []string
	var word []rune
	flush := func() {
		if len(word) > 0 {
			out = append(out, string(word))
			word = word[:0]
		}
	}
	runes := []rune(name)
	for i, r := range runes {
		switch {
		case r == '_' || r == '-' || r == '.' || r == ' ':
			flush()
		case r >= 'A' && r <= 'Z':
			if i > 0 && !(runes[i-1] >= 'A' && runes[i-1] <= 'Z') {
				flush()
			}
			word = append(word, r)
		default:
			word = append(word, r)
		}
	}
	flush()
	return out
}

// acronyms are the words Go writes whole in upper case.
var acronyms = map[string]bool{"id": true, "url": true, "ip": true, "html": true, "dns": true, "spf": true, "dkim": true, "ttl": true, "api": true}

// goName writes a JSON name as an exported Go identifier: emailId is EmailID,
// html is HTML, _count is Count.
func goName(wire string) string {
	var b strings.Builder
	for _, word := range words(wire) {
		lower := strings.ToLower(word)
		if acronyms[lower] {
			b.WriteString(strings.ToUpper(word))
			continue
		}
		b.WriteString(strings.ToUpper(word[:1]) + word[1:])
	}
	return b.String()
}

// enumPascal writes an allowed value as the end of a constant's name:
// DELIVERY_DELAYED is DeliveryDelayed, OnAccountSuppressionList stays as it is.
func enumPascal(value string) string {
	var b strings.Builder
	for _, word := range words(value) {
		if word == strings.ToUpper(word) {
			word = strings.ToLower(word)
		}
		b.WriteString(strings.ToUpper(word[:1]) + word[1:])
	}
	return b.String()
}
