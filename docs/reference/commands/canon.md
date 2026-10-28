## canon

Engineering standards as tools an agent can call

### Synopsis

canon turns engineering opinions into tools: check code against rules,
answer "how do we do X" from decision records, review diffs, and scaffold
projects that already follow the standards. It runs as an MCP server
(canon serve), a CLI, and a Claude Code hook (canon hook).

### Options

```
      --canon-dir string   load the canon from this directory instead of the embedded one
      --config string      config file (default: first of $XDG_CONFIG_HOME/canon/config.yaml, ~/.canon.yaml, ./.canon.yaml)
  -h, --help               help for canon
      --log-level string   log level: debug, info, warn or error (default "warn")
      --root string        repository root that paths are relative to (default ".")
```

### SEE ALSO

* [canon check](canon_check.md)	 - Check files against the canon and report findings
* [canon decide](canon_decide.md)	 - Answer a "how do we do X" question from decision records
* [canon explain](canon_explain.md)	 - Show the record behind a rule or finding, and what enforces it
* [canon hook](canon_hook.md)	 - Run as a Claude Code or git hook on the files just changed
* [canon lint](canon_lint.md)	 - Validate the canon: records, rules, profiles and templates
* [canon profile](canon_profile.md)	 - Detect the repo's profile, the records in force and missing checkers
* [canon review](canon_review.md)	 - Review a change: findings on changed files plus concerns linters miss
* [canon scaffold](canon_scaffold.md)	 - Generate a project or component that already follows the canon
* [canon serve](canon_serve.md)	 - Run the MCP server over stdio, or streamable HTTP with --http
* [canon version](canon_version.md)	 - Print the version, commit and build date

