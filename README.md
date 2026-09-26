# SourceEvidence

## Evidence-Driven Debugging for Real Git Repositories

**SourceEvidence** is a deterministic, offline repository intelligence tool for developers who need reliable repository evidence before changing code.

It analyzes repository structure, dependencies, Git history, search results, hotspots, and file-level signals, then packages those findings into an **Evidence Pack** that can be used by **IBM Bob 2.0** during debugging and application-maintenance workflows.

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
Verified change
```

**SourceEvidence provides the evidence. Bob provides the reasoning and development actions. The developer remains the reviewer.**

---

## The Problem

Debugging an unfamiliar repository often requires a large amount of manual investigation before a developer can safely make a change.

A developer may need to understand the repository structure, find relevant files, inspect dependencies, search for related code, inspect Git history, identify frequently changed areas, and collect enough context before modifying anything.

SourceEvidence turns that repetitive evidence-gathering process into a deterministic workflow.

---

## The Solution

SourceEvidence creates a local evidence layer over a Git repository.

It does **not** embed an LLM or attempt to replace AI reasoning.

Instead:

```text
SourceEvidence
    |
    v
Repository facts and signals
    |
    v
Evidence Pack
    |
    v
IBM Bob 2.0
    |
    v
Repository-aware investigation
    |
    v
Implementation
    |
    v
Regression test
```

This separates deterministic repository evidence from AI reasoning and implementation assistance.

---

# What SourceEvidence Provides

## 1. Scan Repository

Analyzes:

- file counts
- repository size
- file-type distribution
- largest files
- conservative secret-pattern signals

Interactive menu: **[1] Scan repository**

CLI:

```bash
sourceevidence scan <repo>
```

JSON:

```bash
sourceevidence scan <repo> --json
```

---

## 2. Dependency Analysis

Analyzes supported dependency manifests and reports:

- declared dependencies
- dependency versions
- dependency scope
- source references
- possible unused direct runtime dependencies

Supported ecosystems include Go, Python, npm, and Rust/Cargo-related manifests.

Interactive menu: **[2] Dependency analysis**

CLI:

```bash
sourceevidence deps <repo>
```

---

## 3. Git Archaeology

Reads repository Git data directly to analyze:

- HEAD
- discovered refs
- bounded commit history
- authors
- changed-file counts

Git analysis does not require invoking a `git` subprocess.

Interactive menu: **[3] Git archaeology**

CLI:

```bash
sourceevidence git <repo>
```

---

## 4. Search Code

Performs ranked, case-insensitive content search.

Results include:

- file path
- line number
- matching source line
- relevance score

Interactive menu: **[4] Search code**

CLI:

```bash
sourceevidence find <repo> <query>
```

Example:

```bash
sourceevidence find ./my-project "authentication"
```

---

## 5. Find Hotspots

Combines Git change history with repository signals to identify files that deserve investigation.

Interactive menu: **[5] Find hotspots**

CLI:

```bash
sourceevidence hotspot <repo>
```

Historical analysis is bounded to discovered repository refs and up to 500 commits.

---

## 6. Explain a File

Provides file-level evidence including:

- whether the file exists
- current size
- Git change count
- history scope
- dependency/content signals

Interactive menu: **[6] Explain a file**

CLI:

```bash
sourceevidence explain <repo> <path>
```

Example:

```bash
sourceevidence explain ./my-project src/auth.go
```

---

## 7. Full Repository Report

Runs the main repository analyses in one workflow and prepares the resulting reports for export.

Interactive menu: **[7] Run full repository report**

---

## 8. Codebase Overview

Provides a deterministic at-a-glance repository map containing:

- total files
- repository size
- directory/root structure
- repository map
- largest files
- Git-derived hotspots
- current HEAD

Interactive menu: **[A] Codebase overview**

CLI:

```bash
sourceevidence overview <repo>
```

---

# Evidence Pack

The **Evidence Pack** is the central workflow feature connecting SourceEvidence to IBM Bob 2.0.

It packages deterministic repository evidence into a reusable artifact.

Interactive menu: **[E] Build Evidence Pack**

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

- investigation task
- repository metadata
- codebase overview
- repository scan
- dependency analysis
- Git history
- changed-file information
- repository hotspots

The Evidence Pack is **evidence, not an AI conclusion**. SourceEvidence does not claim to determine the root cause automatically.

---

# Using SourceEvidence on Your Own Repository

SourceEvidence is not limited to the included demonstration project.

Point it at the Git repository you want to investigate.

Interactive mode:

```bash
sourceevidence
```

Then provide a repository path:

```text
Repository path [.] : C:\Projects\MyApplication
```

Or use the CLI directly:

```bash
sourceevidence scan C:\Projects\MyApplication
sourceevidence deps C:\Projects\MyApplication
sourceevidence git C:\Projects\MyApplication
sourceevidence find C:\Projects\MyApplication "login"
sourceevidence hotspot C:\Projects\MyApplication
sourceevidence explain C:\Projects\MyApplication src/login.go
sourceevidence overview C:\Projects\MyApplication
```

For an AI-assisted debugging workflow:

```bash
sourceevidence evidence C:\Projects\MyApplication "Investigate login failures" --format=markdown
```

The generated Evidence Pack can then be provided to IBM Bob 2.0 together with the repository context.

---

# Interactive Menu

Running SourceEvidence without a command opens the interactive workflow:

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

The interactive session keeps the selected repository and accumulated reports together.

---

# IBM Bob 2.0 Workflow

SourceEvidence does not embed IBM Bob or require a Bob API at runtime.

The demonstrated workflow is:

### Step 1 - Identify the developer problem

Example:

```text
Payment retries are causing duplicate charges.
```

### Step 2 - Generate evidence

```bash
sourceevidence evidence ./repository "Investigate duplicate payment"
```

### Step 3 - Open the same repository in IBM Bob 2.0

Bob provides full repository context.

### Step 4 - Provide the Evidence Pack

Bob can use the deterministic evidence together with the actual source files.

### Step 5 - Investigate and plan

Use Bob's repository-aware reasoning to identify the relevant code path and proposed change.

### Step 6 - Implement

Use Bob Agent mode to apply the approved fix.

### Step 7 - Verify

Run the repository's regression tests and review the resulting change.

```text
Evidence
   |
   v
Bob investigation
   |
   v
Bob implementation
   |
   v
Tests
   |
   v
Developer review
```

---

# Reproducible IBM Bob 2.0 Demo

The `hackathon_demo/` directory contains a small reproducible Go payment-processing example used to demonstrate the debugging workflow.

The original bug causes a retry of the same payment to bypass the duplicate-payment guard.

### Before the fix

```text
go test ./...

FAIL
duplicate charge: expected total 1000, got 2000
```

### SourceEvidence

SourceEvidence generates an Evidence Pack describing the repository and investigation task.

### IBM Bob 2.0

Bob reads the repository together with the Evidence Pack, identifies the faulty retry guard, and applies the minimal fix.

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

for the reproducible workflow and Bob session evidence.

---

# Report Export

### Text Report

Interactive menu: **[8] Save session report (.txt)**

Reports are written to the user's `SourceEvidence Reports` directory, with a temporary-directory fallback where required.

The report writer does not write inside the inspected repository.

### Markdown Report

Interactive menu: **[B] Save session report (.md)**

Markdown reports are useful for GitHub documentation, issues, pull requests, and other developer tooling.

---

# Safety Model

SourceEvidence is intentionally read-only during repository analysis.

It:

- does not modify source files in the inspected repository
- does not delete repository files
- does not rewrite repository content
- reads source files, manifests, and Git objects
- does not require network access

Secret detection is deliberately conservative.

A detected pattern is a **signal**, not proof that a credential is valid or leaked.

Test and fixture paths can be classified separately from high-risk secret signals, and high-entropy values remain visible for developer review.

Repository path handling also rejects traversal and symlink escapes for file-focused operations.

---

# Technology

- Go
- Go standard library only
- zero third-party runtime dependencies
- offline repository analysis
- direct Git object/history analysis
- Windows / Linux / macOS support

---

# Project Structure

```text
SourceEvidence/
├── main.go
├── ui.go
├── repo.go
├── scan.go
├── deps.go
├── git.go
├── search.go
├── overview.go
├── evidence.go
│
├── *_test.go
│
├── hackathon_demo/
│   ├── payment.go
│   ├── payment_test.go
│   ├── service.go
│   ├── go.mod
│   └── SOURCEEVIDENCE_EVIDENCE.md
│
├── docs/
│   └── bob-session/
│
├── HACKATHON_DEMO.md
└── README.md
```

---

# Building and Testing

Build:

```bash
go build -o sourceevidence .
```

Run tests:

```bash
go test ./...
```

Static analysis:

```bash
go vet ./...
```

---

# Design Principles

### Deterministic Evidence

Repository facts and signals are generated from the repository itself rather than by AI inference.

### Read-Only Analysis

Repository investigation does not modify the target repository.

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

SourceEvidence was developed for the IBM Bob 2.0 Hackathon as a debugging and application-maintenance workflow.

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

See `HACKATHON_DEMO.md` for the complete reproducible example and `docs/bob-session/` for IBM Bob session evidence.
