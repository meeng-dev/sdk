package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// hostile is text a contract could carry to end a comment or a docstring and
// start code of its own: block and line comment terminators, a docstring
// quote, a new line, string interpolation, a backslash and a bidirectional
// override.
const hostile = "first */ BREAKOUT_A /* \"\"\" BREAKOUT_B '''\r\nBREAKOUT_C #{BREAKOUT_D} ${BREAKOUT_E} \\u000a \\ \" ‮ end"

// load reads the repository's contract after mutate has changed it.
func load(t *testing.T, mutate func(doc map[string]any)) (*IR, error) {
	t.Helper()
	raw, err := os.ReadFile("../openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	mutate(doc)
	changed, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "openapi.json")
	if err := os.WriteFile(path, changed, 0o644); err != nil {
		t.Fatal(err)
	}
	return Load(path, "0.1.0")
}

func dig(doc map[string]any, keys ...string) map[string]any {
	node := doc
	for _, key := range keys {
		node = node[key].(map[string]any)
	}
	return node
}

// A description cannot leave its comment, in any language: what ends a comment
// is broken, a new line stays a comment line, and what a language reads inside
// a string is escaped.
func TestHostileTextStaysInsideItsComment(t *testing.T) {
	ir, err := load(t, func(doc map[string]any) {
		body := dig(doc, "paths", "/api/v1/emails", "post", "requestBody", "content", "application/json", "schema")
		body["description"] = hostile
		dig(body, "properties", "subject")["description"] = hostile
		domain := dig(doc, "components", "schemas", "Domain")
		domain["description"] = hostile
		dig(domain, "properties", "name")["description"] = hostile
		query := dig(doc, "paths", "/api/v1/emails", "get")["parameters"].([]any)[0].(map[string]any)
		query["description"] = hostile
	})
	if err != nil {
		t.Fatal(err)
	}
	var params *Param
	for _, service := range ir.Services {
		for _, method := range service.Methods {
			if method.OperationID == "emails.list" {
				params = method.Query[0]
			}
		}
	}
	if params == nil || strings.ContainsAny(params.Description, "\r") {
		t.Fatalf("a query description spans lines: %+v", params)
	}
	lineComments := map[string][]string{
		"go": {"//"}, "rust": {"///"}, "swift": {"///"}, "dart": {"///"}, "ruby": {"#"}, "csharp": {"///"},
		"python": {"#"},
	}
	for language := range Languages {
		out := t.TempDir()
		if err := Generate(ir, language, out); err != nil {
			t.Fatalf("%s: %v", language, err)
		}
		seen := 0
		_ = filepath.WalkDir(out, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			raw, _ := os.ReadFile(path)
			source := string(raw)
			for _, broken := range []string{"*/ BREAKOUT_A", `""" BREAKOUT_B`, "''' ", "‮", "\r"} {
				if strings.Contains(source, broken) && !strings.HasSuffix(path, ".md") {
					t.Errorf("%s: %s keeps %q", language, path, broken)
				}
			}
			if language == "elixir" && strings.Contains(strings.ReplaceAll(source, `\#{BREAKOUT_D`, ""), "#{BREAKOUT_D") {
				t.Errorf("%s: %s interpolates the description", language, path)
			}
			for _, line := range strings.Split(source, "\n") {
				if !strings.Contains(line, "BREAKOUT_C") {
					continue
				}
				seen++
				prefixes := lineComments[language]
				if prefixes == nil {
					continue
				}
				trimmed := strings.TrimSpace(line)
				ok := false
				for _, prefix := range prefixes {
					ok = ok || strings.HasPrefix(trimmed, prefix)
				}
				if !ok {
					t.Errorf("%s: %s has a description line that is not a comment: %q", language, path, line)
				}
			}
			return nil
		})
		// PHP writes arrays, and describes only what the surface says.
		if seen == 0 && language != "elixir" && language != "php" {
			t.Errorf("%s: the test description reached no file, so the test checks nothing", language)
		}
	}
}

// What becomes an identifier, a path or a literal is checked when the
// contract is read; anything outside the alphabet stops generation.
func TestHostileIdentifiersAndValuesStopGeneration(t *testing.T) {
	cases := map[string]func(doc map[string]any){
		"an enum value with a quote": func(doc map[string]any) {
			status := dig(doc, "components", "schemas", "Domain", "properties", "status")
			status["enum"] = []any{"PENDING", `SUCCESS"; evil(); "`}
		},
		"an enum value with a new line": func(doc map[string]any) {
			status := dig(doc, "components", "schemas", "Domain", "properties", "status")
			status["enum"] = []any{"PENDING\nevil"}
		},
		"a field that is not an identifier": func(doc map[string]any) {
			body := dig(doc, "paths", "/api/v1/emails", "post", "requestBody", "content", "application/json", "schema", "properties")
			body[`x"]; evil(); ["`] = map[string]any{"type": "string"}
		},
		"a path with a quote": func(doc map[string]any) {
			paths := dig(doc, "paths")
			paths[`/api/v1/em"ails`] = paths["/api/v1/emails"]
		},
		"a path parameter that is not an identifier": func(doc map[string]any) {
			paths := dig(doc, "paths")
			paths["/api/v1/emails/{a b}"] = paths["/api/v1/emails/{emailId}"]
		},
		"a route the surface does not name": func(doc map[string]any) {
			paths := dig(doc, "paths")
			paths["/api/v1/secrets"] = paths["/api/v1/domains"]
		},
		"a server address with a quote": func(doc map[string]any) {
			doc["servers"] = []any{map[string]any{"url": `https://meeng.dev"; evil(); "`}}
		},
		"a contract version with a new line": func(doc map[string]any) {
			dig(doc, "info")["version"] = "1.0\nevil"
		},
		"a reference to a schema the contract does not have": func(doc map[string]any) {
			dig(doc, "components", "schemas", "Domain", "properties")["status"] = map[string]any{"$ref": "#/components/schemas/Nope"}
		},
		"a schema that contains itself": func(doc map[string]any) {
			dig(doc, "components", "schemas", "Domain", "properties")["dnsRecords"] = map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/Domain"}}
		},
		"a list that refers to itself with no item named for it": func(doc map[string]any) {
			schemas := dig(doc, "components", "schemas")
			schemas["TeamList"] = map[string]any{"$ref": "#/components/schemas/TeamList"}
			response := dig(doc, "paths", "/api/v1/domains", "get", "responses", "200", "content", "application/json", "schema")
			response["$ref"] = "#/components/schemas/TeamList"
		},
		"a header the SDKs do not send": func(doc map[string]any) {
			params := dig(doc, "paths", "/api/v1/emails", "post")["parameters"].([]any)
			params[0].(map[string]any)["name"] = "X-Evil"
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := load(t, mutate); err == nil {
				t.Fatal("generation went ahead")
			}
		})
	}
}

// A route the contract lost is as loud as one it gained: the surface names
// every route the SDKs promise.
func TestARouteTheContractLostStopsGeneration(t *testing.T) {
	_, err := load(t, func(doc map[string]any) {
		delete(dig(doc, "paths"), "/api/v1/analytics/reputation-metrics")
	})
	if err == nil || !strings.Contains(err.Error(), "reputation-metrics") {
		t.Fatalf("generation went ahead or named the wrong route: %v", err)
	}
}

// Each language's quoting writes one literal that reads back as the text.
func TestQuotingEscapesWhatEachLanguageReads(t *testing.T) {
	text := "a\"b'c\\d$e#{f}#$g\n\x01é"
	want := map[string]string{
		"go":         `"a\"b'c\\d$e#{f}#$g\n\x01é"`,
		"typescript": `"a\"b'c\\d$e#{f}#$g\n\u0001é"`,
		"python":     `"a\"b'c\\d$e#{f}#$g\n\x01é"`,
		"ruby":       `"a\"b'c\\d$e\#{f}\#$g\n\x01é"`,
		"elixir":     `"a\"b'c\\d$e\#{f}#$g\n\x01é"`,
		"php":        "'a\"b\\'c\\\\d$e#{f}#$g\n\x01é'",
		"java":       `"a\"b'c\\d$e#{f}#$g\n\001é"`,
		"csharp":     `"a\"b'c\\d$e#{f}#$g\n\u0001é"`,
		"rust":       `"a\"b'c\\d$e#{f}#$g\n\u{1}é"`,
		"swift":      `"a\"b'c\\d$e#{f}#$g\n\u{1}é"`,
		"dart":       `'a"b\'c\\d\$e#{f}#\$g\n\x01é'`,
	}
	for language, expected := range want {
		quote, ok := Languages[language].Funcs["quote"].(func(string) string)
		if !ok {
			t.Fatalf("%s has no quote function", language)
		}
		if got := quote(text); got != expected {
			t.Errorf("%s quoted %s, want %s", language, got, expected)
		}
	}
	for language, lang := range Languages {
		if lang.Funcs["quote"] == nil {
			t.Errorf("%s has no quote function", language)
		}
	}
}

func TestCleanTextBreaksWhatEndsAComment(t *testing.T) {
	got := cleanText("a */ b /* c \"\"\" d ''' e ]]> f\r\ng h\u0000‮")
	for _, bad := range []string{"*/", "/*", `"""`, "'''", "]]>", "\r", " ", "\x00", "‮"} {
		if strings.Contains(got, bad) {
			t.Errorf("%q still has %q", got, bad)
		}
	}
	if !strings.Contains(got, "f\ng\nh") {
		t.Errorf("line breaks lost: %q", got)
	}
	if line := cleanLine("one\ntwo\n\nthree"); line != "one two three" {
		t.Errorf("a summary is %q", line)
	}
}
