package generator

import (
	"strconv"
	"strings"
)

func init() {
	callers["java"] = javaCall
}

func javaName(wire string) string {
	name := idiomCamel(wire)
	if javaKeywords[name] {
		return name + "Value"
	}
	return name
}

// javaCall is the call of a contract test in Java: meeng is the Meeng client.
func javaCall(c *ContractCall) string {
	m := c.Method
	service := Pascal(m.Service) + "Service"
	args := []string{}
	for _, value := range c.Path {
		args = append(args, javaQuote(value))
	}
	if m.Body != nil {
		builder := service + "." + Pascal(m.Name) + "Params.builder()"
		for _, a := range c.Body {
			value := javaQuote(a.Text)
			if a.List != nil {
				value = "List.of(" + quoteAll(javaQuote, a.List) + ")"
			}
			builder += "." + javaName(a.Field.Wire) + "(" + value + ")"
		}
		item := builder + ".build()"
		if m.BodyList {
			item = "List.of(" + item + ")"
		}
		args = append(args, item)
	}
	if len(m.Query) > 0 {
		builder := service + "." + Pascal(m.Name) + "Params.builder()"
		for _, q := range c.Query {
			name := javaName(q.Param.Name)
			switch {
			case q.Param.Number():
				builder += "." + name + "(" + strconv.Itoa(q.Number) + ")"
			case q.Param.List():
				builder += "." + name + "(List.of(" + quoteAll(javaQuote, q.List) + "))"
			default:
				builder += "." + name + "(" + javaQuote(q.Text) + ")"
			}
		}
		args = append(args, builder+".build()")
	}
	if c.Key != "" {
		args = append(args, "RequestOptions.builder().idempotencyKey("+javaQuote(c.Key)+").build()")
	}
	return "meeng." + javaName(m.Service) + "()." + javaName(m.Name) + "(" + strings.Join(args, ", ") + ")"
}
