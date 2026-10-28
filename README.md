<p align="center">
  <img src="https://raw.githubusercontent.com/rossedman/canon/main/docs/assets/banner.png" alt="canon: your engineering standards, as tools agents call" width="900">
</p>

<p align="center">
  <a href="https://github.com/rossedman/canon/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/rossedman/canon/actions/workflows/ci.yml/badge.svg"></a>
  <a href="CONTRIBUTING.md#quality-bars"><img alt="Coverage gate: 90%" src="https://img.shields.io/badge/coverage-%E2%89%A590%25-3FB97F"></a>
  <a href="https://goreportcard.com/report/github.com/rossedman/canon"><img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/rossedman/canon"></a>
  <a href="https://github.com/rossedman/canon/releases"><img alt="Release" src="https://img.shields.io/github/v/release/rossedman/canon?color=E8A33D"></a>
  <a href="LICENSE"><img alt="License: Apache-2.0" src="https://img.shields.io/badge/license-Apache--2.0-4C9BE8"></a>
  <a href="https://rossedman.io/canon/"><img alt="Docs" src="https://img.shields.io/badge/docs-rossedman.io%2Fcanon-E8A33D"></a>
</p>

canon turns engineering opinions into tools an agent calls mid-task. It checks
code against rules, answers "how do we do X" from decision records, reviews
diffs, and scaffolds projects that already follow the standards.

The decision records are the single source of truth. Every rule traces to a
record, and every finding carries `rule → record → why`, so the agent gets the
reason along with the problem. One engine has four front doors: an MCP server,
a CLI, a Claude Code hook, and CI.

```console
$ canon check Dockerfile --min-severity error
...
Dockerfile:6 error Final stage uses debian:bookworm
  canon-final-stage-base → containers-distroless-base: Runtime images carry no shell or package manager.
  fix: FROM gcr.io/distroless/static-debian12:nonroot
Dockerfile:6 error Final stage sets no USER, so it runs as root
  canon-final-stage-user → containers-non-root: A process that escapes a non-root container starts with no privileges on the host.
  fix: USER 65532:65532

canon 4 errors, 0 warnings, 0 suggestions in 1 file (profile go-cli)
```

## Install

```sh
go install github.com/rossedman/canon@latest
```

Or download an archive for your platform from
[Releases](https://github.com/rossedman/canon/releases). Each release's
checksums are signed with cosign; [SECURITY.md](SECURITY.md#verifying-releases)
shows how to verify them.

canon drives the linters you already use. It runs golangci-lint, ruff,
hadolint, tflint, checkov and yamllint when they are on your `PATH`, and
reports any that are missing as skipped. Its own built-in checks always run.
`canon profile` lists which ones your repo needs.

## Quick start

Add canon to Claude Code as an MCP server:

```sh
claude mcp add canon -- canon serve
```

The agent now has six tools (`check`, `decide`, `explain`, `review`,
`scaffold`, `profile`), resources for every record (`canon://records/{id}`),
and three prompts (`canon-review`, `canon-new-record`, `canon-adopt`).

The same engine from a terminal:

```sh
canon profile                          # which profile and records apply here
canon check                            # findings for the whole repo
canon decide how do we handle errors in Go
canon explain canon-final-stage-base   # the record behind a rule
canon review                           # findings and concerns for this branch
canon scaffold go-cli hello --write    # a project that passes its own canon
```

To block edits that break a record, add canon as a Claude Code hook in
`.claude/settings.json`:

```json
{"hooks": {"PostToolUse": [{"matcher": "Edit|Write|MultiEdit",
  "hooks": [{"type": "command", "command": "canon hook"}]}]}}
```

In CI, `canon check --changed-only --sarif > canon.sarif` writes results that
GitHub code scanning can show inline.

## Configuration

Settings resolve as **flag, then environment, then config file**, then the
default. Every flag has an environment variable: `CANON_` plus the command and
flag name, with `-` and `.` turned into `_`. For example, `check
--min-severity` is `CANON_CHECK_MIN_SEVERITY`, and the global `--root` is
`CANON_ROOT`. The config file is the first that exists of
`$XDG_CONFIG_HOME/canon/config.yaml`, `~/.canon.yaml` and `./.canon.yaml`, or
the path given with `--config`:

```yaml
check:
  min-severity: suggestion
  fail-on: warning
hook:
  min-severity: error
```

`--canon-dir` loads records from a directory instead of the copy embedded in
the binary, so a team can keep its canon in its own repo.

## The canon

`canon/` is plain data that a docs site, a hook or another agent can read
without the server:

| Path | What it holds |
| --- | --- |
| `canon/records/<scope>/*.md` | Decision records: Markdown with YAML frontmatter, one opinion each |
| `canon/rules/rules.yaml` | Rules that map checker codes to the records they enforce, plus each checker's config |
| `canon/profiles/*.yaml` | Repo kinds (go-cli, go-service, go-library, py-service, tf-module, generic) and the scopes they apply |
| `canon/templates/<kind>/` | Scaffold sources; every generated project passes `canon check` |
| `canon/principles.md` | The beliefs every record ladders up to |

`canon lint` validates all of it. [Writing records](https://rossedman.io/canon/guides/writing-records/)
covers the format.

## Documentation

The full docs are at <https://rossedman.io/canon/>: getting started,
concepts, the MCP reference, hooks and CI, and a generated page for every
command and every record.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). `make check` runs everything CI does.
Security issues go through [SECURITY.md](SECURITY.md).

## License

[Apache-2.0](LICENSE)
