# SourceEvidence Validation

This document records validation checks performed for the current SourceEvidence release.

---

## Automated Checks

The current source tree was verified with:

```text
go test ./...        PASS
go vet ./...         PASS
go build .           PASS
go list -m all       sourceevidence only
```

A fresh clone of the public GitHub repository was also verified successfully with:

```text
go test ./...        PASS
go vet ./...         PASS
go build .           PASS
```

The fresh-clone verification confirmed that the checked-in `demo_repo` fixture and its Git pack data are available from the public repository without relying on external local files.

---

## Regression Coverage

The test suite covers:

- Byte-to-KB/MB/GB formatting.
- Go, Python, npm, and Cargo dependency matching.
- `pyproject.toml` dependency scopes.
- Requirements-file dependency sections where supported.
- File-level dependency mentions based on source imports.
- Git-history analysis and bounded history traversal.
- Dependency duplicate handling.
- Conservative secret-pattern detection.
- Repository path and symlink-escape validation.
- Text report generation.
- Markdown report generation.
- Packed Git history parsing.
- Codebase Overview generation.
- Evidence Pack aggregation.

---

## Reproducible Linux/amd64 Build

Linux/amd64 reproducibility was verified using:

| Setting | Value |
| --- | --- |
| Go version | `go1.27.1 windows/amd64` |
| Target OS | `linux` |
| Target architecture | `amd64` |
| CGO | disabled |

Build command:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o sourceevidence .
```

Two consecutive builds produced the same SHA-256 digest:

```text
16766CD37DDAC989646CDA66693C2E43C86DF546F7D021A37F6DF0F36EA6D128
```

The two generated Linux/amd64 artifacts were byte-identical.

See `REPRODUCIBLE_BUILD.md` for the complete build record.

---

## Functional Smoke Tests

The core SourceEvidence workflow was exercised against the included Git demo repository.

| Workflow | Result |
| --- | --- |
| Scan repository | PASS |
| Dependency analysis | PASS |
| Git archaeology | PASS |
| Code search | PASS |
| Hotspot analysis | PASS |
| File explanation | PASS |
| Full repository report | PASS |
| Codebase overview | PASS |
| Text report export | PASS |
| Markdown output | PASS |
| Evidence Pack | PASS |

The interactive menu was exercised through the repository-analysis, reporting, overview, and Evidence Pack workflows.

---

## Dependency Verification

The project uses only the Go standard library at runtime.

```text
go list -m all
```

reports only:

```text
sourceevidence
```

No third-party Go modules are required for the runtime.

---

## Notes

Secret detection is intentionally conservative. A detected value is a signal for review rather than proof that a credential is valid or exposed.

The checked-in `demo_repo` is used as a deterministic test fixture. Tests prepare an isolated temporary copy when a working `.git` directory is required for Git-history analysis.
