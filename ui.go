package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

const uiWidth = 78

const (
	ansiBrightRed = "\x1b[91m"
	ansiDim       = "\x1b[2m"
	ansiReset     = "\x1b[0m"
)

func uiColor(code, s string) string {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return s
	}
	return code + s + ansiReset
}

func uiLine(ch byte) string { return strings.Repeat(string(ch), uiWidth) }

func uiBanner() {
	fmt.Println()

	red := func(s string) string {
		return uiColor(ansiBrightRed, s)
	}

	fmt.Println(red("  SSSS   OOO   U   U  RRRR    CCCC  EEEEE"))
	fmt.Println(red("  S      O   O  U   U  R   R  C      E"))
	fmt.Println(red("  SSSS   O   O  U   U  RRRR   C      EEE"))
	fmt.Println(red("     S   O   O  U   U  R  R   C      E"))
	fmt.Println(red("  SSSS    OOO    UUU   R   RR  CCCC  EEEEE"))

	fmt.Println()

	fmt.Println(red("  EEEEE  V   V  III  DDDD   EEEEE  N   N  CCCC  EEEEE"))
	fmt.Println(red("  E      V   V   I   D   D  E      NN  N  C      E"))
	fmt.Println(red("  EEE     V V    I   D   D  EEE    N N N  C      EEE"))
	fmt.Println(red("  E        V     I   D   D  E      N  NN  C      E"))
	fmt.Println(red("  EEEEE    V    III  DDDD   EEEEE  N   N  CCCC  EEEEE"))

	fmt.Println()
	fmt.Println(red("                    SOURCE // EVIDENCE"))
	fmt.Println(red("                    ~ aaryan765"))
	fmt.Println()
}

func uiHeader(command, subtitle string) {
	fmt.Println()
	fmt.Println(uiLine('='))
	fmt.Printf("SOURCE EVIDENCE  /  %s\n", strings.ToUpper(command))
	if subtitle != "" {
		fmt.Printf("Target: %s\n", truncate(subtitle, uiWidth-8))
	}
	fmt.Println(uiLine('='))
}

func uiSection(title string) {
	fmt.Printf("\n%s\n", title)
	fmt.Println(uiLine('-'))
}

func uiValue(label string, value any) {
	fmt.Printf("  %-22s %v\n", label, value)
}

func uiMenu(repo string, reportCount int) {
	fmt.Println()
	fmt.Println(uiLine('-'))
	fmt.Println(uiColor(ansiBrightRed, "SOURCE // EVIDENCE ~ aaryan765"))
	fmt.Printf("Repository: %s\n", truncate(repo, uiWidth-12))
	fmt.Printf("Reports ready: %d\n", reportCount)
	fmt.Println(uiLine('-'))
	fmt.Println("Choose an action")
	fmt.Println("  [1]  Scan repository")
	fmt.Println("  [2]  Dependency analysis")
	fmt.Println("  [3]  Git archaeology")
	fmt.Println("  [4]  Search code")
	fmt.Println("  [5]  Find hotspots")
	fmt.Println("  [6]  Explain a file")
	fmt.Println("  [7]  Run full repository report")
	fmt.Println("  [8]  Save session report (.txt)")
	fmt.Println("  [9]  Change repository")
	fmt.Println("  [A]  Codebase overview (at-a-glance map)")
	fmt.Println("  [B]  Save session report (.md)")
	fmt.Println("  [0]  Exit")
	fmt.Println()
}

func uiPause(readLine func(string) string) { _ = readLine("\nPress Enter to return to the menu...") }

func formatBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	value := float64(n)
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	i := 0
	for value >= 1024 && i < len(units)-1 {
		value /= 1024
		i++
	}
	if value >= 100 {
		return fmt.Sprintf("%.0f %s", value, units[i])
	}
	if value >= 10 {
		return fmt.Sprintf("%.1f %s", value, units[i])
	}
	return fmt.Sprintf("%.2f %s", value, units[i])
}

func formatCount(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	s := fmt.Sprintf("%d", n)
	first := len(s) % 3
	if first == 0 {
		first = 3
	}
	out := s[:first]
	for i := first; i < len(s); i += 3 {
		out += "," + s[i:i+3]
	}
	return out
}

func sortedExtensionCounts(m map[string]int) []Count {
	out := make([]Count, 0, len(m))
	for k, v := range m {
		out = append(out, Count{k, v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Name < out[j].Name
		}
		return out[i].Count > out[j].Count
	})
	return out
}

func compactSignals(signals []string) string {
	if len(signals) == 0 {
		return "-"
	}
	return strings.Join(signals, ", ")
}

func reportBody(v any) string {
	switch x := v.(type) {
	case ScanReport:
		return scanBody(x)
	case DependencyReport:
		return dependencyBody(x)
	case GitReport:
		return gitBody(x)
	case SearchReport:
		return searchBody(x)
	case HotspotReport:
		return hotspotBody(x)
	case ExplainReport:
		return explainBody(x)
	case OverviewReport:
		return overviewBody(x)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func printReport(command, subtitle string, v any) {
	uiHeader(command, subtitle)
	fmt.Print(reportBody(v))
}

func scanBody(r ScanReport) string {
	var b strings.Builder
	uiSectionTo(&b, "SUMMARY")
	fmt.Fprintf(&b, "  %-22s %s\n", "Files", formatCount(r.Files))
	fmt.Fprintf(&b, "  %-22s %s\n", "Total size", formatBytes(r.Bytes))
	highRisk, mockTest := secretSignalCounts(r.SecretHits)
	fmt.Fprintf(&b, "  %-22s %d\n", "High-risk secrets", highRisk)
	fmt.Fprintf(&b, "  %-22s %d\n", "Mock/test signals", mockTest)

	ext := sortedExtensionCounts(r.Extensions)
	uiSectionTo(&b, "TOP FILE TYPES")
	limit := 12
	if len(ext) < limit {
		limit = len(ext)
	}
	for i := 0; i < limit; i++ {
		fmt.Fprintf(&b, "  %-24s %5d\n", ext[i].Name, ext[i].Count)
	}
	if len(ext) > limit {
		fmt.Fprintf(&b, "  ... %d more types (use --json for full data)\n", len(ext)-limit)
	}

	uiSectionTo(&b, "LARGEST FILES")
	for _, f := range r.LargeFiles {
		fmt.Fprintf(&b, "  %-50s %10s\n", f.Path, formatBytes(f.Bytes))
	}

	uiSectionTo(&b, "SECRET-PATTERN SIGNALS")
	if len(r.SecretHits) == 0 {
		fmt.Fprintln(&b, "  none detected")
	} else {
		for _, h := range r.SecretHits {
			label := h.Kind
			if h.Kind == "mock-test-secret" {
				label = "mock/test secret"
			}
			fmt.Fprintf(&b, "  %-45s %-18s line %d\n", h.Path, label, h.Line)
		}
	}
	return b.String()
}

func secretSignalCounts(hits []SecretHit) (highRisk, mockTest int) {
	for _, h := range hits {
		if h.Kind == "mock-test-secret" {
			mockTest++
		} else {
			highRisk++
		}
	}
	return highRisk, mockTest
}

func dependencyBody(d DependencyReport) string {
	var b strings.Builder
	uiSectionTo(&b, "MANIFESTS")
	if len(d.Files) == 0 {
		fmt.Fprintln(&b, "  none detected")
	} else {
		for _, f := range d.Files {
			fmt.Fprintf(&b, "  %-10s %4d packages  %s\n", f.Ecosystem, f.Packages, f.Path)
		}
	}

	uiSectionTo(&b, fmt.Sprintf("DEPENDENCIES (%d)", len(d.Dependencies)))
	if len(d.Dependencies) == 0 {
		fmt.Fprintln(&b, "  none detected")
	} else {
		fmt.Fprintf(&b, "  %-10s %-34s %-12s %-10s\n", "SCOPE", "PACKAGE", "VERSION", "STATUS")
		fmt.Fprintf(&b, "  %-10s %-34s %-12s %-10s\n", "----------", "----------------------------------", "------------", "----------")
		for _, x := range d.Dependencies {
			status := "used"
			if !x.Referenced && x.Declared {
				if x.Scope == "runtime" || x.Scope == "" {
					status = "possibly-unused"
				} else {
					status = "not-source-ref"
				}
			}
			if !x.Declared {
				status = "transitive"
			}
			if x.Script {
				status += ", install-script"
			}
			scope := x.Scope
			if scope == "" {
				scope = "runtime"
			}
			fmt.Fprintf(&b, "  %-10s %-34s %-12s %-10s\n", scope, truncate(x.Name, 34), truncate(x.Version, 12), status)
		}
	}

	uiSectionTo(&b, "FINDINGS")
	if len(d.Findings) == 0 {
		fmt.Fprintln(&b, "  none")
	} else {
		for _, f := range d.Findings {
			fmt.Fprintf(&b, "  %-6s %-28s %s\n", strings.ToUpper(f.Severity), f.Title, f.Package)
			if f.Detail != "" {
				fmt.Fprintf(&b, "         %s\n", f.Detail)
			}
		}
	}
	return b.String()
}

func gitBody(r GitReport) string {
	var b strings.Builder
	uiSectionTo(&b, "REPOSITORY")
	fmt.Fprintf(&b, "  %-22s %s\n", "HEAD", r.Head)
	fmt.Fprintf(&b, "  %-22s %s\n", "Commits inspected", formatCount(r.Commits))
	fmt.Fprintf(&b, "  %-22s %s\n", "Refs discovered", formatCount(len(r.Refs)))

	uiSectionTo(&b, "REF SAMPLE")
	limit := len(r.Refs)
	if limit > 15 {
		limit = 15
	}
	for i := 0; i < limit; i++ {
		fmt.Fprintf(&b, "  %-48s %s\n", truncate(r.Refs[i].Name, 48), truncate(r.Refs[i].Object, 40))
	}
	if len(r.Refs) > limit {
		fmt.Fprintf(&b, "  ... %d more refs (use --json for full data)\n", len(r.Refs)-limit)
	}

	uiSectionTo(&b, "TOP AUTHORS")
	for i, x := range r.Authors {
		if i >= 10 {
			break
		}
		fmt.Fprintf(&b, "  %-38s %4d commits\n", truncate(x.Name, 38), x.Count)
	}

	uiSectionTo(&b, "CHANGED-FILE HOTSPOTS")
	for _, x := range r.ChangedFiles {
		fmt.Fprintf(&b, "  %-52s %4d changes\n", truncate(x.Name, 52), x.Count)
	}
	return b.String()
}

func searchBody(r SearchReport) string {
	var b strings.Builder
	uiSectionTo(&b, "SEARCH")
	fmt.Fprintf(&b, "  %-22s %s\n", "Query", fmt.Sprintf("%q", r.Query))
	fmt.Fprintf(&b, "  %-22s %d\n", "Files scanned", r.FilesScanned)
	fmt.Fprintf(&b, "  %-22s %d\n", "Matches returned", len(r.Results))
	uiSectionTo(&b, "MATCHES")
	if len(r.Results) == 0 {
		fmt.Fprintln(&b, "  No matches.")
		return b.String()
	}
	for i, h := range r.Results {
		fmt.Fprintf(&b, "  #%02d  %-45s score %d\n", i+1, fmt.Sprintf("%s:%d", h.Path, h.Line), h.Score)
		fmt.Fprintf(&b, "        %s\n", truncate(h.Text, 104))
	}
	return b.String()
}

func hotspotBody(r HotspotReport) string {
	var b strings.Builder
	uiSectionTo(&b, "REPOSITORY HOTSPOTS")
	fmt.Fprintf(&b, "  History scope: %s\n", r.HistoryScope)
	if len(r.Files) == 0 {
		fmt.Fprintln(&b, "  No history-derived hotspots.")
		return b.String()
	}
	for i, x := range r.Files {
		fmt.Fprintf(&b, "  #%02d  %-48s %4d changes\n", i+1, truncate(x.Path, 48), x.Changes)
		fmt.Fprintf(&b, "        %s\n", compactSignals(x.Signals))
	}
	return b.String()
}

func explainBody(r ExplainReport) string {
	var b strings.Builder
	uiSectionTo(&b, "FILE")
	fmt.Fprintf(&b, "  %-22s %t\n", "Exists", r.Exists)
	fmt.Fprintf(&b, "  %-22s %s\n", "Size", formatBytes(r.Size))
	fmt.Fprintf(&b, "  %-22s %d\n", "Git changes", r.GitChanges)
	fmt.Fprintf(&b, "  %-22s %s\n", "History scope", r.GitHistoryScope)
	uiSectionTo(&b, "DEPENDENCY MENTIONS")
	if len(r.DependencyMentions) == 0 {
		fmt.Fprintln(&b, "  none detected")
	} else {
		for _, x := range r.DependencyMentions {
			fmt.Fprintf(&b, "  %s\n", x)
		}
	}
	uiSectionTo(&b, "CONTENT SIGNALS")
	if len(r.ContentSignals) == 0 {
		fmt.Fprintln(&b, "  none detected")
	} else {
		for _, x := range r.ContentSignals {
			fmt.Fprintf(&b, "  %s\n", x)
		}
	}
	return b.String()
}

func overviewBody(r OverviewReport) string {
	var b strings.Builder
	uiSectionTo(&b, "CODEBASE AT A GLANCE")
	fmt.Fprintf(&b, "  %-22s %s\n", "Repository", r.Repo)
	fmt.Fprintf(&b, "  %-22s %s\n", "HEAD", truncate(r.Head, 40))
	fmt.Fprintf(&b, "  %-22s %s\n", "Files", formatCount(r.Files))
	fmt.Fprintf(&b, "  %-22s %s\n", "Total size", formatBytes(r.Bytes))
	fmt.Fprintf(&b, "  %-22s %s\n", "Directories", formatCount(r.Directories))
	fmt.Fprintf(&b, "  %-22s %s\n", "Root-level files", formatCount(r.RootFiles))

	uiSectionTo(&b, "REPOSITORY MAP")
	for _, area := range r.Areas {
		label := area.Name
		if label == "[root]" {
			label = "."
		}
		fmt.Fprintf(&b, "  %-28s %5d files  %10s\n", label, area.Files, formatBytes(area.Bytes))
	}
	if len(r.Areas) == 0 {
		fmt.Fprintln(&b, "  no files detected")
	}

	uiSectionTo(&b, "HISTORICAL HOTSPOTS")
	if len(r.Hotspots) == 0 {
		fmt.Fprintln(&b, "  none detected")
	} else {
		maxChanges := r.Hotspots[0].Changes
		if maxChanges < 1 {
			maxChanges = 1
		}
		for _, h := range r.Hotspots {
			barLen := h.Changes * 24 / maxChanges
			if barLen < 1 {
				barLen = 1
			}
			bar := strings.Repeat("#", barLen)
			fmt.Fprintf(&b, "  %-42s %3d %s\n", truncate(h.Path, 42), h.Changes, bar)
		}
	}

	uiSectionTo(&b, "LARGEST CURRENT FILES")
	if len(r.LargestFiles) == 0 {
		fmt.Fprintln(&b, "  none detected")
	} else {
		for _, f := range r.LargestFiles {
			fmt.Fprintf(&b, "  %-50s %10s\n", truncate(f.Path, 50), formatBytes(f.Bytes))
		}
	}
	return b.String()
}

func uiSectionTo(b *strings.Builder, title string) {
	fmt.Fprintf(b, "\n%s\n", title)
	fmt.Fprintln(b, uiLine('-'))
}

func truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func markdownReport(command string, v any) string {
	var b strings.Builder
	fmt.Fprintf(&b, "### %s\n\n", strings.Title(command))
	switch r := v.(type) {
	case ScanReport:
		fmt.Fprintf(&b, "| Metric | Value |\n|---|---:|\n| Files | %s |\n| Total size | %s |\n| Security signals | %d |\n\n", formatCount(r.Files), formatBytes(r.Bytes), len(r.SecretHits))
		fmt.Fprintln(&b, "#### Largest files")
		fmt.Fprintln(&b, "| File | Size |\n|---|---:|")
		for _, f := range r.LargeFiles {
			fmt.Fprintf(&b, "| `%s` | %s |\n", f.Path, formatBytes(f.Bytes))
		}
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, "#### Security signals")
		if len(r.SecretHits) == 0 {
			fmt.Fprintln(&b, "None detected.")
		} else {
			fmt.Fprintln(&b, "| File | Line | Type |\n|---|---:|---|")
			for _, h := range r.SecretHits {
				fmt.Fprintf(&b, "| `%s` | %d | `%s` |\n", h.Path, h.Line, h.Kind)
			}
		}
	case DependencyReport:
		fmt.Fprintln(&b, "| Scope | Package | Version | Status |\n|---|---|---|---|")
		for _, d := range r.Dependencies {
			status := "used"
			if !d.Referenced && d.Declared {
				if d.Scope == "runtime" || d.Scope == "" {
					status = "possibly-unused"
				} else {
					status = "not-source-ref"
				}
			}
			if !d.Declared {
				status = "transitive"
			}
			fmt.Fprintf(&b, "| %s | `%s` | `%s` | %s |\n", d.Scope, d.Name, d.Version, status)
		}
		if len(r.Findings) > 0 {
			fmt.Fprintln(&b, "\n#### Findings")
			for _, f := range r.Findings {
				fmt.Fprintf(&b, "- **%s:** %s (%s)\n", f.Severity, f.Title, f.Package)
			}
		}
	case GitReport:
		fmt.Fprintf(&b, "- **HEAD:** `%s`\n- **Commits inspected:** %d\n- **Refs discovered:** %d\n- **History scope:** %s\n\n", r.Head, r.Commits, len(r.Refs), gitHistoryScopeLabel)
		fmt.Fprintln(&b, "#### Authors")
		fmt.Fprintln(&b, "| Author | Commits |\n|---|---:|")
		for _, a := range r.Authors {
			fmt.Fprintf(&b, "| %s | %d |\n", a.Name, a.Count)
		}
		fmt.Fprintln(&b, "\n#### Changed-file hotspots")
		fmt.Fprintln(&b, "| File | Changes |\n|---|---:|")
		for _, c := range r.ChangedFiles {
			fmt.Fprintf(&b, "| `%s` | %d |\n", c.Name, c.Count)
		}
	case SearchReport:
		fmt.Fprintf(&b, "Query: `%s`  \nFiles scanned: %d  \nMatches returned: %d\n\n", r.Query, r.FilesScanned, len(r.Results))
		fmt.Fprintln(&b, "| File | Line | Score | Match |\n|---|---:|---:|---|")
		for _, h := range r.Results {
			fmt.Fprintf(&b, "| `%s` | %d | %d | %s |\n", h.Path, h.Line, h.Score, strings.ReplaceAll(h.Text, "|", "\\|"))
		}
	case HotspotReport:
		fmt.Fprintf(&b, "**History scope:** %s\n\n", r.HistoryScope)
		fmt.Fprintln(&b, "| File | Changes | Signals |\n|---|---:|---|")
		for _, h := range r.Files {
			fmt.Fprintf(&b, "| `%s` | %d | %s |\n", h.Path, h.Changes, compactSignals(h.Signals))
		}
	case ExplainReport:
		fmt.Fprintf(&b, "| Metric | Value |\n|---|---|\n| Exists | %t |\n| Size | %s |\n| Git changes | %d |\n| History scope | %s |\n\n", r.Exists, formatBytes(r.Size), r.GitChanges, r.GitHistoryScope)
		fmt.Fprintln(&b, "#### Dependency mentions")
		if len(r.DependencyMentions) == 0 {
			fmt.Fprintln(&b, "None detected.")
		} else {
			for _, x := range r.DependencyMentions {
				fmt.Fprintf(&b, "- `%s`\n", x)
			}
		}
		fmt.Fprintln(&b, "\n#### Content signals")
		if len(r.ContentSignals) == 0 {
			fmt.Fprintln(&b, "None detected.")
		} else {
			for _, x := range r.ContentSignals {
				fmt.Fprintf(&b, "- %s\n", x)
			}
		}
	case OverviewReport:
		fmt.Fprintf(&b, "| Metric | Value |\n|---|---|\n| Files | %s |\n| Total size | %s |\n| Directories | %s |\n| Root-level files | %s |\n| HEAD | `%s` |\n\n", formatCount(r.Files), formatBytes(r.Bytes), formatCount(r.Directories), formatCount(r.RootFiles), r.Head)
		fmt.Fprintln(&b, "#### Repository map")
		fmt.Fprintln(&b, "| Area | Files | Size |\n|---|---:|---:|")
		for _, a := range r.Areas {
			name := a.Name
			if name == "[root]" {
				name = "."
			}
			fmt.Fprintf(&b, "| `%s` | %d | %s |\n", name, a.Files, formatBytes(a.Bytes))
		}
		fmt.Fprintln(&b, "\n#### Historical hotspots")
		fmt.Fprintln(&b, "| File | Changes | Current size |\n|---|---:|---:|")
		for _, h := range r.Hotspots {
			fmt.Fprintf(&b, "| `%s` | %d | %s |\n", h.Path, h.Changes, formatBytes(h.Bytes))
		}
		fmt.Fprintln(&b, "\n#### Largest current files")
		fmt.Fprintln(&b, "| File | Size |\n|---|---:|")
		for _, f := range r.LargestFiles {
			fmt.Fprintf(&b, "| `%s` | %s |\n", f.Path, formatBytes(f.Bytes))
		}
	default:
		fmt.Fprintln(&b, "```text")
		fmt.Fprint(&b, reportBody(v))
		fmt.Fprintln(&b, "```")
	}
	return b.String()
}
