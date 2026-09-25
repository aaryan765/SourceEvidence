# STDLIB.md — dependency proof and substitutions

SourceEvidence is a Go project whose runtime uses only the Go standard library.

## Manifest proof

`go.mod` contains no `require` block and the project has no `go.sum`, no `vendor/`, and no vendored third-party source.

The verification command is:

```bash
go list -m all
```

Expected output contains only this module:

```text
sourceevidence
```

## Standard-library substitutions

| Problem normally delegated to a package | SourceEvidence implementation |
|---|---|
| Git repository access / object graph (`go-git`) | `os`, `encoding/hex`, `encoding/binary`, `bytes`, `compress/zlib` |
| Recursive fast file search (`ripgrep`-style tooling) | `filepath.WalkDir`, `bufio.Scanner`, `strings`, `sort` |
| Search ranking/index primitives (`bleve`-style search) | `map`, `strings`, `sort` and a bounded in-memory result set |
| JSON handling (`jsoniter`-style helpers) | `encoding/json` |
| CLI framework (`cobra` / `urfave/cli`) | a small command/flag parser using `os.Args` and the standard library |
| CLI flag parsing (`pflag`) | standard-library argument handling and explicit validation |
| Test assertions (`testify`) | the built-in `testing` package and direct assertions |
| Compression support for Git packs (`klauspost/compress`-style helpers) | `compress/zlib` |
| Path and filesystem utilities (`afero`) | `os` and `path/filepath` |
| Structured terminal/report output (`tablewriter`-style helpers) | `fmt`, `strings`, and explicit formatting |
| Dependency metadata parsing helpers | hand-written parsing on top of `encoding/json`, `bufio`, `strings`, and `regexp` |

These are documented implementation comparisons; none of the named third-party packages is imported by this project.

## What we wrote by hand

The project implements its own Git object-store traversal, pack index v2 lookup, Git pack object decoding, REF/OFS delta application, dependency-manifest inspection, ranked source search, hotspot aggregation, and report formatting.

No third-party source was copied into the repository.

## Read-only claim

The production program contains no file-write APIs and never launches the `git` executable. It reads repository files and `.git` objects directly. The tests also use the checked-in demo repository rather than generating files at runtime.
