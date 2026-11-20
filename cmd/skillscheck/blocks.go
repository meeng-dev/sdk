package main

import (
	"fmt"
	"regexp"
	"strings"
)

// Block is one fenced code block of a skill.
type Block struct {
	Skill string // the skill's folder
	File  string // the markdown file, relative to the skill
	Line  int    // the line of the first line of code, 1-based
	Lang  string // go, ts, python, php, json, or other
	// NoCheck is set by the info string "<lang> nocheck": the block is a
	// fragment, shown and not compiled.
	NoCheck bool
	Code    string
}

// Where says where the block is, for a message.
func (b Block) Where() string {
	return fmt.Sprintf("%s/%s:%d", b.Skill, b.File, b.Line)
}

// Checked says whether the check compiles or parses the block.
func (b Block) Checked() bool {
	return !b.NoCheck && b.Lang != "other"
}

var goPackage = regexp.MustCompile(`(?m)^package\s+main\b`)

// language is the name a fence's first word stands for.
func language(word string) string {
	switch strings.ToLower(word) {
	case "go", "golang":
		return "go"
	case "ts", "typescript":
		return "ts"
	case "py", "python":
		return "python"
	case "php":
		return "php"
	case "json":
		return "json"
	}
	return "other"
}

// ExtractBlocks reads the fenced code blocks of a markdown text. It also
// returns what is wrong with them: a fence never closed, an unknown word after
// the language, a Go block that is not a whole program.
func ExtractBlocks(text string) (blocks []Block, problems []string) {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimLeft(line, " ")
		if len(line)-len(trimmed) > 3 || !strings.HasPrefix(trimmed, "```") {
			continue
		}
		info := strings.Fields(strings.TrimLeft(trimmed, "`"))
		b := Block{Line: i + 2, Lang: "other"}
		if len(info) > 0 {
			b.Lang = language(info[0])
		}
		for _, word := range info[min(1, len(info)):] {
			if word == "nocheck" {
				b.NoCheck = true
			} else {
				problems = append(problems, fmt.Sprintf("line %d: unknown word %q after the language of a code block (only nocheck)", i+1, word))
			}
		}
		open := i + 1
		closed := false
		var code []string
		for i++; i < len(lines); i++ {
			if strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
				closed = true
				break
			}
			code = append(code, lines[i])
		}
		if !closed {
			problems = append(problems, fmt.Sprintf("line %d: the code block is never closed", open))
			break
		}
		b.Code = strings.Join(code, "\n") + "\n"
		if b.Lang == "go" && !b.NoCheck && !goPackage.MatchString(b.Code) {
			problems = append(problems, fmt.Sprintf("line %d: a Go example must be a whole program (package main), or marked `go nocheck`", open))
			continue
		}
		blocks = append(blocks, b)
	}
	return blocks, problems
}
