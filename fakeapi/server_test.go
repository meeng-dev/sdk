package fakeapi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func fake(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	raw, err := os.ReadFile("../openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	server, err := New(raw)
	if err != nil {
		t.Fatal(err)
	}
	http := httptest.NewServer(server)
	t.Cleanup(http.Close)
	return server, http
}

func call(t *testing.T, base, method, path, body string, headers ...string) (int, http.Header, map[string]any, []any) {
	t.Helper()
	req, err := http.NewRequest(method, base+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer mg_test_key")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var object map[string]any
	var list []any
	if json.Unmarshal(raw, &object) != nil {
		_ = json.Unmarshal(raw, &list)
	}
	return resp.StatusCode, resp.Header, object, list
}

func errorCode(object map[string]any) string {
	e, _ := object["error"].(map[string]any)
	code, _ := e["code"].(string)
	return code
}

func TestEveryOperationOfTheContractIsServed(t *testing.T) {
	server, _ := fake(t)
	if got := len(server.Operations()); got != 33 {
		t.Fatalf("%d operations", got)
	}
}

func TestEveryResponseSatisfiesItsOwnSchema(t *testing.T) {
	server, _ := fake(t)
	for _, rt := range server.routes {
		responses, _ := rt.op["responses"].(map[string]any)
		ok, _ := responses["200"].(map[string]any)
		content, _ := ok["content"].(map[string]any)
		media, _ := content["application/json"].(map[string]any)
		schema := media["schema"]
		value := server.sch.sample(schema, "")
		// Read as JSON, so that numbers are what a client decodes.
		var decoded any
		_ = json.Unmarshal(mustJSON(value), &decoded)
		var issues []issue
		server.sch.validate(schema, decoded, nil, &issues)
		if len(issues) > 0 {
			t.Errorf("%s: %+v", rt.key(), issues)
		}
	}
}

func TestARouteTheContractDoesNotHaveIsNotFoundAndAWrongVerbIsNotAllowed(t *testing.T) {
	_, srv := fake(t)
	if status, _, body, _ := call(t, srv.URL, "GET", "/api/v1/nothing", ""); status != 404 || errorCode(body) != "NOT_FOUND" {
		t.Errorf("unknown route: %d %v", status, body)
	}
	if status, _, body, _ := call(t, srv.URL, "DELETE", "/api/v1/emails", ""); status != 405 || errorCode(body) != "METHOD_NOT_ALLOWED" {
		t.Errorf("wrong verb: %d %v", status, body)
	}
	if status, _, _, _ := call(t, srv.URL, "GET", "/v1/emails", ""); status != 404 {
		t.Errorf("a path without the API prefix answered %d", status)
	}
}

func TestTheKeyIsRequired(t *testing.T) {
	_, srv := fake(t)
	req, _ := http.NewRequest("GET", srv.URL+"/api/v1/domains", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 401 {
		t.Errorf("no key: %v %v", resp, err)
	}
	if status, _, body, _ := call(t, srv.URL, "GET", "/api/v1/domains", "", "Authorization", "Bearer nope"); status != 403 || errorCode(body) != "FORBIDDEN" {
		t.Errorf("a bad key: %d %v", status, body)
	}
}

func TestABodyThatBreaksTheContractIsRefusedWithTheIssues(t *testing.T) {
	_, srv := fake(t)
	cases := map[string]string{
		"a missing required field":           `{"to":"a@x.com"}`,
		"a field the contract does not have": `{"from":"a@x.com","to":"b@x.com","subjet":"typo"}`,
		"a wrong type":                       `{"from":"a@x.com","to":5}`,
		"a bad date":                         `{"from":"a@x.com","to":"b@x.com","scheduledAt":"tomorrow"}`,
		"not JSON":                           `{"from":`,
		"an attachment (ADR-0008)":           `{"from":"a@x.com","to":"b@x.com","subject":"Hi","text":"x","attachments":[{"filename":"a.pdf","content":"QQ=="}]}`,
	}
	for name, body := range cases {
		status, _, object, _ := call(t, srv.URL, "POST", "/api/v1/emails", body)
		e, _ := object["error"].(map[string]any)
		if status != 400 || e["name"] != "ZodError" || len(e["issues"].([]any)) == 0 {
			t.Errorf("%s: %d %v", name, status, object)
		}
	}
	if status, _, object, _ := call(t, srv.URL, "POST", "/api/v1/emails", `{"from":"a@x.com","to":["b@x.com"],"subject":"Hi","headers":{"X-A":"1"}}`); status != 200 || object["emailId"] == nil {
		t.Errorf("a valid send: %d %v", status, object)
	}
}

func TestAnEmptyListOfAttachmentsPassesAndAFullOneSaysUseLinks(t *testing.T) {
	_, srv := fake(t)
	if status, _, object, _ := call(t, srv.URL, "POST", "/api/v1/emails", `{"from":"a@x.com","to":["b@x.com"],"subject":"Hi","text":"x","attachments":[]}`); status != 200 || object["emailId"] == nil {
		t.Errorf("an empty list: %d %v", status, object)
	}
	status, _, object, _ := call(t, srv.URL, "POST", "/api/v1/emails", `{"from":"a@x.com","to":["b@x.com"],"subject":"Hi","text":"x","attachments":[{"filename":"a.pdf","content":"QQ=="}]}`)
	e, _ := object["error"].(map[string]any)
	issues, _ := e["issues"].([]any)
	if status != 400 || len(issues) != 1 || !strings.Contains(issues[0].(map[string]any)["message"].(string), "links") {
		t.Errorf("a file: %d %v", status, object)
	}
}

func TestAQueryFilterTheContractDoesNotHaveIsRefused(t *testing.T) {
	_, srv := fake(t)
	if status, _, _, _ := call(t, srv.URL, "GET", "/api/v1/emails?page=2&limit=10&domainId=1&domainId=2", ""); status != 200 {
		t.Errorf("valid filters: %d", status)
	}
	if status, _, _, _ := call(t, srv.URL, "GET", "/api/v1/emails?colour=red", ""); status != 400 {
		t.Errorf("an unknown filter: %d", status)
	}
	if status, _, _, _ := call(t, srv.URL, "GET", "/api/v1/analytics/email-time-series?days=9", ""); status != 400 {
		t.Errorf("a value outside the enumeration: %d", status)
	}
	if status, _, _, _ := call(t, srv.URL, "GET", "/api/v1/campaigns?page=abc", ""); status == 200 {
		// The contract writes page as text: any text is accepted by the schema.
		t.Log("page as text is accepted")
	}
}

func TestTheIdempotencyKeyIsReadOnlyWhereTheContractReadsIt(t *testing.T) {
	_, srv := fake(t)
	first, _, one, _ := call(t, srv.URL, "POST", "/api/v1/emails", `{"from":"a@x.com","to":"b@x.com"}`, "Idempotency-Key", "k1")
	again, header, two, _ := call(t, srv.URL, "POST", "/api/v1/emails", `{"from":"a@x.com","to":"b@x.com"}`, "Idempotency-Key", "k1")
	if first != 200 || again != 200 || one["emailId"] != two["emailId"] || header.Get("Idempotent-Replayed") != "true" {
		t.Errorf("a replay: %d %v / %d %v", first, one, again, two)
	}
	if status, _, body, _ := call(t, srv.URL, "POST", "/api/v1/emails", `{"from":"a@x.com","to":"c@x.com"}`, "Idempotency-Key", "k1"); status != 409 || errorCode(body) != "NOT_UNIQUE" {
		t.Errorf("another body under the same key: %d %v", status, body)
	}
	_, _, fresh, _ := call(t, srv.URL, "POST", "/api/v1/emails", `{"from":"a@x.com","to":"b@x.com"}`)
	if fresh["emailId"] == one["emailId"] {
		t.Error("two sends with no key got the same id")
	}
	if status, _, _, _ := call(t, srv.URL, "POST", "/api/v1/domains", `{"name":"a.com","region":"us-east-1"}`, "Idempotency-Key", "k"); status != 400 {
		t.Errorf("a key on an operation that does not read it: %d", status)
	}
}

func TestAPathParameterChoosesTheAnswer(t *testing.T) {
	_, srv := fake(t)
	if status, _, body, _ := call(t, srv.URL, "GET", "/api/v1/emails/fake-404", ""); status != 404 || errorCode(body) != "NOT_FOUND" {
		t.Errorf("fake-404: %d %v", status, body)
	}
	if status, header, body, _ := call(t, srv.URL, "GET", "/api/v1/emails/fake-429", ""); status != 429 || errorCode(body) != "RATE_LIMITED" || header.Get("Retry-After") != "0" {
		t.Errorf("fake-429: %d %v %v", status, body, header)
	}
	for i, want := range []int{503, 503, 200} {
		if status, _, _, _ := call(t, srv.URL, "GET", "/api/v1/emails/flaky-503-2", ""); status != want {
			t.Errorf("flaky-503-2 call %d answered %d, want %d", i+1, status, want)
		}
	}
}

func TestListsArePagedLikeTheOperationsPageThem(t *testing.T) {
	_, srv := fake(t)
	_, _, emails, _ := call(t, srv.URL, "GET", "/api/v1/emails?page=3&limit=50", "")
	if data, _ := emails["data"].([]any); len(data) != 20 || emails["count"] != float64(120) {
		t.Errorf("emails page 3: %d items, count %v", len(data), emails["count"])
	}
	_, _, campaigns, _ := call(t, srv.URL, "GET", "/api/v1/campaigns?page=1", "")
	if list, _ := campaigns["campaigns"].([]any); len(list) != 30 || campaigns["totalPage"] != float64(3) {
		t.Errorf("campaigns: %d items, %v pages", len(list), campaigns["totalPage"])
	}
	_, _, _, contacts := call(t, srv.URL, "GET", "/api/v1/contactBooks/cb_1/contacts?page=2&limit=100", "")
	if len(contacts) != 20 {
		t.Errorf("contacts page 2: %d items", len(contacts))
	}
	_, _, _, domains := call(t, srv.URL, "GET", "/api/v1/domains", "")
	if len(domains) != 1 {
		t.Errorf("domains: %d items", len(domains))
	}
}

func TestCoverageNamesTheOperationsNeverCalledAndResetForgetsThem(t *testing.T) {
	server, srv := fake(t)
	if len(server.Missing()) != 33 {
		t.Fatalf("%d missing before any call", len(server.Missing()))
	}
	call(t, srv.URL, "GET", "/api/v1/domains", "")
	call(t, srv.URL, "GET", "/api/v1/domains/fake-404", "")
	missing := server.Missing()
	if len(missing) != 32 || contains(missing, "GET /api/v1/domains") || !contains(missing, "GET /api/v1/domains/{id}") {
		t.Errorf("missing %v", missing)
	}
	if calls := server.Calls(); len(calls) != 2 || calls[1].Status != 404 || calls[0].Operation != "GET /api/v1/domains" {
		t.Errorf("calls %+v", calls)
	}
	call(t, srv.URL, "DELETE", "/__requests", "")
	if len(server.Calls()) != 0 || len(server.Missing()) != 33 {
		t.Error("reset kept something")
	}
}

func contains(list []string, want string) bool {
	for _, each := range list {
		if each == want {
			return true
		}
	}
	return false
}
