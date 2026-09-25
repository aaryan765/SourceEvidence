package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Repo struct {
	Root string
	Git  string
}

func NewRepo(root string) (*Repo, error) {
	st, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", root)
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	gitPath := filepath.Join(root, ".git")
	gst, err := os.Stat(gitPath)
	if err != nil {
		return nil, fmt.Errorf("not a Git repository: %w", err)
	}
	if gst.IsDir() {
		return &Repo{Root: root, Git: gitPath}, nil
	}
	// Worktrees/submodules can have a .git file pointing at the real git dir.
	data, err := os.ReadFile(gitPath)
	if err != nil {
		return nil, err
	}
	line := strings.TrimSpace(string(data))
	if !strings.HasPrefix(strings.ToLower(line), "gitdir:") {
		return nil, errors.New("unsupported .git file")
	}
	p := strings.TrimSpace(line[len("gitdir:"):])
	if !filepath.IsAbs(p) {
		p = filepath.Join(root, p)
	}
	return &Repo{Root: root, Git: filepath.Clean(p)}, nil
}

func (r *Repo) readText(name string) (string, error) {
	b, err := os.ReadFile(filepath.Join(r.Git, name))
	return string(b), err
}

// resolveRepoPath converts a user-supplied path into a path guaranteed to stay
// inside the inspected repository. Existing symlinks are resolved before the
// containment check so a symlink cannot escape the repository root.
func resolveRepoPath(r *Repo, input string) (string, string, error) {
	if strings.TrimSpace(input) == "" {
		return "", "", errors.New("path must not be empty")
	}
	rootAbs, err := filepath.Abs(r.Root)
	if err != nil {
		return "", "", err
	}
	full := input
	if !filepath.IsAbs(full) {
		full = filepath.Join(rootAbs, full)
	}
	full, err = filepath.Abs(full)
	if err != nil {
		return "", "", err
	}
	full = filepath.Clean(full)

	rel, err := filepath.Rel(rootAbs, full)
	if err != nil {
		return "", "", fmt.Errorf("invalid repository path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", "", fmt.Errorf("path %q is outside the repository", input)
	}

	// Resolve symlink components even when the final path does not exist.
	// Otherwise a request such as "link-to-outside/new.txt" could escape
	// through an existing symlinked parent directory.
	realRoot, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return "", "", err
	}
	existing := full
	var suffix []string
	for {
		if _, statErr := os.Lstat(existing); statErr == nil {
			break
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			break
		}
		suffix = append([]string{filepath.Base(existing)}, suffix...)
		existing = parent
	}
	if _, statErr := os.Lstat(existing); statErr == nil {
		realExisting, err := filepath.EvalSymlinks(existing)
		if err != nil {
			return "", "", err
		}
		realFull := realExisting
		for _, part := range suffix {
			realFull = filepath.Join(realFull, part)
		}
		realRel, err := filepath.Rel(realRoot, realFull)
		if err != nil {
			return "", "", err
		}
		if realRel == ".." || strings.HasPrefix(realRel, ".."+string(os.PathSeparator)) {
			return "", "", fmt.Errorf("path %q resolves outside the repository", input)
		}
	}
	return filepath.ToSlash(rel), full, nil
}

func safeRel(root, p string) string {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return ""
	}
	return filepath.ToSlash(rel)
}
