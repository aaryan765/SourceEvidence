# SourceEvidence

## Evidence-Driven Debugging for Real Git Repositories

**SourceEvidence** is a deterministic, offline repository-intelligence tool for developers who need reliable evidence before changing a codebase.

It analyzes repository structure, dependencies, Git history, search results, hotspots, and file-level signals, then packages those findings into an **Evidence Pack** that can be used with **IBM Bob 2.0** during debugging and application-maintenance workflows.

### Core workflow

```text
Developer problem
      |
      v
SourceEvidence
      |
      v
Deterministic Evidence Pack
      |
      v
IBM Bob 2.0
      |
      v
Investigate -> Plan -> Implement -> Test
      |
      v
Developer review
```

**SourceEvidence provides the evidence.
IBM Bob 2.0 provides repository-aware reasoning and development actions.
The developer remains the reviewer.**

---

## Why SourceEvidence?

Debugging an unfamiliar repository often begins with repetitive investigation:

- understanding the repository structure;
- locating relevant files;
- inspecting dependencies;
- searching for related code;
- examining Git history;
- identifying frequently changed areas;
- collecting enough context before making a safe change.

SourceEvidence turns that evidence-gathering step into a repeatable, deterministic workflow.

It does **not** embed an LLM and does not attempt to replace the reasoning performed by IBM Bob 2.0 or the developer.

---

# Features

### Repository Scan

Analyzes:

- file counts and repository size;
- file-type distribution;
- largest files;
- conservative secret-pattern signals.

Interactive:

```text
[1] Scan repository
```

CLI:

```bash
sourceevidence scan <repo>
```

JSON:

```bash
sourceevidence scan <repo> --json
```

---

### Dependency Analysis

Analyzes supported dependency manifests and reports:

- declared dependencies;
- versions;
- dependency scope;
- source references;
- possible unused direct runtime dependencies.

Supported ecosystems include Go, Python, npm, and Rust/Cargo-related manifests.

Interactive:

```text
[2] Dependency analysis
```

CLI:

```bash
sourceevidence deps <repo>
```

---

### Git Archaeology

Analyzes repository history and metadata, including:

- current HEAD;
- discovered refs;
- bounded commit history;
- authors;
- changed-file counts.

Git analysis reads repository Git data directly rather than invoking a `git` subprocess.

Interactive:

```text
[3] Git archaeology
```

CLI:

```bash
sourceevidence git <repo>
```

History traversal is bounded to discovered refs and up to 500 commits.

---

### Search Code

Performs ranked, case-insensitive content search.

Results include:

- file path;
- line number;
- matching source line;
- relevance score.

Interactive:

```text
[4] Search code
```

CLI:

```bash
sourceevidence find <repo> <query>
```

Example:

```bash
sourceevidence find ./my-project "authentication"
```

---

### Find Hotspots

Combines Git change history with repository signals to identify files that deserve closer investigation.

Interactive:

```text
[5] Find hotspots
```

CLI:

```bash
sourceevidence hotspot <repo>
```

---

### Explain a File

Provides file-level evidence such as:

- whether the file exists;
- current size;
- Git change count;
- history scope;
- dependency/content signals.

Interactive:

```text
[6] Explain a file
```

CLI:

```bash
sourceevidence explain <repo> <path>
```

Example:

```bash
sourceevidence explain ./my-project src/auth.go
```

---

### Full Repository Report

Runs the core repository analyses in one workflow and prepares the results for export.

Interactive:

```text
[7] Run full repository report
```

---

### Codebase Overview

Provides a deterministic, at-a-glance summary containing:

- total files;
- repository size;
- directory/root structure;
- repository map;
- largest files;
- Git-derived hotspots;
- current HEAD.

Interactive:

```text
[A] Codebase overview
```

CLI:

```bash
sourceevidence overview <repo>
```

---

# Evidence Pack

The **Evidence Pack** is the central workflow feature connecting SourceEvidence with IBM Bob 2.0.

Given a developer task, SourceEvidence combines deterministic repository information into a reusable report.

Interactive:

```text
[E] Build Evidence Pack
```

CLI:

```bash
sourceevidence evidence <repo> "<task>"
```

Markdown:

```bash
sourceevidence evidence <repo> "<task>" --format=markdown
```

JSON:

```bash
sourceevidence evidence <repo> "<task>" --json
```

An Evidence Pack contains:

- investigation task;
- repository metadata;
- codebase overview;
- scan results;
- dependency information;
- Git history;
- changed-file information;
- repository hotspots.

Example:

```bash
sourceevidence evidence ./my-project "Investigate login failures" --format=markdown
```

### Important boundary

The Evidence Pack is **deterministic evidence**, not an AI-generated diagnosis.

SourceEvidence gathers facts and signals from the repository.

IBM Bob 2.0 can then use that evidence together with its full repository context to investigate, plan, implement, and verify a change.

---

# Use SourceEvidence on Your Own Repository

SourceEvidence is not limited to the included demonstration.

It is designed to analyze a Git repository supplied by the developer.

## Interactive workflow

Start the program without a command:

```bash
sourceevidence
```

On Windows:

```powershell
.\sourceevidence.exe
```

When prompted:

```text
Repository path [.] :
```

enter the Git repository you want to inspect.

Example:

```text
Repository path [.] : C:\Projects\MyApplication
```

The application then presents:

```text
[1] Scan repository
[2] Dependency analysis
[3] Git archaeology
[4] Search code
[5] Find hotspots
[6] Explain a file
[7] Run full repository report
[8] Save session report (.txt)
[9] Change repository
[A] Codebase overview
[B] Save session report (.md)
[E] Build Evidence Pack
[0] Exit
```

The interactive session remembers the selected repository and lets you run multiple analyses without repeatedly typing the repository path.

---

# Command-Line Usage

The direct command interface is useful for scripts and repeatable workflows:

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

Machine-readable JSON is available with:

```text
--json
```

Markdown output is available with:

```text
--format=markdown
```

Use the command-specific help/output for options supported by an individual command.

---

# Report Export

### Text report

Interactive option:

```text
[8] Save session report (.txt)
```

Reports are written to the user's:

```text
SourceEvidence Reports
```

directory, with a temporary-directory fallback when required.

### Markdown report

Interactive option:

```text
[B] Save session report (.md)
```

Markdown reports are suitable for sharing in GitHub issues, documentation, pull-request workflows, and developer tooling.

The report writer does not intentionally write reports inside the repository being analyzed.

---

# IBM Bob 2.0 Workflow

SourceEvidence does not embed IBM Bob or require a Bob API at runtime.

The demonstrated developer workflow is:

### 1. Define the problem

Example:

```text
A payment retry can charge the same payment twice.
```

### 2. Generate deterministic evidence

```bash
sourceevidence evidence ./repository "Investigate duplicate payment"
```

### 3. Open the same repository in IBM Bob 2.0

Bob has full repository context.

### 4. Provide the Evidence Pack

Bob can use the generated evidence together with the actual repository files.

### 5. Investigate and plan

Use Bob's repository-aware reasoning to identify the relevant code path and propose a change.

### 6. Implement

Use Bob Agent mode to apply the approved change.

### 7. Verify

Run the repository's regression tests.

```text
Evidence
   |
   v
Bob investigation
   |
   v
Implementation
   |
   v
Regression test
   |
   v
Developer review
```

---

# Reproducible IBM Bob 2.0 Demo

The `hackathon_demo/` directory contains a small Go payment-processing example used to demonstrate the debugging workflow.

The example starts with a retry/idempotency bug where the same payment can be charged twice.

### Before the fix

```text
go test ./...

FAIL
duplicate charge: expected total 1000, got 2000
```

### SourceEvidence

SourceEvidence generates an Evidence Pack containing repository structure, scan information, Git history, dependency information, and hotspot information for the debugging task.

### IBM Bob 2.0

Bob reads the repository together with the Evidence Pack, identifies the faulty retry guard, and applies the minimal correction.

### After the fix

```text
go test ./...

ok demo-payment
```

See:

```text
HACKATHON_DEMO.md
hackathon_demo/
docs/bob-session/
```

for the complete demonstration and Bob session evidence.

---

# Clone and Build

Clone the public repository:

```bash
git clone https://github.com/aaryan765/SourceEvidence.git
cd SourceEvidence
```

Check the working tree:

```bash
git status
```

Build:

```bash
go build .
```

Run static analysis:

```bash
go vet ./...
```

Run the test suite:

```bash
go test ./...
```

Run SourceEvidence:

```bash
sourceevidence
```

On Windows:

```powershell
.\sourceevidence.exe
```

### Windows note

The commands above assume Go is correctly installed and available on `PATH`. If Windows resolves `go` to an invalid or unrelated executable, use the actual Go installation path or correct the system `PATH`; this is a host environment issue rather than a SourceEvidence dependency.

---

# Safety Model

SourceEvidence is intentionally read-only during repository analysis.

It:

- does not create, modify, delete, or rewrite source files in the inspected repository;
- reads source files, manifests, and Git data;
- does not require network access for repository analysis;
- does not require a third-party runtime dependency.

### Secret detection

Secret detection is deliberately conservative.

A detected pattern is a **signal**, not proof that a credential is valid or leaked.

Test and fixture paths can be classified separately from high-risk secret signals, while high-entropy values remain visible for developer review.

### Repository path handling

File-focused operations validate repository paths and reject traversal or symlink escapes.

---

# Technology

- **Language:** Go
- **Runtime dependencies:** none
- **External packages:** none
- **Network requirement:** none for repository analysis
- **Git subprocess:** not required for Git analysis
- **Platforms:** Windows, Linux, macOS

SourceEvidence uses the Go standard library.

---

# Project Structure

```text
SourceEvidence/
├── main.go              CLI dispatch and interactive workflow
├── ui.go                terminal rendering and report formatting
├── repo.go              repository validation and initialization
├── scan.go              repository scanning and security signals
├── deps.go              dependency analysis
├── git.go               Git object/history analysis
├── search.go            content search and hotspot analysis
├── overview.go          codebase overview
├── evidence.go          Evidence Pack generation
│
├── *_test.go             automated tests
├── test_helpers_test.go  self-contained test-fixture setup
├── demo_repo/            Git-history test fixture used by repository tests
│
├── hackathon_demo/       reproducible IBM Bob 2.0 debugging example
│
├── docs/
│   └── bob-session/      Bob task/session evidence
│
├── HACKATHON_DEMO.md     hackathon workflow walkthrough
└── README.md
```

---

# Testing and Validation

Standard checks:

```bash
go test ./...
go vet ./...
go build .
```

The project also contains tests covering repository scanning, dependency handling, Git analysis, search, reports, and Evidence Pack behavior.

### Self-contained repository tests

The repository includes `demo_repo/` as a tracked Git-history fixture used by a small number of tests. Its Git metadata is stored as `.git-fixture` so it can be committed normally without becoming a nested Git repository.

During tests, `test_helpers_test.go` copies the fixture into a temporary directory and restores `.git` there before the repository-analysis tests run. This keeps the public clone self-contained while preserving the fixture's real Git history for tests such as `TestPackedGitHistory`.

A fresh clone should therefore be able to run:

```bash
go test ./...
```

without requiring any files from the developer's local machine.

---

# Design Principles

### Deterministic Evidence

Repository facts and signals are derived from repository data rather than AI inference.

### Read-Only Analysis

Repository investigation does not intentionally modify the target repository.

### Offline-First

No network service is required for repository analysis.

### Standard Library Only

No third-party runtime packages are required.

### AI as a Reasoning Layer

IBM Bob 2.0 handles repository-aware reasoning and development actions while SourceEvidence remains focused on deterministic evidence.

### Developer in the Loop

The developer reviews the proposed and implemented change and verifies the result with tests.

---

# IBM Bob 2.0 Hackathon

SourceEvidence was developed for the IBM Bob 2.0 Hackathon as a **debugging and application-maintenance workflow**.

The demonstrated workflow is:

```text
Developer problem
       |
       v
SourceEvidence
       |
       v
Evidence Pack
       |
       v
IBM Bob 2.0
       |
       v
Investigation
       |
       v
Implementation
       |
       v
Regression test
       |
       v
Verified change
```

The hackathon demonstration is intentionally reproducible and separate from the core SourceEvidence implementation.

See `HACKATHON_DEMO.md` for the workflow and `docs/bob-session/` for Bob session evidence.

---

# License

See `LICENSE`.
