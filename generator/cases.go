package generator

import (
	"fmt"
	"sort"
	"strings"
)

// A contract test calls every method of an SDK against the fake API that is
// built from the contract (package fakeapi). The calls are described here once,
// the same for every language: the arguments a method takes, with values the
// contract accepts. Each language renders them as a call in its own syntax
// (calls.go and the calls_<language>.go files), so every SDK is exercised the
// same way and none is skipped when an operation is added.

// ContractCall is one call a contract test makes.
type ContractCall struct {
	Method *Method
	// Path are the values of the method's path parameters, in order.
	Path []string
	// Query are the filters the call sends, one for each filter the method has.
	Query []*QueryArg
	// Body are the required fields of the request: the call sends those and
	// leaves every optional field out. A list body is a list of one such item.
	Body []*BodyArg
	// Key is the Idempotency-Key the call carries, or empty.
	Key string
}

// QueryArg is the value of one filter.
type QueryArg struct {
	Param *Param
	// Text is the value of a string filter; Number of an integer one; List of
	// a list of strings.
	Text   string
	Number int
	List   []string
}

// BodyArg is the value of one required field of a request.
type BodyArg struct {
	Field *Field
	// Text is the value of a string field; List of a list of strings.
	Text string
	List []string
}

// queryValues are the values the contract test sends for filters that a plain
// "sample" would not satisfy.
var queryValues = map[string]string{
	"startDate": "2026-11-01T00:00:00Z",
	"endDate":   "2026-11-30T00:00:00Z",
	"status":    "DRAFT",
	"days":      "7",
	"domainId":  "1",
	"emails":    "a@example.com",
	"ids":       "c1,c2",
	"search":    "news",
}

// ContractCalls are the calls of the contract test, one for each method, in
// the order of the surface.
func (ir *IR) ContractCalls() []*ContractCall {
	var out []*ContractCall
	for _, service := range ir.Services {
		for _, m := range service.Methods {
			call := &ContractCall{Method: m}
			for _, name := range m.PathParams {
				call.Path = append(call.Path, "id_"+strings.ToLower(name))
			}
			for _, p := range m.Query {
				arg := &QueryArg{Param: p}
				switch {
				case p.Number():
					arg.Number = 2
					if p.Name == "limit" {
						arg.Number = 10
					}
				case p.List():
					arg.List = []string{"1", "2"}
				default:
					arg.Text = queryValues[p.Name]
					if arg.Text == "" {
						arg.Text = "sample"
					}
				}
				call.Query = append(call.Query, arg)
			}
			if m.Body != nil {
				for _, f := range m.Body.Fields {
					if !f.Required {
						continue
					}
					arg := &BodyArg{Field: f}
					switch {
					case f.Type.Kind == "string" && len(f.Type.Enum) == 0:
						arg.Text = bodyText(f.Wire)
					case f.Type.Kind == "array" && f.Type.Items != nil && f.Type.Items.Kind == "string":
						arg.List = []string{bodyText(f.Wire)}
						if f.Wire == "contactIds" {
							arg.List = []string{"c1", "c2"}
						}
					default:
						panic(fmt.Sprintf("generator: %s requires %s, a %s, and the contract tests only build strings and lists of strings: teach generator/cases.go", m.OperationID, f.Wire, f.Type.Kind))
					}
					call.Body = append(call.Body, arg)
				}
			}
			if m.Idempotent {
				call.Key = "contract-" + m.Service + "-" + m.Name
			}
			out = append(out, call)
		}
	}
	return out
}

func bodyText(wire string) string {
	switch wire {
	case "to", "from", "email", "replyTo", "cc", "bcc":
		return "hello@example.com"
	case "scheduledAt":
		return "2026-12-01T09:00:00Z"
	case "region":
		return "us-east-1"
	case "contactBookId":
		return "id_contactbookid"
	case "name":
		return "example.com"
	}
	return "sample"
}

// Callers render a ContractCall as the call that makes it, by language. The
// call is an expression of the language that answers what the method answers
// and can fail the way a method fails, so the test around it is the same
// shape in every language.
var callers = map[string]func(*ContractCall) string{}

// ContractTypes are the names of the request and filter types the contract
// test builds, sorted, once each, for the languages that import them.
func (ir *IR) ContractTypes() []string {
	seen := map[string]bool{}
	var out []string
	add := func(name string) {
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	for _, call := range ir.ContractCalls() {
		if call.Method.Body != nil {
			add(call.Method.Body.Name)
		}
		if len(call.Method.Query) > 0 {
			add(Pascal(call.Method.Service) + Pascal(call.Method.Name) + "Query")
		}
	}
	sort.Strings(out)
	return out
}
