package commands

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

// raffToml is the optional per-project function config ("Layer 1" of the
// deploy model — Layer 0 is zero-config, Layer 2 is a Dockerfile).
type raffToml struct {
	Name           string            `toml:"name"`
	Runtime        string            `toml:"runtime"`
	MemoryMB       int               `toml:"memory_mb"`
	TimeoutSeconds int               `toml:"timeout_seconds"`
	// ExtendedTimeout opts into timeouts above 1h (max 24h) — usage-billed.
	ExtendedTimeout bool `toml:"extended_timeout"`
	MinScale       int               `toml:"min_scale"`
	MaxScale       int               `toml:"max_scale"`
	Env            map[string]string `toml:"env"`
	// Parsed for forward-compatibility; managed in the dashboard today.
	Triggers []map[string]any `toml:"triggers"`
	Bindings []map[string]any `toml:"bindings"`
}

// loadRaffToml reads raff.toml from dir; returns nil when the file is absent.
func loadRaffToml(dir string) (*raffToml, error) {
	path := filepath.Join(dir, "raff.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading raff.toml: %w", err)
	}
	var cfg raffToml
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing raff.toml: %w", err)
	}
	return &cfg, nil
}

// detectRuntime infers the function runtime from project files.
func detectRuntime(dir string) (string, error) {
	exists := func(name string) bool {
		_, err := os.Stat(filepath.Join(dir, name))
		return err == nil
	}
	switch {
	case exists("Dockerfile"):
		return "docker", nil
	case exists("go.mod"):
		return "go", nil
	case exists("package.json"):
		return "node", nil
	case exists("requirements.txt") || exists("pyproject.toml") || exists("main.py"):
		return "python", nil
	}
	return "", fmt.Errorf("could not detect runtime in %s — add raff.toml with `runtime = \"python|node|go|docker\"`", dir)
}

// defaultIgnores are always excluded from source zips. `.env` is deliberate:
// local secrets never ship — set env vars on the function instead.
var defaultIgnores = []string{
	".git", "node_modules", ".venv", "venv", "__pycache__",
	".DS_Store", ".env", ".raffignore", ".next", "dist", "bin",
}

// loadIgnorePatterns reads .raffignore (one pattern per line, `#` comments;
// matched against slash-separated paths relative to the project root).
func loadIgnorePatterns(dir string) []string {
	data, err := os.ReadFile(filepath.Join(dir, ".raffignore"))
	if err != nil {
		return nil
	}
	var patterns []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, strings.TrimSuffix(line, "/"))
	}
	return patterns
}

func ignored(rel string, patterns []string) bool {
	rel = filepath.ToSlash(rel)
	base := filepath.Base(rel)
	for _, p := range append(defaultIgnores, patterns...) {
		p = filepath.ToSlash(p)
		if rel == p || base == p || strings.HasPrefix(rel, p+"/") {
			return true
		}
		if ok, _ := filepath.Match(p, rel); ok {
			return true
		}
		if ok, _ := filepath.Match(p, base); ok {
			return true
		}
	}
	return false
}

const maxSourceZipBytes = 100 << 20 // matches the server-side source cap

// zipSourceDir packs dir into a zip, honoring .raffignore + default ignores.
func zipSourceDir(dir string) ([]byte, int, error) {
	patterns := loadIgnorePatterns(dir)
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	count := 0

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if ignored(rel, patterns) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.IsDir() || !info.Mode().IsRegular() {
			return nil
		}
		fw, err := w.Create(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		_, err = io.Copy(fw, f)
		_ = f.Close()
		if err != nil {
			return err
		}
		count++
		if buf.Len() > maxSourceZipBytes {
			return fmt.Errorf("source exceeds %dMB — add build artifacts to .raffignore", maxSourceZipBytes>>20)
		}
		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	if err := w.Close(); err != nil {
		return nil, 0, err
	}
	if count == 0 {
		return nil, 0, fmt.Errorf("nothing to deploy: no files found in %s", dir)
	}
	return buf.Bytes(), count, nil
}

var slugifyRe = regexp.MustCompile(`[^a-z0-9-]+`)

// slugifyName derives a DNS-label-ish name from a directory name.
func slugifyName(name string) string {
	s := slugifyRe.ReplaceAllString(strings.ToLower(name), "-")
	s = strings.Trim(s, "-")
	if len(s) > 63 {
		s = s[:63]
	}
	return s
}
