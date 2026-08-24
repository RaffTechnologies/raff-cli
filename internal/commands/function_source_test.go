package commands

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestIgnored(t *testing.T) {
	patterns := []string{"*.log", "tmp"}
	cases := map[string]bool{
		".git/config":            true,
		"node_modules/x/y.js":    true,
		"__pycache__/a.pyc":      true,
		".env":                   true,
		"app.log":                true,
		"tmp/scratch.txt":        true,
		"main.py":                false,
		"src/handler.ts":         false,
		"requirements.txt":       false,
		"lib/.env":               true, // basename match
		"assets/logs-readme.md":  false,
	}
	for rel, want := range cases {
		if got := ignored(rel, patterns); got != want {
			t.Errorf("ignored(%q) = %v, want %v", rel, got, want)
		}
	}
}

func TestZipSourceDirRespectsIgnores(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, content string) {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("main.py", "print('hi')")
	write("requirements.txt", "fastapi")
	write(".env", "SECRET=1")
	write("__pycache__/m.pyc", "junk")
	write(".git/HEAD", "ref")
	write("build/out.bin", "bin")
	write(".raffignore", "build\n")

	data, count, err := zipSourceDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("packed %d files, want 2", count)
	}
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, f := range r.File {
		got[f.Name] = true
	}
	if !got["main.py"] || !got["requirements.txt"] {
		t.Fatalf("missing expected files, got %v", got)
	}
	if got[".env"] || got["build/out.bin"] {
		t.Fatalf("ignored files leaked into zip: %v", got)
	}
}

func TestLoadRaffToml(t *testing.T) {
	dir := t.TempDir()
	toml := `name = "my-api"
runtime = "python"
memory_mb = 512
max_scale = 10

[env]
LOG_LEVEL = "debug"
`
	if err := os.WriteFile(filepath.Join(dir, "raff.toml"), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadRaffToml(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Name != "my-api" || cfg.Runtime != "python" || cfg.MemoryMB != 512 || cfg.MaxScale != 10 {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if cfg.Env["LOG_LEVEL"] != "debug" {
		t.Fatalf("env not parsed: %+v", cfg.Env)
	}

	missing, err := loadRaffToml(t.TempDir())
	if err != nil || missing != nil {
		t.Fatalf("absent raff.toml should return nil, nil — got %+v, %v", missing, err)
	}
}

func TestDetectRuntime(t *testing.T) {
	cases := map[string]string{
		"requirements.txt": "python",
		"package.json":     "node",
		"go.mod":           "go",
		"Dockerfile":       "docker",
	}
	for marker, want := range cases {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, marker), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := detectRuntime(dir)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("detectRuntime(%s) = %s, want %s", marker, got, want)
		}
	}
	if _, err := detectRuntime(t.TempDir()); err == nil {
		t.Error("empty dir should not detect a runtime")
	}
}

func TestSlugifyName(t *testing.T) {
	cases := map[string]string{
		"My API":        "my-api",
		"cli_demo":      "cli-demo",
		"--weird--":     "weird",
		"UPPER.case.py": "upper-case-py",
	}
	for in, want := range cases {
		if got := slugifyName(in); got != want {
			t.Errorf("slugifyName(%q) = %q, want %q", in, got, want)
		}
	}
}
