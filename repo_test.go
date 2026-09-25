package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestApplyDelta(t *testing.T) {
	base := []byte("hello world")
	delta := []byte{11, 11, 0x91, 0, 11}
	got, err := applyDelta(base, delta)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello world" {
		t.Fatalf("got %q", got)
	}
}

func TestSearchDemoRepo(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(wd, "demo_repo")
	r, err := NewRepo(root)
	if err != nil {
		t.Fatal(err)
	}
	got, err := SearchRepo(r, "API_KEY", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Results) != 1 {
		t.Fatalf("want 1 result, got %d", len(got.Results))
	}
}

func TestDependencyImportMatching(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		"go.mod": `module example.com/app

go 1.23

require (
	github.com/pkg/errors v0.9.1
	example.com/unused v1.2.3
	example.com/indirect v1.0.0 // indirect
)
`,
		"main.go": `package main

import (
	"fmt"
	"github.com/pkg/errors"
)

func main() {
	fmt.Println(errors.New("boom"))
}
`,
		"requirements.txt": "requests==2.31.0\nrich==13.7.0\nunused-package==1.0.0\n",
		"app.py":           "import requests\nfrom rich import print\n",
		"package-lock.json": `{
  "lockfileVersion": 3,
  "packages": {
    "": {"dependencies": {"axios": "^1.0.0", "unused-js": "^1.0.0"}},
    "node_modules/axios": {"version": "1.1.0"},
    "node_modules/unused-js": {"version": "1.0.0"},
    "node_modules/transitive": {"version": "2.0.0"}
  }
}
`,
		"app.js": `const axios = require("axios");\n`,
	}
	for name, body := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	r, err := NewRepo(root)
	if err != nil {
		t.Fatal(err)
	}
	report, err := AnalyzeDependencies(r)
	if err != nil {
		t.Fatal(err)
	}

	byKey := map[string]Dependency{}
	for _, d := range report.Dependencies {
		byKey[d.Ecosystem+":"+d.Name] = d
	}

	if !byKey["go:github.com/pkg/errors"].Referenced {
		t.Fatal("Go dependency imported from source should be referenced")
	}
	if byKey["go:example.com/unused"].Referenced {
		t.Fatal("unimported Go dependency should not be referenced")
	}
	if byKey["go:example.com/indirect"].Declared {
		t.Fatal("Go indirect dependency should not be marked direct")
	}
	if !byKey["python:requests"].Referenced || !byKey["python:rich"].Referenced {
		t.Fatal("Python imports should reference their declared distributions")
	}
	if byKey["python:unused-package"].Referenced {
		t.Fatal("dependency manifest text must not count as Python import usage")
	}
	if !byKey["npm:axios"].Referenced {
		t.Fatal("JavaScript require should reference the npm dependency")
	}
	if byKey["npm:unused-js"].Referenced {
		t.Fatal("unimported npm dependency should not be referenced")
	}
	if byKey["npm:transitive"].Declared {
		t.Fatal("transitive npm dependency should not be marked direct")
	}

	for _, f := range report.Files {
		if f.Path == "" || f.Path == "." || strings.Contains(f.Path, "..") {
			t.Fatalf("dependency file path is not repository-relative: %#v", f)
		}
	}
}

func TestExplainDependencyMentionsAcrossEcosystems(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"go.mod": `module example.com/app

go 1.23

require github.com/pkg/errors v0.9.1
`,
		"main.go": `package main

import "github.com/pkg/errors"

func main() { _ = errors.New("boom") }
`,
		"requirements.txt":  "requests==2.31.0\n",
		"app.py":            "from requests import get\n",
		"package-lock.json": `{"lockfileVersion":3,"packages":{"":{"dependencies":{"axios":"^1.0.0"}},"node_modules/axios":{"version":"1.1.0"}}}`,
		"app.js": `const axios = require("axios")
`,
		"Cargo.toml": `[package]
name = "demo"
version = "0.1.0"

[dependencies]
serde = "1"
`,
		"Cargo.lock": `[[package]]
name = "demo"
version = "0.1.0"
dependencies = ["serde"]

[[package]]
name = "serde"
version = "1.0.0"
`,
		"src/main.rs": `use serde::Serialize;

fn main() {}
`,
	}
	for name, body := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r, err := NewRepo(root)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		path string
		want string
	}{
		{"main.go", "github.com/pkg/errors"},
		{"app.py", "requests"},
		{"app.js", "axios"},
		{"src/main.rs", "serde"},
	}
	for _, tc := range cases {
		got, err := Explain(r, tc.path)
		if err != nil {
			t.Fatalf("Explain(%q): %v", tc.path, err)
		}
		found := false
		for _, dep := range got.DependencyMentions {
			if dep == tc.want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Explain(%q) dependency mentions = %#v, want %q", tc.path, got.DependencyMentions, tc.want)
		}
	}
}

func TestExplainCargoRenamedDependency(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"Cargo.toml": `[package]
name = "demo"
version = "0.1.0"

[dependencies]
serde_alias = { package = "serde", version = "1" }
`,
		"Cargo.lock": `[[package]]
name = "demo"
version = "0.1.0"
dependencies = ["serde"]

[[package]]
name = "serde"
version = "1.0.0"
`,
		"src/main.rs": `use serde_alias::Serialize;

fn main() {}
`,
	}
	for name, body := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r, err := NewRepo(root)
	if err != nil {
		t.Fatal(err)
	}
	report, err := AnalyzeDependencies(r)
	if err != nil {
		t.Fatal(err)
	}
	var serde Dependency
	for _, dep := range report.Dependencies {
		if dep.Name == "serde" && dep.Ecosystem == "rust" {
			serde = dep
			break
		}
	}
	if !serde.Declared || !serde.Referenced {
		t.Fatalf("renamed Cargo dependency = %#v", serde)
	}
	got, err := Explain(r, "src/main.rs")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.DependencyMentions) != 1 || got.DependencyMentions[0] != "serde" {
		t.Fatalf("renamed Cargo Explain mentions = %#v", got.DependencyMentions)
	}
}

func TestExplainCargoRenamedDependencyMultiline(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"Cargo.toml": `[package]
name = "demo"
version = "0.1.0"

[dependencies]
serde_alias = {
    package = "serde",
    version = "1"
}
`,
		"Cargo.lock": `[[package]]
name = "demo"
version = "0.1.0"
dependencies = ["serde"]

[[package]]
name = "serde"
version = "1.0.0"
`,
		"src/main.rs": `use serde_alias::Serialize;

fn main() {}
`,
	}
	for name, body := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r, err := NewRepo(root)
	if err != nil {
		t.Fatal(err)
	}
	report, err := AnalyzeDependencies(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, dep := range report.Dependencies {
		if dep.Ecosystem == "rust" && dep.Name == "package" {
			t.Fatalf("multiline Cargo table was parsed as a dependency: %#v", report.Dependencies)
		}
	}
	var serde Dependency
	for _, dep := range report.Dependencies {
		if dep.Name == "serde" && dep.Ecosystem == "rust" {
			serde = dep
			break
		}
	}
	if !serde.Declared || !serde.Referenced {
		t.Fatalf("multiline renamed Cargo dependency = %#v", serde)
	}
	got, err := Explain(r, "src/main.rs")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.DependencyMentions) != 1 || got.DependencyMentions[0] != "serde" {
		t.Fatalf("multiline renamed Cargo Explain mentions = %#v", got.DependencyMentions)
	}
}

func TestCargoDependencyDirectnessAndReference(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"Cargo.toml": `[package]
name = "demo"
version = "0.1.0"

[dependencies]
serde = "1"
unused-crate = "1"
`,
		"Cargo.lock": `[[package]]
name = "demo"
version = "0.1.0"
dependencies = ["serde", "unused-crate"]

[[package]]
name = "serde"
version = "1.0.0"

[[package]]
name = "unused-crate"
version = "1.0.0"

[[package]]
name = "transitive"
version = "2.0.0"
`,
		"src/main.rs": `use serde::Serialize;
fn main() {}
`,
	}
	for name, body := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r, err := NewRepo(root)
	if err != nil {
		t.Fatal(err)
	}
	report, err := AnalyzeDependencies(r)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]Dependency{}
	for _, d := range report.Dependencies {
		byName[d.Name] = d
	}
	if !byName["serde"].Declared || !byName["serde"].Referenced {
		t.Fatal("serde should be a referenced direct dependency")
	}
	if !byName["unused-crate"].Declared || byName["unused-crate"].Referenced {
		t.Fatal("unused-crate should be a direct but unreferenced dependency")
	}
	if byName["transitive"].Declared {
		t.Fatal("transitive Cargo.lock entry should not be marked direct")
	}
}

func TestExplainRejectsPathsOutsideRepository(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	inside := filepath.Join(root, "inside.txt")
	outside := filepath.Join(filepath.Dir(root), "outside-sourceevidence-test.txt")
	if err := os.WriteFile(inside, []byte("inside"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(outside)

	r, err := NewRepo(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Explain(r, "../outside-sourceevidence-test.txt"); err == nil {
		t.Fatal("expected path traversal to be rejected")
	}
}

func TestExplainRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(filepath.Dir(root), "outside-sourceevidence-symlink-test.txt")
	if err := os.WriteFile(outside, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(outside)

	link := filepath.Join(root, "link.txt")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	linkDir := filepath.Join(root, "linkdir")
	if err := os.Symlink(filepath.Dir(outside), linkDir); err != nil {
		t.Skipf("directory symlinks unavailable: %v", err)
	}

	r, err := NewRepo(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Explain(r, "link.txt"); err == nil {
		t.Fatal("expected symlink file escape to be rejected")
	}
	if _, err := Explain(r, "linkdir/outside-sourceevidence-symlink-test.txt"); err == nil {
		t.Fatal("expected symlink directory escape to be rejected")
	}
}

func TestGitChangesForCurrentPathMatchesRevListSemantics(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable not available")
	}
	root := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "SourceEvidence Test")
	file := filepath.Join(root, "src", "demo.go")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	for i, body := range []string{"one\n", "two\n", "three\n"} {
		if err := os.WriteFile(file, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		run("add", "src/demo.go")
		run("commit", "-m", fmt.Sprintf("change %d", i+1))
	}
	r, err := NewRepo(root)
	if err != nil {
		t.Fatal(err)
	}
	got, err := GitChangesForPath(r, "src/demo.go")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "rev-list", "--count", "HEAD", "--", "src/demo.go")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	want, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("GitChangesForPath = %d, rev-list = %d", got, want)
	}
}

func TestFormatBytesUsesCorrectUnits(t *testing.T) {
	cases := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{1024, "1.00 KB"},
		{2189478, "2.09 MB"},
		{883794, "863 KB"},
		{306086, "299 KB"},
	}
	for _, tc := range cases {
		if got := formatBytes(tc.bytes); got != tc.want {
			t.Fatalf("formatBytes(%d) = %q, want %q", tc.bytes, got, tc.want)
		}
	}
}

func TestDedupeDependenciesMergesSameScope(t *testing.T) {
	in := []Dependency{
		{Name: "chardet", Ecosystem: "python", Version: ">=3.0.2,<8", Scope: "optional", Declared: true},
		{Name: "chardet", Ecosystem: "python", Scope: "optional", Declared: true, Referenced: true},
		{Name: "chardet", Ecosystem: "python", Version: "7.1", Scope: "runtime", Declared: true},
	}
	out := dedupeDependencies(in)
	if len(out) != 2 {
		t.Fatalf("got %d dependencies, want 2: %#v", len(out), out)
	}
	for _, d := range out {
		if d.Scope == "optional" {
			if d.Version != ">=3.0.2,<8" || !d.Referenced {
				t.Fatalf("unexpected optional dependency: %#v", d)
			}
		}
	}
}

func TestPyprojectDependencyScopesAndReferences(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"pyproject.toml": `[project]
name = "demo"
version = "0.1.0"
dependencies = ["requests>=2.31", "urllib3>=2"]

[project.optional-dependencies]
dev = ["pytest>=8"]
docs = ["Sphinx==7.2.6"]
`,
		"app.py":       "import requests\nimport urllib3\n",
		"docs/conf.py": "import sphinx\nfrom sphinx.ext import autodoc\n",
	}
	for name, body := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r, err := NewRepo(root)
	if err != nil {
		t.Fatal(err)
	}
	report, err := AnalyzeDependencies(r)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]Dependency{}
	for _, d := range report.Dependencies {
		byName[d.Name] = d
	}
	if !byName["requests"].Referenced || byName["requests"].Scope != "runtime" {
		t.Fatalf("requests = %#v", byName["requests"])
	}
	if byName["pytest"].Scope != "dev/test" || !byName["pytest"].Declared {
		t.Fatalf("pytest = %#v", byName["pytest"])
	}
	if byName["Sphinx"].Scope != "docs" || !byName["Sphinx"].Referenced {
		t.Fatalf("Sphinx = %#v", byName["Sphinx"])
	}
	for _, f := range report.Findings {
		if f.Package == "Sphinx" {
			t.Fatal("Sphinx should not be flagged as possibly-unused")
		}
	}
	got, err := Explain(r, "docs/conf.py")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.DependencyMentions) != 1 || got.DependencyMentions[0] != "Sphinx" {
		t.Fatalf("docs dependency mentions = %#v", got.DependencyMentions)
	}
}

func TestSecretDetectorAvoidsCommonFalsePositives(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "sample.py")
	content := `password=password
TOKEN = "hunter2"
url = f"#token=hunter2"
api_key="abcdefghijklmnopQRSTuvwx1234"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	hits := detectSecrets(root, path)
	if len(hits) != 1 {
		t.Fatalf("got %d secret hits, want 1: %#v", len(hits), hits)
	}
	if hits[0].Kind != "generic-secret" || hits[0].Line != 4 {
		t.Fatalf("unexpected hit: %#v", hits[0])
	}
}

func TestReportExportCreatesCleanTextFile(t *testing.T) {
	tmp := t.TempDir()
	oldHome := os.Getenv("HOME")
	oldUserProfile := os.Getenv("USERPROFILE")
	_ = os.Setenv("HOME", tmp)
	_ = os.Setenv("USERPROFILE", tmp)
	defer os.Setenv("HOME", oldHome)
	defer os.Setenv("USERPROFILE", oldUserProfile)

	repoRoot := filepath.Join(tmp, "repo")
	if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := NewRepo(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	s := &interactiveSession{Repo: r}
	s.set("Repository scan", ScanReport{Repo: repoRoot, Files: 1, Bytes: 5, Extensions: map[string]int{".md": 1}})
	if err := s.export(); err != nil {
		t.Fatal(err)
	}
	reportDir, err := reportDirectory()
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(reportDir)
	if err != nil {
		t.Fatal(err)
	}
	var reportPath string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "sourceevidence-report-repo-") && strings.HasSuffix(e.Name(), ".txt") {
			reportPath = filepath.Join(reportDir, e.Name())
		}
	}
	if reportPath == "" {
		t.Fatal("report file was not created")
	}
	b, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, needle := range []string{"SOURCE EVIDENCE REPORT", "REPOSITORY SCAN", "5 B", "read-only inspection"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("report missing %q: %s", needle, text)
		}
	}
	_ = os.Remove(reportPath)
}

func TestTopLevelArea(t *testing.T) {
	cases := map[string]string{
		"main.go":             "[root]",
		"README.md":           "[root]",
		"src/main.go":         "src/",
		"src/internal/app.go": "src/",
		"docs/guide.md":       "docs/",
	}
	for input, want := range cases {
		if got := topLevelArea(input); got != want {
			t.Fatalf("topLevelArea(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestOverviewDemoRepo(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewRepo(filepath.Join(wd, "demo_repo"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := OverviewRepo(r, 10)
	if err != nil {
		t.Fatal(err)
	}
	if got.Files == 0 || got.Bytes == 0 {
		t.Fatalf("overview summary is empty: %#v", got)
	}
	if len(got.Areas) == 0 {
		t.Fatal("expected at least one repository area")
	}
	foundRoot := false
	for _, area := range got.Areas {
		if area.Name == "[root]" {
			foundRoot = true
			break
		}
	}
	if !foundRoot {
		t.Fatal("expected root-level files area")
	}
	if len(got.LargestFiles) == 0 {
		t.Fatal("expected largest files in overview")
	}
	if got.Head == "" {
		t.Fatal("expected HEAD in overview")
	}
}

func TestSecretDetectorClassifiesLowEntropyTestCredential(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "tests"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "tests", "test_auth.py")
	content := `auth = httpx.DigestAuth(username="Mufasa", password="CircleOfLife")
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	hits := detectSecrets(root, path)
	if len(hits) != 1 || hits[0].Kind != "mock-test-secret" {
		t.Fatalf("expected mock-test-secret, got %#v", hits)
	}
}

func TestSecretDetectorKeepsHighRiskTestKey(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "fixtures"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "fixtures", "test_secrets.py")
	content := `api_key="a1B2c3D4e5F6g7H8i9J0k1L2m3N4o5P6"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	hits := detectSecrets(root, path)
	if len(hits) != 1 || hits[0].Kind != "generic-secret" {
		t.Fatalf("expected high-risk generic-secret, got %#v", hits)
	}
}

func TestRequirementsCommentsSetDependencyScope(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := `requests==2.31.0

# Optional charset auto-detection
chardet==5.2.0

# Documentation
mkdocs==1.6.1
mkautodoc==0.2.0

# Packaging build
build==1.3.0

# Tests & Linting
pytest==8.4.1
ruff==0.12.11
`
	if err := os.WriteFile(filepath.Join(root, "requirements.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := NewRepo(root)
	if err != nil {
		t.Fatal(err)
	}
	report, err := AnalyzeDependencies(r)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]Dependency{}
	for _, d := range report.Dependencies {
		byName[d.Name] = d
	}
	wantScopes := map[string]string{
		"requests":  "runtime",
		"chardet":   "optional",
		"mkdocs":    "docs",
		"mkautodoc": "docs",
		"build":     "dev/test",
		"pytest":    "dev/test",
		"ruff":      "dev/test",
	}
	for name, want := range wantScopes {
		if got := byName[name].Scope; got != want {
			t.Fatalf("%s scope = %q, want %q", name, got, want)
		}
	}
	for _, f := range report.Findings {
		if f.Package == "mkdocs" || f.Package == "build" || f.Package == "pytest" || f.Package == "ruff" {
			t.Fatalf("non-runtime dependency should not be flagged: %#v", f)
		}
	}
}

func TestGitExplainAndHotspotUseSameHistoryScope(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable not available")
	}
	root := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "SourceEvidence Test")
	run("branch", "-M", "main")
	file := filepath.Join(root, "src", "demo.go")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	for i, body := range []string{"one\n", "two\n"} {
		if err := os.WriteFile(file, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		run("add", "src/demo.go")
		run("commit", "-m", fmt.Sprintf("main change %d", i+1))
	}
	run("checkout", "-b", "feature")
	if err := os.WriteFile(file, []byte("feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "src/demo.go")
	run("commit", "-m", "feature change")

	r, err := NewRepo(root)
	if err != nil {
		t.Fatal(err)
	}
	hotspots, err := Hotspots(r, 30)
	if err != nil {
		t.Fatal(err)
	}
	gotExplain, err := Explain(r, "src/demo.go")
	if err != nil {
		t.Fatal(err)
	}
	var hotspotCount int
	for _, h := range hotspots.Files {
		if h.Path == "src/demo.go" {
			hotspotCount = h.Changes
		}
	}
	if hotspotCount == 0 {
		t.Fatal("expected demo.go hotspot")
	}
	if gotExplain.GitChanges != hotspotCount {
		t.Fatalf("Explain GitChanges = %d, hotspot count = %d", gotExplain.GitChanges, hotspotCount)
	}
	if gotExplain.GitHistoryScope != gitHistoryScopeLabel || hotspots.HistoryScope != gitHistoryScopeLabel {
		t.Fatalf("history scopes not aligned: explain=%q hotspots=%q", gotExplain.GitHistoryScope, hotspots.HistoryScope)
	}
}

func TestMarkdownReportContainsStructuredOutput(t *testing.T) {
	got := markdownReport("scan", ScanReport{Files: 3, Bytes: 2048, LargeFiles: []FileStat{{Path: "main.go", Bytes: 1024}}})
	for _, needle := range []string{"### Scan", "| Files | 3 |", "| Total size | 2.00 KB |", "| `main.go` | 1.00 KB |"} {
		if !strings.Contains(got, needle) {
			t.Fatalf("markdown report missing %q: %s", needle, got)
		}
	}
}
