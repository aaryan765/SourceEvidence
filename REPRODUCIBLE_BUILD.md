# Reproducible build proof

Build command:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o sourceevidence .
```

The source tree was built twice with the same toolchain and flags. The resulting Linux/amd64 artifacts were byte-identical.

SHA-256:

- build 1: `2d4a258f6059a008d6c15a9d2c384410f1ac34e200b71074311e68bbce7eee6e`
- build 2: `2d4a258f6059a008d6c15a9d2c384410f1ac34e200b71074311e68bbce7eee6e`
