# raff

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/rafftechnologies/raff-cli.svg)](https://pkg.go.dev/github.com/rafftechnologies/raff-cli)

```
Raff CLI — manage cloud resources from the terminal

Usage:
  raff [command]

Available Commands:
  api-key        Manage API keys
  backup         Manage VM backups (incl. `raff backup schedule`)
  completion     Generate the autocompletion script for the specified shell
  configure      Configure API credentials and defaults
  help           Help about any command
  invitation     Send and cancel email invitations
  ip             Manage floating IPs
  member         Manage account-level members
  permission     List the permission catalog
  pricing        Browse the public pricing catalog
  project        Manage projects (incl. `raff project member`)
  region         List datacenter regions
  role           Manage IAM roles
  security-group Manage security groups
  snapshot       Manage VM and volume snapshots
  ssh-key        Manage SSH keys
  template       List OS templates
  version        Show the current version
  vm             Manage virtual machines
  volume         Manage block storage volumes
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
  - [Switching between multiple profiles](#switching-between-multiple-profiles)
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

Visit the [Releases page](https://github.com/RaffTechnologies/raff-cli/releases) for archives matching your OS / architecture. Builds are published for **linux**, **macos**, **windows** × **amd64**, **arm64**.

#### Quickest — auto-detect the latest version

```bash
# Linux x86_64
curl -sL "https://api.github.com/repos/RaffTechnologies/raff-cli/releases/latest" \
  | grep "browser_download_url.*linux_x86_64.tar.gz" \
  | cut -d'"' -f4 | xargs curl -sL | tar -xzv
sudo mv raff /usr/local/bin/
```

Replace `linux_x86_64` with `linux_arm64`, `macos_x86_64`, or `macos_arm64` as needed.

#### Pinning to a specific version

Replace `0.3.0` below with the version you want from the [Releases page](https://github.com/RaffTechnologies/raff-cli/releases).

```bash
# Linux x86_64
curl -sL https://github.com/RaffTechnologies/raff-cli/releases/download/v0.3.0/raff_0.3.0_linux_x86_64.tar.gz | tar -xzv
sudo mv raff /usr/local/bin/
```

```bash
# macOS arm64 (Apple Silicon)
curl -sL https://github.com/RaffTechnologies/raff-cli/releases/download/v0.3.0/raff_0.3.0_macos_arm64.tar.gz | tar -xzv
sudo mv raff /usr/local/bin/
```

For Windows, download the matching `.zip` from the Releases page in your browser, extract `raff.exe`, then follow [How to: Add Tool Locations to the PATH Environment Variable](https://learn.microsoft.com/en-us/previous-versions/office/developer/sharepoint-2010/ee537574(v=office.14)) to put it on your `PATH`.

Verify the install:

```bash
raff --version
```

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

To use `raff`, you need an API key. Generate one in the dashboard at [rafftechnologies.com](https://rafftechnologies.com) under **Team & Projects → API Keys**.

Run the `configure` command to set up a profile:

```bash
raff configure
```

You'll be prompted for three fields, each with the existing value as the default (just press Enter to keep it):

```
API URL [https://api.rafftechnologies.com]:
API Key [...]: raff_pub_xxx
Default Project ID (optional) [...]: 11111111-2222-3333-4444-555555555555
```

When you finish, you'll see:

```
Profile "default" saved to /home/you/.raff/config.yaml
```

The active profile becomes the one you just configured. Subsequent `raff` commands read credentials from this file.

> **Note:** `configure` does not currently validate the key against the API — it just saves to disk. The first real call (e.g. `raff project list`) will surface an auth error if the key is wrong.

### Switching between multiple profiles

`raff` supports multiple profiles in the same config file so you can keep separate keys (e.g. staging vs production):

```bash
raff configure                    # writes/updates the "default" profile
raff configure --profile staging  # writes/updates the "staging" profile and makes it active
raff configure --profile default  # switch back to "default" (just press Enter through the prompts)
```

Other commands (`vm list`, `vpc list`, …) do **not** take a `--profile` flag — they always use whichever profile is set as `current-profile` in `~/.raff/config.yaml`. To use a non-active profile for a single command, override with environment variables:

```bash
RAFF_API_KEY=raff_pub_yyy RAFF_PROJECT_ID=<staging-project-uuid> raff vm list
```

Or with command-line flags:

```bash
raff vm list --api-key raff_pub_yyy --project-id <staging-project-uuid>
```

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

If you installed via `go install` or `make install`, remove the binary from your Go bin directory:

```bash
rm "$(go env GOPATH)/bin/raff"
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
  raff security-group templates                          # list available template IDs
  raff security-group create --name web --template-id web-server
  raff vm sg attach <vm-id> --security-group-id <sg-id> --nic-id 0
  ```

- **Register an SSH key from a file (the typical bootstrap path):**
  ```bash
  raff ssh-key create --name laptop --public-key-file ~/.ssh/id_ed25519.pub
  ```

- **Create an API key for CI (secret printed once — copy it immediately):**
  ```bash
  raff api-key create --name "production-ci" --rate-limit-tier standard
  ```

- **Invite a teammate to a project:**
  ```bash
  raff role list --scope project                            # find the right role UUID
  raff invitation create-project \
    --project-id <project-uuid> \
    --email teammate@example.com \
    --role-id <project-role-uuid>
  ```

- **Add an existing account user to another project:**
  ```bash
  raff project member add <project-id> \
    --target-user-id <user-uuid> \
    --role-id <project-role-uuid>
  ```

- **Browse the permission catalog when designing a custom role:**
  ```bash
  raff permission list --scope project
  raff role create --name "vm-operator" --slug vm-operator --scope project \
    --permission vm.read --permission vm.start --permission vm.stop
  ```

- **Attach a new volume to a VM at create time:**
  ```bash
  raff volume create \
    --name data-vol \
    --size 100 --type standard --region us-east \
    --vm-id <vm-uuid>
  ```

- **Snapshot a VM before a risky change, restore later:**
  ```bash
  raff snapshot create --resource-type vm --resource-id <vm-uuid> --name pre-upgrade
  raff snapshot restore <snapshot-id>      # prompts for confirmation
  ```

- **Set up daily backups on a VM, keep last 14:**
  ```bash
  raff backup schedule create \
    --vm-id <vm-uuid> \
    --frequency daily --time 03:00 --keep 14
  ```

- **Browse VM pricing, find the cheapest plan in a region:**
  ```bash
  raff pricing vm --region us-east --output json | jq 'min_by(.monthly_price)'
  raff template list --category linux --vm-type standard
  raff region list
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
