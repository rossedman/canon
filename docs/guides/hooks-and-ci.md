# Hooks and CI

Run canon where mistakes are cheapest to fix: as the agent edits, before a commit, and on every pull request.

## Claude Code hook

`canon hook` reads a PostToolUse payload on stdin, checks the file the agent
just edited, and exits 2 with the findings on stderr so the agent fixes them
before moving on. Add it to `.claude/settings.json`:

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Edit|Write|MultiEdit",
        "hooks": [{ "type": "command", "command": "canon hook" }]
      }
    ]
  }
}
```

What the agent sees:

```text
canon found 1 issue(s) in Dockerfile. Fix them before continuing; run `canon explain <rule>` for the full reasoning.
- Dockerfile:6 [error canon-final-stage-user] Final stage sets no USER, so it runs as root (why: A process that escapes a non-root container starts with no privileges on the host.) fix: USER 65532:65532
```

Only findings at or above `--min-severity` (default `error`) block. Set it
with `CANON_HOOK_MIN_SEVERITY=warning` to be stricter. The hook uses the
payload's working directory as the root unless you pass `--root`.

## Git pre-commit

Given paths, `canon hook` checks those files:

```bash
#!/usr/bin/env bash
# .git/hooks/pre-commit
canon hook $(git diff --cached --name-only --diff-filter=ACM)
```

## GitHub Actions

Check only the files a pull request changed and upload the result as SARIF, so
findings appear as code scanning alerts:

```yaml
name: canon
on:
  pull_request:
permissions: {}
jobs:
  canon:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      security-events: write
    steps:
      - uses: actions/checkout@<40-char commit SHA> # <version>
        with:
          fetch-depth: 0
          persist-credentials: false
      - uses: actions/setup-go@<40-char commit SHA> # <version>
        with:
          go-version: stable
      - run: go install github.com/rossedman/canon@latest
      - run: canon check --changed-only --base origin/${{ github.base_ref }} --sarif > canon.sarif
      - if: always()
        uses: github/codeql-action/upload-sarif@<40-char commit SHA> # <version>
        with:
          sarif_file: canon.sarif
```

`canon check` exits 1 when a finding reaches `--fail-on` (default `error`), which
fails the job; the SARIF step still runs. Pin the actions to full commit SHAs,
as canon's own `ci-pin-actions-sha` record requires.

!!! tip "Review before you push"
    `canon review` adds the concerns no linter reports. Run it locally, or feed it a diff:
    `git diff main | canon review --diff-file -`.
