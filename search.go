package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type SearchReport struct {
	Query        string      `json:"query"`
	Results      []SearchHit `json:"results"`
	FilesScanned int         `json:"files_scanned"`
}
type SearchHit struct {
	Path  string `json:"path"`
	Line  int    `json:"line"`
	Text  string `json:"text"`
	Score int    `json:"score"`
}

func SearchRepo(r *Repo, query string, max int) (SearchReport, error) {
	out := SearchReport{Query: query}
	q := strings.ToLower(query)
	if q == "" {
		return out, nil
	}
	var hits []SearchHit
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
		if !isTextCandidate(d.Name()) {
			return nil
		}
		info, e := d.Info()
		if e != nil || info.Size() > 4*1024*1024 {
			return nil
		}
		out.FilesScanned++
		f, e := os.Open(path)
		if e != nil {
			return nil
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		line := 0
		for sc.Scan() {
			line++
			txt := sc.Text()
			low := strings.ToLower(txt)
			idx := strings.Index(low, q)
			if idx < 0 {
				continue
			}
			score := 1 + strings.Count(low, q)*2
			base := strings.ToLower(filepath.Base(path))
			if strings.Contains(base, q) {
				score += 5
			}
			hits = append(hits, SearchHit{safeRel(r.Root, path), line, strings.TrimSpace(txt), score})
			if len(hits) > max*5 {
				break
			}
		}
		return nil
	})
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			if hits[i].Path == hits[j].Path {
				return hits[i].Line < hits[j].Line
			}
			return hits[i].Path < hits[j].Path
		}
		return hits[i].Score > hits[j].Score
	})
	if len(hits) > max {
		hits = hits[:max]
	}
	out.Results = hits
	return out, err
}

type HotspotReport struct {
	Files        []Hotspot `json:"files"`
	HistoryScope string    `json:"history_scope,omitempty"`
}

type Hotspot struct {
	Path    string   `json:"path"`
	Changes int      `json:"changes"`
	Signals []string `json:"signals"`
}

func Hotspots(r *Repo, max int) (HotspotReport, error) {
	gr, err := AnalyzeGit(r)
	if err != nil {
		return HotspotReport{}, err
	}
	sr, err := ScanRepo(r, max)
	if err != nil {
		return HotspotReport{}, err
	}
	return buildHotspots(gr, sr, max), nil
}

func buildHotspots(gr GitReport, sr ScanReport, max int) HotspotReport {
	counts := map[string]int{}
	for _, c := range gr.ChangedFiles {
		counts[c.Name] = c.Count
	}
	var out []Hotspot
	for p, c := range counts {
		sig := []string{"high git churn"}
		for _, h := range sr.SecretHits {
			if h.Path != p {
				continue
			}
			if h.Kind == "mock-test-secret" {
				sig = append(sig, "mock/test secret signal")
			} else {
				sig = append(sig, "secret-pattern hit")
			}
		}
		if strings.HasSuffix(strings.ToLower(p), ".env") || strings.Contains(strings.ToLower(p), "config") {
			sig = append(sig, "configuration path")
		}
		out = append(out, Hotspot{p, c, sig})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Changes == out[j].Changes {
			return out[i].Path < out[j].Path
		}
		return out[i].Changes > out[j].Changes
	})
	if len(out) > max {
		out = out[:max]
	}
	return HotspotReport{Files: out, HistoryScope: gitHistoryScopeLabel}
}

type ExplainReport struct {
	Path               string   `json:"path"`
	Exists             bool     `json:"exists"`
	Size               int64    `json:"size"`
	GitChanges         int      `json:"git_changes"`
	GitHistoryScope    string   `json:"git_history_scope,omitempty"`
	DependencyMentions []string `json:"dependency_mentions"`
	ContentSignals     []string `json:"content_signals"`
}

func Explain(r *Repo, path string) (ExplainReport, error) {
	rel, full, err := resolveRepoPath(r, path)
	if err != nil {
		return ExplainReport{}, err
	}
	st, e := os.Stat(full)
	out := ExplainReport{Path: rel, GitHistoryScope: gitHistoryScopeLabel}
	if e == nil {
		out.Exists = true
		out.Size = st.Size()
	}
	gitChanges, e := GitChangesForPath(r, rel)
	if e != nil {
		return out, e
	}
	out.GitChanges = gitChanges
	b, e := os.ReadFile(full)
	if e == nil && len(b) <= 4*1024*1024 {
		for _, hit := range detectSecrets(r.Root, full) {
			out.ContentSignals = append(out.ContentSignals, fmt.Sprintf("%s at line %d", hit.Kind, hit.Line))
		}
	}
	dr, e := AnalyzeDependencies(r)
	if e != nil {
		return out, e
	}
	out.DependencyMentions = fileDependencyMentions(rel, b, dr.Dependencies)
	return out, nil
}

func fileDependencyMentions(rel string, b []byte, deps []Dependency) []string {
	usage := importUsage{NPM: map[string]bool{}, Go: map[string]bool{}, Python: map[string]bool{}, Rust: map[string]bool{}}
	name := filepath.Base(rel)
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".go":
		collectGoImports(rel, b, usage.Go)
	case ".py":
		collectPythonImports(string(b), usage.Python)
	case ".js", ".ts", ".tsx", ".jsx":
		collectJSImports(string(b), usage.NPM)
	case ".rs":
		collectRustImports(string(b), usage.Rust)
	}
	seen := map[string]bool{}
	out := make([]string, 0)
	for _, d := range deps {
		matched := false
		switch d.Ecosystem {
		case "go":
			matched = goReferenced(usage.Go, d.Name)
		case "python":
			matched = pythonReferenced(usage.Python, d.Name)
		case "npm":
			matched = npmReferenced(usage.NPM, d.Name)
		case "rust":
			refs := append([]string{d.Name}, d.referenceNames...)
			for _, ref := range uniqueStrings(refs) {
				if usage.Rust[cargoCanonical(ref)] {
					matched = true
					break
				}
			}
		}
		if matched && !seen[d.Ecosystem+":"+d.Name] {
			seen[d.Ecosystem+":"+d.Name] = true
			out = append(out, d.Name)
		}
	}
	sort.Strings(out)
	return out
}
