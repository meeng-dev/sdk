package generator

import (
	"strings"
	"text/template"
)

func init() {
	register(Language{Name: "php", Funcs: template.FuncMap{
		// quote writes a string literal, escaped the way php reads it.
		"quote":   phpQuote,
		"phpName": phpName,
		// phpQuery is the PHPStan type of a query filter.
		"phpQuery": func(p *Param) string {
			switch p.Kind {
			case "integer":
				return "int"
			case "stringlist":
				return "string|list<string>"
			}
			return "string"
		},
	}, ForIR: func(ir *IR) template.FuncMap {
		models := map[string]*Model{}
		for _, m := range ir.Models {
			models[m.Name] = m
		}
		shape := func(m *Model) string { return phpShape(models, m) }
		return template.FuncMap{
			"phpShape": shape,
			// phpAnswer is the type a method returns: the shape of its model, or
			// a list of them.
			"phpAnswer": func(m *Method) string {
				if m.ResponseList {
					return "list<" + shape(models[m.Response]) + ">"
				}
				return shape(models[m.Response])
			},
			// phpBody is the type of a method's body.
			"phpBody": func(m *Method) string {
				if m.BodyList {
					return "list<" + shape(m.Body) + ">"
				}
				return shape(m.Body)
			},
			"phpData": func(t *TypeRef) string { return phpDoc(models, t) },
		}
	}})
}

// phpDoc is a type as phpdoc and PHPStan read it: lists, maps, the allowed
// values of a string and the shape of a model, written where it is used.
func phpDoc(models map[string]*Model, t *TypeRef) string {
	switch t.Kind {
	case "string":
		if len(t.Enum) > 0 {
			values := make([]string, len(t.Enum))
			for i, value := range t.Enum {
				values[i] = "'" + value + "'"
			}
			return strings.Join(values, "|")
		}
		return "string"
	case "integer":
		return "int"
	case "number":
		return "int|float"
	case "boolean":
		return "bool"
	case "model":
		return phpShape(models, models[t.Model])
	case "array":
		if t.OrScalar {
			return "string|list<string>"
		}
		return "list<" + phpDoc(models, t.Items) + ">"
	case "map":
		return "array<string, " + phpDoc(models, t.Items) + ">"
	}
	return "mixed"
}

// phpShape is a model as a PHPStan array shape.
func phpShape(models map[string]*Model, m *Model) string {
	fields := make([]string, len(m.Fields))
	for i, f := range m.Fields {
		optional := ""
		if !f.Required {
			optional = "?"
		}
		typ := phpDoc(models, f.Type)
		if f.Nullable {
			typ += "|null"
		}
		fields[i] = "'" + f.Wire + "'" + optional + ": " + typ
	}
	return "array{" + strings.Join(fields, ", ") + "}"
}

// phpName writes a name as a PHP variable or property: contact_book_id and
// contactBookId are contactBookId.
func phpName(wire string) string {
	parts := strings.FieldsFunc(wire, func(r rune) bool { return r == '_' || r == '-' })
	for i, part := range parts {
		if i > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}
