package ignore

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Matcher evaluates ignore rules in order, supporting:
// - "#": comment lines
// - "!": negation (unignore)
// - basic glob tokens (*, ?, [], and **)
// - patterns without "/" match anywhere (as "**/<pattern>")
// This is intentionally a small, Syncthing-inspired subset.
type Matcher struct {
	rules []rule
}

type rule struct {
	include bool // "!": unignore
	re      *regexp.Regexp
	dirOnly bool
	raw     string
}

func New(root string, ignoreDefaults bool, patterns []string, ignoreFiles ...string) *Matcher {
	var lines []string
	if ignoreDefaults {
		// Default ignore list:
		// - ".git/": ignore directory and contents.
		// - "runtime/**": ignore contents but still allow the directory itself to exist when it exists on source.
		lines = append(lines, ".git/", "runtime/**")
	}
	lines = append(lines, patterns...)
	for _, ignoreFile := range ignoreFiles {
		ignoreFile = strings.TrimSpace(ignoreFile)
		if ignoreFile == "" {
			continue
		}
		// Do not implicitly support Git ignore files; sync ignores should be explicit to avoid
		// common cases where vendor/.env are in .gitignore but must be synced.
		base := filepath.Base(ignoreFile)
		if base == ".gitignore" {
			continue
		}
		if strings.Contains(filepath.ToSlash(ignoreFile), ".git/info/exclude") {
			continue
		}
		lines = append(lines, loadIgnoreFile(root, ignoreFile)...)
	}

	m := &Matcher{}
	for _, line := range lines {
		r, ok := parseRule(line)
		if !ok {
			continue
		}
		m.rules = append(m.rules, r)
	}
	return m
}

func (m *Matcher) Match(rel string, isDir bool) bool {
	rel = strings.TrimSpace(rel)
	if rel == "" || rel == "." {
		return false
	}
	rel = filepath.ToSlash(rel)

	ignored := false
	for _, r := range m.rules {
		if r.matches(rel, isDir) {
			if r.include {
				ignored = false
			} else {
				ignored = true
			}
		}
	}
	return ignored
}

func (r rule) matches(rel string, isDir bool) bool {
	if r.re == nil {
		return false
	}
	return r.re.MatchString(rel)
}

func parseRule(line string) (rule, bool) {
	raw := strings.TrimSpace(line)
	if raw == "" || strings.HasPrefix(raw, "#") {
		return rule{}, false
	}
	r := rule{raw: raw}
	if strings.HasPrefix(raw, "!") {
		r.include = true
		raw = strings.TrimSpace(strings.TrimPrefix(raw, "!"))
		if raw == "" {
			return rule{}, false
		}
	}

	raw = filepath.ToSlash(raw)
	anchored := strings.HasPrefix(raw, "/")
	if anchored {
		raw = strings.TrimPrefix(raw, "/")
	}

	dirOnly := strings.HasSuffix(raw, "/")
	if dirOnly {
		r.dirOnly = true
		raw = strings.TrimSuffix(raw, "/")
	}
	if raw == "" {
		return rule{}, false
	}

	// Syncthing-like: patterns without "/" apply to any directory level.
	if !anchored && !strings.Contains(raw, "/") {
		raw = "**/" + raw
	}

	expr := globToRegexp(raw)
	if expr == "" {
		return rule{}, false
	}
	if r.dirOnly {
		expr = expr + "(?:/.*)?"
	}
	re, err := regexp.Compile("^" + expr + "$")
	if err != nil {
		return rule{}, false
	}
	r.re = re
	return r, true
}

func globToRegexp(pattern string) string {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return ""
	}

	var b strings.Builder
	for i := 0; i < len(pattern); i++ {
		ch := pattern[i]
		switch ch {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				b.WriteString(".*")
				i++
				continue
			}
			b.WriteString(`[^/]*`)
		case '?':
			b.WriteString(`[^/]`)
		case '[':
			// Copy a character class verbatim where possible.
			j := i + 1
			for j < len(pattern) && pattern[j] != ']' {
				j++
			}
			if j >= len(pattern) {
				b.WriteString(`\[`)
				continue
			}
			class := pattern[i : j+1]
			// Escape backslashes inside the class to keep regexp valid.
			class = strings.ReplaceAll(class, `\`, `\\`)
			b.WriteString(class)
			i = j
		case '.', '+', '(', ')', '|', '^', '$', '{', '}', '\\':
			b.WriteByte('\\')
			b.WriteByte(ch)
		default:
			b.WriteByte(ch)
		}
	}
	return b.String()
}

func loadIgnoreFile(projectRoot, ignorePath string) []string {
	path := ignorePath
	if !filepath.IsAbs(path) {
		path = filepath.Join(projectRoot, ignorePath)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, strings.TrimRight(scanner.Text(), "\r\n"))
	}
	return lines
}
