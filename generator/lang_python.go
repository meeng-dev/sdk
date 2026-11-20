package generator

import (
	"strings"
	"text/template"
)

// pyKeywords are names a Python parameter cannot take.
var pyKeywords = map[string]bool{"from": true, "class": true, "def": true, "global": true, "import": true, "in": true, "is": true, "lambda": true, "pass": true, "return": true, "with": true, "yield": true}

func init() {
	register(Language{Name: "python", Funcs: template.FuncMap{
		// quote writes a string literal, escaped the way python reads it.
		"quote":  pythonQuote,
		"pyType": pyType,
		// pyText escapes text for a docstring: a backslash starts an escape
		// there, and a quote could end it.
		"pyText": pyText,
		// pyDocBlock writes text as the lines of a docstring, indented.
		"pyDocBlock": func(indent, text string) string { return docBlock(indent, text, pyText) },
		// pyOrdered are the models with each one after the ones it holds, so a
		// TypedDict can name them when it is written.
		"pyOrdered": pyOrdered,
		// pyField is the annotation of a model's field.
		"pyField": func(f *Field) string {
			t := pyType(f.Type)
			if f.Nullable {
				t += " | None"
			}
			if !f.Required {
				t = "NotRequired[" + t + "]"
			}
			return t
		},
		// pyQuery is the annotation of a query filter.
		"pyQuery": func(p *Param) string {
			switch p.Kind {
			case "integer":
				return "int | None"
			case "stringlist":
				return "str | builtins.list[str] | None"
			}
			return "str | None"
		},
		"pySnake": Snake,
		"pyName": func(name string) string {
			name = Snake(name)
			if pyKeywords[name] {
				return name + "_"
			}
			return name
		},
		// pyDoc writes a description as the docstring line under a field, or
		// nothing.
		"pyDoc": func(indent, text string) string {
			text = strings.TrimSpace(text)
			if text == "" {
				return ""
			}
			return "\n" + indent + `"""` + pyText(text) + `"""`
		},
	}})
}

// pyType is a field's Python annotation; a string with allowed values is
// their Literal.
func pyType(t *TypeRef) string {
	switch t.Kind {
	case "string":
		if len(t.Enum) > 0 {
			values := make([]string, len(t.Enum))
			for i, value := range t.Enum {
				values[i] = pythonQuote(value)
			}
			return "Literal[" + strings.Join(values, ", ") + "]"
		}
		return "str"
	case "integer":
		return "int"
	case "number":
		return "float"
	case "boolean":
		return "bool"
	case "model":
		return t.Model
	case "array":
		if t.OrScalar {
			return "str | list[str]"
		}
		return "list[" + pyType(t.Items) + "]"
	case "map":
		return "dict[str, " + pyType(t.Items) + "]"
	case "any":
		return "Any"
	}
	return "Any"
}

func pyText(text string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(strings.TrimSpace(text))
}

// pyOrdered sorts models so that every model comes after the models its fields
// name.
func pyOrdered(models []*Model) []*Model {
	byName := map[string]*Model{}
	for _, m := range models {
		byName[m.Name] = m
	}
	var out []*Model
	done := map[string]bool{}
	var visit func(m *Model)
	var uses func(t *TypeRef)
	uses = func(t *TypeRef) {
		if t == nil {
			return
		}
		if t.Kind == "model" && byName[t.Model] != nil {
			visit(byName[t.Model])
		}
		uses(t.Items)
	}
	visit = func(m *Model) {
		if done[m.Name] {
			return
		}
		done[m.Name] = true
		for _, f := range m.Fields {
			uses(f.Type)
		}
		out = append(out, m)
	}
	for _, m := range models {
		visit(m)
	}
	return out
}
