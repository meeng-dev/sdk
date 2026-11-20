// Package fakeapi is a fake of meeng's public API, built from the API's
// contract and from nothing else. It answers every operation of the contract
// with a response that satisfies the contract's schema, and it refuses what the
// contract does not allow: a route, a verb, a query filter, a header or a body
// field the contract does not document, a missing required field, a wrong type.
// The SDKs are run against it, so an SDK that drifts from the contract fails
// the contract test of its language.
//
// Its own endpoints, which the real API does not have:
//
//	GET    /__coverage   the operations answered, and those never called
//	GET    /__requests   what was received, in order
//	DELETE /__requests   forgets both
//
// A path parameter chooses an answer, so that a test can reach an error through
// any SDK, which has no way to send a header of its own:
//
//	fake-<status>        always that status: fake-404, fake-429, fake-503
//	flaky-<status>-<n>   that status n times, then the answer of the contract
package fakeapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Call is one request the server received.
type Call struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Query  string `json:"query,omitempty"`
	Status int    `json:"status"`
	// Operation is "METHOD /template" when the request matched a route.
	Operation      string `json:"operation,omitempty"`
	IdempotencyKey string `json:"idempotencyKey,omitempty"`
	Body           string `json:"body,omitempty"`
}

type route struct {
	method   string
	template string
	segments []string
	op       map[string]any
}

func (r *route) key() string { return r.method + " " + r.template }

type replay struct {
	hash string
	body []byte
}

// Server is the fake API. It is an http.Handler.
type Server struct {
	sch    *schemas
	routes []*route

	mu     sync.Mutex
	calls  []Call
	hits   map[string]int
	replay map[string]replay
	flaky  map[string]int
	serial int
}

// New reads the contract, an OpenAPI 3.x document in JSON.
func New(contract []byte) (*Server, error) {
	var doc map[string]any
	if err := json.Unmarshal(contract, &doc); err != nil {
		return nil, fmt.Errorf("fakeapi: read the contract: %w", err)
	}
	s := &Server{hits: map[string]int{}, replay: map[string]replay{}, flaky: map[string]int{}}
	components, _ := doc["components"].(map[string]any)
	s.sch = &schemas{components: map[string]any{}}
	if all, ok := components["schemas"].(map[string]any); ok {
		s.sch.components = all
	}
	paths, _ := doc["paths"].(map[string]any)
	for template, item := range paths {
		methods, _ := item.(map[string]any)
		for verb, op := range methods {
			operation, ok := op.(map[string]any)
			if !ok || verb == "parameters" {
				continue
			}
			s.routes = append(s.routes, &route{method: strings.ToUpper(verb), template: template, segments: strings.Split(strings.Trim(template, "/"), "/"), op: operation})
		}
	}
	if len(s.routes) == 0 {
		return nil, fmt.Errorf("fakeapi: the contract has no operation")
	}
	sort.Slice(s.routes, func(i, j int) bool { return s.routes[i].key() < s.routes[j].key() })
	return s, nil
}

// Operations are the contract's operations, "METHOD /template", sorted.
func (s *Server) Operations() []string {
	out := make([]string, len(s.routes))
	for i, r := range s.routes {
		out[i] = r.key()
	}
	return out
}

// Calls are the requests received so far.
func (s *Server) Calls() []Call {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Call(nil), s.calls...)
}

// Missing are the operations that were never answered with success.
func (s *Server) Missing() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for _, r := range s.routes {
		if s.hits[r.key()] == 0 {
			out = append(out, r.key())
		}
	}
	return out
}

// Reset forgets the requests, the coverage and what the idempotency keys and
// the flaky answers remember.
func (s *Server) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls, s.hits, s.replay, s.flaky, s.serial = nil, map[string]int{}, map[string]replay{}, map[string]int{}, 0
}

func (s *Server) match(method, path string) (found *route, params map[string]string, other bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for _, r := range s.routes {
		if len(r.segments) != len(parts) {
			continue
		}
		values := map[string]string{}
		ok := true
		for i, segment := range r.segments {
			if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
				if parts[i] == "" {
					ok = false
					break
				}
				values[segment[1:len(segment)-1]] = parts[i]
			} else if segment != parts[i] {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		if r.method == method {
			return r, values, false
		}
		other = true
	}
	return nil, nil, other
}

var errorCodes = map[int]string{400: "BAD_REQUEST", 401: "UNAUTHORIZED", 403: "FORBIDDEN", 404: "NOT_FOUND", 405: "METHOD_NOT_ALLOWED", 409: "NOT_UNIQUE", 429: "RATE_LIMITED"}

func (s *Server) fail(w http.ResponseWriter, status int, message string) {
	code := errorCodes[status]
	if code == "" {
		code = "INTERNAL_SERVER_ERROR"
	}
	if status == http.StatusTooManyRequests {
		w.Header().Set("Retry-After", "0")
	}
	s.write(w, status, mustJSON(map[string]any{"error": map[string]any{"code": code, "message": message}}))
}

func (s *Server) write(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func mustJSON(v any) []byte {
	out, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return out
}

var bearer = regexp.MustCompile(`^Bearer mg_[A-Za-z0-9_]+$`)

var simulated = regexp.MustCompile(`^(fake)-(\d{3})$|^(flaky)-(\d{3})-(\d+)$`)

// ServeHTTP answers one request.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/__requests":
		if r.Method == http.MethodDelete {
			s.Reset()
		}
		s.write(w, 200, mustJSON(s.Calls()))
		return
	case "/__coverage":
		s.write(w, 200, mustJSON(map[string]any{"operations": s.Operations(), "missing": nonNil(s.Missing())}))
		return
	}
	raw, _ := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	call := Call{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, IdempotencyKey: r.Header.Get("Idempotency-Key"), Body: string(raw)}
	status := s.answer(w, r, raw, &call)
	call.Status = status
	s.mu.Lock()
	if len(s.calls) < 10000 {
		s.calls = append(s.calls, call)
	}
	s.mu.Unlock()
}

func nonNil(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}

func (s *Server) answer(w http.ResponseWriter, r *http.Request, raw []byte, call *Call) int {
	reply := func(status int, message string) int {
		s.fail(w, status, message)
		return status
	}
	found, params, other := s.match(r.Method, r.URL.Path)
	if found == nil {
		if other {
			return reply(405, "Method not allowed: "+r.Method+" "+r.URL.Path)
		}
		return reply(404, "Not found: "+r.Method+" "+r.URL.Path)
	}
	call.Operation = found.key()
	header := r.Header.Get("Authorization")
	if header == "" {
		return reply(401, "No Authorization header provided")
	}
	if !bearer.MatchString(header) {
		return reply(403, "Invalid API token")
	}
	names := make([]string, 0, len(params))
	for name := range params {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if m := simulated.FindStringSubmatch(params[name]); m != nil {
			if m[1] == "fake" {
				code, _ := strconv.Atoi(m[2])
				return reply(code, "Simulated "+m[2])
			}
			code, _ := strconv.Atoi(m[4])
			times, _ := strconv.Atoi(m[5])
			s.mu.Lock()
			seen := s.flaky[r.URL.Path]
			s.flaky[r.URL.Path]++
			s.mu.Unlock()
			if seen < times {
				return reply(code, "Simulated "+m[4]+", "+strconv.Itoa(seen+1)+" of "+m[5])
			}
		}
	}
	if issues := s.checkRequest(found, r, raw); len(issues) > 0 {
		s.write(w, 400, mustJSON(map[string]any{"success": false, "error": map[string]any{"issues": issues, "name": "ZodError"}}))
		return 400
	}
	key := r.Header.Get("Idempotency-Key")
	var body []byte
	if key != "" {
		hash := sha256.Sum256(raw)
		id := key + "|" + r.URL.Path
		s.mu.Lock()
		previous, again := s.replay[id]
		s.mu.Unlock()
		if again {
			if previous.hash != hex.EncodeToString(hash[:]) {
				return reply(409, "Idempotency-Key already used with a different payload")
			}
			s.mu.Lock()
			s.hits[found.key()]++
			s.mu.Unlock()
			w.Header().Set("Idempotent-Replayed", "true")
			s.write(w, 200, previous.body)
			return 200
		}
		body = s.response(found, r)
		s.mu.Lock()
		s.replay[id] = replay{hash: hex.EncodeToString(hash[:]), body: body}
		s.mu.Unlock()
	} else {
		body = s.response(found, r)
	}
	s.mu.Lock()
	s.hits[found.key()]++
	s.mu.Unlock()
	s.write(w, 200, body)
	return 200
}

// parameters are the operation's parameters, resolved.
func (s *Server) parameters(op map[string]any) []map[string]any {
	var out []map[string]any
	list, _ := op["parameters"].([]any)
	for _, each := range list {
		if p, _ := each.(map[string]any); p != nil {
			out = append(out, p)
		}
	}
	return out
}

func (s *Server) checkRequest(rt *route, r *http.Request, raw []byte) []issue {
	var issues []issue
	declared := map[string]map[string]any{}
	headers := map[string]bool{}
	for _, p := range s.parameters(rt.op) {
		name, _ := p["name"].(string)
		switch in, _ := p["in"].(string); in {
		case "query":
			declared[name] = p
		case "header":
			headers[strings.ToLower(name)] = true
		}
	}
	query := r.URL.Query()
	names := make([]string, 0, len(query))
	for name := range query {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		p, ok := declared[name]
		if !ok {
			issues = append(issues, issue{Code: "unrecognized_keys", Path: []any{name}, Message: "Unrecognized query parameter: " + name})
			continue
		}
		for _, value := range query[name] {
			schema, _ := p["schema"].(map[string]any)
			if anyOf, ok := schema["anyOf"].([]any); ok && len(anyOf) > 0 {
				continue
			}
			var typed any = value
			if types, _ := kinds(orEmpty(s.sch.resolve(schema))); len(types) > 0 && (types[0] == "integer" || types[0] == "number") {
				number, err := strconv.ParseFloat(value, 64)
				if err != nil {
					issues = append(issues, issue{Code: "invalid_type", Path: []any{name}, Message: "Expected number, received nan"})
					continue
				}
				typed = number
			}
			s.sch.validate(schema, typed, []any{name}, &issues)
		}
		if _, many := declaredList(p); !many && len(query[name]) > 1 {
			issues = append(issues, issue{Code: "invalid_type", Path: []any{name}, Message: "Expected one value, received a list"})
		}
	}
	if key := r.Header.Get("Idempotency-Key"); key != "" {
		switch {
		case !headers["idempotency-key"]:
			issues = append(issues, issue{Code: "unrecognized_keys", Path: []any{"Idempotency-Key"}, Message: "Idempotency-Key is not read by this operation"})
		case len(key) > 256:
			issues = append(issues, issue{Code: "too_big", Path: []any{"Idempotency-Key"}, Message: "String must contain at most 256 character(s)"})
		}
	}
	rb, _ := rt.op["requestBody"].(map[string]any)
	trimmed := bytes.TrimSpace(raw)
	if rb == nil {
		if len(trimmed) > 0 && string(trimmed) != "{}" {
			issues = append(issues, issue{Code: "unrecognized_keys", Path: []any{}, Message: "This operation takes no body"})
		}
		return issues
	}
	if len(trimmed) == 0 {
		return append(issues, issue{Code: "invalid_type", Path: []any{}, Message: "Required"})
	}
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(strings.ToLower(ct), "application/json") {
		return append(issues, issue{Code: "invalid_type", Path: []any{}, Message: "Content-Type must be application/json"})
	}
	var decoded any
	if err := json.Unmarshal(trimmed, &decoded); err != nil {
		return append(issues, issue{Code: "invalid_type", Path: []any{}, Message: "The body is not JSON"})
	}
	content, _ := rb["content"].(map[string]any)
	media, _ := content["application/json"].(map[string]any)
	s.sch.validate(media["schema"], decoded, []any{}, &issues)
	return issues
}

func orEmpty(m map[string]any, _ string) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	return m
}

// declaredList says the query filter is written as a union that includes a list.
func declaredList(p map[string]any) (map[string]any, bool) {
	schema, _ := p["schema"].(map[string]any)
	anyOf, _ := schema["anyOf"].([]any)
	for _, each := range anyOf {
		if m, _ := each.(map[string]any); m != nil && m["type"] == "array" {
			return m, true
		}
	}
	return nil, false
}

// response is the operation's answer: a sample that satisfies the 200 schema,
// with the pages of the lists the operation pages, and a new id for each email.
func (s *Server) response(rt *route, r *http.Request) []byte {
	responses, _ := rt.op["responses"].(map[string]any)
	ok, _ := responses["200"].(map[string]any)
	content, _ := ok["content"].(map[string]any)
	media, _ := content["application/json"].(map[string]any)
	value := s.sch.sample(media["schema"], "")
	s.mu.Lock()
	s.serial++
	serial := s.serial
	s.mu.Unlock()
	value = s.paginate(rt, r, value)
	value = renumber(value, serial)
	return mustJSON(value)
}

const (
	totalEmails    = 120
	totalCampaigns = 70
	totalContacts  = 120
)

// paginate gives a paged list its pages: emails 120 and a count of them,
// campaigns 70 and the number of pages, contacts 120 and no count, as the
// operations answer them.
func (s *Server) paginate(rt *route, r *http.Request, value any) any {
	query := r.URL.Query()
	page := atoi(query.Get("page"), 1)
	switch obj := value.(type) {
	case map[string]any:
		if data, ok := obj["data"].([]any); ok && len(data) == 1 && obj["count"] != nil {
			limit := atoi(query.Get("limit"), 50)
			obj["data"] = slice(data[0], totalEmails, page, limit)
			obj["count"] = float64(totalEmails)
		}
		if campaigns, ok := obj["campaigns"].([]any); ok && len(campaigns) == 1 && obj["totalPage"] != nil {
			obj["campaigns"] = slice(campaigns[0], totalCampaigns, page, 30)
			obj["totalPage"] = float64((totalCampaigns + 29) / 30)
		}
	case []any:
		if _, paged := queryParam(rt, "page"); paged && len(obj) == 1 {
			return slice(obj[0], totalContacts, page, atoi(query.Get("limit"), 50))
		}
	}
	return value
}

func queryParam(rt *route, name string) (map[string]any, bool) {
	list, _ := rt.op["parameters"].([]any)
	for _, each := range list {
		if p, _ := each.(map[string]any); p != nil && p["name"] == name && p["in"] == "query" {
			return p, true
		}
	}
	return nil, false
}

func atoi(text string, fallback int) int {
	if n, err := strconv.Atoi(text); err == nil && n > 0 {
		return n
	}
	return fallback
}

// slice is the page of total copies of an item, each with its own id.
func slice(item any, total, page, limit int) []any {
	out := []any{}
	for n := (page-1)*limit + 1; n <= total && n <= page*limit; n++ {
		out = append(out, renumber(clone(item), n))
	}
	return out
}

func clone(v any) any {
	var out any
	_ = json.Unmarshal(mustJSON(v), &out)
	return out
}

// renumber makes the ids of an item its own: the id of the 7th is ..._7.
func renumber(v any, n int) any {
	switch value := v.(type) {
	case map[string]any:
		for key, each := range value {
			if text, ok := each.(string); ok && (key == "id" || key == "emailId" || key == "contactId") && strings.HasPrefix(text, "sample_") {
				value[key] = text + "_" + strconv.Itoa(n)
				continue
			}
			value[key] = renumber(each, n)
		}
	case []any:
		for i, each := range value {
			value[i] = renumber(each, n)
		}
	}
	return v
}
