// Package generator builds meeng's SDKs from the public API's contract: the
// OpenAPI document is read into an intermediate representation that no
// language knows about -- services, methods, models -- and one emitter per
// language renders it. The SDKs under sdk/<language> are output and are never
// edited by hand; what is hand-written lives in this package's templates.
//
// The contract is the one the app exports (docs/contracts/api.openapi.json),
// which names its schemas and gives each operation an operationId of the app's
// own routing. The names the SDKs give to services, methods and models are
// written once in surface.go instead, the same in every language, and the
// webhook events, which the contract does not describe, in webhooks.go.
package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// IR is the API as every SDK sees it.
type IR struct {
	// Version is the SDK's version, the same in every language.
	Version string
	// APIVersion is the contract's info.version.
	APIVersion string
	// BaseURL is the address the SDKs call by default, without the API
	// prefix: https://app.meeng.dev. The environment variable MEENG_BASE_URL and
	// the client's base URL option name the same thing.
	BaseURL string
	// APIPrefix is the path every request starts with, from the contract's
	// server: /api. The paths of the operations start with /v1.
	APIPrefix string
	Models    []*Model
	Services  []*Service
	// Webhooks are the events the API posts to an endpoint, grouped by the
	// object their data carries.
	Webhooks []*WebhookGroup
}

// WebhookGroup is the events whose data is one kind of object: email.queued
// and email.sent carry an EmailBasePayload. Each SDK declares one event type a
// group, and the union of them is what a verified webhook answers; the event's
// type tells the members of a group apart.
type WebhookGroup struct {
	// Name names the group's event type: Email is EmailWebhookEvent.
	Name string
	// Data is the type of the event's data: a model.
	Data *TypeRef
	// Types are the event types the group carries, in the order they are
	// declared.
	Types []string
}

// Model is a named object: a resource the API answers, or the body of a
// request.
type Model struct {
	Name        string
	Description string
	Fields      []*Field
	// Input says the model is sent, not read.
	Input bool
	// Refused are the fields the contract lists and the API takes only empty
	// (attachments, ADR-0008); no SDK has them.
	Refused []string
}

// Field is one property of a model.
type Field struct {
	// Wire is the JSON name.
	Wire        string
	Description string
	Type        *TypeRef
	Required    bool
	Nullable    bool
}

// TypeRef is a field's type.
type TypeRef struct {
	// Kind is string, integer, number, boolean, model, array, map or any: a
	// free-form JSON value, which a schema with no type allows.
	Kind  string
	Model string
	Items *TypeRef
	// Enum are a string's allowed values.
	Enum   []string
	Format string
	// OrScalar says an array of strings that the API also takes, and may
	// answer, as one string: to, cc, bcc and replyTo. An SDK sends the list; it
	// reads a lone string as a list of one.
	OrScalar bool
	// Nested is the model itself when it is written inline in a request body
	// and exists nowhere else: a language that has no type of its own for it
	// describes it where it is used.
	Nested *Model
}

// Service groups the methods of one resource: emails, domains.
type Service struct {
	Name    string
	Methods []*Method
}

// Method is one operation.
type Method struct {
	// Service is the resource the method belongs to: emails.
	Service string
	// Name is the method's name in its service: send, list, bulkCreate.
	Name string
	// OperationID is service.name: emails.send.
	OperationID string
	Summary     string
	// SummaryPT is the summary as the READMEs say it, in Brazilian Portuguese.
	SummaryPT  string
	HTTPMethod string
	Path       string
	PathParams []string
	// Query are the optional filters.
	Query []*Param
	// Body is the request model, or nil. When BodyList is true the request
	// body is a list of them.
	Body     *Model
	BodyList bool
	// Nested are the models Body holds that are written inline, innermost
	// first, for the languages that write a request body where it is used.
	Nested []*Model
	// Response is the model answered. When ResponseList is true the answer is a
	// list of them, with no object around it.
	Response     string
	ResponseList bool
	// Aliases are other names the dynamic clients give the method.
	Aliases []string
	// Idempotent says it takes an Idempotency-Key.
	Idempotent bool
	// Write says the verb is not GET.
	Write bool
	// EmptyBody says the call is a POST or a PUT with no body of its own, which
	// the SDKs send as {}.
	EmptyBody bool
}

// Param is a query filter.
type Param struct {
	Name, Description string
	// Kind is string, integer or stringlist: a list is sent once for each
	// value, as name=a&name=b.
	Kind string
}

// List says the filter is a list of strings.
func (p *Param) List() bool { return p.Kind == "stringlist" }

// Number says the filter is an integer.
func (p *Param) Number() bool { return p.Kind == "integer" }

type spec struct {
	Info struct {
		Version string `json:"version"`
	} `json:"info"`
	Servers []struct {
		URL string `json:"url"`
	} `json:"servers"`
	Paths      map[string]map[string]*operation `json:"paths"`
	Components struct {
		Schemas map[string]*schema `json:"schemas"`
	} `json:"components"`
}

type parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"`
	Description string  `json:"description"`
	Schema      *schema `json:"schema"`
}

type operation struct {
	Parameters  []*parameter `json:"parameters"`
	RequestBody *struct {
		Content map[string]struct {
			Schema *schema `json:"schema"`
		} `json:"content"`
	} `json:"requestBody"`
	Responses map[string]struct {
		Content map[string]struct {
			Schema *schema `json:"schema"`
		} `json:"content"`
	} `json:"responses"`
}

// properties is an object's properties in the order the contract writes them.
type properties struct {
	Keys []string
	Map  map[string]*schema
}

func (p *properties) UnmarshalJSON(data []byte) error {
	p.Map = map[string]*schema{}
	dec := json.NewDecoder(bytes.NewReader(data))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return fmt.Errorf("properties is not an object")
	}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		key, _ := tok.(string)
		var s schema
		if err := dec.Decode(&s); err != nil {
			return err
		}
		if _, dup := p.Map[key]; !dup {
			p.Keys = append(p.Keys, key)
		}
		p.Map[key] = &s
	}
	_, err := dec.Token()
	return err
}

type schema struct {
	Ref         string `json:"$ref"`
	Type        any    `json:"type"`
	Format      string `json:"format"`
	Description string `json:"description"`
	Nullable    bool   `json:"nullable"`
	// Enum has nil for the null a nullable schema lists among its values.
	Enum       []*string  `json:"enum"`
	Required   []string   `json:"required"`
	Properties properties `json:"properties"`
	Items      *schema    `json:"items"`
	// MaxItems is the longest a list may be; 0 with no items says a field the
	// API does not take (attachments, ADR-0008).
	MaxItems *int `json:"maxItems"`
	// AdditionalProperties is a schema, or a boolean for a closed or open object.
	AdditionalProperties json.RawMessage `json:"additionalProperties"`
	AnyOf                []*schema       `json:"anyOf"`
}

// refused says the schema is a list the API takes only empty: a field it does
// not support, which the contract keeps in the body so a caller is told why.
func (s *schema) refused() bool {
	return s.Ref == "" && s.kind() == "array" && s.Items == nil && s.MaxItems != nil && *s.MaxItems == 0
}

// isNull says the schema is the JSON null: {"type": "null"}.
func (s *schema) isNull() bool {
	text, _ := s.Type.(string)
	return text == "null"
}

// hasNull says the schema's type lists null beside another type, as OpenAPI
// 3.1 writes a nullable value: {"type": ["string", "null"]}.
func (s *schema) hasNull() bool {
	list, _ := s.Type.([]any)
	for _, each := range list {
		if text, _ := each.(string); text == "null" {
			return true
		}
	}
	return false
}

// free says the schema constrains nothing: no type, no shape, no values. Its
// value is any JSON.
func (s *schema) free() bool {
	return s.Type == nil && s.Ref == "" && len(s.AnyOf) == 0 && len(s.Properties.Keys) == 0 && s.Items == nil && len(s.Enum) == 0 && s.Format == "" && s.additional() == nil
}

// additional is the schema of a map's values, or nil.
func (s *schema) additional() *schema {
	raw := strings.TrimSpace(string(s.AdditionalProperties))
	if raw == "" || !strings.HasPrefix(raw, "{") {
		return nil
	}
	var values schema
	if json.Unmarshal(s.AdditionalProperties, &values) != nil {
		return nil
	}
	return &values
}

func (s *schema) kind() string {
	switch t := s.Type.(type) {
	case string:
		return t
	case []any:
		for _, each := range t {
			if text, _ := each.(string); text != "null" {
				return text
			}
		}
	}
	return ""
}

// Load reads the contract at path into the IR.
func Load(path, version string) (*IR, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc spec
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("generator: read %s: %w", path, err)
	}
	return build(&doc, version)
}

// loader reads schemas into models, naming the ones the contract writes
// inline.
type loader struct {
	models map[string]*Model
	// extra are the models a schema writes inline -- an object nested in a
	// request body -- named after where they sit, in the order they were met.
	extra []*Model
	// inline are those models by name.
	inline map[string]*Model
	// components are the schemas the contract names, by name. A schema that
	// refers to one is read as that schema.
	components map[string]*schema
}

func build(doc *spec, version string) (*IR, error) {
	ir := &IR{Version: version, APIVersion: doc.Info.Version}
	if len(doc.Servers) == 0 {
		return nil, fmt.Errorf("generator: the contract has no server")
	}
	if err := checkMatch("contract version", doc.Info.Version, versionText); err != nil {
		return nil, err
	}
	if err := checkMatch("SDK version", version, versionText); err != nil {
		return nil, err
	}
	if err := checkMatch("server address", doc.Servers[0].URL, baseURL); err != nil {
		return nil, err
	}
	origin, prefix := splitServer(doc.Servers[0].URL)
	paths := make([]string, 0, len(doc.Paths))
	for path := range doc.Paths {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	// The contract writes the whole path, /api/v1/emails; the SDKs keep the
	// API prefix apart, so that a client can be pointed at the app alone.
	pathPrefix, err := versionPrefix(paths)
	if err != nil {
		return nil, err
	}
	ir.BaseURL, ir.APIPrefix = origin, prefix+pathPrefix
	l := &loader{models: map[string]*Model{}, inline: map[string]*Model{}, components: doc.Components.Schemas}
	services := map[string]*Service{}
	seen := map[string]bool{}
	for _, full := range paths {
		path := strings.TrimPrefix(full, pathPrefix)
		if err := checkPath(path); err != nil {
			return nil, err
		}
		verbs := make([]string, 0, len(doc.Paths[full]))
		for verb := range doc.Paths[full] {
			verbs = append(verbs, verb)
		}
		// Sorted, so that what two routes share is described the same way every run.
		sort.Strings(verbs)
		for _, httpMethod := range verbs {
			op := doc.Paths[full][httpMethod]
			if !httpMethods[httpMethod] {
				return nil, fmt.Errorf("generator: %s is not an HTTP method the SDKs send (%s)", httpMethod, path)
			}
			key := strings.ToUpper(httpMethod) + " " + path
			def, ok := surface[key]
			if !ok {
				return nil, fmt.Errorf("generator: %s is not named in surface.go: add it, so every SDK gives it the same name", key)
			}
			seen[key] = true
			if err := checkMatch("service", def.Service, slug); err != nil {
				return nil, err
			}
			if err := checkMatch("method", def.Name, slug); err != nil {
				return nil, err
			}
			method, err := l.methodOf(op, strings.ToUpper(httpMethod), path, def)
			if err != nil {
				return nil, err
			}
			service := services[def.Service]
			if service == nil {
				service = &Service{Name: def.Service}
				services[def.Service] = service
			}
			service.Methods = append(service.Methods, method)
		}
	}
	for key := range surface {
		if !seen[key] {
			return nil, fmt.Errorf("generator: surface.go names %s, which the contract does not have", key)
		}
	}
	names := make([]string, 0, len(services))
	for name := range services {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool { return serviceRank(names[i]) < serviceRank(names[j]) })
	for _, name := range names {
		service := services[name]
		sort.SliceStable(service.Methods, func(i, j int) bool {
			return methodRank(service.Name, service.Methods[i].Name) < methodRank(service.Name, service.Methods[j].Name)
		})
		ir.Services = append(ir.Services, service)
	}
	groups, models, err := webhookSurface()
	if err != nil {
		return nil, err
	}
	for _, model := range models {
		if err := l.add(model); err != nil {
			return nil, err
		}
	}
	ir.Webhooks = groups
	all := make([]*Model, 0, len(l.models))
	for _, model := range l.models {
		all = append(all, model)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Name < all[j].Name })
	ir.Models = all
	return ir, nil
}

// versionPrefix is what every path of the contract has before /v1: /api. The
// contract's operations all live under one prefix, or the SDKs would have to
// guess which one a client adds.
func versionPrefix(paths []string) (string, error) {
	prefix := ""
	for i, path := range paths {
		at := strings.Index(path, "/v1/")
		if at < 0 {
			return "", fmt.Errorf("generator: path %s has no /v1/ in it", path)
		}
		if i == 0 {
			prefix = path[:at]
		} else if path[:at] != prefix {
			return "", fmt.Errorf("generator: paths do not share one prefix before /v1: %q and %q", prefix, path[:at])
		}
	}
	return prefix, nil
}

// splitServer writes https://app.meeng.dev/api as its origin and its prefix.
func splitServer(server string) (origin, prefix string) {
	rest := server
	scheme := ""
	if i := strings.Index(rest, "://"); i >= 0 {
		scheme, rest = rest[:i+3], rest[i+3:]
	}
	host, path, _ := strings.Cut(rest, "/")
	if path != "" {
		prefix = "/" + strings.TrimRight(path, "/")
	}
	return scheme + host, prefix
}

var httpMethods = map[string]bool{"get": true, "post": true, "put": true, "patch": true, "delete": true}

// add records a model; the same name twice is fine when it is the same shape.
func (l *loader) add(model *Model) error {
	if existing := l.models[model.Name]; existing != nil {
		if signature(existing) != signature(model) {
			return fmt.Errorf("generator: two different shapes are both called %s", model.Name)
		}
		return nil
	}
	l.models[model.Name] = model
	return nil
}

// signature is what makes two models the same: their fields and types, not
// their descriptions.
func signature(m *Model) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%v|", m.Input)
	for _, f := range m.Fields {
		fmt.Fprintf(&b, "%s:%s:%v:%v;", f.Wire, typeSignature(f.Type), f.Required, f.Nullable)
	}
	return b.String()
}

func typeSignature(t *TypeRef) string {
	s := t.Kind + "/" + t.Model + "/" + strings.Join(t.Enum, ",") + "/" + t.Format + fmt.Sprint(t.OrScalar)
	if t.Items != nil {
		s += "[" + typeSignature(t.Items) + "]"
	}
	return s
}

func (l *loader) methodOf(op *operation, httpMethod, path string, def operationDef) (*Method, error) {
	id := def.Service + "." + def.Name
	method := &Method{Service: def.Service, Name: def.Name, OperationID: id, Summary: def.Summary, SummaryPT: def.Readme, Aliases: def.Aliases, HTTPMethod: httpMethod, Path: path, Write: httpMethod != "GET"}
	// The path names its parameters; a contract that forgets to declare one
	// (the upsert of a contact does) still has it.
	for _, part := range pathParts(path) {
		if part.Param != "" {
			method.PathParams = append(method.PathParams, part.Param)
		}
	}
	for _, param := range op.Parameters {
		switch param.In {
		case "path":
			found := false
			for _, name := range method.PathParams {
				found = found || name == param.Name
			}
			if !found {
				return nil, fmt.Errorf("generator: %s declares the path parameter %s, which its path does not have", id, param.Name)
			}
		case "header":
			if param.Name != "Idempotency-Key" {
				return nil, fmt.Errorf("generator: %s: header %s is not one the SDKs send", id, param.Name)
			}
			method.Idempotent = true
		case "query":
			if err := checkIdentifier("query parameter", param.Name); err != nil {
				return nil, err
			}
			if param.Schema != nil {
				resolved, err := l.deref(param.Schema, map[string]bool{})
				if err != nil {
					return nil, fmt.Errorf("generator: %s: query parameter %s: %w", id, param.Name, err)
				}
				param.Schema = resolved
			}
			kind, err := queryKind(param)
			if err != nil {
				return nil, fmt.Errorf("generator: %s: %w", id, err)
			}
			text := cleanText(param.Description)
			if text == "" {
				text = queryDocs[param.Name]
			}
			method.Query = append(method.Query, &Param{Name: param.Name, Description: text, Kind: kind})
		default:
			return nil, fmt.Errorf("generator: %s: parameter %s is in %s, which the SDKs do not send", id, param.Name, param.In)
		}
	}
	method.EmptyBody = (httpMethod == "POST" || httpMethod == "PUT") && op.RequestBody == nil
	var answer *schema
	for status, response := range op.Responses {
		if len(status) == 0 || status[0] != '2' {
			continue
		}
		if status != "200" {
			return nil, fmt.Errorf("generator: %s answers %s; the contract is expected to answer 200", id, status)
		}
		answer = response.Content["application/json"].Schema
	}
	if answer == nil {
		return nil, fmt.Errorf("generator: %s answers no JSON body", id)
	}
	answer, err := l.answerOf(answer, def)
	if err != nil {
		return nil, fmt.Errorf("generator: %s: %w", id, err)
	}
	before := len(l.extra)
	response, list, err := l.rootModel(def.responseName(), answer, false)
	if err != nil {
		return nil, fmt.Errorf("generator: %s: %w", id, err)
	}
	method.Response, method.ResponseList = response.Name, list
	if err := l.add(response); err != nil {
		return nil, err
	}
	if err := l.flush(before); err != nil {
		return nil, err
	}
	if op.RequestBody != nil {
		body := op.RequestBody.Content["application/json"].Schema
		if body == nil {
			return nil, fmt.Errorf("generator: %s takes a body that is not JSON", id)
		}
		body, err := l.deref(body, map[string]bool{})
		if err != nil {
			return nil, fmt.Errorf("generator: %s: %w", id, err)
		}
		before := len(l.extra)
		model, list, err := l.rootModel(Pascal(def.Service)+Pascal(def.Name)+"Params", body, true)
		if err != nil {
			return nil, fmt.Errorf("generator: %s: %w", id, err)
		}
		model.Description = "The body of " + id + "."
		if list {
			model.Description = "One item of the list " + id + " takes."
		}
		method.Body, method.BodyList = model, list
		method.Nested = append([]*Model(nil), l.extra[before:]...)
		if err := l.add(model); err != nil {
			return nil, err
		}
		if err := l.flush(before); err != nil {
			return nil, err
		}
	}
	return method, nil
}

// componentPrefix starts the reference to a schema the contract names.
const componentPrefix = "#/components/schemas/"

// answerOf is the schema an operation answers, with every reference to a named
// schema read as that schema.
//
// The contract writes a list as a named schema that refers to itself:
// DomainList is {"$ref": "#/components/schemas/DomainList"}, which names no
// shape at all. A list is read as the list of its item, which surface.go names
// in Response (Domain for DomainList), and the contract must have that item as a
// schema of its own. Any other reference to itself stops generation.
func (l *loader) answerOf(answer *schema, def operationDef) (*schema, error) {
	if strings.HasPrefix(answer.Ref, componentPrefix) {
		name := strings.TrimPrefix(answer.Ref, componentPrefix)
		if target := l.components[name]; target != nil && target.Ref == answer.Ref {
			if def.Response == "" || name != def.Response+"List" || l.components[def.Response] == nil {
				return nil, fmt.Errorf("the contract answers %s, a schema that refers to itself: surface.go must name the item of the list in Response", name)
			}
			answer = &schema{Type: "array", Items: &schema{Ref: componentPrefix + def.Response}}
		}
	}
	return l.deref(answer, map[string]bool{})
}

// deref reads every reference to a named schema inside s as the schema it
// names, and writes the OpenAPI 3.1 forms of a nullable value the way the
// generator reads one: {"type": ["string", "null"]} and
// {"anyOf": [schema, {"type": "null"}]}. trail holds the schemas being read, so a
// schema that contains itself stops generation instead of running forever.
func (l *loader) deref(s *schema, trail map[string]bool) (*schema, error) {
	if s == nil {
		return nil, nil
	}
	if strings.HasPrefix(s.Ref, componentPrefix) {
		name := strings.TrimPrefix(s.Ref, componentPrefix)
		target := l.components[name]
		if target == nil {
			return nil, fmt.Errorf("the contract refers to the schema %s, which it does not define", name)
		}
		if trail[name] {
			return nil, fmt.Errorf("the schema %s contains itself", name)
		}
		trail[name] = true
		defer delete(trail, name)
		resolved, err := l.deref(target, trail)
		if err != nil {
			return nil, err
		}
		out := *resolved
		if s.Description != "" {
			out.Description = s.Description
		}
		if s.Nullable {
			out.Nullable = true
		}
		return &out, nil
	}
	out := *s
	if len(s.Properties.Keys) > 0 {
		out.Properties = properties{Keys: s.Properties.Keys, Map: map[string]*schema{}}
		for _, key := range s.Properties.Keys {
			property, err := l.deref(s.Properties.Map[key], trail)
			if err != nil {
				return nil, err
			}
			out.Properties.Map[key] = property
		}
	}
	if s.Items != nil {
		items, err := l.deref(s.Items, trail)
		if err != nil {
			return nil, err
		}
		out.Items = items
	}
	if len(s.AnyOf) > 0 {
		out.AnyOf = make([]*schema, len(s.AnyOf))
		for i, each := range s.AnyOf {
			member, err := l.deref(each, trail)
			if err != nil {
				return nil, err
			}
			out.AnyOf[i] = member
		}
		// A value that is a schema or null is that schema, nullable.
		if len(out.AnyOf) == 2 && (out.AnyOf[0].isNull() != out.AnyOf[1].isNull()) {
			member := out.AnyOf[0]
			if member.isNull() {
				member = out.AnyOf[1]
			}
			merged := *member
			merged.Nullable = true
			if merged.Description == "" {
				merged.Description = out.Description
			}
			return &merged, nil
		}
	}
	if out.hasNull() {
		out.Nullable = true
	}
	return &out, nil
}

// flush records the models met since before.
func (l *loader) flush(before int) error {
	for _, model := range l.extra[before:] {
		if err := l.add(model); err != nil {
			return err
		}
	}
	l.extra = l.extra[:before]
	return nil
}

// rootModel reads the schema of a body or an answer: an object, or a list of
// objects, which is answered as list.
func (l *loader) rootModel(name string, s *schema, input bool) (*Model, bool, error) {
	if s.kind() == "array" && s.Items != nil {
		model, err := l.modelOf(name, s.Items, input)
		return model, true, err
	}
	if s.kind() != "object" || len(s.Properties.Keys) == 0 {
		return nil, false, fmt.Errorf("a body or an answer that is neither an object nor a list of them")
	}
	model, err := l.modelOf(name, s, input)
	return model, false, err
}

func (l *loader) modelOf(name string, s *schema, input bool) (*Model, error) {
	if err := checkIdentifier("model", name); err != nil {
		return nil, err
	}
	model := &Model{Name: name, Description: cleanText(s.Description), Input: input}
	required := map[string]bool{}
	for _, field := range s.Required {
		required[field] = true
	}
	for _, wire := range s.Properties.Keys {
		property := s.Properties.Map[wire]
		if err := checkIdentifier("field of "+name, wire); err != nil {
			return nil, err
		}
		if input && property.refused() {
			// The API refuses this field with a 400 when it holds anything, so no
			// SDK offers it: a caller cannot send what the type does not have.
			model.Refused = append(model.Refused, wire)
			continue
		}
		property, err := l.nest(name, wire, property, input)
		if err != nil {
			return nil, err
		}
		typ, nullable, err := l.typeOf(wire, property)
		if err != nil {
			return nil, fmt.Errorf("%s.%s: %w", name, wire, err)
		}
		model.Fields = append(model.Fields, &Field{Wire: wire, Description: cleanText(property.Description), Type: typ, Required: required[wire], Nullable: nullable || property.Nullable})
	}
	return model, nil
}

// nest turns an object a schema writes inline -- a property that is an object
// with properties of its own, or an array of such objects -- into a model of
// its own, named after the model it sits in and the property: the attachments
// of a body would be its Attachment, and the dnsRecords of Domain
// its DomainDnsRecord. The property is read as a reference to that model.
func (l *loader) nest(parent, wire string, property *schema, input bool) (*schema, error) {
	base := strings.TrimSuffix(strings.TrimSuffix(parent, "Params"), "Response")
	switch {
	case property.Ref == "" && property.kind() == "object" && len(property.Properties.Keys) > 0:
		return l.nested(nestedName(base, Pascal(wire)), property, input, "The "+wire+" of "+parent+".")
	case property.Ref == "" && property.kind() == "array" && property.Items != nil && property.Items.kind() == "object" && len(property.Items.Properties.Keys) > 0:
		noun := "Item"
		if wire != "data" {
			noun = Pascal(singularWord(wire))
		}
		item, err := l.nested(nestedName(base, noun), property.Items, input, "One item of "+wire+" in "+parent+".")
		if err != nil {
			return nil, err
		}
		return &schema{Type: "array", Nullable: property.Nullable, Description: property.Description, Items: item}, nil
	}
	return property, nil
}

// nestedName is base and noun joined, unless noun already starts with base:
// Email and EmailEvent are EmailEvent, not EmailEmailEvent.
func nestedName(base, noun string) string {
	if strings.HasPrefix(noun, base) {
		return noun
	}
	return base + noun
}

func singularWord(word string) string {
	switch {
	case strings.HasSuffix(word, "ies"):
		return strings.TrimSuffix(word, "ies") + "y"
	case strings.HasSuffix(word, "s"):
		return strings.TrimSuffix(word, "s")
	}
	return word
}

func (l *loader) nested(name string, s *schema, input bool, fallback string) (*schema, error) {
	model, err := l.modelOf(name, s, input)
	if err != nil {
		return nil, err
	}
	if model.Description == "" {
		model.Description = fallback
	}
	l.extra = append(l.extra, model)
	l.inline[name] = model
	return &schema{Ref: "#/nested/" + name, Nullable: s.Nullable, Description: s.Description}, nil
}

// idNames are the numbers the contract writes as number that the SDKs read as
// integers: an id is not a fraction.
var idNames = map[string]bool{"id": true, "count": true, "contacts": true}

func (l *loader) typeOf(wire string, s *schema) (*TypeRef, bool, error) {
	if s.Ref != "" {
		name := strings.TrimPrefix(s.Ref, "#/nested/")
		if l.inline[name] == nil {
			return nil, false, fmt.Errorf("a reference the contract does not define")
		}
		return &TypeRef{Kind: "model", Model: name, Nested: l.inline[name]}, false, nil
	}
	if len(s.AnyOf) > 0 {
		return anyOfType(s)
	}
	nullable := s.Nullable
	kind := s.kind()
	if kind == "" && s.free() {
		return &TypeRef{Kind: "any"}, nullable, nil
	}
	switch kind {
	case "string", "integer", "number", "boolean":
		var enum []string
		for _, value := range s.Enum {
			if value == nil {
				nullable = true
				continue
			}
			if err := checkMatch("allowed value", *value, enumValue); err != nil {
				return nil, false, err
			}
			enum = append(enum, *value)
		}
		if kind == "number" && (idNames[wire] || strings.HasSuffix(wire, "Id")) {
			kind = "integer"
		}
		return &TypeRef{Kind: kind, Enum: enum, Format: s.Format}, nullable, nil
	case "array":
		if s.Items == nil {
			return nil, false, fmt.Errorf("an array with no items")
		}
		items, _, err := l.typeOf(wire, s.Items)
		return &TypeRef{Kind: "array", Items: items}, nullable, err
	case "object":
		if values := s.additional(); values != nil {
			items, _, err := l.typeOf(wire, values)
			return &TypeRef{Kind: "map", Items: items}, nullable, err
		}
		if len(s.Properties.Keys) == 0 {
			return &TypeRef{Kind: "map", Items: &TypeRef{Kind: "any"}}, nullable, nil
		}
	}
	return nil, false, fmt.Errorf("unsupported schema type %v", s.Type)
}

// anyOfType reads the one union the contract uses: a string or a list of
// strings, which may also be null.
func anyOfType(s *schema) (*TypeRef, bool, error) {
	nullable := s.Nullable
	var scalar, list bool
	for _, each := range s.AnyOf {
		switch {
		case each.isNull(), each.Nullable && each.free():
			nullable = true
		case each.kind() == "string" && len(each.Enum) == 0:
			scalar = true
		case each.kind() == "array" && each.Items != nil && each.Items.kind() == "string":
			list = true
		default:
			return nil, false, fmt.Errorf("anyOf is read for one shape only: a string or a list of strings")
		}
	}
	if !scalar || !list {
		return nil, false, fmt.Errorf("anyOf is read for one shape only: a string or a list of strings")
	}
	return &TypeRef{Kind: "array", Items: &TypeRef{Kind: "string"}, OrScalar: true}, nullable, nil
}

// queryKind is the kind of a query filter.
func queryKind(p *parameter) (string, error) {
	if p.Name == "page" || p.Name == "limit" {
		return "integer", nil
	}
	if p.Schema == nil {
		return "", fmt.Errorf("query parameter %s has no schema", p.Name)
	}
	if len(p.Schema.AnyOf) > 0 {
		if typ, _, err := anyOfType(p.Schema); err == nil && typ.OrScalar {
			return "stringlist", nil
		}
		return "", fmt.Errorf("query parameter %s is a union the SDKs cannot send", p.Name)
	}
	switch p.Schema.kind() {
	case "string":
		return "string", nil
	case "number", "integer":
		return "integer", nil
	}
	return "", fmt.Errorf("query parameter %s is neither a string, a number nor a list of strings", p.Name)
}

// Pascal writes payment_methods as PaymentMethods and contactBooks as
// ContactBooks.
func Pascal(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool { return r == '_' || r == '-' || r == '.' || r == ' ' })
	for i, part := range parts {
		switch strings.ToLower(part) {
		case "id", "url":
			parts[i] = strings.ToUpper(part)
		default:
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

// Camel writes payment_methods as paymentMethods.
func Camel(name string) string {
	pascal := Pascal(name)
	// The leading acronym is lowercase whole: id, idToken, url, urlPath.
	for _, acronym := range []string{"ID", "URL"} {
		if strings.HasPrefix(pascal, acronym) {
			return strings.ToLower(acronym) + pascal[len(acronym):]
		}
	}
	return strings.ToLower(pascal[:1]) + pascal[1:]
}

// Snake writes contactBooks as contact_books and firstName as first_name.
func Snake(name string) string {
	var b strings.Builder
	runes := []rune(name)
	for i, r := range runes {
		switch {
		case r == '-' || r == ' ' || r == '.':
			b.WriteRune('_')
		case r >= 'A' && r <= 'Z':
			if i > 0 && runes[i-1] != '_' && !(runes[i-1] >= 'A' && runes[i-1] <= 'Z') {
				b.WriteRune('_')
			}
			b.WriteRune(r + 32)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
