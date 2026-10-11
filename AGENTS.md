# AGENTS.md

## Overview

`devctl` is a cross-platform **Developer Environment Control** CLI: one entry point to
validate, bootstrap, and operate a local dev environment from a declarative YAML config.
Built-in commands: `doctor` (env/tool/network checks), `bootstrap` (clone repos),
`run --task` (shell tasks), `ai` (Ollama-backed log analysis). External `devctl-<name>`
binaries are auto-registered as subcommands (plugin system).

## Tech Stack

- **Language:** Go `1.27.x` — module `github.com/rs-box/devctl`
- **CLI:** `spf13/cobra`
- **Config:** `spf13/viper` (YAML, schema-validated)
- **Logging:** stdlib `log/slog` (structured, `-v`/`-vv` verbosity)
- **Testing:** stdlib `testing` **only** — no testify/mock frameworks
- **Tooling:** GoReleaser (release), git-cliff (changelog), GitHub Actions (CI/CD)

## Project Structure

```
devctl/
├── main.go              # Entry point -> cmd.Execute(ctx)
├── cmd/                 # Cobra command layer (root, doctor, bootstrap, run, ai)
├── internal/            # Business logic (private packages)
│   ├── config/          # Config model + viper loader
│   ├── runner/          # Core: Command interface + concurrency engine
│   ├── doctor/          # Config / URL / tool checks
│   ├── bootstrap/       # Git repo cloning
│   ├── task/            # Task lookup + shell execution
│   ├── plugin/          # External plugin discovery & wrapping
│   ├── ai/              # AI client factory (Ollama) + log-analyzer
│   ├── logger/          # slog-based structured logger
│   └── timing/          # Command timing helper
├── configs/             # config.example.yaml, config.schema.json
├── plugins/             # External plugin binaries (gitignored)
└── scripts/             # Cross-platform build script
```

**Core pattern:** every multi-item operation (doctor, bootstrap, run) builds
`[]runner.Command` and executes through the shared `internal/runner` engine, which
supports `sequential` / `parallel` / `semaphore` concurrency strategies.

## Constraints

- **stdlib testing only.** Never introduce testify, gomock, or any external
  assertion/mock library. Use `testing`, `net/http/httptest`, `t.TempDir()`.
- **`internal/` is private.** Keep business logic there; never expose it as a public API.
- **Wrap errors** with `fmt.Errorf("context: %w", err)` — preserve the chain, assert with `errors.Is`.
- **Plugins** must follow the `devctl-<name>` naming convention and must never shadow a
  built-in command.
- **Conventional Commits** are mandatory (`feat`/`fix`/`refactor`/…); git-cliff derives the
  changelog from them.

## Commands

```bash
go build ./...                      # compile everything
go test ./...                       # run the full test suite
go run . doctor                     # validate the environment
go run . bootstrap                  # clone configured repos
go run . run --task <name>          # execute a task
```

Global flags: `--config`, `--concurrency`, `--max-concurrency`, `--dry-run`, `-v`/`-vv`.
Default config: `./configs/config.yaml`.

## Development Workflow — TDD

TDD applies to **`internal/` packages only.** `cmd/` is a thin Cobra wiring layer and
does not require a test suite.

Always work the **Red → Green → Refactor** cycle inside `internal/`:

1. **Red:** write a failing `*_test.go` that specifies the behavior *before* implementing.
2. **Green:** write the minimum code to make it pass.
3. **Refactor:** clean up while keeping tests green.

- Tests are **co-located** in the same package (white-box), named `*_test.go`.
- Prefer **table-driven tests** with `t.Run(name, ...)` subtests.
- Report failures with `t.Fatalf` / `t.Errorf`.

## Style / Conventions

- New multi-item operations implement the `runner.Command` interface
  (`Name() string`, `Run(ctx context.Context) error`).
- Each package logs through a component-scoped logger:
  `logger.GetLogger(logger.WithComponent("Name"))`.
- Action commands support `--dry-run` (preview without side effects).
- Comments explain **why**, not **what**. Error messages are actionable and specific.
- Packages: lowercase, single word, matching the directory name.
