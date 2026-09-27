# Reproducible Build Verification

SourceEvidence can be cross-compiled for Linux/amd64 with a fixed Go toolchain and reproducible build flags.

## Build environment

- Go version: `go1.27.1 windows/amd64`
- Target OS: `linux`
- Target architecture: `amd64`
- CGO: disabled

## Build command

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o sourceevidence .
```

The source tree was built twice with the same toolchain, target settings, and build flags.

## SHA-256 Verification

Build 1:

```text
16766CD37DDAC989646CDA66693C2E43C86DF546F7D021A37F6DF0F36EA6D128
```

Build 2:

```text
16766CD37DDAC989646CDA66693C2E43C86DF546F7D021A37F6DF0F36EA6D128
```

The two generated Linux/amd64 binaries were byte-identical, producing the same SHA-256 digest.

## Result

The current SourceEvidence source tree produced identical Linux/amd64 build artifacts across two consecutive builds using the same Go toolchain and build configuration.
