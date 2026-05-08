# raff-cli — Raff CLI Tool

Command-line interface for the Raff cloud platform. Built on `raff-go` client library.

## Critical Rules

1. **Built on raff-go** — never call the API directly. All API calls go through the `raff-go` client.
2. **Public API only** — follows `docs/api-reference/openapi.yaml`. Never expose admin/internal concepts.
3. **Sync order**: spec → raff-go → **raff-cli** → terraform-provider-raff
4. **No account-id needed** — account is derived from the API key.

## Structure

```
cmd/raff/           # Entry point
internal/
├── commands/       # Cobra command definitions
├── config/         # API key, endpoint config (env vars or config file)
└── output/         # Table + JSON output formatting
```

## Patterns

- **Cobra** framework for commands
- Commands: `raff vm list`, `raff vm create`, `raff project list`, etc.
- Output: table format (default) or `--output json`
- Config: `RAFF_API_KEY` env var or `~/.raff/config`

## Quick Commands

```bash
go build ./...     # Build
go test ./...      # Test
make build         # Build binary to bin/
```
