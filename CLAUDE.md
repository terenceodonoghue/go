# CLAUDE.md

## What is this?

A Go workspace with services in `services/*`. Module paths use `go.local/` as a non-resolvable TLD to prevent public publishing.

## Commands

- Build a service: `go build ./services/<name>/...`
- Run tests: `go test ./services/<name>/...`
- Regenerate sqlc: `cd services/<name> && sqlc generate` (output goes to `internal/db/`, do not edit generated files)

## Conventions

- Go 1.26+
- Services live in `services/`, each with its own `go.mod`
- After adding a service, add it to `go.work` locally (`go.work` is gitignored)
- Commit messages use imperative present tense (e.g., "Add feature", "Fix bug")

## Code style

Handled entirely by `gofmt` — do not manually enforce formatting rules.
