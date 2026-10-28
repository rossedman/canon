# Contributing to canon

Thanks for helping. canon has two kinds of contribution: changes to the
engine (Go code) and changes to the canon itself (decision records, rules,
profiles and templates under `canon/`). Both go through the same checks.

## Set up

You need Go 1.27 or later. For the full `make check` you also need
[golangci-lint](https://golangci-lint.run) v2, [shellcheck](https://www.shellcheck.net),
[GoReleaser](https://goreleaser.com) v2 and [uv](https://docs.astral.sh/uv/) (for the docs site).

```sh
git clone https://github.com/rossedman/canon.git
cd canon
go build ./...
go test ./...
```

The external linters canon drives (hadolint, ruff, tflint, checkov,
yamllint) are optional for development: tests use a scripted runner, and
`canon check` reports a checker it cannot find as skipped.

## Check your work

```sh
make check
```

runs everything CI runs: build, lint, tests with the race detector and the
coverage gate, the one-command-per-file check, doc comments, `canon lint`,
benchmark and fuzz smoke runs, the strict docs build and `goreleaser check`.
`make fmt` fixes formatting, and `make docs` regenerates `docs/reference/`
after you change a command or a record.

## Quality bars

- **Coverage:** at least 90% of hand-written code, measured by `make cover-gate`.
  Generated files are excluded. Don't write tests for trivial getters to move the number.
- **Tests:** table-driven, `t.Parallel()` where safe. Commands are tested
  through `rootCmd.SetArgs` with captured output (see `cmd/cmd_test.go`).
- **Doc comments:** every function and method, exported or not, has one that
  starts with its name, and every package has a package comment
  (`make doccheck`).
- **One cobra command per file** in `cmd/`, named after the command
  (`make check-cmds`).
- **Errors** are wrapped with `%w` and context where they cross a package.
- **Benchmarks and fuzzing:** every critical path below has a `BenchmarkXxx`;
  every one that reads external input has a `FuzzXxx` with a real seed
  corpus that asserts properties. Commit failing fuzz inputs under
  `testdata/fuzz/`.

## Critical paths

These parse input, transform data, run in hot loops or resolve config.
Changes to them need a benchmark result (`make bench`, then compare with
`benchstat`) and, for parsers, a clean 30-second fuzz run.

| Function | Why it is critical | Benchmark | Fuzz |
| --- | --- | --- | --- |
| `records.Parse` | parses every record file | `BenchmarkParse` | `FuzzParse` |
| `glob.Match` | runs for every file against every pattern | `BenchmarkMatch` | `FuzzMatch` |
| `search.Tokenize`, `(*search.Index).Search` | every `decide` call | `BenchmarkSearch` | `FuzzTokenize` |
| `review.ParseDiff` | parses diffs from agents and git | `BenchmarkParseDiff` | `FuzzParseDiff` |
| `review.Classify` | runs over every changed line | `BenchmarkClassify` | via `FuzzParseDiff` |
| `checkers.ParseDockerfile` and the Dockerfile checks | parse user Dockerfiles | `BenchmarkParseDockerfile` | `FuzzParseDockerfile` |
| `checkers.Pinned`, `checkers.FindUses` | parse workflow files | covered by tests | `FuzzPinned` |
| `checkers.ParseGolangci`, `ParseRuff`, `ParseHadolint`, `ParseTFLint`, `ParseCheckov`, `ParseYamllint` | parse linter output | `BenchmarkParseGolangci` | `FuzzParseToolOutput` |
| `findings.Normalize` | sorts and dedupes every result | `BenchmarkNormalize` | n/a |
| `hook.ParseInput` | parses Claude Code hook payloads | n/a | `FuzzParseInput` |
| `(*engine.Engine).Check`, `Decide` | the path every front door takes | `BenchmarkCheckBuiltin`, `BenchmarkDecide` | n/a |
| `cmd.initConfig` | config precedence: flag > env > file | tested in `cmd/cmd_test.go` | n/a |

## Changing the canon

- One opinion per record, in `canon/records/<scope>/<id>.md`. The file name
  matches the `id`. Required sections: Context, Decision, Consequences.
- New records start as `status: proposed`. The `canon-new-record` MCP prompt
  drafts one in the right shape.
- A record's `rules:` and each rule's `record:` in `canon/rules/rules.yaml`
  must point at each other. Every rule traces to a record; there are no
  orphan lint rules.
- To retire advice, add a new record that lists the old one in `supersedes:`
  and set the old one's status to `superseded`.
- Templates under `canon/templates/` must pass their own canon: the tests
  scaffold every kind and fail on any error.
- Run `go run . lint` and `make docs`.

## Commits and pull requests

Commit messages and PR titles follow [Conventional Commits](https://www.conventionalcommits.org):

```text
feat(checkers): add a built-in check for unpinned pip installs
fix(review): count renamed files as modified
docs(records): explain why distroless supersedes alpine
```

PRs are squash-merged, so the PR title becomes the commit. Every PR needs a
review from a code owner (see `.github/CODEOWNERS`) and green CI. Fill in
the pull request template's checklist.

## Questions

Open an issue with the feature request form, or ask in the PR you're
working on.
