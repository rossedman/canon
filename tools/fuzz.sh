#!/usr/bin/env bash
# Run every Fuzz* target in the module for the given time (default 30s).
set -euo pipefail

fuzztime="${1:-30s}"
status=0
while read -r pkg; do
  dir=$(go list -f '{{.Dir}}' "$pkg")
  for target in $(go test -list '^Fuzz' "$pkg" | grep '^Fuzz' || true); do
    echo "fuzz $pkg $target for $fuzztime"
    if ! (cd "$dir" && go test -run='^$' -fuzz="^${target}\$" -fuzztime="$fuzztime" .); then
      status=1
    fi
  done
done < <(go list ./...)
exit "$status"
