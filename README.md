# raff

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Official command-line interface for the [Raff Cloud](https://rafftechnologies.com) platform. Built on top of [raff-go](https://github.com/RaffTechnologies/raff-go).

```bash
raff vm list
raff vm create --name web-01 --template-id <id> --pricing-id 3 --region us-east
raff vpc create --name prod --cidr 10.0.0.0/20
raff ip reserve --type ipv4
raff security-group create --name web --template-id allow-http
```

## Install

### Binary download (recommended)

Download the archive for your OS/arch from the [latest release](https://github.com/RaffTechnologies/raff-cli/releases/latest), extract `raff`, put it on your `PATH`.

```bash
# macOS (Apple Silicon)
curl -sL https://github.com/RaffTechnologies/raff-cli/releases/latest/download/raff_$(curl -sL https://api.github.com/repos/RaffTechnologies/raff-cli/releases/latest | grep tag_name | cut -d'"' -f4 | sed 's/v//')_macos_arm64.tar.gz | tar -xz
sudo mv raff /usr/local/bin/
```

Builds are published for **linux**, **macos**, **windows** × **amd64**, **arm64**.

### From `go install`

```bash
go install github.com/rafftechnologies/raff-cli/cmd/raff@latest
```

Requires Go 1.25+.

### From source

```bash
git clone https://github.com/RaffTechnologies/raff-cli.git
cd raff-cli
make install
```

## Quick start

```bash
# 1. Authenticate
raff configure
# (interactive prompt for API key — generate one in the dashboard at
#  Team & Projects → API Keys)

# 2. Pick a project to work in
raff project list
export RAFF_PROJECT_ID=<project-uuid>

# 3. Create a VM
raff vm create \
  --name web-01 \
  --template-id 5ac21891-32e6-41ce-8a93-b5d6ab708b0d \
  --pricing-id 3 \
  --region us-east \
  --ssh-keys "ssh-ed25519 AAAA..."

# 4. Manage it
raff vm list
raff vm stop <vm-id>
raff vm start <vm-id>
raff vm delete <vm-id>
```

## Commands

```
raff configure         Configure API credentials and defaults
raff project           Manage projects
raff vm                Manage virtual machines (create, list, start, stop, resize, ...)
raff vm vpc            Attach/detach a VM to a VPC
raff vm ip             Attach/detach floating IPs
raff vm sg             Attach/detach security groups
raff vm tags           Manage VM tags
raff vm notes          Manage VM notes
raff vpc               Manage VPCs
raff ip                Manage floating IPs (reserve, release, change)
raff security-group    Manage security groups (create, update, delete, templates)
```

Run `raff <cmd> --help` for full options on any command.

## Configuration

Config lives in `~/.raff/config.yaml`. Multiple profiles are supported:

```bash
raff configure                       # default profile
raff configure --profile staging     # named profile
raff vm list --profile staging
```

### Environment variables

| Variable | Description |
|----------|-------------|
| `RAFF_API_KEY` | API key (required) |
| `RAFF_API_URL` | API base URL (default `https://api.rafftechnologies.com`) |
| `RAFF_PROJECT_ID` | Default project for project-scoped commands |

### Precedence

CLI flag > environment variable > profile config file.

## Output formats

```bash
raff vm list                    # table (default)
raff vm list --output json      # JSON, for scripting
raff vm get <id> -o json | jq '.name'
```

## Versioning

Semantic versioning. v0.x is allowed to introduce breaking flag/output changes; v1.0.0+ implies a stable CLI surface. See [CHANGELOG](https://github.com/RaffTechnologies/raff-cli/releases) for per-release details.

## Contributing

```bash
make build       # build binary to bin/raff
make test        # run tests
go vet ./...
```

## License

[MIT](LICENSE)
