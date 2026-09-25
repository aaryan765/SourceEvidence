package main

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type DependencyReport struct {
	Files        []LockfileInfo `json:"files"`
	Dependencies []Dependency   `json:"dependencies"`
	Findings     []Finding      `json:"findings"`
}
type LockfileInfo struct {
	Path      string `json:"path"`
	Ecosystem string `json:"ecosystem"`
	Packages  int    `json:"packages"`
}
type Dependency struct {
	Name       string `json:"name"`
	Ecosystem  string `json:"ecosystem"`
	Version    string `json:"version,omitempty"`
	Declared   bool   `json:"declared"`
	Referenced bool   `json:"referenced"`
	Script     bool   `json:"install_script"`
	Scope      string `json:"scope,omitempty"`
	// referenceNames contains source-level names that can refer to this package.
	// It is internal metadata used by Explain and intentionally omitted from JSON.
	referenceNames []string
}
type Finding struct {
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Detail   string `json:"detail"`
	Package  string `json:"package,omitempty"`
}

type importUsage struct {
	NPM    map[string]bool
	Go     map[string]bool
	Python map[string]bool
	Rust   map[string]bool
}

var (
	pythonImportRe = regexp.MustCompile(`(?m)^\s*import\s+([^#\n]+)`)
	pythonFromRe   = regexp.MustCompile(`(?m)^\s*from\s+([A-Za-z_][A-Za-z0-9_.]*)\s+import\b`)
	jsImportRe     = regexp.MustCompile(`(?m)^\s*import\s+(?:[^"'\n]*?\sfrom\s+)?["']([^"']+)["']`)
	jsDynamicRe    = regexp.MustCompile(`\bimport\s*\(\s*["']([^"']+)["']\s*\)`)
	jsRequireRe    = regexp.MustCompile(`\brequire\s*\(\s*["']([^"']+)["']\s*\)`)
	jsExportRe     = regexp.MustCompile(`(?m)^\s*export\s+(?:[^"'\n]*?\sfrom\s+)["']([^"']+)["']`)
	rustUseRe      = regexp.MustCompile(`(?m)^\s*(?:pub\s+)?use\s+([A-Za-z_][A-Za-z0-9_]*)`)
	rustExternRe   = regexp.MustCompile(`(?m)^\s*extern\s+crate\s+([A-Za-z_][A-Za-z0-9_]*)`)
	reqLineRe      = regexp.MustCompile(`^([A-Za-z0-9_.-]+)\s*(?:==|>=|<=|~=|>|<)?\s*([A-Za-z0-9_.+-]*)`)
	cargoDepRe     = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_-]*)\s*=`)
	cargoPackageRe = regexp.MustCompile(`(?i)\bpackage\s*=\s*["']([^"']+)["']`)
)

func AnalyzeDependencies(r *Repo) (DependencyReport, error) {
	out := DependencyReport{Files: []LockfileInfo{}, Dependencies: []Dependency{}, Findings: []Finding{}}
	used, err := collectImports(r)
	if err != nil {
		return out, err
	}

	err = filepath.WalkDir(r.Root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if d.IsDir() {
			if path != r.Root && skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		switch d.Name() {
		case "package-lock.json":
			parsePackageLock(r.Root, path, &out, used)
		case "npm-shrinkwrap.json":
			parsePackageLock(r.Root, path, &out, used)
		case "go.mod":
			parseGoMod(r.Root, path, &out, used)
		case "Cargo.lock":
			parseCargoLock(r.Root, path, &out, used)
		case "requirements.txt", "requirements-dev.txt", "requirements-test.txt", "requirements-docs.txt", "dev-requirements.txt", "test-requirements.txt":
			parseRequirements(r.Root, path, &out, used)
		case "pyproject.toml":
			parsePyproject(r.Root, path, &out, used)
		}
		return nil
	})
	if err != nil {
		return out, err
	}

	sort.Slice(out.Dependencies, func(i, j int) bool {
		if out.Dependencies[i].Ecosystem == out.Dependencies[j].Ecosystem {
			return out.Dependencies[i].Name < out.Dependencies[j].Name
		}
		return out.Dependencies[i].Ecosystem < out.Dependencies[j].Ecosystem
	})

	versions := map[string]map[string]bool{}
	for _, d := range out.Dependencies {
		if d.Version == "" {
			continue
		}
		key := d.Ecosystem + "\x00" + strings.ToLower(d.Name)
		if versions[key] == nil {
			versions[key] = map[string]bool{}
		}
		versions[key][d.Version] = true
	}
	for key, vs := range versions {
		if len(vs) <= 1 {
			continue
		}
		parts := strings.SplitN(key, "\x00", 2)
		list := make([]string, 0, len(vs))
		for v := range vs {
			list = append(list, v)
		}
		sort.Strings(list)
		out.Findings = append(out.Findings, Finding{"medium", "Multiple versions", "The dependency appears at multiple versions in the local manifest/lock data: " + strings.Join(list, ", "), parts[1]})
	}

	for _, d := range out.Dependencies {
		if d.Declared && !d.Referenced && d.Scope == "runtime" {
			out.Findings = append(out.Findings, Finding{"low", "Possibly unused dependency", "No matching source import/reference was found; dynamic/generated use can cause false positives.", d.Name})
		}
	}
	return out, nil
}

func dedupeDependencies(in []Dependency) []Dependency {
	type key struct {
		ecosystem string
		name      string
		scope     string
	}
	index := make(map[key]int, len(in))
	out := make([]Dependency, 0, len(in))
	for _, d := range in {
		k := key{strings.ToLower(d.Ecosystem), strings.ToLower(d.Name), strings.ToLower(d.Scope)}
		if i, ok := index[k]; ok {
			existing := &out[i]
			if existing.Version == "" {
				existing.Version = d.Version
			}
			existing.Declared = existing.Declared || d.Declared
			existing.Referenced = existing.Referenced || d.Referenced
			existing.Script = existing.Script || d.Script
			continue
		}
		index[k] = len(out)
		out = append(out, d)
	}
	return out
}

func parsePackageLock(root, path string, out *DependencyReport, used importUsage) {
	b, e := os.ReadFile(path)
	if e != nil {
		return
	}
	var v map[string]any
	if json.Unmarshal(b, &v) != nil {
		return
	}

	direct := map[string]bool{}
	if packages, ok := v["packages"].(map[string]any); ok {
		if rootPkg, ok := packages[""].(map[string]any); ok {
			collectNPMDirect(rootPkg, direct)
		}
		pkgs := 0
		for key, val := range packages {
			if key == "" {
				continue
			}
			pkgs++
			name := npmNameFromLockPath(key)
			obj, _ := val.(map[string]any)
			ver, _ := obj["version"].(string)
			script, _ := obj["hasInstallScript"].(bool)
			dep := Dependency{
				Name:       name,
				Ecosystem:  "npm",
				Version:    ver,
				Declared:   direct[npmCanonical(name)],
				Referenced: npmReferenced(used.NPM, name),
				Script:     script,
				Scope:      "runtime",
			}
			out.Dependencies = append(out.Dependencies, dep)
			if script {
				out.Findings = append(out.Findings, Finding{"medium", "Install/build script", "Package declares an install-time script flag.", name})
			}
		}
		out.Files = append(out.Files, LockfileInfo{safeRel(root, path), "npm", pkgs})
		return
	}

	// package-lock v1 has a top-level dependencies tree. Keep directness accurate
	// for the first level and include nested lock entries as transitive packages.
	if deps, ok := v["dependencies"].(map[string]any); ok {
		for name := range deps {
			direct[npmCanonical(name)] = true
		}
		pkgs := appendNPMV1Dependencies(deps, "", out, used, direct)
		out.Files = append(out.Files, LockfileInfo{safeRel(root, path), "npm", pkgs})
	}
}

func collectNPMDirect(rootPkg map[string]any, direct map[string]bool) {
	for _, field := range []string{"dependencies", "devDependencies", "optionalDependencies", "peerDependencies"} {
		if deps, ok := rootPkg[field].(map[string]any); ok {
			for name := range deps {
				direct[npmCanonical(name)] = true
			}
		}
	}
}

func appendNPMV1Dependencies(deps map[string]any, parent string, out *DependencyReport, used importUsage, direct map[string]bool) int {
	count := 0
	for name, raw := range deps {
		obj, _ := raw.(map[string]any)
		ver, _ := obj["version"].(string)
		script := false
		if scripts, ok := obj["scripts"].(map[string]any); ok {
			_, script = scripts["install"]
			if !script {
				_, script = scripts["postinstall"]
			}
		}
		canon := npmCanonical(name)
		out.Dependencies = append(out.Dependencies, Dependency{
			Name:       name,
			Ecosystem:  "npm",
			Version:    ver,
			Declared:   direct[canon] && parent == "",
			Referenced: npmReferenced(used.NPM, name),
			Script:     script,
			Scope:      "runtime",
		})
		count++
		if script {
			out.Findings = append(out.Findings, Finding{"medium", "Install/build script", "Package declares an install-time script flag.", name})
		}
		if nested, ok := obj["dependencies"].(map[string]any); ok {
			count += appendNPMV1Dependencies(nested, name, out, used, direct)
		}
	}
	return count
}

func npmNameFromLockPath(key string) string {
	parts := strings.Split(strings.TrimPrefix(key, "node_modules/"), "/node_modules/")
	return parts[len(parts)-1]
}

func npmCanonical(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func npmReferenced(used map[string]bool, name string) bool {
	return used[npmCanonical(name)]
}

func parseGoMod(root, path string, out *DependencyReport, used importUsage) {
	b, e := os.ReadFile(path)
	if e != nil {
		return
	}
	s := string(b)
	n := 0
	in := false
	for _, line := range strings.Split(s, "\n") {
		raw := strings.TrimSpace(line)
		if strings.HasPrefix(raw, "require (") {
			in = true
			continue
		}
		if in && raw == ")" {
			in = false
			continue
		}
		if !in && !strings.HasPrefix(raw, "require ") {
			continue
		}
		t := strings.TrimSpace(strings.TrimPrefix(raw, "require "))
		if t == "" || strings.HasPrefix(t, "//") {
			continue
		}
		p := strings.Fields(t)
		if len(p) == 0 {
			continue
		}
		name := p[0]
		ver := ""
		if len(p) >= 2 {
			ver = p[1]
		}
		n++
		indirect := strings.Contains(t, "// indirect")
		out.Dependencies = append(out.Dependencies, Dependency{
			Name:       name,
			Ecosystem:  "go",
			Version:    ver,
			Declared:   !indirect,
			Referenced: goReferenced(used.Go, name),
			Scope:      "runtime",
		})
	}
	out.Files = append(out.Files, LockfileInfo{safeRel(root, path), "go", n})
}

func goReferenced(imports map[string]bool, module string) bool {
	module = strings.TrimSuffix(module, "/")
	for imp := range imports {
		if imp == module || strings.HasPrefix(imp, module+"/") {
			return true
		}
	}
	return false
}

func parseCargoLock(root, path string, out *DependencyReport, used importUsage) {
	b, e := os.ReadFile(path)
	if e != nil {
		return
	}
	depInfo := cargoDirectDependencyInfo(filepath.Join(filepath.Dir(path), "Cargo.toml"))
	name, ver := "", ""
	n := 0
	flush := func() {
		if name == "" {
			return
		}
		canonical := cargoCanonical(name)
		references := append([]string{canonical}, depInfo[canonical]...)
		references = uniqueStrings(references)
		referenced := false
		for _, ref := range references {
			if used.Rust[cargoCanonical(ref)] {
				referenced = true
				break
			}
		}
		n++
		out.Dependencies = append(out.Dependencies, Dependency{
			Name:           name,
			Ecosystem:      "rust",
			Version:        ver,
			Declared:       depInfo[canonical] != nil,
			Referenced:     referenced,
			Scope:          "runtime",
			referenceNames: references,
		})
		name = ""
		ver = ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		t := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(t, "[[package]]"):
			flush()
		case strings.HasPrefix(t, "name = "):
			name, _ = strconv.Unquote(strings.TrimSpace(strings.TrimPrefix(t, "name = ")))
		case strings.HasPrefix(t, "version = "):
			ver, _ = strconv.Unquote(strings.TrimSpace(strings.TrimPrefix(t, "version = ")))
		}
	}
	flush()
	out.Files = append(out.Files, LockfileInfo{safeRel(root, path), "rust", n})
}

// cargoDirectDependencyInfo maps each canonical package name to any renamed
// source import names declared for that dependency in Cargo.toml.
func cargoDirectDependencyInfo(path string) map[string][]string {
	info := map[string][]string{}
	b, err := os.ReadFile(path)
	if err != nil {
		return info
	}
	section := ""
	pendingAlias := ""
	pendingValue := ""
	add := func(alias, value string) {
		alias = cargoCanonical(alias)
		if alias == "" {
			return
		}
		actual := alias
		if pkg := cargoPackageField(value); pkg != "" {
			actual = cargoCanonical(pkg)
		}
		if actual == "" {
			return
		}
		if info[actual] == nil {
			info[actual] = []string{}
		}
		if alias != actual {
			info[actual] = append(info[actual], alias)
		}
	}
	flushPending := func() {
		if pendingAlias != "" {
			add(pendingAlias, pendingValue)
			pendingAlias = ""
			pendingValue = ""
		}
	}
	for _, line := range strings.Split(string(b), "\n") {
		t := strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
		if t == "" {
			continue
		}
		if strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") {
			flushPending()
			section = strings.Trim(t, "[]")
			continue
		}
		if !strings.Contains(section, "dependencies") {
			continue
		}
		if pendingAlias != "" {
			pendingValue += " " + t
			if strings.Contains(t, "}") {
				flushPending()
			}
			continue
		}
		m := cargoDepRe.FindStringSubmatch(t)
		if len(m) <= 1 || m[1] == "" {
			continue
		}
		eq := strings.IndexByte(t, '=')
		value := ""
		if eq >= 0 {
			value = strings.TrimSpace(t[eq+1:])
		}
		if strings.Contains(value, "{") && !strings.Contains(value, "}") {
			pendingAlias = m[1]
			pendingValue = value
			continue
		}
		add(m[1], value)
	}
	flushPending()
	for name, aliases := range info {
		info[name] = uniqueStrings(aliases)
	}
	return info
}

func cargoPackageField(value string) string {
	m := cargoPackageRe.FindStringSubmatch(value)
	if len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, value := range in {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func cargoCanonical(name string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), "-", "_"))
}

func parseRequirements(root, path string, out *DependencyReport, used importUsage) {
	b, e := os.ReadFile(path)
	if e != nil {
		return
	}
	defaultScope := pythonManifestScope(safeRel(root, path))
	currentScope := defaultScope
	n := 0
	for _, line := range strings.Split(string(b), "\n") {
		raw := strings.TrimSpace(line)
		if strings.HasPrefix(raw, "#") {
			if scope, ok := requirementsCommentScope(strings.TrimSpace(strings.TrimPrefix(raw, "#"))); ok {
				currentScope = scope
			}
			continue
		}
		t := strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
		if t == "" || strings.HasPrefix(t, "-") {
			continue
		}
		m := reqLineRe.FindStringSubmatch(t)
		if len(m) > 2 && m[1] != "" {
			n++
			name := m[1]
			out.Dependencies = append(out.Dependencies, Dependency{
				Name:       name,
				Ecosystem:  "python",
				Version:    m[2],
				Declared:   true,
				Referenced: pythonReferenced(used.Python, name),
				Scope:      currentScope,
			})
		}
	}
	out.Files = append(out.Files, LockfileInfo{safeRel(root, path), "python", n})
}

func requirementsCommentScope(comment string) (string, bool) {
	g := strings.ToLower(strings.TrimSpace(comment))
	switch {
	case strings.Contains(g, "documentation"), strings.Contains(g, "docs"):
		return "docs", true
	case strings.Contains(g, "test"), strings.Contains(g, "lint"), strings.Contains(g, "development"), strings.Contains(g, "dev"), strings.Contains(g, "tooling"), strings.Contains(g, "packaging"), strings.Contains(g, "build"):
		return "dev/test", true
	case strings.Contains(g, "optional"):
		return "optional", true
	default:
		return "", false
	}
}

func pythonManifestScope(rel string) string {
	low := strings.ToLower(strings.ReplaceAll(rel, "\\", "/"))
	base := strings.ToLower(filepath.Base(rel))
	if strings.Contains(low, "/docs/") || strings.HasPrefix(low, "docs/") || strings.Contains(low, "/documentation/") {
		return "docs"
	}
	if strings.Contains(base, "docs") || strings.Contains(base, "doc") {
		return "docs"
	}
	if strings.Contains(base, "dev") || strings.Contains(base, "test") {
		return "dev/test"
	}
	return "runtime"
}

func parsePyproject(root, path string, out *DependencyReport, used importUsage) {
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	lines := strings.Split(string(b), "\n")
	section := ""
	arrayKey := ""
	arrayScope := "runtime"
	arrayBuf := ""
	count := 0
	flushArray := func() {
		if arrayKey == "" || arrayBuf == "" {
			arrayKey, arrayBuf = "", ""
			return
		}
		for _, spec := range tomlQuotedValues(arrayBuf) {
			name, ver := parsePythonRequirementSpec(spec)
			if name == "" {
				continue
			}
			count++
			out.Dependencies = append(out.Dependencies, Dependency{
				Name:       name,
				Ecosystem:  "python",
				Version:    ver,
				Declared:   true,
				Referenced: pythonReferenced(used.Python, name),
				Scope:      arrayScope,
			})
		}
		arrayKey, arrayBuf = "", ""
	}
	for _, raw := range lines {
		t := strings.TrimSpace(raw)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		if strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") {
			flushArray()
			section = strings.TrimSuffix(strings.TrimPrefix(t, "["), "]")
			arrayScope = pyprojectSectionScope(section)
			continue
		}
		if arrayKey != "" {
			arrayBuf += " " + t
			if strings.Contains(t, "]") {
				flushArray()
			}
			continue
		}
		if section == "project" && strings.HasPrefix(t, "dependencies") && strings.Contains(t, "[") {
			arrayKey = "dependencies"
			arrayScope = "runtime"
			arrayBuf = t[strings.Index(t, "["):]
			if strings.Contains(arrayBuf, "]") {
				flushArray()
			}
			continue
		}
		if (section == "project.optional-dependencies" || strings.HasPrefix(section, "project.optional-dependencies.")) && strings.Contains(t, "[") && strings.Contains(t, "=") {
			parts := strings.SplitN(t, "=", 2)
			group := strings.TrimSpace(parts[0])
			arrayKey = group
			if strings.HasPrefix(section, "project.optional-dependencies.") {
				arrayScope = optionalGroupScope(strings.TrimPrefix(section, "project.optional-dependencies."))
			} else {
				arrayScope = optionalGroupScope(group)
			}
			arrayBuf = parts[1]
			if strings.Contains(arrayBuf, "]") {
				flushArray()
			}
			continue
		}
		if section == "dependency-groups" && strings.Contains(t, "[") && strings.Contains(t, "=") {
			parts := strings.SplitN(t, "=", 2)
			group := strings.TrimSpace(parts[0])
			arrayKey = group
			arrayScope = optionalGroupScope(group)
			arrayBuf = parts[1]
			if strings.Contains(arrayBuf, "]") {
				flushArray()
			}
		}
	}
	flushArray()
	out.Files = append(out.Files, LockfileInfo{safeRel(root, path), "python", count})
}

func pyprojectSectionScope(section string) string {
	if section == "project" {
		return "runtime"
	}
	if strings.HasPrefix(section, "project.optional-dependencies") || section == "dependency-groups" {
		return "dev/test"
	}
	return "runtime"
}

func optionalGroupScope(group string) string {
	g := strings.ToLower(group)
	switch {
	case strings.Contains(g, "doc"), strings.Contains(g, "sphinx"):
		return "docs"
	case strings.Contains(g, "dev"), strings.Contains(g, "test"), strings.Contains(g, "lint"), strings.Contains(g, "build"):
		return "dev/test"
	default:
		return "optional"
	}
}

var pyprojectQuoteRe = regexp.MustCompile(`"([^"]+)"|'([^']+)'`)

func tomlQuotedValues(s string) []string {
	matches := pyprojectQuoteRe.FindAllStringSubmatch(s, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) > 1 && m[1] != "" {
			out = append(out, m[1])
		} else if len(m) > 2 {
			out = append(out, m[2])
		}
	}
	return out
}

func parsePythonRequirementSpec(spec string) (string, string) {
	spec = strings.TrimSpace(strings.SplitN(spec, ";", 2)[0])
	if spec == "" {
		return "", ""
	}
	name := spec
	for i, r := range spec {
		if strings.ContainsRune("<>=!~[ ", r) {
			name = spec[:i]
			break
		}
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ""
	}
	version := strings.TrimSpace(strings.TrimPrefix(spec, name))
	version = strings.TrimSpace(version)
	return name, version
}

func pythonReferenced(modules map[string]bool, distribution string) bool {
	canon := pythonCanonical(distribution)
	if modules[canon] {
		return true
	}
	// A small built-in compatibility table handles common distributions whose
	// published name differs from their import name without adding runtime deps.
	aliases := map[string][]string{
		"beautifulsoup4":  {"bs4"},
		"opencv-python":   {"cv2"},
		"pillow":          {"pil"},
		"pyyaml":          {"yaml"},
		"scikit-learn":    {"sklearn"},
		"python-dateutil": {"dateutil"},
	}
	for _, alias := range aliases[canon] {
		if pythonCanonical(alias) == canon || modules[pythonCanonical(alias)] {
			return true
		}
	}
	return false
}

func pythonCanonical(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.NewReplacer("-", "_", ".", "_", " ", "_").Replace(name)
	return name
}

func collectImports(r *Repo) (importUsage, error) {
	used := importUsage{
		NPM:    map[string]bool{},
		Go:     map[string]bool{},
		Python: map[string]bool{},
		Rust:   map[string]bool{},
	}
	err := filepath.WalkDir(r.Root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if d.IsDir() {
			if path != r.Root && skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !isSourceCandidate(d.Name()) {
			return nil
		}
		b, e := os.ReadFile(path)
		if e != nil || len(b) > 2*1024*1024 {
			return nil
		}
		collectGoImports(path, b, used.Go)
		collectPythonImports(string(b), used.Python)
		collectJSImports(string(b), used.NPM)
		collectRustImports(string(b), used.Rust)
		return nil
	})
	return used, err
}

func isSourceCandidate(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".go", ".rs", ".py", ".js", ".ts", ".tsx", ".jsx", ".java", ".kt", ".c", ".h", ".cpp", ".hpp", ".cs", ".rb", ".php", ".swift", ".scala":
		return true
	}
	return false
}

func collectGoImports(path string, b []byte, used map[string]bool) {
	f, err := parser.ParseFile(token.NewFileSet(), path, b, parser.ImportsOnly)
	if err != nil {
		return
	}
	for _, spec := range f.Imports {
		name, err := strconv.Unquote(spec.Path.Value)
		if err == nil && name != "" {
			used[name] = true
		}
	}
}

func collectPythonImports(s string, used map[string]bool) {
	for _, m := range pythonImportRe.FindAllStringSubmatch(s, -1) {
		parts := strings.Split(m[1], ",")
		for _, part := range parts {
			name := strings.Fields(strings.TrimSpace(part))
			if len(name) == 0 {
				continue
			}
			top := strings.Split(strings.TrimPrefix(name[0], "."), ".")[0]
			if top != "" && !strings.HasPrefix(name[0], ".") {
				used[pythonCanonical(top)] = true
			}
		}
	}
	for _, m := range pythonFromRe.FindAllStringSubmatch(s, -1) {
		name := m[1]
		if strings.HasPrefix(name, ".") {
			continue
		}
		top := strings.Split(name, ".")[0]
		if top != "" {
			used[pythonCanonical(top)] = true
		}
	}
}

func collectJSImports(s string, used map[string]bool) {
	for _, re := range []*regexp.Regexp{jsImportRe, jsDynamicRe, jsRequireRe, jsExportRe} {
		for _, m := range re.FindAllStringSubmatch(s, -1) {
			if len(m) < 2 {
				continue
			}
			if pkg := npmBarePackage(m[1]); pkg != "" {
				used[npmCanonical(pkg)] = true
			}
		}
	}
}

func npmBarePackage(spec string) string {
	if spec == "" || strings.HasPrefix(spec, ".") || strings.HasPrefix(spec, "/") || strings.HasPrefix(spec, "node:") {
		return ""
	}
	parts := strings.Split(spec, "/")
	if strings.HasPrefix(spec, "@") {
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
		return ""
	}
	return parts[0]
}

func collectRustImports(s string, used map[string]bool) {
	for _, re := range []*regexp.Regexp{rustUseRe, rustExternRe} {
		for _, m := range re.FindAllStringSubmatch(s, -1) {
			if len(m) > 1 {
				used[cargoCanonical(m[1])] = true
			}
		}
	}
}
