# Codebase Overview

SourceEvidence includes a deterministic **Codebase Overview** workflow that gives a developer a fast mental model of a repository before they dive into individual files.

It remains:

- read-only with respect to the inspected repository
- standard-library only at runtime
- offline with no network requirement
- independent of the `git` executable

## What it shows

`sourceevidence overview <repo>` combines existing SourceEvidence analysis with a lightweight repository map:

- repository file count and total size
- directory count and root-level file count
- top-level repository areas with file counts and sizes
- historical Git hotspots with current file sizes where available
- largest current files
- current HEAD commit

Use `--json` for machine-readable output.

## Evidence Pack

`sourceevidence evidence <repo> [task] --json` bundles deterministic repository evidence into one compact structure for AI-agent and automation workflows.

The Evidence Pack includes:

- codebase overview
- repository scan results
- dependency analysis
- Git history information
- historical hotspot information

It is evidence only: SourceEvidence does not infer a root cause or generate a fix.

## Reporting

Interactive **[B] Save session report (.md)** produces structured Markdown tables, making collected evidence easy to attach to GitHub Actions artifacts, pull-request workflows, issues, or other developer tooling.

The existing **[8] Save session report (.txt)** export remains available.

## Interactive Menu

The interactive menu includes:

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

## Hackathon Role

The Codebase Overview and Evidence Pack are deliberately deterministic. They do not add an AI runtime to SourceEvidence.

In the IBM Bob workflow:

```text
SourceEvidence
     |
     v
Deterministic repository evidence
     |
     v
Evidence Pack
     |
     v
IBM Bob 2.0
     |
     v
Investigation -> Planning -> Implementation -> Testing
```

SourceEvidence provides the local evidence layer, IBM Bob 2.0 provides repository-aware reasoning and development actions, and the developer remains responsible for reviewing and validating the resulting change.
