package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func prepareDemoRepo(t *testing.T) string {
	t.Helper()

	source := filepath.Join("demo_repo")
	tempRoot := t.TempDir()
	target := filepath.Join(tempRoot, "demo_repo")

	if err := copyDir(source, target); err != nil {
		t.Fatalf("copy demo repo: %v", err)
	}

	fixtureGit := filepath.Join(target, ".git-fixture")
	realGit := filepath.Join(target, ".git")

	if err := os.Rename(fixtureGit, realGit); err != nil {
		t.Fatalf("restore demo repo .git directory: %v", err)
	}

	return target
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()

		_, err = io.Copy(out, in)
		return err
	})
}