package generator

import (
	"strconv"
	"strings"
)

func init() {
	callers["csharp"] = csharpCall
}

// csharpCall is the call of a contract test in C#: meeng is the MeengClient,
// and the call is a Task to await.
func csharpCall(c *ContractCall) string {
	m := c.Method
	args := []string{}
	for _, value := range c.Path {
		args = append(args, csharpQuote(value))
	}
	if m.Body != nil {
		var fields []string
		for _, a := range c.Body {
			value := csharpQuote(a.Text)
			if a.List != nil {
				value = "[" + quoteAll(csharpQuote, a.List) + "]"
			}
			fields = append(fields, csProp(m.Body.Name, a.Field.Wire)+" = "+value)
		}
		item := "new " + m.Body.Name + " { " + strings.Join(fields, ", ") + " }"
		if len(fields) == 0 {
			item = "new " + m.Body.Name + "()"
		}
		if m.BodyList {
			item = "[" + item + "]"
		}
		args = append(args, item)
	}
	if len(m.Query) > 0 {
		var fields []string
		for _, q := range c.Query {
			name := idiomPascal(q.Param.Name)
			switch {
			case q.Param.Number():
				fields = append(fields, name+" = "+strconv.Itoa(q.Number))
			case q.Param.List():
				fields = append(fields, name+" = ["+quoteAll(csharpQuote, q.List)+"]")
			default:
				fields = append(fields, name+" = "+csharpQuote(q.Text))
			}
		}
		args = append(args, "new "+Pascal(m.Service)+Pascal(m.Name)+"Query { "+strings.Join(fields, ", ")+" }")
	}
	if c.Key != "" {
		args = append(args, "new RequestOptions { IdempotencyKey = "+csharpQuote(c.Key)+" }")
	}
	return "meeng." + Pascal(m.Service) + "." + Pascal(m.Name) + "Async(" + strings.Join(args, ", ") + ")"
}

func csProp(model, wire string) string {
	name := idiomPascal(wire)
	if name == model {
		return name + "Value"
	}
	return name
}
