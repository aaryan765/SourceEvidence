package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// OverviewReport is a deterministic, read-only "codebase at a glance" view.
// It combines repository shape, current file sizes, and the bounded Git churn
// information SourceEvidence already computes. No network or external tools are used.
type OverviewReport struct {
	Repo         string         `json:"repo"`
	Head         string         `json:"head"`
	Files        int            `json:"files"`
	Bytes        int64          `json:"bytes"`
	Directories  int            `json:"directories"`
	RootFiles    int            `json:"root_files"`
	Areas        []OverviewArea `json:"areas"`
	Hotspots     []OverviewFile `json:"hotspots"`
	LargestFiles []FileStat     `json:"largest_files"`
}

type OverviewArea struct {
	Name  string `json:"name"`
	Files int    `json:"files"`
	Bytes int64  `json:"bytes"`
}

type OverviewFile struct {
	Path    string `json:"path"`
	Bytes   int64  `json:"bytes"`
	Changes int    `json:"changes"`
}

// OverviewRepo builds a lightweight repository map from existing SourceEvidence
// primitives. The analysis remains read-only and stays inside the repository.
func OverviewRepo(r *Repo, max int) (OverviewReport, error) {
	if max < 1 {
		max = 1
	}

	scan, err := ScanRepo(r, max)
	if err != nil {
		return OverviewReport{}, err
	}
	git, err := AnalyzeGit(r)
	if err != nil {
		return OverviewReport{}, err
	}

	out := OverviewReport{
		Repo:         r.Root,
		Head:         git.Head,
		Files:        scan.Files,
		Bytes:        scan.Bytes,
		LargestFiles: append([]FileStat(nil), scan.LargeFiles...),
	}

	areas := map[string]*OverviewArea{}
	rootPath := filepath.Clean(r.Root)
	err = filepath.WalkDir(r.Root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if d.IsDir() {
			if path != rootPath {
				if skipDirs[d.Name()] {
					return filepath.SkipDir
				}
				out.Directories++
			}
			return nil
		}

		info, statErr := d.Info()
		if statErr != nil {
			return nil
		}
		rel := safeRel(r.Root, path)
		area := topLevelArea(rel)
		bucket := areas[area]
		if bucket == nil {
			bucket = &OverviewArea{Name: area}
			areas[area] = bucket
		}
		bucket.Files++
		bucket.Bytes += info.Size()
		if area == "[root]" {
			out.RootFiles++
		}
		return nil
	})
	if err != nil {
		return OverviewReport{}, err
	}

	out.Areas = make([]OverviewArea, 0, len(areas))
	for _, area := range areas {
		out.Areas = append(out.Areas, *area)
	}
	sort.Slice(out.Areas, func(i, j int) bool {
		if out.Areas[i].Files == out.Areas[j].Files {
			if out.Areas[i].Bytes == out.Areas[j].Bytes {
				return out.Areas[i].Name < out.Areas[j].Name
			}
			return out.Areas[i].Bytes > out.Areas[j].Bytes
		}
		return out.Areas[i].Files > out.Areas[j].Files
	})

	sizes := make(map[string]int64, len(scan.LargeFiles))
	for _, f := range scan.LargeFiles {
		sizes[filepath.ToSlash(f.Path)] = f.Bytes
	}
	// ScanRepo only keeps the 10 largest files, so resolve hotspot sizes directly
	// rather than assuming every hotspot is present in that limited list.
	for _, h := range git.ChangedFiles {
		if len(out.Hotspots) >= max {
			break
		}
		path := filepath.ToSlash(h.Name)
		if size, ok := sizes[path]; ok {
			out.Hotspots = append(out.Hotspots, OverviewFile{Path: path, Bytes: size, Changes: h.Count})
			continue
		}
		if size, ok := currentFileSize(r.Root, path); ok {
			out.Hotspots = append(out.Hotspots, OverviewFile{Path: path, Bytes: size, Changes: h.Count})
			continue
		}
		// Historical Git paths may no longer exist in the working tree. Keep the
		// churn signal, but report zero current bytes rather than inventing a size.
		out.Hotspots = append(out.Hotspots, OverviewFile{Path: path, Changes: h.Count})
	}
	return out, nil
}

func topLevelArea(rel string) string {
	rel = filepath.ToSlash(filepath.Clean(rel))
	if rel == "." || rel == "" {
		return "[root]"
	}
	if i := strings.IndexByte(rel, '/'); i >= 0 {
		return rel[:i] + "/"
	}
	return "[root]"
}

func currentFileSize(root, rel string) (int64, bool) {
	full := filepath.Join(root, filepath.FromSlash(rel))
	st, err := os.Stat(full)
	if err != nil || !st.Mode().IsRegular() {
		return 0, false
	}
	return st.Size(), true
}

func overviewSummary(r OverviewReport) string {
	return fmt.Sprintf("%s files, %s", formatCount(r.Files), formatBytes(r.Bytes))
}
