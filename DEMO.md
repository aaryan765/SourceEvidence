# SourceEvidence Demo

This repository includes a small `demo_repo` fixture used for reproducible testing and demonstration.

## Interactive Mode

Build SourceEvidence, start the executable, and choose a repository to inspect.

```text
sourceevidence
```

For a repository with a normal `.git` directory, the interactive menu provides:

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

### Repository Fixture

The checked-in `demo_repo/` directory is a test fixture. Its Git metadata is stored separately under:

```text
demo_repo/.git-fixture/
```

The automated tests prepare an isolated temporary copy and restore the Git metadata when Git-history analysis is required.

For interactive demonstrations involving Git archaeology, hotspots, or Evidence Packs, use a normal Git repository with a `.git` directory.

## Core Workflow

The main repository-analysis workflow is:

```text
Repository
    |
    v
SourceEvidence analysis
    |
    +--> Scan
    +--> Dependencies
    +--> Git archaeology
    +--> Search
    +--> Hotspots
    +--> Explain
    +--> Codebase Overview
    |
    v
Evidence Pack
```

The Evidence Pack combines deterministic repository evidence for AI-agent and automation workflows.

## Direct Command Mode

Build the project:

```bash
go build -trimpath -o sourceevidence .
```

Run individual analyses against a normal Git repository:

```bash
./sourceevidence scan <repo>
./sourceevidence deps <repo>
./sourceevidence git <repo>
./sourceevidence find <repo> "API_KEY"
./sourceevidence hotspot <repo>
./sourceevidence explain <repo> src/auth.go
./sourceevidence overview <repo>
```

Use JSON output when needed:

```bash
./sourceevidence scan <repo> --json
./sourceevidence overview <repo> --json
```

Generate an Evidence Pack:

```bash
./sourceevidence evidence <repo> "Investigate authentication failure" --json
```

## Report Export

### Text Report

Interactive option **[8] Save session report (.txt)** creates a timestamped text report outside the inspected repository.

### Markdown Report

Interactive option **[B] Save session report (.md)** creates a structured Markdown report suitable for issues, pull-request workflows, CI artifacts, and developer review.

### Evidence Pack

Interactive option **[E] Build Evidence Pack** creates a compact package of deterministic repository evidence for AI-agent and automation workflows.

The Evidence Pack includes:

- Codebase Overview
- repository scan results
- dependency analysis
- Git history information
- historical hotspot information

SourceEvidence provides evidence only. It does not infer a root cause or generate a code fix.

## IBM Bob 2.0 Workflow

SourceEvidence can be used as the evidence layer before IBM Bob 2.0 begins investigation and implementation.

```text
Developer task
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
Investigate -> Plan -> Implement -> Test
      |
      v
Developer review
```

This keeps deterministic repository analysis separate from AI-assisted reasoning and development actions.

## Hackathon Demonstration

The included `hackathon_demo/` directory contains the reproducible payment-processing example used to demonstrate the IBM Bob 2.0 workflow.

See:

```text
HACKATHON_DEMO.md
```

for the complete debugging demonstration and Bob-session evidence.
