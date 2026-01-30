package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type SSH struct {
	User string `yaml:"user"`
	Host string `yaml:"host"`           // hostname or IP (e.g., mymac.local)
	Port int    `yaml:"port,omitempty"` // default 22
	Key  string `yaml:"key,omitempty"`  // path to private key (optional)
}

type Category struct {
	Local   string   `yaml:"local"`             // absolute path recommended
	Remote  string   `yaml:"remote"`            // absolute path on remote
	Exclude []string `yaml:"exclude,omitempty"` // extra excludes for this category
}

type Defaults struct {
	Delete   *bool `yaml:"delete,omitempty"`   // mirror deletions
	Checksum *bool `yaml:"checksum,omitempty"` // compare by checksum (slower, safer)
	Verbose  *bool `yaml:"verbose,omitempty"`  // rsync -v
}

type Config struct {
	SSH        SSH                 `yaml:"ssh"`
	Categories map[string]Category `yaml:"categories"`
	Defaults   Defaults            `yaml:"defaults,omitempty"`
}

// Overridden at build time with: -ldflags "-X main.version=vX.Y.Z"
var version = "dev"

type RunOptions struct {
	DryRun    bool
	Delete    bool
	Checksum  bool
	NoVerbose bool
	Direction string
}

func main() {
	// Flags
	cfgPath := flag.String("config", defaultConfigPath(), "path to config YAML (default: ~/.belterlink/config.yaml)")
	dryRun := flag.Bool("dry-run", false, "show what would change without writing")
	deleteFlag := flag.Bool("delete", false, "delete files on destination that were deleted at source (can be defaulted in config)")
	checksum := flag.Bool("checksum", false, "use checksums to detect changes (slower, can be defaulted in config)")
	noVerbose := flag.Bool("no-verbose", false, "disable verbose output even if defaulted on")
	showHelp := flag.Bool("help", false, "show help")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("belterlink: ", version)
		return
	}

	args := flag.Args()
	if *showHelp || len(args) < 2 {
		printHelp()
		return
	}

	categoryName, direction, err := parseArgs(args)
	if err != nil {
		fail("%v", err)
	}

	// Load config
	cfg, err := loadConfig(*cfgPath)
	if err != nil {
		fail("load config: %v", err)
	}

	cat, ok := cfg.Categories[categoryName]
	if !ok {
		fail("category %q not found in config", categoryName)
	}

	if cfg.SSH.User == "" || cfg.SSH.Host == "" {
		fail("ssh.user and ssh.host are required in config")
	}

	opts := RunOptions{
		DryRun:    *dryRun,
		Delete:    *deleteFlag,
		Checksum:  *checksum,
		NoVerbose: *noVerbose,
		Direction: direction,
	}
	rsArgs, err := buildRsyncArgs(cfg, cat, opts)
	if err != nil {
		fail("build rsync args: %v", err)
	}

	fmt.Println("Running:", "rsync", strings.Join(rsArgs, " "))

	cmd := exec.Command("rsync", rsArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fail("rsync failed: %v", err)
	}
}

func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "./config.yaml"
	}
	return filepath.Join(home, ".belterlink", "config.yaml")
}

func buildRsyncArgs(cfg *Config, cat Category, opts RunOptions) ([]string, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}
	switch opts.Direction {
	case "push", "pull":
	default:
		return nil, fmt.Errorf("invalid direction %q", opts.Direction)
	}

	// Resolve defaults
	useDelete := getBool(opts.Delete, cfg.Defaults.Delete, false)
	useChecksum := getBool(opts.Checksum, cfg.Defaults.Checksum, false)
	useVerbose := getBool(!opts.NoVerbose, cfg.Defaults.Verbose, true)

	// Base rsync args
	rsArgs := []string{"-aH", "--protect-args", "--update"} // archive + hardlinks + don't clobber newer
	if useVerbose {
		rsArgs = append(rsArgs, "-v")
	}
	if opts.DryRun {
		rsArgs = append(rsArgs, "--dry-run")
	}
	if useChecksum {
		rsArgs = append(rsArgs, "--checksum")
	}
	if useDelete {
		rsArgs = append(rsArgs, "--delete", "--delete-excluded")
	}

	// Built-in safe excludes for Obsidian/macOS; users can add more in category
	builtinExcludes := []string{
		".DS_Store",
		"._*",
		".Trash*",
		".obsidian/cache",
		".git",
		"*.icloud", // iCloud placeholders
	}
	for _, e := range append(builtinExcludes, cat.Exclude...) {
		rsArgs = append(rsArgs, "--exclude", e)
	}

	// ssh transport
	sshCmd := "ssh"
	if cfg.SSH.Key != "" {
		sshCmd += " -i " + shellEscape(cfg.SSH.Key)
	}
	if cfg.SSH.Port != 0 && cfg.SSH.Port != 22 {
		sshCmd += fmt.Sprintf(" -p %d", cfg.SSH.Port)
	}
	rsArgs = append(rsArgs, "-e", sshCmd)

	// Source/Destination
	local := ensureTrailingSlash(cat.Local)
	remote := fmt.Sprintf("%s@%s:%s/", cfg.SSH.User, cfg.SSH.Host, strings.TrimRight(cat.Remote, "/"))

	switch opts.Direction {
	case "push": // local → remote
		rsArgs = append(rsArgs, local, remote)
	case "pull": // remote → local
		rsArgs = append(rsArgs, remote, local)
	}

	return rsArgs, nil
}

func loadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	if cfg.SSH.Port == 0 {
		cfg.SSH.Port = 22
	}
	if cfg.Categories == nil || len(cfg.Categories) == 0 {
		return nil, errors.New("no categories defined")
	}
	return &cfg, nil
}

func parseArgs(args []string) (string, string, error) {
	if len(args) < 2 {
		return "", "", errors.New("missing required arguments: <CategoryName> <push|pull>")
	}
	if len(args) > 2 {
		extra := args[2:]
		for _, arg := range extra {
			if strings.HasPrefix(arg, "-") {
				return "", "", fmt.Errorf("unexpected flag %q after positional args; flags must come before <CategoryName> <push|pull>", arg)
			}
		}
		return "", "", fmt.Errorf("unexpected extra arguments: %s", strings.Join(extra, " "))
	}
	direction := strings.ToLower(args[1])
	if direction != "push" && direction != "pull" {
		return "", "", errors.New("direction must be 'push' or 'pull'")
	}
	return args[0], direction, nil
}

func ensureTrailingSlash(p string) string {
	p = strings.TrimRight(p, "/")
	return p + "/"
}

func getBool(cli bool, def *bool, fallback bool) bool {
	// If user passed CLI true, honor it; if false + def is set, use def; else fallback
	if cli {
		return true
	}
	if def != nil {
		return *def
	}
	return fallback
}

func fail(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", a...)
	os.Exit(1)
}

// very light "escape" for showing in the printed command (rsync gets --protect-args)
func shellEscape(s string) string {
	if strings.ContainsAny(s, " \t") && !strings.HasPrefix(s, "'") && !strings.HasSuffix(s, "'") {
		return "'" + s + "'"
	}
	return s
}

func printHelp() {
	fmt.Print(`belterlink — simple, config-driven rsync wrapper (one-way by choice)

USAGE:
  belterlink [flags] <CategoryName> <push|pull>

FLAGS:
  -config <path>     Path to YAML config (default: ~/.belterlink/config.yaml)
  -dry-run           Show what would change (no writes)
  -delete            Mirror deletions (can be defaulted in config)
  -checksum          Compare by checksums instead of size+mtime (slower; can be defaulted)
  -no-verbose        Disable verbose rsync output (config default can enable it)
  -help              Show this help
  -version           Print version

EXAMPLES:
  belterlink Notes push
  belterlink -delete Notes push

DIRECTION:
  push  : local → remote
  pull  : remote → local

CONFIG SETUP (local machine):
  1) Create folder:  ~/.belterlink/
  2) Create file:    ~/.belterlink/config.yaml
  3) Fill SSH + categories (see example below).
  4) Ensure you can SSH between machines with key auth (no passwords).
  5) Run: belterlink Notes push (or pull)

CONFIG YAML EXAMPLE:

ssh:
  user: macuser
  host: mymac.local     # or a reserved LAN IP like 192.168.1.50
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

NOTES:
 - 'push' and 'pull' are one-way by design. If you edited both sides, the newer side wins
   because rsync is called with --update (and optionally --checksum).
 - Keep both machines' clocks in sync (NTP) to avoid timestamp confusion.
 - For iCloud paths on macOS, make sure files are downloaded (no .icloud placeholders).

`)
}
