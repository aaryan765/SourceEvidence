package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type sessionSection struct {
	Title string
	Body  string
	Value any
}

type interactiveSession struct {
	Repo     *Repo
	Sections []sessionSection
}

func usage() {
	fmt.Print(`SourceEvidence - zero-dependency codebase intelligence

Interactive mode:
  sourceevidence

Direct commands:
  sourceevidence scan <repo>
  sourceevidence deps <repo>
  sourceevidence git <repo>
  sourceevidence find <repo> <query>
  sourceevidence hotspot <repo>
  sourceevidence explain <repo> <path>
  sourceevidence overview <repo>
  sourceevidence evidence <repo> [task]

Options:
  --json               Emit machine-readable JSON. Human UI is bypassed.
  --format=markdown    Emit a clean Markdown report.
  --max=N              Limit ranked/search-style results (default 50).
  --help               Show this help.

Design:
  * read-only inspection
  * standard-library only
  * no network access required
  * no Git subprocess required
  * interactive menu with report export
`)
}

func main() {
	if len(os.Args) == 1 {
		interactive()
		return
	}
	if os.Args[1] == "--help" || os.Args[1] == "-h" || os.Args[1] == "help" {
		usage()
		return
	}
	if os.Args[1] == "menu" || os.Args[1] == "interactive" {
		interactive()
		return
	}

	executeCommand(os.Args[1:])
}

func executeCommand(argv []string) {
	if len(argv) == 0 {
		interactive()
		return
	}
	args := append([]string(nil), argv[1:]...)
	jsonOut := false
	markdownOut := false
	maxResults := 50
	filtered := args[:0]
	for _, a := range args {
		if a == "--json" {
			jsonOut = true
			continue
		}
		if a == "--format=markdown" {
			markdownOut = true
			continue
		}
		if strings.HasPrefix(a, "--format=") {
			format := strings.TrimPrefix(a, "--format=")
			if format != "text" && format != "markdown" {
				fatal(fmt.Errorf("unsupported --format value %q", format))
			}
			markdownOut = format == "markdown"
			continue
		}
		if strings.HasPrefix(a, "--max=") {
			if _, err := fmt.Sscanf(strings.TrimPrefix(a, "--max="), "%d", &maxResults); err != nil {
				fatal(fmt.Errorf("invalid --max value %q", a))
			}
			if maxResults < 1 {
				maxResults = 1
			}
			continue
		}
		filtered = append(filtered, a)
	}
	args = filtered

	command := argv[0]
	repo := ""
	if len(args) > 0 {
		repo = args[0]
	}
	if repo == "" {
		repo = "."
	}
	abs, err := filepath.Abs(repo)
	if err != nil {
		fatal(err)
	}
	repoInfo, err := NewRepo(abs)
	if err != nil {
		fatal(err)
	}

	var value any
	switch command {
	case "scan":
		value, err = ScanRepo(repoInfo, maxResults)
	case "deps":
		value, err = AnalyzeDependencies(repoInfo)
	case "git":
		value, err = AnalyzeGit(repoInfo)
	case "find":
		if len(args) < 2 {
			fatal(fmt.Errorf("find requires <repo> <query>"))
		}
		value, err = SearchRepo(repoInfo, args[1], maxResults)
	case "hotspot":
		value, err = Hotspots(repoInfo, maxResults)
	case "explain":
		if len(args) < 2 {
			fatal(fmt.Errorf("explain requires <repo> <path>"))
		}
		value, err = Explain(repoInfo, args[1])
	case "overview":
		value, err = OverviewRepo(repoInfo, maxResults)
	case "evidence":
		task := ""
		if len(args) >= 2 {
			task = strings.TrimSpace(strings.Join(args[1:], " "))
		}
		value, err = BuildEvidencePack(repoInfo, task, maxResults)
	default:
		fatal(fmt.Errorf("unknown command %q", command))
	}
	if err != nil {
		fatal(err)
	}
	if jsonOut && markdownOut {
		fatal(fmt.Errorf("--json and --format=markdown cannot be used together"))
	}
	printValue(command, abs, value, jsonOut, markdownOut)
}

func printValue(command, target string, v any, asJSON, asMarkdown bool) {
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(v); err != nil {
			fatal(err)
		}
		return
	}
	if asMarkdown {
		fmt.Print(markdownReport(command, v))
		return
	}
	if command == "explain" {
		if report, ok := v.(ExplainReport); ok {
			printReport(command, report.Path, report)
			return
		}
	}
	printReport(command, target, v)
}

func interactive() {
	reader := bufio.NewReader(os.Stdin)
	readLine := func(prompt string) string {
		fmt.Print(prompt)
		text, _ := reader.ReadString('\n')
		return strings.TrimSpace(text)
	}

	uiBanner()
	path := readLine("Repository path [.] : ")
	if path == "" {
		path = "."
	}
	repo, err := NewRepoFromInput(path)
	if err != nil {
		fmt.Printf("\nError: %v\n", err)
		return
	}
	session := &interactiveSession{Repo: repo}

	for {
		uiMenu(repo.Root, len(session.Sections))
		choice := strings.ToUpper(readLine("Select [0-9/A/B] : "))
		switch choice {
		case "1":
			scan, err := ScanRepo(repo, 50)
			if err != nil {
				fmt.Printf("\nError: %v\n", err)
			} else {
				printReport("scan", repo.Root, scan)
				session.set("Repository scan", scan)
			}
			uiPause(readLine)
		case "2":
			deps, err := AnalyzeDependencies(repo)
			if err != nil {
				fmt.Printf("\nError: %v\n", err)
			} else {
				printReport("deps", "discovered dependency manifests", deps)
				session.set("Dependency analysis", deps)
			}
			uiPause(readLine)
		case "3":
			git, err := AnalyzeGit(repo)
			if err != nil {
				fmt.Printf("\nError: %v\n", err)
			} else {
				printReport("git", repo.Root, git)
				session.set("Git archaeology", git)
			}
			uiPause(readLine)
		case "4":
			q := readLine("Search query : ")
			if q == "" {
				fmt.Println("Query cannot be empty.")
				uiPause(readLine)
				continue
			}
			search, err := SearchRepo(repo, q, 50)
			if err != nil {
				fmt.Printf("\nError: %v\n", err)
			} else {
				printReport("find", fmt.Sprintf("query %q", q), search)
				session.set("Search: "+q, search)
			}
			uiPause(readLine)
		case "5":
			hotspots, err := Hotspots(repo, 30)
			if err != nil {
				fmt.Printf("\nError: %v\n", err)
			} else {
				printReport("hotspot", "history-derived risk signals", hotspots)
				session.set("Repository hotspots", hotspots)
			}
			uiPause(readLine)
		case "6":
			p := readLine("File path inside repository : ")
			if p == "" {
				fmt.Println("Path cannot be empty.")
				uiPause(readLine)
				continue
			}
			explain, err := Explain(repo, p)
			if err != nil {
				fmt.Printf("\nError: %v\n", err)
			} else {
				printReport("explain", explain.Path, explain)
				session.set("Explain: "+explain.Path, explain)
			}
			uiPause(readLine)
		case "7":
			runFullReport(repo, session)
			uiPause(readLine)
		case "8":
			if err := session.export(); err != nil {
				fmt.Printf("\nError: %v\n", err)
			}
			uiPause(readLine)
		case "9":
			newPath := readLine("New repository path [.] : ")
			if newPath == "" {
				newPath = "."
			}
			newRepo, err := NewRepoFromInput(newPath)
			if err != nil {
				fmt.Printf("\nError: %v\n", err)
				uiPause(readLine)
				continue
			}
			repo = newRepo
			session = &interactiveSession{Repo: repo}
			fmt.Printf("\nRepository changed to %s\n", repo.Root)
			uiPause(readLine)
		case "A":
			overview, err := OverviewRepo(repo, 15)
			if err != nil {
				fmt.Printf("\nError: %v\n", err)
			} else {
				printReport("overview", repo.Root, overview)
				session.set("Codebase overview", overview)
			}
			uiPause(readLine)
		case "B":
			if err := session.exportMarkdown(); err != nil {
				fmt.Printf("\nError: %v\n", err)
			}
			uiPause(readLine)
		case "0":
			fmt.Println("\nSourceEvidence closed.")
			return
		default:
			fmt.Println("\nPlease choose 0-9, A, or B.")
		}
	}
}

func NewRepoFromInput(path string) (*Repo, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	return NewRepo(abs)
}

func runFullReport(repo *Repo, session *interactiveSession) {
	fmt.Printf("\nRunning full report for %s ...\n", repo.Root)
	var scan ScanReport
	var git GitReport
	scanOK, gitOK := false, false
	if value, err := ScanRepo(repo, 50); err != nil {
		fmt.Printf("  Scan: error: %v\n", err)
	} else {
		scan = value
		scanOK = true
		session.set("Repository scan", scan)
		fmt.Println("  [OK] scan")
	}
	if deps, err := AnalyzeDependencies(repo); err != nil {
		fmt.Printf("  Deps: error: %v\n", err)
	} else {
		session.set("Dependency analysis", deps)
		fmt.Println("  [OK] dependencies")
	}
	if value, err := AnalyzeGit(repo); err != nil {
		fmt.Printf("  Git: error: %v\n", err)
	} else {
		git = value
		gitOK = true
		session.set("Git archaeology", git)
		fmt.Println("  [OK] git archaeology")
	}
	if scanOK && gitOK {
		hotspots := buildHotspots(git, scan, 30)
		session.set("Repository hotspots", hotspots)
		fmt.Println("  [OK] hotspots")
	} else {
		fmt.Println("  [SKIP] hotspots (scan and Git data are required)")
	}
	fmt.Printf("\nFull report ready: %d sections. Choose [8] for .txt or [B] for .md.\n", len(session.Sections))
}

func (s *interactiveSession) set(title string, value any) {
	body := reportBody(value)
	for i := range s.Sections {
		if s.Sections[i].Title == title {
			s.Sections[i].Body = body
			return
		}
	}
	s.Sections = append(s.Sections, sessionSection{Title: title, Body: body, Value: value})
}

func (s *interactiveSession) export() error {
	if len(s.Sections) == 0 {
		return fmt.Errorf("no report data yet; run at least one analysis first")
	}
	stamp := time.Now().Format("20060102-150405")
	dir, err := reportDirectory()
	if err != nil {
		return err
	}
	name := "sourceevidence-report-" + safeFilename(filepath.Base(s.Repo.Root)) + "-" + stamp + ".txt"
	path := filepath.Join(dir, name)
	for i := 2; ; i++ {
		if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
			break
		} else if statErr != nil {
			return statErr
		}
		name = fmt.Sprintf("sourceevidence-report-%s-%s-%d.txt", safeFilename(filepath.Base(s.Repo.Root)), stamp, i)
		path = filepath.Join(dir, name)
	}
	var b strings.Builder
	fmt.Fprintln(&b, "SOURCE EVIDENCE REPORT")
	fmt.Fprintln(&b, "=====================")
	fmt.Fprintf(&b, "Repository : %s\n", s.Repo.Root)
	fmt.Fprintf(&b, "Generated  : %s\n", time.Now().Format(time.RFC1123))
	fmt.Fprintln(&b, "Mode       : read-only inspection")
	fmt.Fprintln(&b)
	for i, section := range s.Sections {
		fmt.Fprintf(&b, "[%d] %s\n", i+1, strings.ToUpper(section.Title))
		fmt.Fprintln(&b, strings.Repeat("-", 78))
		fmt.Fprint(&b, section.Body)
		if !strings.HasSuffix(section.Body, "\n") {
			b.WriteByte('\n')
		}
		fmt.Fprintln(&b)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return err
	}
	fmt.Printf("\nReport saved to:\n  %s\n", path)
	return nil
}

func (s *interactiveSession) exportMarkdown() error {
	if len(s.Sections) == 0 {
		return fmt.Errorf("no report data yet; run at least one analysis first")
	}
	stamp := time.Now().Format("20060102-150405")
	dir, err := reportDirectory()
	if err != nil {
		return err
	}
	name := "sourceevidence-report-" + safeFilename(filepath.Base(s.Repo.Root)) + "-" + stamp + ".md"
	path := filepath.Join(dir, name)
	for i := 2; ; i++ {
		if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
			break
		} else if statErr != nil {
			return statErr
		}
		name = fmt.Sprintf("sourceevidence-report-%s-%s-%d.md", safeFilename(filepath.Base(s.Repo.Root)), stamp, i)
		path = filepath.Join(dir, name)
	}

	var b strings.Builder
	fmt.Fprintln(&b, "# SourceEvidence Report")
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "**Repository:** `%s`  \n", s.Repo.Root)
	fmt.Fprintf(&b, "**Generated:** %s  \n", time.Now().Format(time.RFC1123))
	fmt.Fprintln(&b, "**Mode:** read-only inspection")
	fmt.Fprintln(&b)
	for i, section := range s.Sections {
		fmt.Fprintf(&b, "## %d. %s\n\n", i+1, section.Title)
		fmt.Fprint(&b, markdownReport(section.Title, section.Value))
		fmt.Fprintln(&b)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return err
	}
	fmt.Printf("\nMarkdown report saved to:\n  %s\n", path)
	return nil
}

func reportDirectory() (string, error) {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		dir := filepath.Join(home, "SourceEvidence Reports")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
		return dir, nil
	}
	dir := filepath.Join(os.TempDir(), "SourceEvidence Reports")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func safeFilename(s string) string {
	if s == "" || s == "." {
		return "repository"
	}
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "repository"
	}
	return out
}

func fatal(err error) { fmt.Fprintf(os.Stderr, "error: %v\n", err); os.Exit(1) }
