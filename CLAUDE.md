# CLAUDE.md

## What is this?

A single Go module (`go.local`) with services in `services/*` and shared packages in `pkg/*`. The module path uses `go.local/` as a non-resolvable TLD to prevent public publishing.

## Commands

- First-time setup: `make setup` (installs Homebrew tools and pre-commit hooks)
- Build a service: `go build ./services/<name>/...`
- Run tests: `go test ./services/<name>/...`
- Regenerate sqlc: `cd services/<name> && sqlc generate` (output goes to `internal/db/`, do not edit generated files)

## Conventions

- Go 1.26+
- Services live in `services/`, shared packages in `pkg/`
- Commit messages use imperative present tense (e.g., "Add feature", "Fix bug")

## CI

Per-service workflows — each runs security scan (Gitleaks, CodeQL), Docker build, Trivy image scan, and publish to ghcr.io on push to main. Both also trigger on shared code changes (`pkg/**`, `go.mod`).

- `auth-api.yml` — path-filtered to `services/auth-api/**`
- `fron-svc.yml` — path-filtered to `services/fron-svc/**`

## Code style

Handled entirely by `gofmt` — do not manually enforce formatting rules.
