// Package ignore loads and evaluates .beamignore rules (gitignore syntax).
package ignore

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Rules holds the parsed patterns from a .beamignore file.
type Rules struct {
	patterns []pattern
}

type pattern struct {
	raw     string
	dirOnly bool // pattern ends with /
	rooted  bool // pattern starts with /
	neg     bool // negation (!)
}

// Load reads .beamignore from localRoot. Returns an empty Rules if the file
// doesn't exist (not an error — the project just has no custom rules).
func Load(localRoot string) *Rules {
	r := &Rules{}
	f, err := os.Open(filepath.Join(localRoot, ".beamignore"))
	if err != nil {
		return r
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), " \t\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p := pattern{raw: line}
		if strings.HasPrefix(line, "!") {
			p.neg = true
			line = line[1:]
		}
		if strings.HasPrefix(line, "/") {
			p.rooted = true
			line = line[1:]
		}
		if strings.HasSuffix(line, "/") {
			p.dirOnly = true
			line = strings.TrimSuffix(line, "/")
		}
		p.raw = line
		r.patterns = append(r.patterns, p)
	}
	return r
}

// Match reports whether the given relative path (always forward slashes)
// should be ignored. isDir must reflect whether the path is a directory.
func (r *Rules) Match(relPath string, isDir bool) bool {
	if len(r.patterns) == 0 {
		return false
	}
	ignored := false
	for _, p := range r.patterns {
		if p.dirOnly && !isDir {
			continue
		}
		if matchPattern(p, relPath) {
			ignored = !p.neg
		}
	}
	return ignored
}

func matchPattern(p pattern, relPath string) bool {
	base := path(relPath).base()

	if p.rooted {
		// anchored: match only from the root
		ok, _ := filepath.Match(p.raw, relPath)
		return ok
	}

	// un-anchored: match against any component or the full path
	if ok, _ := filepath.Match(p.raw, base); ok {
		return true
	}
	if ok, _ := filepath.Match(p.raw, relPath); ok {
		return true
	}
	// Also check each path segment to handle patterns like "vendor"
	// matching inside sub-trees (e.g. "src/vendor/foo" → "vendor" matches)
	for _, seg := range strings.Split(relPath, "/") {
		if ok, _ := filepath.Match(p.raw, seg); ok {
			return true
		}
	}
	return false
}

type path string

func (p path) base() string {
	s := string(p)
	if i := strings.LastIndex(s, "/"); i >= 0 {
		return s[i+1:]
	}
	return s
}
