# STDLIB.md — Dependency Proof & Implementation Substitutions

SourceEvidence is a Go project whose runtime uses only the **Go standard library**.

---

## Dependency Proof

The project's `go.mod` contains no `require` block, and the repository has no:

- `go.sum`
- `vendor/`
- vendored third-party source

The module dependency set can be verified with:

```bash
go list -m all
```

Expected output contains only the SourceEvidence module:

```text
sourceevidence
```

This confirms that the project does not depend on third-party Go modules at runtime.

---

## Standard-Library Substitutions

SourceEvidence implements functionality commonly provided by external packages or utilities using the Go standard library and project-specific code.

| Problem normally delegated to a package or external tool | SourceEvidence implementation |
| --- | --- |
| Git repository access / object graph (`go-git`) | Direct repository and Git object access using `os`, `encoding/hex`, `encoding/binary`, `bytes`, and `compress/zlib` |
| Recursive fast file search (`ripgrep`-style tooling) | `filepath.WalkDir`, `bufio.Scanner`, `strings`, and `sort` |
| Search ranking / index primitives (`bleve`-style search) | `map`, `strings`, `sort`, and a bounded in-memory result set |
| JSON handling (`jsoniter`-style helpers) | `encoding/json` |
| CLI framework (`cobra` / `urfave/cli`) | A project-specific command and flag parser using `os.Args` and standard-library functionality |
| CLI flag parsing (`pflag`) | Standard-library argument handling with explicit validation |
| Test assertions (`testify`) | The built-in `testing` package and direct assertions |
| Compression support for Git packs (`klauspost/compress`-style helpers) | `compress/zlib` |
| Path and filesystem utilities (`afero`) | `os` and `path/filepath` |
| Structured terminal and report formatting (`tablewriter`-style helpers) | `fmt`, `strings`, and explicit formatting |
| Dependency metadata parsing helpers | Hand-written parsing using `encoding/json`, `bufio`, `strings`, and `regexp` |

> These are implementation comparisons only. None of the named third-party packages is imported by SourceEvidence.

---

## What SourceEvidence Implements Directly

The project contains its own implementations for:

- Git repository traversal
- Git object-store inspection
- Git pack index v2 lookup
- Git pack object decoding
- REF delta application
- OFS delta application
- Dependency-manifest inspection
- Ranked source search
- Hotspot aggregation
- Codebase overview generation
- File-level explanation
- Evidence Pack generation
- Text and Markdown report formatting

The project does not launch the `git` executable for repository analysis.

No third-party source code was copied into the repository.

---

## Repository Access & Write Behavior

SourceEvidence reads repository files and `.git` data directly rather than invoking the `git` executable.

The core repository-analysis operations are read-only with respect to the target repository's source code and Git metadata. SourceEvidence does not modify the target repository's source files as part of:

- scanning
- searching
- dependency analysis
- Git archaeology
- hotspot analysis
- codebase overview
- file-level analysis

Separate reporting features may write generated output artifacts when the user explicitly requests them, including:

- saved session reports
- Markdown reports
- generated Evidence Packs

These outputs are generated artifacts and are separate from modifying the analyzed repository's source code.

---

## Test Fixture Behavior

The repository includes a checked-in `demo_repo` fixture used by the automated test suite.

The fixture contains Git pack data under:

```text
demo_repo/.git-fixture/
```

Tests do not depend on a developer's external local repository.

When a test needs a working `.git` directory, `test_helpers_test.go` prepares an isolated temporary copy of the checked-in fixture and restores the Git metadata there.

This keeps the tests reproducible from a fresh clone while preserving the packed Git history required by the Git-analysis tests.

---

## Verification

The standard-library and dependency claims can be checked with:

```bash
go list -m all
go test ./...
go vet ./...
go build .
```

The expected module output from `go list -m all` contains only:

```text
sourceevidence
```

The test suite exercises:

- repository scanning
- source search
- codebase overview generation
- Git history analysis
- report generation
- the checked-in demo repository fixture

---

## Summary

SourceEvidence deliberately keeps its runtime dependency surface small.

Instead of relying on third-party Go modules for repository analysis, search, reporting, CLI handling, or Git pack processing, the project uses the **Go standard library together with its own implementations**.

This design keeps the repository self-contained, simplifies reproducible builds, and allows the core repository-intelligence workflow to operate without invoking external Git commands or requiring third-party runtime packages.
