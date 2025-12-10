# mysql-user-migrate

Go CLI tool to migrate MySQL users (username, auth/password, host, privileges) from one source instance to one or many targets. Supports explicit include/exclude with wildcards, dry-run planning, overwrite controls, and post-run reports.

## Quick start
- Prereqs: Go 1.21+, network access to source/target MySQL.
- Install deps: `make deps`
- Run directly:  
  `go run ./cmd/mysql-user-migrate --source "user:pass@tcp(src:3306)/" --target "name=stg=user:pass@tcp(stg:3306)/" --include app_user --dry-run --report report.json`
- Run via config:  
  `go run ./cmd/mysql-user-migrate --config config.example.yaml`

## Key features
- Filtering: `--include user1,user2`, `--exclude root,test`; supports wildcards (`mysql.*`) and host patterns (`app@10.0.%`).
- Multi-target: repeat `--target` or define in config; supports one-to-many with `--concurrency`.
- Modes: `--dry-run` produces a plan/report only; default applies changes; `--drop-missing`/`--force-overwrite` control overwrite behavior.
- Reporting: terminal summary plus optional JSON via `--report`.
- Safety: DSN passwords are masked in logs/reports; root/system users not migrated unless explicitly included.

## Config file (YAML/JSON)
See `config.example.yaml`; common fields:
- `source`: source DSN
- `targets`: list of `{ name, dsn }`
- `include` / `exclude`
- `dry_run`, `drop_missing`, `force_overwrite`, `report_path`, `concurrency`, `verbose`

## Useful commands
- `make deps` install dependencies
- `make fmt` run gofmt/goimports
- `make lint` run golangci-lint (if installed)
- `make test` run unit tests
- `make run ARGS="--source ... --target ..."` run the CLI

## Environment variables
- `SOURCE_DSN`, `TARGET_DSN`, or `TARGET_DSN_LIST` (comma-separated) can provide DSNs.
- See `.env.example`; passwords are never logged in cleartext.

## Status and next steps
Current scope covers config/CLI parsing, source user load, target application, and reporting. Future improvements: full integration tests (Docker Compose MySQL), finer-grained privilege diffs, retries/recovery for partial failures.
