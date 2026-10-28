# Getting started

Install canon, give your agent its tools, and run a first check in about five minutes.

## Install

=== "go install"

    ```bash
    go install github.com/rossedman/canon@latest
    ```

    Needs Go 1.27 or newer. Make sure `$(go env GOPATH)/bin` is on your `PATH`.

=== "Release archive"

    Download the archive for your platform from the
    [releases page](https://github.com/rossedman/canon/releases), unpack it and put
    `canon` on your `PATH`. Verify it first, as described below.

Check it works:

```bash
canon version
```

### Verify a release

Release archives are listed in `checksums.txt`, which the release workflow
(`.github/workflows/release.yml`) signs with [cosign](https://docs.sigstore.dev/)
keylessly. The signature bundle is `checksums.txt.sigstore.json`. Download both
files next to the archive, then:

```bash
cosign verify-blob \
  --bundle checksums.txt.sigstore.json \
  --certificate-identity-regexp '^https://github.com/rossedman/canon/\.github/workflows/release\.yml@refs/tags/v.*$' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  checksums.txt

sha256sum --ignore-missing -c checksums.txt
```

The first command proves `checksums.txt` came from this repository's release
workflow. The second proves your archive matches it.

## Add canon to Claude Code

```bash
claude mcp add canon -- canon serve
```

This registers the MCP server over stdio. Claude can now call the six tools
(`check`, `decide`, `explain`, `review`, `scaffold`, `profile`). See the
[MCP guide](guides/mcp.md) for what each does, and
[Hooks and CI](guides/hooks-and-ci.md) to make Claude check every file it edits.

To serve over HTTP instead, run `canon serve --http :8080`.

## Run your first check

From the root of a repository:

```bash
canon profile   # which kind of repo is this, and which records apply
canon check     # run every rule in force
```

A finding looks like this:

```text
Dockerfile:6 error Final stage uses debian:bookworm
  canon-final-stage-base → containers-distroless-base: Runtime images carry no shell or package manager.
  fix: FROM gcr.io/distroless/static-debian12:nonroot
```

Read it as `file:line severity message`, then `rule → record: why`, then a fix
when canon knows one. Run `canon explain canon-final-stage-base` for the full
record behind it.

!!! note "Linters are optional"
    `canon profile` lists which checkers are installed. A missing linter is
    skipped with a message; canon's own checks always run.

## Ask a question

```bash
canon decide how do we handle errors in Go
```

returns the ranked records that answer it, with their Decision and Exceptions,
or a gap when the canon has no answer.

## Next steps

- [Concepts](guides/concepts.md): records, rules, profiles and findings.
- [Configuration](configuration.md): flags, environment variables and the config file.
- [Writing records](guides/writing-records.md): add your own standards.
