package generator

import (
	"strings"
	"text/template"
)

// exReserved are names an Elixir variable cannot take.
var exReserved = map[string]bool{
	"do": true, "end": true, "fn": true, "when": true, "and": true, "or": true, "not": true, "in": true,
	"true": true, "false": true, "nil": true, "catch": true, "rescue": true, "after": true, "else": true,
}

func init() {
	register(Language{Name: "elixir", Funcs: template.FuncMap{
		// quote writes a string literal, escaped the way elixir reads it.
		"quote": elixirQuote,
		// exName is a name as an Elixir variable: snake case.
		"exName": func(name string) string {
			name = Snake(name)
			if exReserved[name] {
				return name + "_"
			}
			return name
		},
		// exOptions are the options of the Client.request call a method makes:
		// the query, the body and the idempotency key.
		"exOptions": exOptions,
		// exDoc writes a sentence as text inside an Elixir string.
		"exDoc": func(text string) string { return strings.ReplaceAll(exDocText(text), "\n", " ") },
		// exDocBlock writes text as the lines of a @doc heredoc.
		"exDocBlock": func(prefix, text string) string { return docBlock(prefix, text, exDocText) },
	}})
}

// exDocText escapes text for a heredoc or a string: a backslash starts an
// escape, a quote could end it and #{ would interpolate.
func exDocText(text string) string {
	return strings.NewReplacer(`\`, `\\`, `"""`, `\"\"\"`, `"`, `\"`, `#{`, `\#{`).Replace(strings.TrimSpace(text))
}

// exOptions is the keyword list a method passes to Client.request, or the
// empty string when it has nothing to pass.
func exOptions(m *Method) string {
	var items []string
	if len(m.Query) > 0 {
		pairs := make([]string, len(m.Query))
		for i, q := range m.Query {
			name := Snake(q.Name)
			if exReserved[name] {
				name += "_"
			}
			pairs[i] = "{" + elixirQuote(q.Name) + ", opts[:" + name + "]}"
		}
		items = append(items, "query: ["+strings.Join(pairs, ", ")+"]")
	}
	switch {
	case m.Body != nil:
		items = append(items, "body: payload")
	case m.EmptyBody:
		items = append(items, "body: %{}")
	}
	if m.Idempotent {
		items = append(items, "idempotency_key: opts[:idempotency_key]")
	}
	if len(items) == 0 {
		return ""
	}
	return ", " + strings.Join(items, ", ")
}
