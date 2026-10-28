# Security policy

canon runs linters on your code and, as an MCP server, takes requests from
agents, so we take reports seriously.

## Supported versions

| Version | Supported |
| --- | --- |
| Latest release (`v0.x`) | Yes |
| Older releases | No; upgrade to the latest |

Until 1.0, only the most recent release gets fixes.

## Reporting a vulnerability

Report privately through GitHub's private vulnerability reporting:
<https://github.com/rossedman/canon/security/advisories/new>. Please do not
open a public issue.

Include the canon version (`canon version`), how you ran it (CLI, `canon
serve`, `canon hook` or CI), and the smallest input that shows the problem.

What to expect:

- An acknowledgement within 3 business days.
- A fix, or a plan with a timeline, within 30 days.
- Credit in the published advisory, unless you prefer to stay anonymous.

Things worth reporting include a path that lets an MCP client read or write
outside the repository root, a scaffold that writes outside its destination,
and a way to make canon run a binary other than the linters it is configured
for.

## Verifying releases

Every release's `checksums.txt` is signed keylessly with cosign by this
repo's release workflow. Once the repository is public, the archives also
carry GitHub build provenance, checked by the `gh attestation verify` line.
Download an archive, `checksums.txt` and `checksums.txt.sigstore.json`, then:

```sh
cosign verify-blob checksums.txt \
  --bundle checksums.txt.sigstore.json \
  --certificate-identity-regexp '^https://github.com/rossedman/canon/.github/workflows/release.yml@refs/tags/v' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
sha256sum --ignore-missing -c checksums.txt
gh attestation verify canon_<version>_linux_amd64.tar.gz --repo rossedman/canon
```

A signature from any other repository or workflow fails the first command.
