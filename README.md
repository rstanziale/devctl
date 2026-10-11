# devctl

> A CLI built by and for developers

**Developer Environment Control CLI** - `devctl` is a command-line tool that helps developers manage their environment, including tools, services, and repositories. It provides commands to validate the environment, bootstrap repositories, and run tasks defined in a configuration file.

## Table of Contents
- [Installation](#installation)
- [Configuration](#configuration)
- [Commands](#commands)
- [Plugins](#plugins)
- [License](#license)
- [Authors](#authors)
- [Support](#support)

## Installation

Download the archive for your OS from the [releases page](https://github.com/rstanziale/devctl/releases):

| OS | Archive |
|----|---------|
| Windows 64-bit | `devctl-<VERSION>-windows-amd64.zip` |
| Linux 64-bit | `devctl-<VERSION>-linux-amd64.tar.gz` |

Extract the archive, open a terminal in the extracted directory, and run:

```powershell
# Windows
.\devctl.exe --help

# Linux
./devctl --help
```

No installation required - the binary is self-contained.

## Configuration

`devctl` reads a YAML configuration file that describes the tools, services, and repositories for your environment.

**Option A - Use the config provided by your team.**
Your team will share a `config.yaml` file. Place it in the `config` directory as `config.example.yaml` or pass its path via `--config`.

**Option B - Write your own config.**
Create a `config.yaml` file with the structure below. Add the schema directive at the top to enable validation and autocompletion in your editor:

```yaml
# yaml-language-server: $schema=./config.schema.json
version: 1
workspace: /path/to/your/workspace
concurrency_mode: parallel       # sequential | parallel | semaphore
max_concurrency: 5               # used only with semaphore mode

healthchecks:
  - name: GitHub
    url: https://github.com
    type: http                   # http | tcp
    timeout: 2s

tools:
  - name: Git
    command: git
  - name: Node
    command: node

repos:
  - name: Octo Template
    url: https://github.com/octo-org/octo-template
    dir: projects                # subdirectory inside workspace
    gitflow: false               # set true to initialize gitflow

tasks:
  - name: build-fe
    mode: parallel               # parallel | sequential
    subtasks:
      - name: app-fe-1
        dir: /dir/app-1
        commands:
          - npm install
          - npm run build
      - name: app-fe-2
        dir: /dir/app-2
        commands:
          - npm install
          - npm run build

settings:
  ai:
    enabled: true                # enable AI-powered commands
    engine: ollama               # AI engine to use
    ollama:
      model: gemma4              # Ollama model name
      base_url: http://localhost:11434
      timeout: 10m               # max wait for a response (default 5m)
```

### Configuration reference

| Field | Type | Description |
|-------|------|-------------|
| `version` | int | Must be `1` |
| `workspace` | string | Root directory where you are going to work |
| `concurrency_mode` | string | `sequential`, `parallel`, or `semaphore` |
| `max_concurrency` | int | Max parallel operations (semaphore mode only) |
| `healthchecks` | array | Connectivity checks (`http` or `tcp`) |
| `tools` | array | System tools to verify |
| `repos` | array | Repositories to clone |
| `tasks` | array | Generic tasks executed by `run` |
| `settings` | object | Global settings for devctl features |

AI settings (`settings.ai`):

| Field | Type | Description |
|-------|------|-------------|
| `enabled` | bool | Enable or disable AI-powered commands (default `false`) |
| `engine` | string | AI engine to use (`ollama`) |
| `ollama.model` | string | Ollama model name (e.g. `gemma4`) |
| `ollama.base_url` | string | Base URL of the Ollama HTTP API (e.g. `http://localhost:11434`) |
| `ollama.timeout` | string | Max duration to wait for a response (e.g. `10m`, default `5m`) |

AI-powered commands require an AI engine configured under `settings.ai` with `enabled: true`. Ollama must be running and reachable at the configured `base_url`.

Health check types:
- `http` - performs a GET request; fails on `4xx`/`5xx` responses
- `tcp` - opens a TCP connection to the configured host and port

## Commands

> Commands marked with ✨ leverage AI and require `settings.ai.enabled: true` in the configuration.

All commands share the following global flags:

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `./configs/config.yaml` | Path to config file |
| `--concurrency` | config value | `sequential`, `parallel`, or `semaphore` |
| `--max-concurrency` | config value | Max concurrent operations (semaphore mode) |
| `--dry-run` | `false` | Preview actions without making changes |
| `-v` / `-vv` | - | Increase log verbosity (debug / trace) |

### `doctor`

Validates the environment: checks the config file, verifies required tools are installed, and tests network connectivity.

```powershell
# Run with default config
devctl doctor

# Run with a custom config
devctl doctor --config /path/to/config.yaml

# Preview what would be checked
devctl doctor --dry-run

# Run checks sequentially
devctl doctor --concurrency sequential
```

### `bootstrap`

Clones all repositories defined in the config into the workspace directory.

```powershell
# Run with default config
devctl bootstrap

# Preview without cloning
devctl bootstrap --dry-run

# Clone with limited parallelism
devctl bootstrap --concurrency semaphore --max-concurrency 3
```

### `run`

Runs a configured task and executes all subtasks in the selected mode.

```powershell
# Run a task defined in config
devctl run --task build-fe

# Preview task execution without running commands
devctl run --task build-fe --dry-run
```

Task modes:
- `parallel` - executes subtasks concurrently (no deterministic order)
- `sequential` - executes subtasks one after another in declared order

### `ai test` ✨

Verifies that the configured AI engine is reachable by sending a short prompt and printing the response.

```powershell
# Test the AI engine connection
devctl ai test
```

### `ai log-analyzer` ✨

Scans a directory of logs (HAR, JSON) coming from different sources (client, gateway, server, ...), extracts the `WARN`/`ERROR` events within a time window, and asks the AI engine to correlate them and suggest fixes.

```powershell
# Analyze logs within a time window
devctl ai log-analyzer --dir /path/to/logs --start 20060102T15:04:05 --end 20060102T16:04:05
```

| Flag | Required | Description |
|------|----------|-------------|
| `--dir` | yes | Directory containing the log files to analyze |
| `--start` | yes | Start of the time window (format `20060102T15:04:05`) |
| `--end` | yes | End of the time window (format `20060102T15:04:05`) |

## Plugins

`devctl` supports external plugins - standalone binaries that are registered as subcommands at startup. This allows teams to integrate existing scripts or tools into `devctl` and have a single entry point for all terminal operations.

### Naming convention

Place plugin binaries in a `plugins/` directory next to the `devctl` binary. Each file must be named with the `devctl-` prefix followed by the command name:

| OS | Filename | Invocation |
|----|----------|------------|
| Linux / macOS | `devctl-<name>` | `devctl <name>` |
| Windows | `devctl-<name>.exe` | `devctl <name>` |

Multi-word command names are supported via hyphens: a binary named `devctl-init-project` is invoked as `devctl init-project`.

### Setup

**Linux / macOS** - the binary must have the executable bit set:

```bash
chmod +x plugins/devctl-init-project
devctl init-project --flag arg
```

**Windows** - the binary must have the `.exe` extension:

```powershell
# plugins\devctl-init-project.exe
devctl init-project --flag arg
```

### Behaviour

- Plugins are discovered at startup from the `plugins/` directory relative to the working directory. If the directory does not exist, `devctl` starts normally without errors.
- All arguments and flags are forwarded as-is to the plugin binary.
- The plugin's exit code is propagated to the calling shell.
- A plugin whose name conflicts with a built-in command (`doctor`, `bootstrap`, `run`, …) is skipped with a warning.

## Support

For issues or questions, please create an issue in the repository.
