// Command skillscheck is the gate of the skills/ folder: every skill has valid
// frontmatter and links that resolve, and every code example in it compiles
// against the real SDKs of this repository (Go and TypeScript; Python and PHP
// are compiled and parsed too), so a skill never teaches a call the SDK does not
// have.
//
//	go run ./cmd/skillscheck              # the whole check
//	go run ./cmd/skillscheck -static      # frontmatter, links and fences only
//
// A toolchain that is not installed stops the check; SKIP_MISSING=1 names it and
// skips it, as check.sh does.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	root := flag.String("sdk", ".", "the SDK repository root")
	dir := flag.String("skills", "skills", "the folder of skills, relative to -sdk unless absolute")
	static := flag.Bool("static", false, "check the text only; compile no example")
	langs := flag.String("lang", "", "comma-separated languages to compile (go, ts, python, php, json); default all")
	flag.Parse()
	os.Exit(runMain(*root, *dir, *static, *langs, os.Getenv("SKIP_MISSING") == "1"))
}

func runMain(root, skillsDir string, static bool, langs string, skipMissing bool) int {
	sdk, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if !filepath.IsAbs(skillsDir) {
		skillsDir = filepath.Join(sdk, skillsDir)
	}
	skills, err := FindSkills(skillsDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if len(skills) == 0 {
		fmt.Fprintf(os.Stderr, "no skill under %s\n", skillsDir)
		return 1
	}
	failed := false
	var blocks []Block
	for _, s := range skills {
		b, problems := CheckSkill(s)
		blocks = append(blocks, b...)
		if len(problems) > 0 {
			failed = true
			fmt.Printf("FAIL %s\n", s.Name)
			for _, p := range problems {
				fmt.Printf("  %s\n", p)
			}
			continue
		}
		fmt.Printf("ok   %s (%d files, %d code blocks)\n", s.Name, len(s.Files), len(b))
	}
	if static {
		if failed {
			return 1
		}
		return 0
	}
	scratch, err := os.MkdirTemp("", "skillscheck-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer os.RemoveAll(scratch)
	r := &Runner{SDK: sdk, Scratch: scratch, SkipMissing: skipMissing, Out: os.Stdout}
	if langs != "" {
		r.Langs = map[string]bool{}
		for _, l := range strings.Split(langs, ",") {
			r.Langs[strings.TrimSpace(l)] = true
		}
	}
	nocheck := 0
	for _, b := range blocks {
		if b.NoCheck {
			nocheck++
		}
	}
	for _, res := range r.Run(blocks) {
		switch {
		case res.Skipped != "" && len(res.Problems) == 0 && strings.HasPrefix(res.Skipped, "mypy"):
			fmt.Printf("ok   %s examples: %d compiled; -- %s\n", res.Lang, res.Checked, res.Skipped)
		case res.Skipped != "":
			fmt.Printf("-- %s examples skipped: %s\n", res.Lang, res.Skipped)
		case len(res.Problems) > 0:
			failed = true
			fmt.Printf("FAIL %s examples (%d)\n", res.Lang, res.Checked)
			for _, p := range res.Problems {
				fmt.Printf("  %s\n", p)
			}
		default:
			fmt.Printf("ok   %s examples: %d checked\n", res.Lang, res.Checked)
		}
	}
	if nocheck > 0 {
		fmt.Printf("--   %d code blocks are marked nocheck: shown, not compiled\n", nocheck)
	}
	if failed {
		return 1
	}
	return 0
}
