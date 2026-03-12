# Raff CLI

Command-line interface for managing [Raff](https://rafftechnologies.com) cloud resources.

## Installation

### From Source

```bash
git clone https://github.com/rafftechnologies/raff-cli.git
cd raff-cli
make install
```

Requires Go 1.25+.

## Quick Start

```bash
# Configure your credentials
raff configure

# List projects
raff project list

# Create a project
raff project create --name "my-project" --region us-east

# Get project details
raff project get <project-id>

# Update a project
raff project update <project-id> --name "new-name"

# Delete a project
raff project delete <project-id>
```

## Configuration

Config is stored in `~/.raff/config.yaml`. You can manage multiple profiles:

```bash
# Default profile
raff configure

# Named profile
raff configure --profile staging
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `RAFF_API_URL` | API base URL |
| `RAFF_API_KEY` | API key |
| `RAFF_PROJECT_ID` | Default project ID |

### Priority Order

CLI flag > environment variable > config file

## Global Flags

| Flag | Description |
|------|-------------|
| `--api-url` | API base URL |
| `--api-key` | API key |
| `--project-id` | Default project ID |
| `-o, --output` | Output format: `table` (default) or `json` |

## Output Formats

```bash
# Default table output
raff project list

# JSON output (for scripting)
raff project list --output json
```

## License

MIT
