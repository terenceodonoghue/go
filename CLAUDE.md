# CLAUDE.md

## What is this?

A single Go module (`go.local`) with services in `services/*` and shared packages in `pkg/*`. The module path uses `go.local/` as a non-resolvable TLD to prevent public publishing.

## Commands

- Build a service: `go build ./services/<name>/...`
- Run tests: `go test ./services/<name>/...`
- Regenerate sqlc: `cd services/<name> && sqlc generate` (output goes to `internal/db/`, do not edit generated files)

## Conventions

- Go 1.26+
- Services live in `services/`, shared packages in `pkg/`
- Commit messages use imperative present tense (e.g., "Add feature", "Fix bug")

## Code style

Handled entirely by `gofmt` — do not manually enforce formatting rules.
