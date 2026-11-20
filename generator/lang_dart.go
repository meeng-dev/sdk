package generator

import (
	"strconv"
	"strings"
	"text/template"
)

// dartKeywords are names a Dart field or parameter cannot take.
var dartKeywords = map[string]bool{
	"assert": true, "break": true, "case": true, "catch": true, "class": true, "const": true, "continue": true,
	"default": true, "do": true, "else": true, "enum": true, "extends": true, "false": true, "final": true,
	"finally": true, "for": true, "if": true, "in": true, "is": true, "new": true, "null": true, "rethrow": true,
	"return": true, "super": true, "switch": true, "this": true, "throw": true, "true": true, "try": true,
	"var": true, "void": true, "while": true, "with": true,
}

// dartLocals are the names a service method gives its own parameters and
// variables, which a path parameter or a filter cannot take.
var dartLocals = map[string]bool{"data": true, "idempotencyKey": true, "params": true, "query": true}

func init() {
	register(Language{Name: "dart", Funcs: template.FuncMap{
		// quote writes a string literal, escaped the way dart reads it.
		"quote":          dartQuote,
		"dartType":       dartFieldType,
		"dartBase":       dartBase,
		"dartName":       dartName,
		"dartParam":      dartParam,
		"dartRead":       dartRead,
		"dartPath":       dartPath,
		"dartParams":     dartParams,
		"dartText":       dartText,
		"dartWriteValue": dartWriteValue,
		"dartDoc":        dartDoc,
		// dartDocBlock writes text as the lines of a doc comment, wrapped.
		"dartDocBlock": func(prefix, text string) string { return docBlock(prefix, text, dartText) },
		// dartFieldDoc documents a field: its description and the values it
		// may take.
		"dartFieldDoc": func(indent string, f *Field) string {
			text := strings.TrimSpace(f.Description)
			if f.Type.Kind == "string" && len(f.Type.Enum) > 0 {
				values := make([]string, len(f.Type.Enum))
				for i, value := range f.Type.Enum {
					values[i] = "`" + value + "`"
				}
				if text != "" {
					text += "\n\n"
				}
				text += "One of " + strings.Join(values, ", ") + "."
			}
			return dartDoc(indent, text)
		},
		// dartOptional says a field may be left out of a constructor: it can
		// be null.
		"dartOptional": func(f *Field) bool { return f.Nullable || !f.Required },
	}})
}

// dartName writes customer_id as customerId and url as url: Dart's camel
// case capitalizes an abbreviation like a word, where the common Camel keeps
// Go's ID and URL.
func dartName(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool { return r == '_' || r == '-' || r == '.' || r == ' ' })
	for i, part := range parts {
		if i == 0 {
			parts[i] = strings.ToLower(part[:1]) + part[1:]
		} else {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	camel := strings.Join(parts, "")
	if dartKeywords[camel] {
		return camel + "Value"
	}
	return camel
}

// dartParam is a path parameter or a filter as a Dart parameter.
func dartParam(name string) string {
	camel := dartName(name)
	if dartLocals[camel] {
		return camel + "Value"
	}
	return camel
}

// dartDoc writes a description as a Dart doc comment wrapped at 80
// columns, or nothing.
func dartDoc(indent, text string) string {
	text = dartText(text)
	if text == "" {
		return ""
	}
	prefix := indent + "///"
	var out []string
	for _, paragraph := range strings.Split(text, "\n") {
		line := prefix
		for _, word := range strings.Fields(paragraph) {
			if line != prefix && len([]rune(line))+1+len([]rune(word)) > 80 {
				out = append(out, line)
				line = prefix
			}
			line += " " + word
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n") + "\n"
}

// dartBase is a type without its nullability.
func dartBase(t *TypeRef) string {
	switch t.Kind {
	case "string":
		return "String"
	case "integer":
		return "int"
	case "number":
		return "double"
	case "boolean":
		return "bool"
	case "model":
		return t.Model
	case "array":
		return "List<" + dartBase(t.Items) + ">"
	case "map":
		return "Map<String, " + dartBase(t.Items) + ">"
	case "any":
		return "Object?"
	}
	return "Object?"
}

// dartFieldType is a field's type: nullable when the API may answer null or
// leave it out.
func dartFieldType(f *Field) string {
	base := dartBase(f.Type)
	if (f.Nullable || !f.Required) && !strings.HasSuffix(base, "?") {
		return base + "?"
	}
	return base
}

// dartReader is a function that reads a JSON value into t, for the items
// of a list or a map.
func dartReader(t *TypeRef) string {
	switch t.Kind {
	case "string":
		return "readString"
	case "integer":
		return "readInt"
	case "number":
		return "readDouble"
	case "boolean":
		return "readBool"
	case "model":
		return "(value) => " + t.Model + ".fromJson(readObject(value))"
	case "array":
		if t.OrScalar {
			return "readStringList"
		}
		return "(value) => readList(value, " + dartReader(t.Items) + ")"
	case "map":
		return "(value) => readMap(value, " + dartReader(t.Items) + ")"
	}
	return "readAny"
}

// dartRead is the expression that reads a field out of a model's JSON. A
// field the API always sends is cast, so a missing one fails loudly.
func dartRead(f *Field) string {
	value := "json['" + f.Wire + "']"
	optional := f.Nullable || !f.Required
	switch f.Type.Kind {
	case "string", "integer", "boolean":
		if optional {
			return value + " as " + dartBase(f.Type) + "?"
		}
		return value + " as " + dartBase(f.Type)
	}
	if f.Type.Kind == "any" {
		return value
	}
	if optional {
		return "readOptional(" + value + ", " + dartReader(f.Type) + ")"
	}
	switch f.Type.Kind {
	case "number":
		return "readDouble(" + value + ")"
	case "model":
		return f.Type.Model + ".fromJson(readObject(" + value + "))"
	case "array":
		if f.Type.OrScalar {
			return "readStringList(" + value + ")"
		}
		return "readList(" + value + ", " + dartReader(f.Type.Items) + ")"
	case "map":
		return "readMap(" + value + ", " + dartReader(f.Type.Items) + ")"
	}
	return value
}

// dartWriteValue is the value of a field's toJson entry.
func dartWriteValue(f *Field) string {
	expr := dartWriter(f.Type, dartName(f.Wire), f.Nullable || !f.Required, 1)
	if !f.Required {
		expr = "?" + expr
	}
	return expr
}

// dartWriter writes expr, of type t, as JSON; nested lists and maps name
// their items by depth.
func dartWriter(t *TypeRef, expr string, nullable bool, depth int) string {
	dot := "."
	if nullable {
		dot = "?."
	}
	suffix := ""
	if depth > 1 {
		suffix = strconv.Itoa(depth)
	}
	switch t.Kind {
	case "model":
		return expr + dot + "toJson()"
	case "array":
		if inner := dartWriter(t.Items, "item"+suffix, false, depth+1); inner != "item"+suffix {
			return expr + dot + "map((item" + suffix + ") => " + inner + ").toList()"
		}
	case "map":
		if inner := dartWriter(t.Items, "value"+suffix, false, depth+1); inner != "value"+suffix {
			return expr + dot + "map((key" + suffix + ", value" + suffix + ") => MapEntry(key" + suffix + ", " + inner + "))"
		}
	}
	return expr
}

// dartPath is a method's path as a Dart string, its parameters escaped.
func dartPath(m *Method) string {
	var b strings.Builder
	b.WriteString("'")
	for _, part := range pathParts(m.Path) {
		if part.Param != "" {
			b.WriteString("${Uri.encodeComponent(" + dartParam(part.Param) + ")}")
			continue
		}
		b.WriteString(inner(dartQuote(part.Literal)))
	}
	b.WriteString("'")
	return b.String()
}

// dartParams is a service method's parameter list: the path parameters and
// the body positional, the filters and the idempotency key named.
func dartParams(m *Method) string {
	var positional, named []string
	for _, param := range m.PathParams {
		positional = append(positional, "String "+dartParam(param))
	}
	if m.Body != nil {
		if m.BodyList {
			positional = append(positional, "List<"+m.Body.Name+"> params")
		} else {
			positional = append(positional, m.Body.Name+" params")
		}
	}
	for _, query := range m.Query {
		switch query.Kind {
		case "integer":
			named = append(named, "int? "+dartParam(query.Name))
		case "stringlist":
			named = append(named, "List<String>? "+dartParam(query.Name))
		default:
			named = append(named, "String? "+dartParam(query.Name))
		}
	}
	if m.Idempotent {
		named = append(named, "String? idempotencyKey")
	}
	var b strings.Builder
	b.WriteString("(")
	b.WriteString(strings.Join(positional, ", "))
	if len(named) > 0 {
		if len(positional) > 0 {
			b.WriteString(", ")
		}
		b.WriteString("{" + strings.Join(named, ", ") + "}")
	}
	b.WriteString(")")
	return b.String()
}

// dartText escapes the angle brackets of a description, which dartdoc would
// read as HTML.
func dartText(text string) string {
	return strings.NewReplacer("<", "&lt;", ">", "&gt;").Replace(strings.TrimSpace(text))
}
