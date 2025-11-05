# Repository Guidelines

## Project Structure & Module Organization
Runtime code centers on `app.go` and packages under `internal/` (router, webhook, stream, version). The bundled React UI lives in `ui/`, configuration templates stay at the repo root (`*.template`, `*.example`), and helper scripts sit in `scripts/`. Tests and fixtures are collected in `test/`, with build outputs funneled into `build/` by the Makefile.

## Build, Test, and Development Commands
Use the provided Make targets to stay consistent:
```bash
make build       # cross-compile Go binaries into build/
make build-js    # install UI deps and produce ui/dist
make check-go    # run golangci-lint with project config
make check-js    # lint + formatting checks for the UI
make test        # go test ./... with verbose output
```
For ad-hoc debugging, `go run ./app.go` starts the service using your local `hooks.*` config.

## Coding Style & Naming Conventions
Run `gofmt` (or `goimports`) before committing; Go files should keep tab indentation and follow mixedCaps naming without underscores. Prefer package-level doc comments only when behavior is non-obvious. UI code must pass the ESLint and Prettier rules baked into `yarn lint` and `yarn testformat`. Keep configuration template filenames camel-cased and update JSON/YAML variants together.

## Testing Guidelines
Unit tests live beside their packages (`*_test.go`), while end-to-end flows rely on fixtures in `test/`. Favor table-driven Go tests and reuse helpers from `testutils.go`. Cover new routes or webhook handlers before PRs and run `make test`. UI changes should ship with Jest or Testing Library coverage and a green `make check-js` run.

## Commit & Pull Request Guidelines
History follows Conventional Commits (e.g., `feat:`, `fix:`, `refactor:`) with concise, action-led summaries—bilingual snippets mirror current practice. Reference related issues in the body and call out config or schema migrations explicitly. Pull requests should outline scope, list verification steps, and include screenshots for UI tweaks; wait for passing CI and a reviewer from the affected area.

## Security & Configuration Tips
Never commit live secrets—refresh the `.template` and `.example` files instead and document overrides in `docs/`. Validate privilege changes with the drop-privilege stubs (`droppriv_*.go`) and list required capabilities in the PR summary. When adding webhook triggers, keep defaults restrictive and note firewall or TLS expectations for operators.
