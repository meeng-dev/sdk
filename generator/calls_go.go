package generator

import (
	"strconv"
	"strings"
)

func init() {
	callers["go"] = goCall
}

// goCall is the call of a contract test in Go: client is the *Client, ctx the
// context, and the call answers (value, error).
func goCall(c *ContractCall) string {
	m := c.Method
	args := []string{"ctx"}
	for _, value := range c.Path {
		args = append(args, goQuote(value))
	}
	if m.Body != nil {
		item := goBodyLiteral(c)
		if m.BodyList {
			item = "[]" + m.Body.Name + "{" + item + "}"
		}
		args = append(args, item)
	}
	if len(m.Query) > 0 {
		var fields []string
		for _, q := range c.Query {
			switch {
			case q.Param.Number():
				fields = append(fields, goName(q.Param.Name)+": "+strconv.Itoa(q.Number))
			case q.Param.List():
				fields = append(fields, goName(q.Param.Name)+": []string{"+quoteAll(goQuote, q.List)+"}")
			default:
				fields = append(fields, goName(q.Param.Name)+": "+goQuote(q.Text))
			}
		}
		args = append(args, "&"+Pascal(m.Service)+Pascal(m.Name)+"Query{"+strings.Join(fields, ", ")+"}")
	}
	if c.Key != "" {
		args = append(args, "WithIdempotencyKey("+goQuote(c.Key)+")")
	}
	return "client." + Pascal(m.Service) + "." + Pascal(m.Name) + "(" + strings.Join(args, ", ") + ")"
}

func goBodyLiteral(c *ContractCall) string {
	var fields []string
	for _, a := range c.Body {
		switch {
		case a.List != nil:
			fields = append(fields, goName(a.Field.Wire)+": "+goBase(a.Field.Type)+"{"+quoteAll(goQuote, a.List)+"}")
		default:
			fields = append(fields, goName(a.Field.Wire)+": "+goQuote(a.Text))
		}
	}
	return c.Method.Body.Name + "{" + strings.Join(fields, ", ") + "}"
}

func quoteAll(quote func(string) string, values []string) string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = quote(v)
	}
	return strings.Join(out, ", ")
}
