# raff

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/rafftechnologies/raff-cli.svg)](https://pkg.go.dev/github.com/rafftechnologies/raff-cli)

```
raff is a command-line interface (CLI) for the Raff Cloud API.

Usage:
  raff [command]

Available Commands:
  completion     Generate the autocompletion script for the specified shell
  configure      Configure API credentials and defaults
  help           Help about any command
  ip             Manage floating IPs
  project        Manage projects
  security-group Manage security groups
  vm             Manage virtual machines
  vpc            Manage virtual private clouds

Flags:
      --api-key string      API key (overrides config)
      --api-url string      API base URL (overrides config)
  -h, --help                help for raff
  -o, --output string       Output format: table or json (default "table")
      --project-id string   Default project ID (overrides config)
  -v, --version             version for raff

Use "raff [command] --help" for more information about a command.
```

See the [full reference documentation](https://docs.rafftechnologies.com) for information about each available command.

- [Installing `raff`](#installing-raff)
  - [Downloading a Release from GitHub](#downloading-a-release-from-github)
  - [Building the Development Version from Source](#building-the-development-version-from-source)
- [Authenticating with Raff](#authenticating-with-raff)
  - [Logging into multiple Raff accounts](#logging-into-multiple-raff-accounts)
- [Configuring Default Values](#configuring-default-values)
  - [Environment Variables](#environment-variables)
- [Enabling Shell Auto-Completion](#enabling-shell-auto-completion)
  - [Linux Auto Completion](#linux-auto-completion)
  - [macOS Auto Completion](#macos-auto-completion)
- [Uninstalling `raff`](#uninstalling-raff)
- [Examples](#examples)
- [Documentation](#documentation)
- [Related projects](#related-projects)

## Installing `raff`

### Downloading a Release from GitHub

Visit the [Releases page](https://github.com/RaffTechnologies/raff-cli/releases) and find the appropriate archive for your operating system and architecture. Download the archive from your browser, or copy its URL and retrieve it with `wget` or `curl`.

For example, with `curl`:

```bash
cd ~
curl -OL https://github.com/RaffTechnologies/raff-cli/releases/download/v<version>/raff_<version>_linux_x86_64.tar.gz
```

Extract the binary:

```bash
tar xf ~/raff_<version>_linux_x86_64.tar.gz
```

Or download and extract in one line:

```bash
curl -sL https://github.com/RaffTechnologies/raff-cli/releases/download/v<version>/raff_<version>_linux_x86_64.tar.gz | tar -xzv
```

Where `<version>` is the full semantic version, e.g. `0.1.0`. Builds are published for **linux**, **macos**, **windows** × **amd64**, **arm64**.

Move the `raff` binary somewhere on your `PATH`. For example, on Linux/macOS:

```bash
sudo mv ~/raff /usr/local/bin
```

Windows users should follow [How to: Add Tool Locations to the PATH Environment Variable](https://learn.microsoft.com/en-us/previous-versions/office/developer/sharepoint-2010/ee537574(v=office.14)) to add `raff` to their `PATH`.

### Building the Development Version from Source

If you have a [Go environment](https://go.dev/doc/install) configured, you can install the development version of `raff` from the command line:

```bash
go install github.com/rafftechnologies/raff-cli/cmd/raff@latest
```

While the development version is a good way to take a peek at the latest features before they're released, be aware that it may have bugs. Officially released versions will generally be more stable.

Or build from a clone:

```bash
git clone https://github.com/RaffTechnologies/raff-cli.git
cd raff-cli
make build       # produces ./bin/raff
make install     # installs into $GOBIN
```

Requires Go 1.25+.

## Authenticating with Raff

To use `raff`, you need to authenticate with Raff by providing an API key. Generate one in the dashboard at [rafftechnologies.com](https://rafftechnologies.com) under **Team & Projects → API Keys**.

Authenticate with the `configure` command:

```bash
raff configure
```

You'll be prompted to enter the API key:

```
Raff API key: raff_pub_xxx
```

After entering the key, the credentials are validated against the API. If the token doesn't validate, double-check that you copied it correctly.

```
Validating key: OK
```

This creates the necessary directory structure and configuration file to store your credentials at `~/.raff/config.yaml`.

### Logging into multiple Raff accounts

`raff` supports multiple authentication profiles so you can switch between accounts (or between staging/production keys) without re-entering credentials.

By default, a profile named `default` is used. To create a new profile:

```bash
raff configure --profile staging
```

Then pass the profile name to any `raff` command:

```bash
raff vm list --profile staging
```

The `--api-key` flag and `RAFF_API_KEY` environment variable take precedence over any profile, so you can also override credentials per-command without changing your active profile.

## Configuring Default Values

The `raff` configuration file stores your API key and default values for command flags. The file is created automatically the first time you run `raff configure`.

| OS | Config path |
|----|-------------|
| Linux | `~/.raff/config.yaml` |
| macOS | `~/.raff/config.yaml` |
| Windows | `%USERPROFILE%\.raff\config.yaml` |

Example file with two profiles:

```yaml
current-profile: default
profiles:
  default:
    api-url: https://api.rafftechnologies.com
    api-key: raff_pub_xxx
    project-id: 11111111-2222-3333-4444-555555555555
  staging:
    api-url: https://api.rafftechnologies.com
    api-key: raff_pub_yyy
```

### Environment Variables

In addition to the config file, you can override values per-session with environment variables:

| Variable | Description |
|----------|-------------|
| `RAFF_API_KEY` | API key (`--api-key`) |
| `RAFF_API_URL` | API base URL (`--api-url`) |
| `RAFF_PROJECT_ID` | Default project ID (`--project-id`) |

**Precedence:** CLI flag > environment variable > profile config file.

## Enabling Shell Auto-Completion

`raff` supports shell tab-completion for commands, subcommands, and flags. Generate a completion script with `raff completion <shell>`:

```bash
raff completion bash       # for bash
raff completion zsh        # for zsh
raff completion fish       # for fish
raff completion powershell # for PowerShell
```

### Linux Auto Completion

The most common way is to source the completion script from your shell config. For bash, add this to `~/.bashrc` (or `~/.profile`):

```bash
source <(raff completion bash)
```

For zsh, add this to `~/.zshrc`:

```bash
source <(raff completion zsh)
compdef _raff raff
```

Then reload:

```bash
source ~/.bashrc   # or ~/.zshrc
```

### macOS Auto Completion

macOS users using bash will have to install the `bash-completion` framework first:

```bash
brew install bash-completion
```

Then add to `~/.bash_profile` or `~/.bashrc`:

```bash
source $(brew --prefix)/etc/bash_completion
source <(raff completion bash)
```

For zsh on macOS (the default shell since Catalina), add to `~/.zshrc`:

```bash
autoload -U +X compinit; compinit
source <(raff completion zsh)
```

Then reload your shell.

## Uninstalling `raff`

If you installed `raff` by downloading a release archive:

```bash
sudo rm /usr/local/bin/raff
```

If you installed via `go install`, remove the binary from `$GOBIN`:

```bash
rm $(go env GOBIN)/raff
# or, if GOBIN is unset, $GOPATH/bin/raff
```

To completely remove the configuration:

```bash
rm -rf ~/.raff
```

## Examples

`raff` can interact with all Raff resources. Below are common usage examples.

- **List all VMs in the current project:**
  ```bash
  raff vm list
  ```

- **Create a VM:**
  ```bash
  raff vm create \
    --name web-01 \
    --template-id 5ac21891-32e6-41ce-8a93-b5d6ab708b0d \
    --pricing-id 3 \
    --region us-east \
    --ssh-keys "ssh-ed25519 AAAA..."
  ```

- **Stop, start, reboot a VM:**
  ```bash
  raff vm stop <vm-id>
  raff vm start <vm-id>
  raff vm reboot <vm-id>
  ```

- **Resize a VM (change pricing plan or grow disk):**
  ```bash
  raff vm resize <vm-id> --pricing-id 5
  raff vm resize-disk <vm-id> --new-size 100
  ```

- **Create a VPC:**
  ```bash
  raff vpc create --name prod --cidr 10.0.0.0/20 --region us-east
  ```

- **Reserve a floating IP and attach it to a VM:**
  ```bash
  raff ip reserve --type ipv4 --region us-east
  raff vm ip attach <vm-id> --ip-id <ip-id>
  ```

- **Create a security group from a template:**
  ```bash
  raff security-group templates                          # list templates
  raff security-group create --name web --template-id allow-http
  raff vm sg attach <vm-id> --sg-id <sg-id> --nic-id 0
  ```

- **JSON output for scripting:**
  ```bash
  raff vm list --output json | jq '.[].name'
  ```

## Documentation

- **CLI reference** — `raff <command> --help` (every command and flag)
- **API reference** — [docs.rafftechnologies.com](https://docs.rafftechnologies.com)
- **Dashboard** — [rafftechnologies.com](https://rafftechnologies.com)
- **Releases / changelog** — [github.com/RaffTechnologies/raff-cli/releases](https://github.com/RaffTechnologies/raff-cli/releases)

## Related projects

- [raff-go](https://github.com/RaffTechnologies/raff-go) — official Go SDK that powers this CLI
- [terraform-provider-raff](https://github.com/RaffTechnologies/terraform-provider-raff) — official Terraform provider

## Contributing

```bash
make build       # build binary to bin/
make test        # run tests
go vet ./...
```

PRs welcome. Please match the existing command patterns (see [internal/commands/](internal/commands/) for examples).

## License

[MIT](LICENSE)
