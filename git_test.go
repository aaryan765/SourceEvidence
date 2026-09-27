package main

import (
	"testing"
)

func TestPackedGitHistory(t *testing.T) {
	demoRepo := prepareDemoRepo(t)
	r, err := NewRepo(demoRepo)
	if err != nil {
		t.Fatal(err)
	}

	g := NewGitStore(r)
	if err := g.indexPacks(); err != nil {
		t.Fatal(err)
	}

	id, _ := hexTo20("6766db9ff173ca6fd12688f7cba9725fa113793d")
	obj, err := g.object(id)
	if err != nil {
		t.Fatal(err)
	}

	if obj.typ != "commit" {
		t.Fatalf("want commit, got %q", obj.typ)
	}

	gr, err := AnalyzeGit(r)
	if err != nil {
		t.Fatal(err)
	}

	if gr.Commits != 3 {
		t.Fatalf("want 3 commits, got %d", gr.Commits)
	}

	if len(gr.ChangedFiles) == 0 {
		t.Fatal("want changed-file hotspots")
	}
}