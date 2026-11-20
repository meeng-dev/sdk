package generator

import (
	"strconv"
	"strings"
)

func init() {
	callers["typescript"] = tsCall
}

// tsCall is the call of a contract test in TypeScript: meeng is the client, and
// the call answers a promise of {data, error}.
func tsCall(c *ContractCall) string {
	m := c.Method
	args := []string{}
	for _, value := range c.Path {
		args = append(args, tsQuote(value))
	}
	if m.Body != nil {
		var fields []string
		for _, a := range c.Body {
			fields = append(fields, tsQuote(a.Field.Wire)+": "+tsValue(a))
		}
		item := "{ " + strings.Join(fields, ", ") + " }"
		if len(fields) == 0 {
			item = "{}"
		}
		if m.BodyList {
			item = "[" + item + "]"
		}
		args = append(args, item)
	}
	if len(m.Query) > 0 {
		var fields []string
		for _, q := range c.Query {
			switch {
			case q.Param.Number():
				fields = append(fields, tsQuote(q.Param.Name)+": "+strconv.Itoa(q.Number))
			case q.Param.List():
				fields = append(fields, tsQuote(q.Param.Name)+": ["+quoteAll(tsQuote, q.List)+"]")
			default:
				fields = append(fields, tsQuote(q.Param.Name)+": "+tsQuote(q.Text))
			}
		}
		args = append(args, "{ "+strings.Join(fields, ", ")+" }")
	}
	if c.Key != "" {
		args = append(args, "{ idempotencyKey: "+tsQuote(c.Key)+" }")
	}
	return "meeng." + Camel(m.Service) + "." + m.Name + "(" + strings.Join(args, ", ") + ")"
}

func tsValue(a *BodyArg) string {
	if a.List != nil {
		return "[" + quoteAll(tsQuote, a.List) + "]"
	}
	return tsQuote(a.Text)
}

func init() {
	callers["python"] = pyCall
	callers["php"] = phpCall
	callers["ruby"] = rubyCall
	callers["elixir"] = elixirCall
}

// pyCall is the call of a contract test in Python: meeng is the client, and
// the call answers (data, error) or raises MeengHTTPError.
func pyCall(c *ContractCall) string {
	m := c.Method
	args := []string{}
	for _, value := range c.Path {
		args = append(args, pythonQuote(value))
	}
	if m.Body != nil {
		var fields []string
		for _, a := range c.Body {
			fields = append(fields, pythonQuote(a.Field.Wire)+": "+pyValue(a))
		}
		item := "{" + strings.Join(fields, ", ") + "}"
		if m.BodyList {
			item = "[" + item + "]"
		}
		args = append(args, item)
	}
	for _, q := range c.Query {
		name := Snake(q.Param.Name)
		switch {
		case q.Param.Number():
			args = append(args, name+"="+strconv.Itoa(q.Number))
		case q.Param.List():
			args = append(args, name+"=["+quoteAll(pythonQuote, q.List)+"]")
		default:
			args = append(args, name+"="+pythonQuote(q.Text))
		}
	}
	if c.Key != "" {
		args = append(args, `options={"idempotency_key": `+pythonQuote(c.Key)+`}`)
	}
	return "meeng." + Snake(m.Service) + "." + Snake(m.Name) + "(" + strings.Join(args, ", ") + ")"
}

func pyValue(a *BodyArg) string {
	if a.List != nil {
		return "[" + quoteAll(pythonQuote, a.List) + "]"
	}
	return pythonQuote(a.Text)
}

// phpCall is the call of a contract test in PHP: $meeng is the client, and the
// call answers the decoded JSON or throws MeengException.
func phpCall(c *ContractCall) string {
	m := c.Method
	args := []string{}
	for _, value := range c.Path {
		args = append(args, phpQuote(value))
	}
	if m.Body != nil {
		var fields []string
		for _, a := range c.Body {
			value := phpQuote(a.Text)
			if a.List != nil {
				value = "[" + quoteAll(phpQuote, a.List) + "]"
			}
			fields = append(fields, phpQuote(a.Field.Wire)+" => "+value)
		}
		item := "[" + strings.Join(fields, ", ") + "]"
		if m.BodyList {
			item = "[" + item + "]"
		}
		args = append(args, item)
	}
	if len(m.Query) > 0 {
		var fields []string
		for _, q := range c.Query {
			switch {
			case q.Param.Number():
				fields = append(fields, phpQuote(q.Param.Name)+" => "+strconv.Itoa(q.Number))
			case q.Param.List():
				fields = append(fields, phpQuote(q.Param.Name)+" => ["+quoteAll(phpQuote, q.List)+"]")
			default:
				fields = append(fields, phpQuote(q.Param.Name)+" => "+phpQuote(q.Text))
			}
		}
		args = append(args, "["+strings.Join(fields, ", ")+"]")
	}
	if c.Key != "" {
		args = append(args, phpQuote(c.Key))
	}
	return "$meeng->" + phpName(m.Service) + "->" + m.Name + "(" + strings.Join(args, ", ") + ")"
}

// rubyCall is the call of a contract test in Ruby: meeng is the client, and
// the call answers the decoded JSON or raises Meeng::Error.
func rubyCall(c *ContractCall) string {
	m := c.Method
	args := []string{}
	for _, value := range c.Path {
		args = append(args, rubyQuote(value))
	}
	if m.Body != nil {
		var fields []string
		for _, a := range c.Body {
			value := rubyQuote(a.Text)
			if a.List != nil {
				value = "[" + quoteAll(rubyQuote, a.List) + "]"
			}
			fields = append(fields, rubyQuote(a.Field.Wire)+" => "+value)
		}
		item := "{ " + strings.Join(fields, ", ") + " }"
		if m.BodyList {
			item = "[" + item + "]"
		}
		args = append(args, item)
	}
	for _, q := range c.Query {
		name := rbName(Snake(q.Param.Name))
		switch {
		case q.Param.Number():
			args = append(args, name+": "+strconv.Itoa(q.Number))
		case q.Param.List():
			args = append(args, name+": ["+quoteAll(rubyQuote, q.List)+"]")
		default:
			args = append(args, name+": "+rubyQuote(q.Text))
		}
	}
	if c.Key != "" {
		args = append(args, "idempotency_key: "+rubyQuote(c.Key))
	}
	return "meeng." + Snake(m.Service) + "." + Snake(m.Name) + "(" + strings.Join(args, ", ") + ")"
}

// elixirCall is the call of a contract test in Elixir: meeng is the client, and
// the call answers {:ok, data} or {:error, error}.
func elixirCall(c *ContractCall) string {
	m := c.Method
	args := []string{"meeng"}
	for _, value := range c.Path {
		args = append(args, elixirQuote(value))
	}
	if m.Body != nil {
		var fields []string
		for _, a := range c.Body {
			value := elixirQuote(a.Text)
			if a.List != nil {
				value = "[" + quoteAll(elixirQuote, a.List) + "]"
			}
			fields = append(fields, elixirQuote(a.Field.Wire)+" => "+value)
		}
		item := "%{" + strings.Join(fields, ", ") + "}"
		if m.BodyList {
			item = "[" + item + "]"
		}
		args = append(args, item)
	}
	var options []string
	for _, q := range c.Query {
		name := Snake(q.Param.Name)
		if exReserved[name] {
			name += "_"
		}
		switch {
		case q.Param.Number():
			options = append(options, name+": "+strconv.Itoa(q.Number))
		case q.Param.List():
			options = append(options, name+": ["+quoteAll(elixirQuote, q.List)+"]")
		default:
			options = append(options, name+": "+elixirQuote(q.Text))
		}
	}
	if c.Key != "" {
		options = append(options, "idempotency_key: "+elixirQuote(c.Key))
	}
	args = append(args, options...)
	return "Meeng." + Pascal(m.Service) + "." + exName(m.Name) + "(" + strings.Join(args, ", ") + ")"
}

func exName(name string) string {
	name = Snake(name)
	if exReserved[name] {
		name += "_"
	}
	return name
}
