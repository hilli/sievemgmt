# sievemgmt

A small CLI for managing [Sieve](https://en.wikipedia.org/wiki/Sieve_(mail_filtering_language))
scripts on a remote mail server over the **ManageSieve** protocol (RFC 5804).

It supports multiple accounts via a YAML config, server-side script validation,
and an `edit` workflow that opens your `$EDITOR`, validates on save, and uploads.

## Install

```sh
go install github.com/hilli/sievemgmt@latest
```

Or build from source:

```sh
go build -o sievemgmt .
```

## Configuration

Accounts are read from two locations and **merged**, with the local file
overriding the user file on a per-field basis:

1. `~/.config/sievemgmt/sievemgmt.yaml`
2. `./sievemgmt.yaml` (current directory)

```yaml
primary:
  email: you@example.com
  password: your-password
  server: mail.example.com     # optional ":port"; default is 4190 (with SRV lookup)
work:
  email: you@work.example.com
  password: your-work-password
  server: mail.work.example.com:4190
```

> **Security:** the config holds plaintext passwords. Restrict its permissions:
> `chmod 600 ~/.config/sievemgmt/sievemgmt.yaml`

### Selecting an account

Precedence (highest first):

1. `--account` / `-a` flag
2. `SIEVEMGMT_ACCOUNT` environment variable
3. the **first** account in the config file

```sh
sievemgmt -a work list
SIEVEMGMT_ACCOUNT=work sievemgmt list
```

## Commands

| Command                       | Description                                              |
| ----------------------------- | -------------------------------------------------------- |
| `accounts`                    | List merged accounts from all config files (selected one marked). |
| `account add\|list\|remove`   | Manage accounts in a config file (see below).            |
| `list`                        | List scripts on the server; the active one is marked `*`.|
| `download [name] [-o file]`   | Download a script (defaults to the active script).       |
| `upload <file> [name] [--activate]` | Validate and upload a script from a local file.    |
| `edit [name]`                 | Edit the active (or named) script in `$EDITOR`.          |
| `activate <name>` / `--none`  | Set the active script, or deactivate all.                |
| `rename <old> <new>`          | Rename a script.                                         |
| `delete <name> [-y]`          | Delete a script (prompts unless `-y`).                   |
| `check <file>`                | Validate a local script against the server.              |

### The `edit` workflow

```sh
sievemgmt edit
```

1. Downloads the active script (or the one you name) to a temp file.
2. Opens it in `$EDITOR` (falling back to `$VISUAL`, then `vim`/`vi`/`nano`).
3. On exit, the script is validated by the server.
4. If validation **fails**, you're asked to **edit again** or **save locally**
   (written to `./<name>.sieve`).
5. If it succeeds, the script is uploaded.

### Managing accounts

The `account` command edits a config file directly. By default it targets the
per-user config (`~/.config/sievemgmt/sievemgmt.yaml`); use `--local` to target
`./sievemgmt.yaml`, or `--file <path>` for an explicit path. Existing key order
and comments are preserved, and files are written with `0600` permissions.

```sh
# Add an account (you are prompted for the password if --password is omitted)
sievemgmt account add primary --email you@example.com --server mail.example.com

# List accounts in the target file
sievemgmt account list

# Overwrite an existing account
sievemgmt account add primary --force --email you@example.com --server mail.example.com

# Remove an account (alias: rm)
sievemgmt account remove primary
```

## Testing

Unit tests:

```sh
go test ./...
```

Integration tests run against a live server using the accounts listed in
`tmp/accounts` (one `email:password` per line; set the server with
`SIEVE_TEST_SERVER`). They require outbound
access to the ManageSieve port (4190) and are skipped when `tmp/accounts` is
absent:

```sh
go test -tags integration ./...
```

The integration suite exercises connect/list and a full
check → upload → download → activate → rename → delete lifecycle on a uniquely
named temporary script, restoring the original active script afterwards.
