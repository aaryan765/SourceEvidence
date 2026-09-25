package main

// EvidencePack is a compact, machine-readable bundle of deterministic
// repository evidence intended for AI agents, code-review tooling, or audit
// workflows. It contains facts produced by SourceEvidence; it does not infer
// a root cause or generate a proposed fix.
type EvidencePack struct {
	SchemaVersion string           `json:"schema_version"`
	Task          string           `json:"task,omitempty"`
	Repo          string           `json:"repo"`
	Scan          ScanReport       `json:"scan"`
	Dependencies  DependencyReport `json:"dependencies"`
	Git           GitReport        `json:"git"`
	Hotspots      HotspotReport    `json:"hotspots"`
}

func BuildEvidencePack(r *Repo, task string, max int) (EvidencePack, error) {
	if max < 1 {
		max = 1
	}
	scan, err := ScanRepo(r, max)
	if err != nil {
		return EvidencePack{}, err
	}
	deps, err := AnalyzeDependencies(r)
	if err != nil {
		return EvidencePack{}, err
	}
	git, err := AnalyzeGit(r)
	if err != nil {
		return EvidencePack{}, err
	}
	return EvidencePack{
		SchemaVersion: "1",
		Task:          task,
		Repo:          r.Root,
		Scan:          scan,
		Dependencies:  deps,
		Git:           git,
		Hotspots:      buildHotspots(git, scan, max),
	}, nil
}
