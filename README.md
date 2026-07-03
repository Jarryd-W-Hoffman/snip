# snip

Lightning-fast CLI snippet manager — store, organise, and run frequently used shell commands from your terminal.

[![Go](https://github.com/Jarryd-W-Hoffman/snip/actions/workflows/test.yml/badge.svg)](https://github.com/Jarryd-W-Hoffman/snip/actions/workflows/test.yml)

## Quick start

```bash
# Save a command
snip save ssh-prod-db \
  --command 'gcloud compute ssh prod-db-01 --tunnel-through-iap --zone=us-central1-a' \
  --desc "SSH into production DB VM" \
  --tag gcp,iap,db

# List all snippets in the interactive TUI
snip list

# Run a snippet by name
snip run ssh-prod-db
```

## Installation

```bash
go install github.com/Jarryd-W-Hoffman/snip@latest
```

Or download a pre-built binary from the [releases page](https://github.com/Jarryd-W-Hoffman/snip/releases).

Requires Go 1.26+. The binary is a single static executable with no runtime dependencies (pure-Go SQLite).

## Usage

### `snip save <name>` — Save a snippet

```bash
snip save <name> --command "<shell command>" [--desc "<description>"] [--tag <tags>]
```

| Flag | Short | Required | Description |
|---|---|---|---|
| `--command` | `-c` | yes | The shell command to store |
| `--desc` | `-d` | no | A short description of what the command does |
| `--tag` | `-t` | no | Comma-separated tags for organising snippets |

**Examples:**

```bash
snip save docker-logs \
  --command 'docker logs -f --tail 100 my-service' \
  --desc "Tail production service logs" \
  --tag docker,prod

snip save iap-proxy \
  --command 'gcloud compute start-iap-tunnel my-instance 3389 --local-host-port=localhost:3390 --zone=us-east1-b' \
  --desc "RDP tunnel to dev VM" \
  --tag gcp,iap,tunnel
```

### `snip list` — Interactive TUI

Opens a full-screen Bubble Tea terminal UI for browsing, running, copying, and deleting snippets.

| Key | Action |
|---|---|
| `enter` | Run the selected snippet |
| `c` | Copy the command to your clipboard |
| `x` / `backspace` | Delete the snippet |
| `q` / `ctrl+c` | Quit the TUI |
| `/` | Filter/search snippets by name, description, or tags |

Snippets are displayed sorted by usage frequency — the commands you run most appear at the top.

When you run a snippet from the TUI, the command executes in your current shell, output streams in real time, and the TUI closes automatically when the command finishes. If the command fails, the error is shown before exiting.

### `snip run <name>` — Run a snippet directly

```bash
snip run ssh-prod-db
```

Executes the saved command in a subshell (bash on Linux/macOS, cmd on Windows). If the command contains `{{var}}` placeholders (see Variables below), you'll be prompted for each value interactively.

### `snip remove <name>` — Delete a snippet

```bash
snip remove ssh-prod-db
# Aliases: snip rm ssh-prod-db, snip delete ssh-prod-db
```

### `snip stats` — Usage analytics

Displays:

```
📊 SNIP TELEMETRY DASHBOARD
======================================
📦 Total Saved Snippets : 12
🚀 Total Executions      : 47
🎯 Active Rotation Ratio : 75%

🔥 Top 5 Most Frequently Used:
--------------------------------------
  ssh-prod-db     | Runs: 23  | SSH into production DB VM
  docker-logs     | Runs: 10  | Tail production service logs
  iap-ssh         | Runs: 8   | Generic IAP SSH tunnel
  deploy-api      | Runs: 4   | Deploy latest API build
  kubectl-pods    | Runs: 2   | List all pods in cluster
```

### `snip help` — Show help

Display the full help text with all available commands and flags.

## Variables and templates

Commands can contain `{{variable}}` placeholders that are substituted at runtime:

```bash
snip save ssh-iap \
  --command 'gcloud compute ssh {{vm}} --tunnel-through-iap --zone={{zone}} --project={{project}}' \
  --desc "Generic IAP SSH tunnel" \
  --tag gcp,iap

snip run ssh-iap
# 📋 Snippet 'ssh-iap' requires context variable values:
#
# ➡️ Enter value for [vm]: prod-api-02
# ➡️ Enter value for [zone]: us-central1-a
# ➡️ Enter value for [project]: my-production-project
```

The TUI also supports placeholders — selecting a snippet with `{{var}}` patterns opens an inline prompt for each variable before execution.

## Real-world example: GCP IAP tunnels

This project was built to manage the many `gcloud compute` commands needed for daily work with GCP VMs through IAP. Instead of remembering intricate zone, project, and instance names:

```bash
# Save your most-used tunnels
snip save iap-dev-db \
  --command 'gcloud compute start-iap-tunnel dev-db-01 3306 --local-host-port=localhost:3307 --zone=us-west2-a --project=dev-project' \
  --desc "MySQL tunnel to dev database" \
  --tag gcp,mysql,dev,iap

snip save iap-prod-web \
  --command 'gcloud compute ssh prod-web-01 --tunnel-through-iap --zone=us-east1-b --project=prod-project -L 8080:localhost:8080' \
  --desc "SSH + port forward to prod web server" \
  --tag gcp,ssh,prod,iap

# Quick access — blur out the name and run
snip run iap-dev-db

# Or browse visually
snip list
```

Combined with template variables, one generic snippet can handle every VM:

```bash
snip save iap-ssh \
  --command 'gcloud compute ssh {{instance}} --tunnel-through-iap --zone={{zone}} --project={{project}}' \
  --tag gcp,iap,template

snip run iap-ssh
# ➡️ Enter value for [instance]: prod-web-01
# ➡️ Enter value for [zone]: us-east1-b
# ➡️ Enter value for [project]: prod-project
```

## Storage

Snippets are stored in a local SQLite database at:

- **Linux:** `~/.config/snip/snippets.db`
- **macOS:** `~/Library/Application Support/snip/snippets.db`
- **Windows:** `%AppData%/snip/snippets.db`

The database is created automatically on first use. No external services, no network calls, no cloud sync — everything stays on your machine.

## Development

```bash
# Build
go build -o snip .

# Test (pure Go, no CGO required)
go test ./...

# Vet
go vet ./...
```

### Test structure

- `storage/sqlite_test.go` — In-memory SQLite tests for all CRUD operations
- `cmd/exec_test.go` — Unit tests for template parsing and substitution
- `cmd/remove_test.go`, `cmd/save_test.go`, `cmd/stats_test.go`, `cmd/run_test.go` — Command-level tests using isolated temp directories

## License

MIT
