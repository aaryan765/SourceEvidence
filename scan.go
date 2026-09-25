package main

import (
	"bufio"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type ScanReport struct {
	Repo       string         `json:"repo"`
	Files      int            `json:"files"`
	Bytes      int64          `json:"bytes"`
	Extensions map[string]int `json:"extensions"`
	LargeFiles []FileStat     `json:"large_files"`
	SecretHits []SecretHit    `json:"secret_hits"`
}

type FileStat struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
}
type SecretHit struct {
	Path string `json:"path"`
	Line int    `json:"line"`
	Kind string `json:"kind"`
}

var skipDirs = map[string]bool{".git": true, ".hg": true, ".svn": true, "node_modules": true, "vendor": true, "target": true, "dist": true, "build": true, ".venv": true, "venv": true, "__pycache__": true}

func ScanRepo(r *Repo, max int) (ScanReport, error) {
	out := ScanReport{Repo: r.Root, Extensions: map[string]int{}}
	var largest []FileStat
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
		info, e := d.Info()
		if e != nil {
			return nil
		}
		out.Files++
		out.Bytes += info.Size()
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext == "" {
			ext = "[none]"
		}
		out.Extensions[ext]++
		largest = append(largest, FileStat{safeRel(r.Root, path), info.Size()})
		if info.Size() <= 2*1024*1024 && isTextCandidate(d.Name()) {
			hits := detectSecrets(r.Root, path)
			out.SecretHits = append(out.SecretHits, hits...)
			if len(out.SecretHits) > max {
				out.SecretHits = out.SecretHits[:max]
			}
		}
		return nil
	})
	sort.Slice(largest, func(i, j int) bool { return largest[i].Bytes > largest[j].Bytes })
	if len(largest) > 10 {
		largest = largest[:10]
	}
	out.LargeFiles = largest
	return out, err
}

func isTextCandidate(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".go", ".rs", ".py", ".js", ".ts", ".tsx", ".jsx", ".java", ".kt", ".c", ".h", ".cpp", ".hpp", ".cs", ".rb", ".php", ".swift", ".scala", ".sh", ".yaml", ".yml", ".json", ".toml", ".xml", ".ini", ".env", ".txt", ".md", ".sql":
		return true
	}
	return strings.HasPrefix(name, ".env") || name == "Dockerfile" || ext == ""
}

var secretRules = []struct {
	kind string
	re   *regexp.Regexp
}{
	{"private-key", regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH |DSA )?PRIVATE KEY-----`)},
	{"aws-access-key", regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)},
	{"github-token", regexp.MustCompile(`\b(?:ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9_]{20,}\b|\bgithub_pat_[A-Za-z0-9_]{20,}\b`)},
	{"generic-secret", regexp.MustCompile(`(?i)\b(?:password|secret|api[_-]?key)\b\s*[:=]\s*["']([A-Za-z0-9+/_=-]{12,})["']`)},
	{"generic-token", regexp.MustCompile(`(?i)\btoken\b\s*[:=]\s*["']([A-Za-z0-9+/_=-]{12,})["']`)},
}

func detectSecrets(root, path string) []SecretHit {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var hits []SecretHit
	s := bufio.NewScanner(io.LimitReader(f, 2*1024*1024))
	rel := safeRel(root, path)
	testLike := isTestFixturePath(rel)
	line := 0
	for s.Scan() {
		line++
		text := s.Text()
		for _, rule := range secretRules {
			if !rule.re.MatchString(text) {
				continue
			}
			kind := rule.kind
			// Test suites frequently contain deliberate example credentials. Keep
			// the signal visible, but downgrade low-entropy, human-readable values
			// in test/fixture paths instead of treating them like real secrets.
			if testLike && (kind == "generic-secret" || kind == "generic-token") {
				value := extractSecretValue(kind, text)
				if value != "" && (len(value) < 24 || shannonEntropy(value) < 3.8) {
					kind = "mock-test-secret"
				}
			}
			hits = append(hits, SecretHit{rel, line, kind})
			break
		}
		if len(hits) >= 10 {
			break
		}
	}
	return hits
}

func isTestFixturePath(rel string) bool {
	low := strings.ToLower(strings.ReplaceAll(rel, "\\", "/"))
	parts := strings.Split(low, "/")
	for _, part := range parts[:maxInt(0, len(parts)-1)] {
		if part == "test" || part == "tests" || part == "fixture" || part == "fixtures" {
			return true
		}
	}
	base := parts[len(parts)-1]
	return strings.HasPrefix(base, "test_") || strings.HasPrefix(base, "test-") ||
		strings.HasSuffix(base, "_test.go") || strings.HasSuffix(base, "_test.py") ||
		strings.HasSuffix(base, ".test.js") || strings.HasSuffix(base, ".test.ts") ||
		strings.Contains(base, ".spec.")
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func extractSecretValue(kind, text string) string {
	var re *regexp.Regexp
	switch kind {
	case "generic-secret":
		re = regexp.MustCompile(`(?i)\b(?:password|secret|api[_-]?key)\b\s*[:=]\s*["']([A-Za-z0-9+/_=-]{12,})["']`)
	case "generic-token":
		re = regexp.MustCompile(`(?i)\btoken\b\s*[:=]\s*["']([A-Za-z0-9+/_=-]{12,})["']`)
	default:
		return ""
	}
	m := re.FindStringSubmatch(text)
	if len(m) == 2 {
		return m[1]
	}
	return ""
}

func shannonEntropy(s string) float64 {
	if len(s) == 0 {
		return 0
	}
	counts := map[rune]int{}
	for _, r := range s {
		counts[r]++
	}
	n := float64(len([]rune(s)))
	var entropy float64
	for _, count := range counts {
		p := float64(count) / n
		entropy -= p * math.Log2(p)
	}
	return entropy
}
