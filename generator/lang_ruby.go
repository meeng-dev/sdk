package generator

import (
	"strings"
	"text/template"
)

// rbReserved are names a Ruby keyword argument or reader cannot take: the
// language's keywords, and the methods every object answers that a reader
// would shadow.
var rbReserved = map[string]bool{
	"alias": true, "and": true, "begin": true, "break": true, "case": true, "class": true, "def": true, "defined?": true,
	"do": true, "else": true, "elsif": true, "end": true, "ensure": true, "false": true, "for": true, "if": true, "in": true,
	"module": true, "next": true, "nil": true, "not": true, "or": true, "redo": true, "rescue": true, "retry": true,
	"return": true, "self": true, "super": true, "then": true, "true": true, "undef": true, "unless": true, "until": true,
	"when": true, "while": true, "yield": true, "hash": true, "send": true, "object_id": true, "freeze": true,
	"to_h": true, "to_s": true, "inspect": true, "initialize": true, "dup": true, "clone": true,
}

func init() {
	register(Language{Name: "ruby", Funcs: template.FuncMap{
		// quote writes a string literal, escaped the way ruby reads it.
		"quote":  rubyQuote,
		"rbName": rbName,
	}})
}

// rbName is a wire name as a Ruby reader and keyword argument.
func rbName(name string) string {
	name = strings.ReplaceAll(name, "-", "_")
	if rbReserved[name] {
		return name + "_"
	}
	return name
}
