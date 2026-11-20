package main

// The part of the check that needs no toolchain: each skill is a folder with a
// SKILL.md whose frontmatter is valid, whose links resolve, and whose code
// blocks can be told apart (and so compiled, in check.go).

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	maxNameLength        = 64
	maxDescriptionLength = 1024
)

// otherProducts are the names of other e-mail products. A skill teaches how to
// use Meeng.dev and never names, compares with or points at another product: the
// showcase is silent about its competitors, and so is every file it publishes.
var otherProducts = regexp.MustCompile(`(?i)\b(resend|usesend|unsend|sendgrid|mailgun|postmark(?:app)?|mailchimp|mandrill|sparkpost|mailjet|mailersend|brevo|sendinblue|amazon ses|aws ses|zeptomail|elastic ?email)\b`)

var (
	nameShape = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	// A markdown link or image whose target is a relative path.
	linkShape = regexp.MustCompile(`\]\(([^)\s]+)\)`)
)

// Skill is a skill folder read from disk.
type Skill struct {
	Dir  string // absolute path of the folder
	Name string // the folder's name
	// Files are the markdown files of the skill, SKILL.md first, then every
	// file under referencias/, with the path relative to Dir.
	Files []string
}

// Frontmatter is what the top of a SKILL.md says.
type Frontmatter struct {
	Name        string
	Description string
}

// ParseFrontmatter reads the block between the two "---" lines at the top of a
// SKILL.md. Only single-line `key: value` pairs are valid, and only the keys
// name and description: the format of the skills of the other HYZIS products.
// It returns the frontmatter and the body that follows it.
func ParseFrontmatter(text string) (Frontmatter, string, error) {
	var fm Frontmatter
	text = strings.ReplaceAll(text, "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return fm, "", fmt.Errorf("frontmatter: the file does not start with a --- line")
	}
	rest := text[len("---\n"):]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		if strings.HasSuffix(rest, "\n---") {
			end = len(rest) - len("\n---")
		} else {
			return fm, "", fmt.Errorf("frontmatter: the closing --- line is missing")
		}
	}
	block := rest[:end]
	body := ""
	if end+len("\n---\n") <= len(rest) {
		body = rest[end+len("\n---\n"):]
	}
	seen := map[string]bool{}
	for n, line := range strings.Split(block, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			return fm, "", fmt.Errorf("frontmatter line %d: want `key: value` on one line, got %q", n+1, line)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if seen[key] {
			return fm, "", fmt.Errorf("frontmatter: %q appears twice", key)
		}
		seen[key] = true
		if len(value) >= 2 && (value[0] == '"' && value[len(value)-1] == '"' || value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
		switch key {
		case "name":
			fm.Name = value
		case "description":
			fm.Description = value
		default:
			return fm, "", fmt.Errorf("frontmatter: unknown key %q (only name and description)", key)
		}
	}
	return fm, body, nil
}

// Validate says what is wrong with the frontmatter of the skill in folder.
func (fm Frontmatter) Validate(folder string) []string {
	var problems []string
	switch {
	case fm.Name == "":
		problems = append(problems, "frontmatter: name is empty")
	case len(fm.Name) > maxNameLength:
		problems = append(problems, fmt.Sprintf("frontmatter: name is %d characters, the limit is %d", len(fm.Name), maxNameLength))
	case !nameShape.MatchString(fm.Name):
		problems = append(problems, fmt.Sprintf("frontmatter: name %q must be lowercase letters, digits and single hyphens", fm.Name))
	case fm.Name != folder:
		problems = append(problems, fmt.Sprintf("frontmatter: name %q must equal the folder name %q", fm.Name, folder))
	}
	switch {
	case strings.TrimSpace(fm.Description) == "":
		problems = append(problems, "frontmatter: description is empty")
	case utf8.RuneCountInString(fm.Description) > maxDescriptionLength:
		problems = append(problems, fmt.Sprintf("frontmatter: description is %d characters, the limit is %d", utf8.RuneCountInString(fm.Description), maxDescriptionLength))
	case strings.ContainsAny(fm.Description, "<>"):
		problems = append(problems, "frontmatter: description must not contain < or >")
	case !strings.Contains(fm.Description, "Use quando"):
		problems = append(problems, `frontmatter: description must say when to use the skill ("Use quando ...")`)
	}
	return problems
}

// FindSkills lists the skill folders under root: every direct subfolder, each of
// which must hold a SKILL.md.
func FindSkills(root string) ([]Skill, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var skills []Skill
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		dir, err := filepath.Abs(filepath.Join(root, e.Name()))
		if err != nil {
			return nil, err
		}
		s := Skill{Dir: dir, Name: e.Name(), Files: []string{"SKILL.md"}}
		refs, err := os.ReadDir(filepath.Join(dir, "referencias"))
		if err == nil {
			for _, r := range refs {
				if !r.IsDir() && strings.HasSuffix(r.Name(), ".md") {
					s.Files = append(s.Files, filepath.Join("referencias", r.Name()))
				}
			}
		}
		sort.Strings(s.Files[1:])
		skills = append(skills, s)
	}
	return skills, nil
}

// CheckSkill reads one skill and says everything wrong with its text: the
// frontmatter, a missing SKILL.md, a link that goes nowhere, a reference file
// SKILL.md never mentions, a fenced block that is not closed.
func CheckSkill(s Skill) (blocks []Block, problems []string) {
	skillPath := filepath.Join(s.Dir, "SKILL.md")
	raw, err := os.ReadFile(skillPath)
	if err != nil {
		return nil, []string{"SKILL.md is missing"}
	}
	fm, body, err := ParseFrontmatter(string(raw))
	if err != nil {
		problems = append(problems, err.Error())
	} else {
		problems = append(problems, fm.Validate(s.Name)...)
		if !strings.HasPrefix(strings.TrimSpace(body), "# ") {
			problems = append(problems, "SKILL.md: the body must start with a # heading")
		}
	}
	mentioned := map[string]bool{}
	for _, rel := range s.Files {
		text := string(raw)
		if rel != "SKILL.md" {
			b, err := os.ReadFile(filepath.Join(s.Dir, rel))
			if err != nil {
				problems = append(problems, fmt.Sprintf("%s: %v", rel, err))
				continue
			}
			text = string(b)
		}
		if m := otherProducts.FindString(text); m != "" {
			problems = append(problems, fmt.Sprintf("%s: names another product (%q); a skill never does", rel, m))
		}
		found, bad := ExtractBlocks(text)
		for _, p := range bad {
			problems = append(problems, fmt.Sprintf("%s: %s", rel, p))
		}
		for _, b := range found {
			b.Skill, b.File = s.Name, rel
			blocks = append(blocks, b)
		}
		for _, link := range relativeLinks(stripFences(text)) {
			target := filepath.Join(filepath.Dir(filepath.Join(s.Dir, rel)), filepath.FromSlash(link))
			if _, err := os.Stat(target); err != nil {
				problems = append(problems, fmt.Sprintf("%s: the link %q goes nowhere", rel, link))
				continue
			}
			if rel == "SKILL.md" {
				mentioned[filepath.ToSlash(filepath.Clean(link))] = true
			}
		}
	}
	for _, rel := range s.Files[1:] {
		if !mentioned[filepath.ToSlash(rel)] {
			problems = append(problems, fmt.Sprintf("SKILL.md never links to %s", rel))
		}
	}
	return blocks, problems
}

// relativeLinks are the targets of the markdown links that point at files of
// the skill: not URLs, not anchors.
func relativeLinks(text string) []string {
	var out []string
	for _, m := range linkShape.FindAllStringSubmatch(text, -1) {
		target := m[1]
		if strings.Contains(target, "://") || strings.HasPrefix(target, "#") || strings.HasPrefix(target, "mailto:") {
			continue
		}
		if i := strings.Index(target, "#"); i >= 0 {
			target = target[:i]
		}
		if target != "" {
			out = append(out, target)
		}
	}
	return out
}

// stripFences removes the fenced code blocks, so a "](" inside an example is
// not read as a link.
func stripFences(text string) string {
	var b strings.Builder
	in := false
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			in = !in
			continue
		}
		if !in {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return b.String()
}
