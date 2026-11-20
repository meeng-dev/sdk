package main

// The part of the check that compiles the examples: each Go example against
// the Go SDK, each TypeScript one against the TypeScript SDK, each Python one
// with py_compile and mypy --strict, each PHP one with php -l, each JSON one
// parsed. A toolchain that is not installed stops the check, unless SKIP_MISSING
// names it and skips it, as check.sh does for every language.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Runner compiles the examples of the skills.
type Runner struct {
	SDK         string // the SDK repository root
	Scratch     string // an empty directory to write the examples into
	SkipMissing bool
	Out         io.Writer
	// Langs limits the languages checked; nil is all.
	Langs map[string]bool
}

// Result is what one language's check did.
type Result struct {
	Lang     string
	Checked  int
	Skipped  string // why the language was not run, when it was not
	Problems []string
}

func (r *Runner) wants(lang string) bool { return r.Langs == nil || r.Langs[lang] }

// Run checks every block, language by language.
func (r *Runner) Run(blocks []Block) []Result {
	by := map[string][]Block{}
	for _, b := range blocks {
		if b.Checked() {
			by[b.Lang] = append(by[b.Lang], b)
		}
	}
	var results []Result
	for _, lang := range []string{"json", "go", "ts", "python", "php"} {
		if !r.wants(lang) || len(by[lang]) == 0 {
			continue
		}
		var res Result
		switch lang {
		case "json":
			res = r.checkJSON(by[lang])
		case "go":
			res = r.checkGo(by[lang])
		case "ts":
			res = r.checkTS(by[lang])
		case "python":
			res = r.checkPython(by[lang])
		case "php":
			res = r.checkPHP(by[lang])
		}
		res.Lang = lang
		results = append(results, res)
	}
	return results
}

func (r *Runner) checkJSON(blocks []Block) Result {
	res := Result{Checked: len(blocks)}
	for _, b := range blocks {
		if !json.Valid([]byte(b.Code)) {
			res.Problems = append(res.Problems, b.Where()+": the JSON example does not parse")
		}
	}
	return res
}

// missing says the tool is not installed, as a skip or as a problem.
func (r *Runner) missing(res *Result, tool, why string) {
	if r.SkipMissing {
		res.Skipped = tool + " is not installed" + why
		return
	}
	res.Problems = append(res.Problems, fmt.Sprintf("%s is not installed%s (SKIP_MISSING=1 skips what is missing)", tool, why))
}

func (r *Runner) examples(lang string, blocks []Block, ext string) (dir string, files []string, err error) {
	dir = filepath.Join(r.Scratch, lang)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", nil, err
	}
	for n, b := range blocks {
		name := fmt.Sprintf("ex%03d", n+1)
		path := filepath.Join(dir, name+ext)
		if lang == "go" {
			if err := os.MkdirAll(filepath.Join(dir, name), 0o755); err != nil {
				return "", nil, err
			}
			path = filepath.Join(dir, name, "main.go")
		}
		if err := os.WriteFile(path, []byte(b.Code), 0o644); err != nil {
			return "", nil, err
		}
		files = append(files, path)
	}
	return dir, files, nil
}

// run starts a command in dir and gives back its output.
func run(dir string, env []string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	return out.String(), err
}

// translate rewrites the compiler's file names into the markdown file and line
// the example came from. pattern's first group is the example's name and the
// second the line inside it.
func translate(output string, blocks []Block, pattern *regexp.Regexp) string {
	return pattern.ReplaceAllStringFunc(output, func(m string) string {
		sub := pattern.FindStringSubmatch(m)
		n, err := strconv.Atoi(sub[1])
		line, err2 := strconv.Atoi(sub[2])
		if err != nil || err2 != nil || n < 1 || n > len(blocks) {
			return m
		}
		return fmt.Sprintf("%s/%s:%d", blocks[n-1].Skill, blocks[n-1].File, blocks[n-1].Line+line-1) + sub[3]
	})
}

func (r *Runner) checkGo(blocks []Block) Result {
	res := Result{Checked: len(blocks)}
	if _, err := exec.LookPath("go"); err != nil {
		r.missing(&res, "go", "")
		return res
	}
	dir, _, err := r.examples("go", blocks, ".go")
	if err != nil {
		res.Problems = append(res.Problems, err.Error())
		return res
	}
	mod := fmt.Sprintf("module skillexamples\n\ngo 1.23\n\nrequire github.com/meeng-dev/sdk/go v0.0.0\n\nreplace github.com/meeng-dev/sdk/go => %s\n", filepath.Join(r.SDK, "go"))
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o644); err != nil {
		res.Problems = append(res.Problems, err.Error())
		return res
	}
	out, err := run(dir, []string{"GOWORK=off", "GOFLAGS=-mod=mod"}, "go", "vet", "./...")
	if err != nil {
		pattern := regexp.MustCompile(`ex0*(\d+)/main\.go:(\d+)(:\d+)?`)
		res.Problems = append(res.Problems, "the Go examples do not compile or vet clean:\n"+indent(translate(out, blocks, pattern)))
	}
	return res
}

func (r *Runner) checkTS(blocks []Block) Result {
	res := Result{Checked: len(blocks)}
	tsc := filepath.Join(r.SDK, "typescript", "node_modules", ".bin", "tsc")
	if _, err := exec.LookPath("node"); err != nil {
		r.missing(&res, "node", "")
		return res
	}
	if _, err := os.Stat(tsc); err != nil {
		r.missing(&res, "typescript/node_modules (run npm ci in typescript/)", "")
		return res
	}
	dir, files, err := r.examples("ts", blocks, ".ts")
	if err != nil {
		res.Problems = append(res.Problems, err.Error())
		return res
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"type":"module"}`), 0o644); err != nil {
		res.Problems = append(res.Problems, err.Error())
		return res
	}
	config := map[string]any{
		"compilerOptions": map[string]any{
			"target": "ES2022", "module": "NodeNext", "moduleResolution": "NodeNext",
			"lib": []string{"ES2022", "DOM"}, "types": []string{"node"},
			"typeRoots": []string{filepath.Join(r.SDK, "typescript", "node_modules", "@types")},
			"strict":    true, "noUncheckedIndexedAccess": true, "noEmit": true, "skipLibCheck": true,
			"allowImportingTsExtensions": true, "verbatimModuleSyntax": true,
			"paths": map[string][]string{"@meeng-dev/sdk": {filepath.Join(r.SDK, "typescript", "src", "index.ts")}},
		},
		"files": baseNames(files),
	}
	raw, _ := json.MarshalIndent(config, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "tsconfig.json"), raw, 0o644); err != nil {
		res.Problems = append(res.Problems, err.Error())
		return res
	}
	out, err := run(dir, nil, tsc, "-p", "tsconfig.json", "--pretty", "false")
	if err != nil {
		pattern := regexp.MustCompile(`ex0*(\d+)\.ts\((\d+)(,\d+\))?`)
		res.Problems = append(res.Problems, "the TypeScript examples do not type-check:\n"+indent(translate(out, blocks, pattern)))
	}
	return res
}

func (r *Runner) checkPython(blocks []Block) Result {
	res := Result{Checked: len(blocks)}
	py, err := exec.LookPath("python3")
	if err != nil {
		r.missing(&res, "python3", "")
		return res
	}
	dir, files, err := r.examples("python", blocks, ".py")
	if err != nil {
		res.Problems = append(res.Problems, err.Error())
		return res
	}
	pattern := regexp.MustCompile(`ex0*(\d+)\.py(?:", line |:)(\d+)()`)
	for _, f := range files {
		if out, err := run(dir, []string{"PYTHONDONTWRITEBYTECODE=1"}, py, "-I", "-c", "import sys; compile(open(sys.argv[1]).read(), sys.argv[1], 'exec')", f); err != nil {
			res.Problems = append(res.Problems, "a Python example does not compile:\n"+indent(translate(out, blocks, pattern)))
		}
	}
	mypy := os.Getenv("MYPY")
	if mypy == "" {
		mypy = "mypy"
	}
	path, err := exec.LookPath(mypy)
	if err != nil {
		if r.SkipMissing {
			res.Skipped = "mypy is not installed, so the Python examples were compiled but their types were not checked (MYPY names it)"
		} else {
			res.Problems = append(res.Problems, "mypy is not installed, so the types of the Python examples were not checked (MYPY names it; SKIP_MISSING=1 skips it)")
		}
		return res
	}
	args := append([]string{"--strict", "--python-version", "3.11", "--no-incremental", "--cache-dir", filepath.Join(r.Scratch, "mypy-cache"), "--show-column-numbers"}, baseNames(files)...)
	out, err := run(dir, []string{"MYPYPATH=" + filepath.Join(r.SDK, "python", "src")}, path, args...)
	if err != nil {
		mypyPattern := regexp.MustCompile(`ex0*(\d+)\.py:(\d+)(:\d+)?`)
		res.Problems = append(res.Problems, "the Python examples do not type-check:\n"+indent(translate(out, blocks, mypyPattern)))
	}
	return res
}

func (r *Runner) checkPHP(blocks []Block) Result {
	res := Result{Checked: len(blocks)}
	php, err := exec.LookPath("php")
	if err != nil {
		r.missing(&res, "php", "")
		return res
	}
	dir, files, err := r.examples("php", blocks, ".php")
	if err != nil {
		res.Problems = append(res.Problems, err.Error())
		return res
	}
	pattern := regexp.MustCompile(`ex0*(\d+)\.php on line (\d+)()`)
	for _, f := range files {
		if out, err := run(dir, nil, php, "-d", "display_errors=1", "-l", f); err != nil {
			res.Problems = append(res.Problems, "a PHP example does not parse:\n"+indent(translate(out, blocks, pattern)))
		}
	}
	if len(res.Problems) > 0 {
		return res
	}
	// Parsing proves the syntax; PHPStan, level 8 as for the SDK itself, proves
	// that every class and method the example uses is in php/src.
	phpstan := os.Getenv("PHPSTAN")
	if phpstan == "" {
		phpstan = "phpstan"
	}
	path, err := exec.LookPath(phpstan)
	if err != nil && strings.HasSuffix(phpstan, ".phar") {
		// A phar is run with php, as check.sh does, and need not be executable.
		if _, statErr := os.Stat(phpstan); statErr == nil {
			path, err = phpstan, nil
		}
	}
	if err != nil {
		if r.SkipMissing {
			res.Skipped = "phpstan is not installed, so the PHP examples were parsed but their types were not checked (PHPSTAN names it)"
		} else {
			res.Problems = append(res.Problems, "phpstan is not installed, so the types of the PHP examples were not checked (PHPSTAN names it; SKIP_MISSING=1 skips it)")
		}
		return res
	}
	neon := "parameters:\n\tlevel: 8\n\tpaths:\n"
	for _, f := range baseNames(files) {
		neon += "\t\t- " + f + "\n"
	}
	neon += "\tscanDirectories:\n\t\t- " + filepath.Join(r.SDK, "php", "src") + "\n\ttreatPhpDocTypesAsCertain: false\n"
	if err := os.WriteFile(filepath.Join(dir, "phpstan.neon"), []byte(neon), 0o644); err != nil {
		res.Problems = append(res.Problems, err.Error())
		return res
	}
	args := []string{"analyse", "--no-progress", "--error-format=raw", "-c", "phpstan.neon"}
	command := path
	if strings.HasSuffix(path, ".phar") {
		command, args = php, append([]string{path}, args...)
	}
	if out, err := run(dir, nil, command, args...); err != nil {
		stanPattern := regexp.MustCompile(`ex0*(\d+)\.php:(\d+)()`)
		res.Problems = append(res.Problems, "the PHP examples do not pass PHPStan level 8:\n"+indent(translate(out, blocks, stanPattern)))
	}
	return res
}

func baseNames(paths []string) []string {
	out := make([]string, len(paths))
	for i, p := range paths {
		out[i] = filepath.Base(p)
	}
	return out
}

func indent(s string) string {
	s = strings.TrimRight(s, "\n")
	return "    " + strings.ReplaceAll(s, "\n", "\n    ")
}
