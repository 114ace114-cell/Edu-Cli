package skills

import (
	"os"
	"path/filepath"
	"strings"
)

// Skill is one SKILL.md. The full loader parses YAML frontmatter
// (internal/skills). Here the file body is the whole skill.
type Skill struct {
	Name string
	Body string
}

// Load reads dataDir/skills/*/SKILL.md. Missing dir yields no skills.
func Load(dataDir string) ([]Skill, error) {
	root := filepath.Join(dataDir, "skills")
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Skill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		body, err := os.ReadFile(filepath.Join(root, e.Name(), "SKILL.md"))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		out = append(out, Skill{Name: e.Name(), Body: strings.TrimSpace(string(body))})
	}
	return out, nil
}

// Find returns a skill by name.
func Find(list []Skill, name string) (Skill, bool) {
	for _, s := range list {
		if s.Name == name {
			return s, true
		}
	}
	return Skill{}, false
}
