package instructions

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// builtinCommands are command names that cannot be overridden by custom commands.
var builtinCommands = map[string]struct{}{
	"help":     {},
	"model":    {},
	"config":   {},
	"sessions": {},
	"session":  {},
	"clear":    {},
	"quit":     {},
	"exit":     {},
}

// maxCommandFileChars is the maximum character count for a single custom command file.
const maxCommandFileChars = 8000

func Loader(name string) (string, error) {
	dir := os.Getenv("EDU_DATA")
	if dir == "" {
		dir = "data"
	}
	path := filepath.Join(dir, "instructions", name+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// LoadInstructions reads instruction markdown and returns the combined text.
// eduDir is the data directory (EDU_DATA or data). projectDir is optional,
// usually .edu in the working directory. maxTokens is an approximate budget
// (1 token ~ 4 chars). Missing files are skipped. Higher-priority files win
// when the same line appears twice.
func LoadInstructions(eduDir string, projectDir string, maxTokens int) (string, error) {
	type source struct {
		path     string
		priority int
	}

	var sources []source
	priority := 0
	if eduDir != "" {
		sources = append(sources, source{filepath.Join(eduDir, "instructions.md"), priority})
		priority++
		for _, rf := range sortedMDFiles(filepath.Join(eduDir, "rules")) {
			sources = append(sources, source{rf, priority})
			priority++
		}
	}
	if projectDir != "" {
		sources = append(sources, source{filepath.Join(projectDir, "instructions.md"), priority})
		priority++
		for _, rf := range sortedMDFiles(filepath.Join(projectDir, "rules")) {
			sources = append(sources, source{rf, priority})
			priority++
		}
		sources = append(sources, source{filepath.Join(projectDir, "instructions.local.md"), priority})
	}

	type fileContent struct {
		path     string
		lines    []string
		priority int
	}
	var loaded []fileContent
	for _, src := range sources {
		data, err := readMDFile(src.path)
		if err != nil {
			continue
		}
		loaded = append(loaded, fileContent{
			path:     src.path,
			lines:    strings.Split(data, "\n"),
			priority: src.priority,
		})
	}

	seenLines := make(map[string]struct{})
	for i := len(loaded) - 1; i >= 0; i-- {
		fc := &loaded[i]
		deduped := make([]string, 0, len(fc.lines))
		for _, line := range fc.lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				deduped = append(deduped, line)
				continue
			}
			if _, exists := seenLines[trimmed]; !exists {
				seenLines[trimmed] = struct{}{}
				deduped = append(deduped, line)
			}
		}
		fc.lines = deduped
	}

	if maxTokens <= 0 {
		maxTokens = 4000
	}
	maxChars := maxTokens * 4
	var parts []string
	for _, fc := range loaded {
		content := strings.TrimSpace(strings.Join(fc.lines, "\n"))
		if content == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("<!-- from: %s -->\n%s", fc.path, content))
	}
	result := strings.Join(parts, "\n\n")
	if len(result) > maxChars {
		result = result[:maxChars]
		result += "\n[Instructions truncated — reduce content in lower-priority files]"
	}
	return result, nil
}

// LoadMemory reads data/Memory.md, or EDU_DATA/Memory.md when that directory
// is set. maxLines keeps the first lines only. A missing file is empty memory.
func LoadMemory(maxLines int) (string, error) {
	dir := os.Getenv("EDU_DATA")
	if dir == "" {
		dir = "data"
	}
	if maxLines <= 0 {
		maxLines = 200
	}
	data, err := readMDFile(filepath.Join(dir, "Memory.md"))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	lines := strings.Split(data, "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return strings.Join(lines, "\n"), nil
}

// LoadCustomCommands reads .md files from eduDir/commands and projectDir/commands.
// Project files override data-dir files with the same name. Built-in names are skipped.
func LoadCustomCommands(eduDir string, projectDir string) (map[string]string, error) {
	commands := make(map[string]string)
	if eduDir != "" {
		loadCommandDir(filepath.Join(eduDir, "commands"), commands)
	}
	if projectDir != "" {
		loadCommandDir(filepath.Join(projectDir, "commands"), commands)
	}
	return commands, nil
}

func loadCommandDir(dir string, commands map[string]string) {
	for _, path := range sortedMDFiles(dir) {
		name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		if _, isBuiltin := builtinCommands[strings.ToLower(name)]; isBuiltin {
			fmt.Fprintf(os.Stderr, "warning: custom command %q skipped — conflicts with built-in command\n", name)
			continue
		}
		data, err := readMDFile(path)
		if err != nil {
			continue
		}
		if len(data) > maxCommandFileChars {
			data = data[:maxCommandFileChars]
		}
		commands[name] = data
	}
}

func sortedMDFiles(dir string) []string {
	matches, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil || len(matches) == 0 {
		return nil
	}
	sort.Strings(matches)
	return matches
}

// readMDFile reads a file if it exists, is a .md file, and contains valid UTF-8.
// Returns the file contents or an error.
func readMDFile(path string) (string, error) {
	if !strings.EqualFold(filepath.Ext(path), ".md") {
		return "", fmt.Errorf("not a .md file: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(data) {
		return "", fmt.Errorf("file is not valid UTF-8: %s", path)
	}
	return string(data), nil
}
