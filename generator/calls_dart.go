package generator

import (
	"strconv"
	"strings"
)

func init() {
	callers["dart"] = dartCall
}

// dartCall is the call of a contract test in Dart: meeng is the Meeng client,
// and the call is a Future to await.
func dartCall(c *ContractCall) string {
	m := c.Method
	args := []string{}
	for _, value := range c.Path {
		args = append(args, dartQuote(value))
	}
	if m.Body != nil {
		var fields []string
		for _, a := range c.Body {
			value := dartQuote(a.Text)
			if a.List != nil {
				value = "[" + quoteAll(dartQuote, a.List) + "]"
			}
			fields = append(fields, dartName(a.Field.Wire)+": "+value)
		}
		item := "const " + m.Body.Name + "(" + strings.Join(fields, ", ") + ")"
		if m.BodyList {
			item = "const [" + item[len("const "):] + "]"
		}
		args = append(args, item)
	}
	for _, q := range c.Query {
		name := dartName(q.Param.Name)
		switch {
		case q.Param.Number():
			args = append(args, name+": "+strconv.Itoa(q.Number))
		case q.Param.List():
			args = append(args, name+": ["+quoteAll(dartQuote, q.List)+"]")
		default:
			args = append(args, name+": "+dartQuote(q.Text))
		}
	}
	if c.Key != "" {
		args = append(args, "idempotencyKey: "+dartQuote(c.Key))
	}
	return "meeng." + dartName(m.Service) + "." + dartName(m.Name) + "(" + strings.Join(args, ", ") + ")"
}
