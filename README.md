# belterlink 🚀

Belterlink is a tiny, config-driven wrapper around `rsync` for one-way syncs over SSH.
It is designed for simple “push” or “pull” workflows (e.g., a local vault to a remote Mac),
with sensible defaults and optional safe deletions.

## Features ✨

- 🔁 One-way sync: `push` (local → remote) or `pull` (remote → local)
- 🔐 SSH transport with optional key and port
- 🧹 Built-in safe excludes (macOS/Obsidian related) plus per-category excludes
- ✅ Optional checksum comparison and delete mirroring
- 🧪 Dry-run mode for safe previews

## Installation 📦

Requires `rsync` and `ssh` on your machine.

### One-line install (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/arcapol/belterlink/main/install.sh | sh
```

If you want to install from a fork:

```bash
curl -fsSL https://raw.githubusercontent.com/arcapol/belterlink/main/install.sh | REPO=yourname/belterlink sh
```

### Local install

```bash
./install.sh
```

## Uninstallation 🧹

### One-line uninstall

```bash
curl -fsSL https://raw.githubusercontent.com/arcapol/belterlink/main/uninstall.sh | sh
```

### Local uninstall

```bash
./uninstall.sh
```

## Update 🔄

Just re-run the installer:

```bash
curl -fsSL https://raw.githubusercontent.com/arcapol/belterlink/main/install.sh | sh
```

## Quick start ⚡

1) Create the config directory:

```bash
mkdir -p ~/.belterlink
```

2) Create the config file:

```bash
touch ~/.belterlink/config.yaml
```

3) Add your SSH + categories (see example below).

4) Run a sync:

```bash
belterlink Notes push
```

## Usage 🧭

Flags must come before positional args (this is how Go’s `flag` package parses):

```bash
belterlink [flags] <CategoryName> <push|pull>
```

Examples:

```bash
belterlink Notes push
belterlink -delete Notes push
belterlink -dry-run -checksum Notes pull
```

## Flags 🏷️

- `-config <path>`: path to YAML config (default: `~/.belterlink/config.yaml`)
- `-dry-run`: show what would change (no writes)
- `-delete`: mirror deletions (can be defaulted in config)
- `-checksum`: compare by checksums (slower, safer; can be defaulted)
- `-no-verbose`: disable verbose rsync output (config default can enable it)
- `-help`: show help
- `-version`: print version

## Configuration 🧩

Config file path: `~/.belterlink/config.yaml`

```yaml
ssh:
  user: macuser
  host: mymac.local     # or a LAN IP like 192.168.1.50
  port: 22
  key: /home/linuxuser/.ssh/id_ed25519   # optional

defaults:
  delete: false
  checksum: false
  verbose: true

categories:
  Piano:
    local:  /home/linuxuser/ObsidianVault/Piano
    remote: /Users/macuser/Library/Mobile Documents/com~apple~CloudDocs/ObsidianVault/Piano
    exclude:
      - "*.wav"
      - ".obsidian/workspace*"

  Notes:
    local:  /home/linuxuser/ObsidianVault/Notes
    remote: /Users/macuser/Library/Mobile Documents/com~apple~CloudDocs/ObsidianVault/Notes
    exclude:
      - ".obsidian/cache"
      - ".DS_Store"
```

### Built-in excludes 🧯

Belterlink always excludes:

- `.DS_Store`
- `._*`
- `.Trash*`
- `.obsidian/cache`
- `.git`
- `*.icloud`

## Notes and behavior 📎

- Syncs are one-way by design. If both sides changed, the newer side wins because `rsync`
  is invoked with `--update` (and optionally `--checksum`).
- Keep both machines’ clocks in sync (NTP) to avoid timestamp confusion.
- For iCloud paths on macOS, make sure files are downloaded (no `.icloud` placeholders).
- `-delete` removes destination files that no longer exist at the source. Use carefully.

## Troubleshooting 🛠️

- If you see “unexpected flag after positional args”, move flags before the category:
  `belterlink -delete Notes push`
- Ensure SSH works without password prompts (key-based auth recommended).
- Confirm `rsync` is installed and available in `PATH`.
