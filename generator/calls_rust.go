package generator

import (
	"strconv"
	"strings"
)

func init() {
	callers["rust"] = rustCall
}

// rustCall is the call of a contract test in Rust: client is the Client, and
// the call answers a Result.
func rustCall(c *ContractCall) string {
	m := c.Method
	args := []string{}
	for _, value := range c.Path {
		args = append(args, rustQuote(value))
	}
	if m.Body != nil {
		var fields []string
		for _, a := range c.Body {
			if a.List != nil {
				items := make([]string, len(a.List))
				for i, v := range a.List {
					items[i] = rustQuote(v) + ".to_owned()"
				}
				fields = append(fields, "vec!["+strings.Join(items, ", ")+"]")
			} else {
				fields = append(fields, rustQuote(a.Text))
			}
		}
		item := m.Body.Name + "::new(" + strings.Join(fields, ", ") + ")"
		if m.BodyList {
			item = "&[" + item + "]"
		} else {
			item = "&" + item
		}
		args = append(args, item)
	}
	if len(m.Query) > 0 {
		query := "&" + Pascal(m.Service) + Pascal(m.Name) + "Query::new()"
		for _, q := range c.Query {
			name := rsName(q.Param.Name)
			switch {
			case q.Param.Number():
				query += "." + name + "(" + strconv.Itoa(q.Number) + ")"
			case q.Param.List():
				query += "." + name + "([" + quoteAll(rustQuote, q.List) + "])"
			default:
				query += "." + name + "(" + rustQuote(q.Text) + ")"
			}
		}
		args = append(args, query)
	}
	service := "client." + rsName(Snake(m.Service)) + "()"
	if c.Key != "" {
		service += ".idempotency_key(" + rustQuote(c.Key) + ")"
	}
	return service + "." + rsName(Snake(m.Name)) + "(" + strings.Join(args, ", ") + ")"
}
