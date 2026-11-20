package generator

import (
	"strconv"
	"strings"
)

func init() {
	callers["swift"] = swiftCall
}

// swiftCall is the call of a contract test in Swift: meeng is the MeengClient,
// and the call is an expression to try await.
func swiftCall(c *ContractCall) string {
	m := c.Method
	args := []string{}
	for _, value := range c.Path {
		args = append(args, swiftQuote(value))
	}
	if m.Body != nil {
		var fields []string
		for _, a := range c.Body {
			value := swiftQuote(a.Text)
			if a.List != nil {
				value = "[" + quoteAll(swiftQuote, a.List) + "]"
			}
			fields = append(fields, swName(a.Field.Wire)+": "+value)
		}
		item := m.Body.Name + "(" + strings.Join(fields, ", ") + ")"
		if m.BodyList {
			item = "[" + item + "]"
		}
		args = append(args, item)
	}
	for _, q := range c.Query {
		name := swName(q.Param.Name)
		switch {
		case q.Param.Number():
			args = append(args, name+": "+strconv.Itoa(q.Number))
		case q.Param.List():
			args = append(args, name+": ["+quoteAll(swiftQuote, q.List)+"]")
		default:
			args = append(args, name+": "+swiftQuote(q.Text))
		}
	}
	if c.Key != "" {
		args = append(args, "idempotencyKey: "+swiftQuote(c.Key))
	}
	return "meeng." + swName(m.Service) + "." + swName(m.Name) + "(" + strings.Join(args, ", ") + ")"
}
