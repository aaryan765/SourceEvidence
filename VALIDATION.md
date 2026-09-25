# SourceEvidence validation

This release is a clean rebuild of SourceEvidence with the interactive menu, report export, dependency-scope improvements, conservative secret signals, unified Git-history semantics, Markdown reporting, Evidence Pack output, and deterministic Codebase Overview.

## Automated checks

```text
go test -count=1 ./...        PASS
go test -race -count=1 ./... PASS
go vet ./...                 PASS
go list -m all               sourceevidence only
go build -trimpath           PASS
```

## Regression coverage

The test suite covers:

- Correct byte-to-KB/MB/GB formatting.
- Go, Python, npm, and Cargo source-aware dependency matching.
- `pyproject.toml` runtime, optional, docs, and dev/test dependency scopes.
- Requirements-file comment sections for optional, documentation, packaging/build, tests, and linting scopes.
- File-level dependency mentions based on actual imports across Go, Python, npm, and Cargo, including renamed Cargo dependencies.
- Hotspots and Explain sharing the same bounded Git-history traversal and exposing the history scope explicitly.
- Dependency duplicate merging within the same ecosystem/scope.
- Conservative secret detection with low-entropy test/fixture credentials classified as `mock-test-secret` while high-entropy values remain visible.
- Repository path traversal and symlink-escape rejection.
- Clean `.txt` report generation outside the inspected repository.
- Structured Markdown report generation.
- Packed Git history parsing and demo-repository hotspots.
- Deterministic codebase overview areas, hotspot mapping, and largest-file presentation.
- Evidence Pack aggregation for AI-agent/automation workflows.

## Cross-platform builds

```text
Windows/amd64  PASS
Linux/amd64    PASS
macOS/arm64    PASS
CGO_ENABLED=0 / standard-library-only build PASS
```

The Linux/amd64 release was built twice with identical flags and produced the same SHA-256:

```text
6f7e8accba83b757832ac2586bac9f5d4364e278874d419f47dc6c3663a82cf4
```

## Functional smoke tests

The analysis commands and interactive menu were exercised against the included Git demo repository, and the real-world validation workflow was exercised against HTTPX on Windows.

```text
scan              PASS
deps              PASS
git               PASS
find              PASS
hotspot           PASS
explain           PASS
full report       PASS
overview           PASS
overview JSON      PASS
.txt export       PASS
Markdown output   PASS
Evidence Pack     PASS
```

A real HTTPX scan exposed one deliberate test credential at `tests/test_auth.py:150`; this is now classified as `mock-test-secret` rather than a high-risk generic secret. The dependency analyzer also uses requirements-file section comments where available to avoid treating documentation, packaging, and test tooling as runtime dependencies.
