# xmnemo

A build tool for compiling a custom `mnemo` binary with the agent adapters you choose. Follows the same pattern as [xcaddy](https://github.com/caddyserver/xcaddy) for [Caddy](https://caddyserver.com).

> **This is a developer tool.** Most mnemo users never need it — install the official binary via `install.sh` instead.

## When to use this

Use `xmnemo` when you want to:

- Include a community or proprietary adapter not shipped with the official binary
- Build a mnemo binary with only the adapters your team needs
- Develop and test your own adapter against a real mnemo build

## Requirements

- Go 1.22 or later
- Internet access (resolves adapter modules from the Go module proxy)

## Installation

```sh
go install github.com/jmeiracorbal/xmnemo@latest
```

## Usage

```sh
xmnemo build --adapter <module[@version]>
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--adapter` | *(required, repeatable)* | Adapter module to include (`module[@version]`) |
| `--mnemo-version` | `latest` | mnemo core version to build against |
| `--output` | `~/.local/bin/mnemo` | Output path for the compiled binary |
| `--no-builtins` | `false` | Exclude the built-in adapters (Claude Code, Codex, Cursor, OpenCode, Pi) |

### Examples

```sh
# Add a single community adapter
xmnemo build --adapter github.com/acme/mnemo-zed@v1.2.0

# Multiple adapters
xmnemo build --adapter github.com/acme/mnemo-zed --adapter github.com/corp/mnemo-internal

# Only your adapter, no built-ins
xmnemo build --adapter github.com/acme/mnemo-zed --no-builtins --output ~/bin/mnemo
```

## Writing an adapter

An adapter is a Go module that blank-imports `github.com/jmeiracorbal/mnemo-adapters/agents` and registers itself via its `init()` function. See [mnemo-adapters](https://github.com/jmeiracorbal/mnemo-adapters) for the agent contract and existing built-in adapters as reference.

## How it works

`xmnemo build` generates a temporary Go workspace that blank-imports the mnemo core alongside the requested adapters, runs `go build`, and places the resulting binary at the output path. The temporary workspace is discarded after the build.
