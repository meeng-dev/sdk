package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const goodSkill = `---
name: meeng-exemplo
description: Ensina um exemplo. Use quando o pedido for testar o verificador de skills.
---

# Exemplo

Veja [a referência](referencias/detalhe.md).
`

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

// skillDir writes a skill named meeng-exemplo into a fresh folder and gives back
// the folder of skills.
func skillDir(t *testing.T, skill, reference string) string {
	t.Helper()
	root := t.TempDir()
	write(t, filepath.Join(root, "meeng-exemplo", "SKILL.md"), skill)
	if reference != "" {
		write(t, filepath.Join(root, "meeng-exemplo", "referencias", "detalhe.md"), reference)
	}
	return root
}

func problemsOf(t *testing.T, root string) (blocks []Block, problems []string) {
	t.Helper()
	skills, err := FindSkills(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range skills {
		b, p := CheckSkill(s)
		blocks = append(blocks, b...)
		problems = append(problems, p...)
	}
	return blocks, problems
}

func contains(problems []string, want string) bool {
	for _, p := range problems {
		if strings.Contains(p, want) {
			return true
		}
	}
	return false
}

func TestParseFrontmatter(t *testing.T) {
	good := []struct{ in, name, description string }{
		{"---\nname: a\ndescription: b\n---\n# T\n", "a", "b"},
		{"---\nname: \"a\"\ndescription: 'b: c'\n---\n# T\n", "a", "b: c"},
		{"---\r\nname: a\r\ndescription: b\r\n---\r\n# T\r\n", "a", "b"},
	}
	for _, c := range good {
		fm, body, err := ParseFrontmatter(c.in)
		if err != nil {
			t.Fatalf("%q: %v", c.in, err)
		}
		if fm.Name != c.name || fm.Description != c.description || !strings.HasPrefix(strings.TrimSpace(body), "# T") {
			t.Errorf("%q: got %+v body %q", c.in, fm, body)
		}
	}
	bad := map[string]string{
		"no frontmatter":     "# T\n",
		"never closed":       "---\nname: a\ndescription: b\n# T\n",
		"unknown key":        "---\nname: a\ndescription: b\nlicense: MIT\n---\n",
		"repeated key":       "---\nname: a\nname: b\ndescription: c\n---\n",
		"a multi-line value": "---\nname: a\ndescription: >\n  long\n---\n",
		"an indented key":    "---\nname: a\n  description: b\n---\n",
		"a line with no key": "---\nname a\n---\n",
	}
	for what, in := range bad {
		if _, _, err := ParseFrontmatter(in); err == nil {
			t.Errorf("%s: accepted %q", what, in)
		}
	}
}

func TestFrontmatterValidate(t *testing.T) {
	long := strings.Repeat("a", maxDescriptionLength+1)
	cases := []struct {
		fm     Frontmatter
		folder string
		want   string // "" is valid
	}{
		{Frontmatter{"meeng-a", "Faz isso. Use quando precisar."}, "meeng-a", ""},
		{Frontmatter{"", "Use quando x"}, "x", "name is empty"},
		{Frontmatter{"Meeng_A", "Use quando x"}, "Meeng_A", "lowercase"},
		{Frontmatter{"meeng-a", "Use quando x"}, "meeng-b", "must equal the folder"},
		{Frontmatter{"meeng-a", ""}, "meeng-a", "description is empty"},
		{Frontmatter{"meeng-a", "Faz isso."}, "meeng-a", "Use quando"},
		{Frontmatter{"meeng-a", "Use quando <x>"}, "meeng-a", "< or >"},
		{Frontmatter{"meeng-a", "Use quando " + long}, "meeng-a", "limit is"},
		{Frontmatter{strings.Repeat("a", maxNameLength+1), "Use quando x"}, strings.Repeat("a", maxNameLength+1), "limit is"},
	}
	for _, c := range cases {
		got := c.fm.Validate(c.folder)
		switch {
		case c.want == "" && len(got) != 0:
			t.Errorf("%+v: unexpected %v", c.fm, got)
		case c.want != "" && !contains(got, c.want):
			t.Errorf("%+v: want a problem with %q, got %v", c.fm, c.want, got)
		}
	}
}

func TestExtractBlocks(t *testing.T) {
	text := "# T\n\n```go\npackage main\n\nfunc main() {}\n```\n\ntexto\n\n```ts nocheck\nlet x\n```\n\n```sh\necho oi\n```\n\n```python\nprint(1)\n```\n"
	blocks, problems := ExtractBlocks(text)
	if len(problems) != 0 {
		t.Fatalf("problems: %v", problems)
	}
	if len(blocks) != 4 {
		t.Fatalf("want 4 blocks, got %d: %+v", len(blocks), blocks)
	}
	if blocks[0].Lang != "go" || !blocks[0].Checked() || blocks[0].Line != 4 {
		t.Errorf("go block: %+v", blocks[0])
	}
	if blocks[1].Lang != "ts" || !blocks[1].NoCheck || blocks[1].Checked() {
		t.Errorf("ts nocheck block: %+v", blocks[1])
	}
	if blocks[2].Lang != "other" || blocks[2].Checked() {
		t.Errorf("sh block: %+v", blocks[2])
	}
	if blocks[3].Lang != "python" || !blocks[3].Checked() {
		t.Errorf("python block: %+v", blocks[3])
	}

	for what, in := range map[string]string{
		"an unclosed fence":    "```go\npackage main\n",
		"a fragment of Go":     "```go\nfmt.Println(1)\n```\n",
		"an unknown info word": "```ts compile\nlet x\n```\n",
		"a typo of nocheck":    "```go nocheck extra\nx\n```\n",
	} {
		if _, problems := ExtractBlocks(in); len(problems) == 0 {
			t.Errorf("%s: no problem for %q", what, in)
		}
	}
}

func TestCheckSkillAcceptsAGoodSkill(t *testing.T) {
	root := skillDir(t, goodSkill, "# Detalhe\n\nTexto.\n")
	if _, problems := problemsOf(t, root); len(problems) != 0 {
		t.Fatalf("problems: %v", problems)
	}
}

func TestCheckSkillFindsWhatIsWrong(t *testing.T) {
	cases := []struct {
		what      string
		skill     string
		reference string
		want      string
	}{
		{"a link that goes nowhere", goodSkill + "\nVeja [aqui](referencias/nao-existe.md).\n", "# D\n", "goes nowhere"},
		{"a reference nobody links", strings.Replace(goodSkill, "[a referência](referencias/detalhe.md)", "nada", 1), "# D\n", "never links to"},
		{"a missing title", strings.Replace(goodSkill, "# Exemplo", "Exemplo", 1), "# D\n", "# heading"},
		{"a bad name", strings.Replace(goodSkill, "name: meeng-exemplo", "name: outro", 1), "# D\n", "must equal the folder"},
		{"another product", goodSkill + "\nTambém funciona como o Resend.\n", "# D\n", "names another product"},
		{"another product in a reference", goodSkill, "# D\n\nIgual ao SendGrid.\n", "names another product"},
	}
	for _, c := range cases {
		_, problems := problemsOf(t, skillDir(t, c.skill, c.reference))
		if !contains(problems, c.want) {
			t.Errorf("%s: want a problem with %q, got %v", c.what, c.want, problems)
		}
	}
}

func TestLinksInsideCodeAreNotLinks(t *testing.T) {
	skill := goodSkill + "\n```ts nocheck\nconst x = a[b](c.md);\n```\n"
	if _, problems := problemsOf(t, skillDir(t, skill, "# D\n")); len(problems) != 0 {
		t.Fatalf("a call inside a code block was read as a link: %v", problems)
	}
}

func TestMissingSkillFile(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "meeng-vazia"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, problems := problemsOf(t, root); !contains(problems, "SKILL.md is missing") {
		t.Fatalf("problems: %v", problems)
	}
}

func mustPattern(s string) *regexp.Regexp { return regexp.MustCompile(s) }

func TestTranslate(t *testing.T) {
	blocks := []Block{{Skill: "meeng-a", File: "SKILL.md", Line: 10}, {Skill: "meeng-b", File: "referencias/x.md", Line: 40}}
	got := translate("ex002/main.go:5:3: undefined: x\nex001/main.go:1: y", blocks, mustPattern(`ex0*(\d+)/main\.go:(\d+)(:\d+)?`))
	want := "meeng-b/referencias/x.md:44:3: undefined: x\nmeeng-a/SKILL.md:10: y"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestJSONBlocks(t *testing.T) {
	r := &Runner{SDK: t.TempDir(), Scratch: t.TempDir()}
	res := r.checkJSON([]Block{
		{Skill: "s", File: "f.md", Line: 3, Lang: "json", Code: `{"a": 1}`},
		{Skill: "s", File: "f.md", Line: 9, Lang: "json", Code: `{"a": }`},
	})
	if len(res.Problems) != 1 || !strings.Contains(res.Problems[0], "f.md:9") {
		t.Fatalf("problems: %v", res.Problems)
	}
}

// repoRoot is the SDK repository, two folders above this one.
func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// The skills of this repository are valid. Compiling their examples is
// check.sh's job (go run ./cmd/skillscheck), because it needs every toolchain.
func TestRepositorySkillsAreValid(t *testing.T) {
	skills, err := FindSkills(filepath.Join(repoRoot(t), "skills"))
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) == 0 {
		t.Fatal("no skill in skills/")
	}
	want := []string{"meeng-enviar-email", "meeng-dominios", "meeng-webhooks", "meeng-contatos-e-campanhas", "meeng-smtp", "meeng-mcp", "meeng-migrar-de-outro-provedor"}
	have := map[string]bool{}
	for _, s := range skills {
		have[s.Name] = true
		if _, problems := CheckSkill(s); len(problems) != 0 {
			t.Errorf("%s:\n  %s", s.Name, strings.Join(problems, "\n  "))
		}
	}
	for _, name := range want {
		if !have[name] {
			t.Errorf("the skill %s is missing", name)
		}
	}
}

// The gate fails when an example calls something the SDK does not have: it is
// what keeps a skill from teaching a method that does not exist.
func TestGoExamplesAgainstTheSDK(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go is not installed")
	}
	r := &Runner{SDK: repoRoot(t), Scratch: t.TempDir()}
	good := Block{Skill: "s", File: "f.md", Line: 5, Lang: "go", Code: "package main\n\nimport meeng \"github.com/meeng-dev/sdk/go\"\n\nfunc main() { _ = meeng.New(\"\") }\n"}
	if res := r.checkGo([]Block{good}); len(res.Problems) != 0 {
		t.Fatalf("a good example failed: %v", res.Problems)
	}
	bad := good
	bad.Code = "package main\n\nimport meeng \"github.com/meeng-dev/sdk/go\"\n\nfunc main() {\n\tclient := meeng.New(\"\")\n\t_ = client.Emails.SendEverything\n}\n"
	res := r.checkGo([]Block{bad})
	if len(res.Problems) == 0 || !strings.Contains(res.Problems[0], "f.md:") {
		t.Fatalf("a call the SDK does not have was accepted, or the place was not translated: %v", res.Problems)
	}
}

func TestTypeScriptExamplesAgainstTheSDK(t *testing.T) {
	root := repoRoot(t)
	if _, err := os.Stat(filepath.Join(root, "typescript", "node_modules", ".bin", "tsc")); err != nil {
		t.Skip("typescript/node_modules is not installed (npm ci in typescript/)")
	}
	r := &Runner{SDK: root, Scratch: t.TempDir()}
	good := Block{Skill: "s", File: "f.md", Line: 5, Lang: "ts", Code: "import { Meeng } from \"@meeng-dev/sdk\";\n\nconst meeng = new Meeng(\"mg_123\");\nconst { data } = await meeng.emails.send({ from: \"a@b.com\", to: \"c@d.com\", text: \"oi\" });\nconsole.log(data?.emailId);\n"}
	if res := r.checkTS([]Block{good}); len(res.Problems) != 0 {
		t.Fatalf("a good example failed: %v", res.Problems)
	}
	bad := good
	bad.Code = "import { Meeng } from \"@meeng-dev/sdk\";\n\nconst meeng = new Meeng(\"mg_123\");\nawait meeng.emails.sendEverything({});\n"
	res := r.checkTS([]Block{bad})
	if len(res.Problems) == 0 || !strings.Contains(res.Problems[0], "f.md:") {
		t.Fatalf("a call the SDK does not have was accepted, or the place was not translated: %v", res.Problems)
	}
}

func TestMissingToolchainStopsUnlessSkipped(t *testing.T) {
	r := &Runner{SDK: t.TempDir(), Scratch: t.TempDir()}
	blocks := []Block{{Skill: "s", File: "f.md", Line: 1, Lang: "ts", Code: "export {};\n"}}
	res := r.checkTS(blocks) // the temporary SDK has no typescript/node_modules
	if len(res.Problems) == 0 {
		t.Fatal("a missing toolchain did not stop the check")
	}
	r.SkipMissing = true
	res = r.checkTS(blocks)
	if len(res.Problems) != 0 || res.Skipped == "" {
		t.Fatalf("SKIP_MISSING=1 should name and skip it: %+v", res)
	}
}

func TestPHPExamplesAgainstTheSDK(t *testing.T) {
	phpstan := os.Getenv("PHPSTAN")
	if phpstan == "" {
		phpstan = "phpstan"
	}
	_, lookErr := exec.LookPath(phpstan)
	if _, err := exec.LookPath("php"); err != nil || (lookErr != nil && !strings.HasSuffix(phpstan, ".phar")) {
		t.Skip("php or phpstan is not installed (PHPSTAN names it)")
	}
	r := &Runner{SDK: repoRoot(t), Scratch: t.TempDir()}
	good := Block{Skill: "s", File: "f.md", Line: 5, Lang: "php", Code: "<?php\n\nuse Meeng\\Meeng;\n\n$meeng = new Meeng('mg_123');\n$sent = $meeng->emails->send(['from' => 'a@b.com', 'to' => 'c@d.com', 'text' => 'oi']);\necho $sent['emailId'] ?? '';\n"}
	if res := r.checkPHP([]Block{good}); len(res.Problems) != 0 {
		t.Fatalf("a good example failed: %v", res.Problems)
	}
	bad := good
	bad.Code = "<?php\n\nuse Meeng\\Meeng;\n\n$meeng = new Meeng('mg_123');\n$meeng->emails->sendEverything([]);\n"
	res := r.checkPHP([]Block{bad})
	if len(res.Problems) == 0 || !strings.Contains(res.Problems[0], "f.md:") {
		t.Fatalf("a call the SDK does not have was accepted, or the place was not translated: %v", res.Problems)
	}
}
