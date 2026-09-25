# SourceEvidence

**SourceEvidence** is zero-dependency codebase intelligence for developers who need evidence before they change a repository.

It stays local, read-only, and standard-library-only while combining repository structure, dependencies, Git history, search, and file-level investigation.

## Ready-to-run binaries

The release bundle includes:

- `sourceevidence.exe` for Windows x86-64.
- `bin/sourceevidence-linux-amd64` for Linux x86-64.
- `bin/sourceevidence-darwin-arm64` for Apple Silicon macOS.

On Windows, double-clicking the executable is not required; run `sourceevidence.exe` from a terminal and the interactive menu opens automatically.

## Simple interactive mode

Run the executable with no command:

```text
sourceevidence
```

You get a small menu:

```text
[1] Scan repository
[2] Dependency analysis
[3] Git archaeology
[4] Search code
[5] Find hotspots
[6] Explain a file
[7] Run full repository report
[A] Codebase overview (at-a-glance map)
[B] Save session report (.md)
[8] Save session report (.txt)
[9] Change repository
[0] Exit
```

The interactive mode remembers the inspected repository and lets you build a report without repeatedly typing long commands. A compact red branding banner is shown at startup; subsequent menu screens use a compact author header to keep the workflow clean.

## Direct commands

The original command interface remains available for scripts and automation:

```text
sourceevidence scan <repo>
sourceevidence deps <repo>
sourceevidence git <repo>
sourceevidence find <repo> <query>
sourceevidence hotspot <repo>
sourceevidence explain <repo> <path>
sourceevidence overview <repo>
sourceevidence evidence <repo> [task]
```

Add `--json` for machine-readable output, `--format=markdown` for Markdown output, and `--max=N` to limit ranked/search results. The `evidence` command bundles scan, dependency, Git, and hotspot evidence into one compact JSON-ready package for AI-agent and automation workflows.

## Report export

Option **[8] Save session report (.txt)** writes a clean, timestamped text report to a per-user `SourceEvidence Reports` directory (for example, `%USERPROFILE%\SourceEvidence Reports` on Windows). The report can include scan, dependency, Git, hotspot, search, and explain sections gathered during the session. This default location is outside the inspected repository.

Option **[B] Save session report (.md)** writes the same session evidence as structured Markdown tables, suitable for GitHub Actions artifacts, pull-request workflows, issues, and other developer tooling.

Option **[7] Run full repository report** runs the core repository analyses in one pass and prepares them for export.

The report writer never writes inside the inspected repository.

## Evidence-driven AI workflow

SourceEvidence does not embed an LLM. The `evidence` command creates a compact JSON bundle of deterministic repository facts for an AI development agent such as IBM Bob to investigate. The boundary is explicit: SourceEvidence gathers evidence; the agent reasons and acts; the developer approves the change.

```text
Developer task
      ↓
SourceEvidence evidence --json
      ↓
Evidence Pack
      ↓
IBM Bob / AI agent
      ↓
investigate → plan → implement → test
```

## What it analyzes

- Repository scan: files, sizes, file types, largest files, and conservative secret-pattern signals.
- Dependencies: npm lockfiles, Go modules, Cargo lockfiles, Python requirements files, and common `pyproject.toml` dependency declarations. Duplicate declarations within the same ecosystem/scope are merged in the report, and renamed Cargo dependencies are matched to their source import names.
- Git archaeology: refs, bounded commit history, authors, and changed-file hotspots, reading `.git` directly rather than invoking Git. Hotspots and file-level **Explain** now share the same bounded history traversal and explicitly report the history scope.
- Search: ranked case-insensitive content search.
- Explain: a file's size, Git churn, actual import-based dependency mentions, and high-confidence content signals.
- Hotspots: combines Git churn with repository signals.
- Codebase overview: a deterministic, at-a-glance map of top-level areas, current file sizes, largest files, and Git-derived hotspots.

Dependency scopes distinguish runtime, optional, docs, and dev/test declarations where the manifest makes that distinction visible. Requirements-file comment sections such as Documentation, Packaging/Build, Tests, and Linting are also used when present. "Possibly unused" is only reported for direct runtime dependencies; generated, dynamic, documentation, and development usage can still require human review.

## Safety model

SourceEvidence is intentionally read-only. It does not create, modify, delete, or rewrite files under the inspected repository. It reads source files, manifests, and `.git` objects only.

Secret detection is deliberately conservative. A match is a **signal**, not proof that a credential is valid or leaked. Test/fixture paths with low-entropy human-readable credentials are classified as `mock-test-secret` instead of being treated as high-risk secrets; high-entropy values remain visible.

Repository path handling rejects traversal and symlink escapes for file-focused operations.

## Zero dependency

The project uses only the Go standard library. No third-party runtime packages are required. No network access is required. No `git` subprocess is required for analysis.

## Tests

```bash
go test ./...
go test -race ./...
go vet ./...
go build -trimpath -o sourceevidence .
```

## Demo

The `demo_repo/` directory is a small Git repository with dependency manifests, source files, configuration, and multiple commits. It exists so the tool can be demonstrated without network access.

See `DEMO.md` for a quick walkthrough.
