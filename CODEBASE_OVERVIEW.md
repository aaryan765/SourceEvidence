# Codebase Overview

SourceEvidence includes a deterministic **Codebase Overview** workflow inspired by repository-mapping tools: it gives a developer a fast mental model before they dive into individual files.

It remains:

- read-only
- standard-library only
- offline/no network requirement
- independent of the `git` executable

## What it shows

`sourceevidence overview <repo>` combines existing SourceEvidence analysis with a lightweight repository map:

- repository file count and total size
- directory count and root-level file count
- top-level repository areas (for example `src/`, `docs/`, `tests/`) with file counts and sizes
- historical Git hotspots with current file sizes where available
- largest current files
- current HEAD commit

Use `--json` for machine-readable output.

## Evidence Pack

`sourceevidence evidence <repo> [task] --json` bundles deterministic scan, dependency, Git, and hotspot evidence into one compact structure for AI-agent and automation workflows. It is evidence only: SourceEvidence does not infer a root cause or generate a fix.

## Reporting

Interactive **[B] Save session report (.md)** produces structured Markdown tables, making the collected evidence easy to attach to a GitHub Actions artifact or pull-request workflow. The existing **[8] Save session report (.txt)** export remains unchanged.

## Interactive menu

The original numeric options remain available. Press **A** for `Codebase overview (at-a-glance map)` and **B** for the Markdown export.

## Hackathon role

The overview and Evidence Pack are deliberately deterministic. They do not add an AI runtime to SourceEvidence. In the IBM Bob workflow, SourceEvidence provides the local evidence layer while Bob provides the reasoning and development layer.
