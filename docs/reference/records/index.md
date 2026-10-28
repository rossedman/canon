# Decision records

Every rule canon enforces traces to one of these records. There are 25: 24 accepted, and 1 kept for history.

## Go

| Record | Strength | Status |
| --- | --- | --- |
| [Terminal UX uses the Charmbracelet libraries](go-charmbracelet-ux.md) | should | accepted |
| [One cobra command per file in cmd/](go-cobra-one-command-per-file.md) | must | accepted |
| [CLI configuration resolves flag, then env, then file](go-config-viper.md) | should | accepted |
| [Every function and method has a doc comment](go-doc-comments.md) | should | accepted |
| [Wrap errors with %w and context; never log and return](go-errors-wrap.md) | must | accepted |
| [Critical paths have benchmarks, and input parsers have fuzz targets](go-fuzz-critical-paths.md) | should | accepted |
| [Every Go repo runs the golangci-lint baseline](go-lint-baseline.md) | must | accepted |
| [HTTP APIs are spec-first with OpenAPI and generated code](go-openapi-spec-first.md) | must | accepted |
| [Hand-written Go code keeps 90% test coverage](go-test-coverage.md) | must | accepted |

## Python

| Record | Strength | Status |
| --- | --- | --- |
| [Python is linted and formatted with ruff](python-ruff.md) | must | accepted |
| [Python projects are managed with uv and a committed lock file](python-uv.md) | should | accepted |

## Containers

| Record | Strength | Status |
| --- | --- | --- |
| [Use alpine for small runtime images](containers-alpine-base.md) | should | superseded |
| [Use distroless or scratch for runtime images](containers-distroless-base.md) | must | accepted |
| [Dockerfiles follow hadolint's best practices](containers-dockerfile-hygiene.md) | should | accepted |
| [Containers run as a non-root user](containers-non-root.md) | must | accepted |
| [Pin every base image by tag and digest](containers-pin-base-images.md) | must | accepted |

## Terraform

| Record | Strength | Status |
| --- | --- | --- |
| [Terraform modules are typed, documented and linted](terraform-module-conventions.md) | should | accepted |
| [Terraform passes a security scan before merge](terraform-security-scan.md) | must | accepted |
| [Pin Terraform and provider versions with bounded constraints](terraform-version-constraints.md) | must | accepted |

## YAML

| Record | Strength | Status |
| --- | --- | --- |
| [YAML is linted with yamllint](yaml-lint.md) | should | accepted |

## CI

| Record | Strength | Status |
| --- | --- | --- |
| [Workflows declare least-privilege token permissions](ci-least-privilege.md) | must | accepted |
| [Pin third-party GitHub Actions to a full commit SHA](ci-pin-actions-sha.md) | must | accepted |

## Open source

| Record | Strength | Status |
| --- | --- | --- |
| [Open-source repos ship the standard community files](oss-community-files.md) | should | accepted |
| [Commits and PR titles follow Conventional Commits](oss-conventional-commits.md) | should | accepted |
| [Release assets are signed keylessly with cosign](oss-release-signing.md) | must | accepted |
