# Repository Guidelines

## Project Structure & Module Organization
Runtime code lives in `cmd/mysql-user-migrate/main.go` (CLI entry) and `internal/` packages: `db/` for MySQL connections and grant readers, `migrate/` for diff/apply logic, and `cli/` for flag parsing and output. Keep shared helpers in `pkg/` only if they should be imported by other binaries. Store SQL fixtures and sample grants in `fixtures/`. Keep env examples in `.env.example`. Tests live in `tests/` for integration (MySQL container) and alongside code for unit tests. `scripts/` holds operational helpers (start db, seed data).

## Build, Test, and Development Commands
- `make deps` → run `go mod tidy` and install tools (e.g., `golangci-lint`).
- `make fmt` → run `gofmt -w` and `goimports` on `cmd/`, `internal/`, `pkg/`, and `tests/`.
- `make lint` → `golangci-lint run ./...`.
- `make test` → `go test ./...` with `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASS`, `DB_NAME` overrides.
- `make itest` → spin up a MySQL container (e.g., `docker compose -f docker-compose.test.yml up --build --abort-on-container-exit`).
- `make run ARGS="--source ... --target ..."` → `go run ./cmd/mysql-user-migrate $(ARGS)`.

## Coding Style & Naming Conventions
Use Go 1.21+ modules; run `gofmt`/`goimports` before commits. Keep packages small and capability-named (`db`, `migrate`, `cli`, `auth`). Prefer context-aware functions and pass `context.Context` to DB calls. Avoid globals; inject dependencies via interfaces for testability. Keep SQL filenames snake_case. Config keys and env vars are upper snake (`SOURCE_DB_URI`, `TARGET_DB_URI`). Never log passwords; redact sensitive fields.

## Testing Guidelines
Write table-driven unit tests with `*_test.go`; favor fakes for DB I/O. Assert idempotency of migrations and privilege parity. Integration tests should start their own MySQL container and clean up after. Include seed SQL in `fixtures/` and use distinct test schemas. Run `go test -race ./...` on changes touching concurrency or connection pooling.

## Commit & Pull Request Guidelines
Follow Conventional Commits (`feat:`, `fix:`, `chore:`, `docs:`, `refactor:`). Keep scope small and behavioral. PRs should list the change summary, before/after behavior, `make lint`/`make test` results, and any schema/config impacts. Attach CLI output snippets when UX changes.

## Security & Configuration Tips
Use `.env.example`; never commit credentials. Default to least-privilege MySQL users; avoid `GRANT ALL` outside disposable test containers. Provide explicit `--dry-run` and `--apply` modes to reduce risk. Mask default hosts/ports so contributors do not point at production by accident.
