# SourceEvidence demo

## Interactive mode

From the SourceEvidence directory:

```bash
./sourceevidence
```

Enter `demo_repo` as the repository path, then try:

```text
[1] Scan repository
[2] Dependency analysis
[3] Git archaeology
[4] Search code
[5] Find hotspots
[6] Explain a file
[7] Run full repository report
[8] Save session report (.txt)
```

Option 8 writes a clean, timestamped `.txt` report outside the inspected repository.

## Direct command mode

```bash
# Build
GO111MODULE=on go build -trimpath -o sourceevidence .

# Scan the included demo repository
./sourceevidence scan demo_repo
./sourceevidence deps demo_repo
./sourceevidence git demo_repo
./sourceevidence find demo_repo "API_KEY"
./sourceevidence hotspot demo_repo
./sourceevidence explain demo_repo src/auth.go

# JSON output
./sourceevidence scan demo_repo --json
```


## New workflow features

- **A — Codebase overview** gives a fast map of repository areas, sizes, Git hotspots, and largest files.
- **B — Markdown export** writes the current session as structured Markdown for CI, GitHub issues, artifacts, and review workflows.
- **Evidence Pack** bundles deterministic scan, dependency, Git, and hotspot evidence for AI-agent workflows:

```text
sourceevidence evidence ./demo_repo "Investigate authentication failure" --json
```
