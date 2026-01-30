package main

import "testing"

func TestGetBool(t *testing.T) {
	tests := []struct {
		name     string
		cli      bool
		def      *bool
		fallback bool
		want     bool
	}{
		{name: "cli true wins", cli: true, def: boolPtr(false), fallback: false, want: true},
		{name: "default used when cli false", cli: false, def: boolPtr(true), fallback: false, want: true},
		{name: "fallback used when no default", cli: false, def: nil, fallback: true, want: true},
		{name: "fallback false", cli: false, def: nil, fallback: false, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getBool(tt.cli, tt.def, tt.fallback); got != tt.want {
				t.Fatalf("getBool(%v, %v, %v) = %v, want %v", tt.cli, tt.def, tt.fallback, got, tt.want)
			}
		})
	}
}

func TestEnsureTrailingSlash(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "/tmp/notes", want: "/tmp/notes/"},
		{in: "/tmp/notes/", want: "/tmp/notes/"},
		{in: "/tmp/notes////", want: "/tmp/notes/"},
	}
	for _, tt := range tests {
		if got := ensureTrailingSlash(tt.in); got != tt.want {
			t.Fatalf("ensureTrailingSlash(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestBuildRsyncArgsDeleteFlag(t *testing.T) {
	cfg := &Config{
		SSH: SSH{User: "alice", Host: "example.com", Port: 22},
	}
	cat := Category{
		Local:   "/local/path",
		Remote: "/remote/path",
		Exclude: []string{
			"*.tmp",
		},
	}
	opts := RunOptions{
		Delete:    true,
		Direction: "push",
	}

	args, err := buildRsyncArgs(cfg, cat, opts)
	if err != nil {
		t.Fatalf("buildRsyncArgs error: %v", err)
	}
	if !containsArg(args, "--delete") || !containsArg(args, "--delete-excluded") {
		t.Fatalf("expected delete flags in args, got: %v", args)
	}
	expectSrc := "/local/path/"
	expectDst := "alice@example.com:/remote/path/"
	if len(args) < 2 || args[len(args)-2] != expectSrc || args[len(args)-1] != expectDst {
		t.Fatalf("unexpected src/dst: got %v, want %q %q", args[len(args)-2:], expectSrc, expectDst)
	}
}

func TestBuildRsyncArgsDeleteDefaultFromConfig(t *testing.T) {
	cfg := &Config{
		SSH:      SSH{User: "bob", Host: "host", Port: 22},
		Defaults: Defaults{Delete: boolPtr(true)},
	}
	cat := Category{Local: "/l", Remote: "/r"}
	opts := RunOptions{Delete: false, Direction: "push"}

	args, err := buildRsyncArgs(cfg, cat, opts)
	if err != nil {
		t.Fatalf("buildRsyncArgs error: %v", err)
	}
	if !containsArg(args, "--delete") {
		t.Fatalf("expected delete flag from defaults, got: %v", args)
	}
}

func TestBuildRsyncArgsNoDeleteWhenDisabled(t *testing.T) {
	cfg := &Config{
		SSH:      SSH{User: "bob", Host: "host", Port: 22},
		Defaults: Defaults{Delete: boolPtr(false)},
	}
	cat := Category{Local: "/l", Remote: "/r"}
	opts := RunOptions{Delete: false, Direction: "push"}

	args, err := buildRsyncArgs(cfg, cat, opts)
	if err != nil {
		t.Fatalf("buildRsyncArgs error: %v", err)
	}
	if containsArg(args, "--delete") || containsArg(args, "--delete-excluded") {
		t.Fatalf("did not expect delete flags, got: %v", args)
	}
}

func TestBuildRsyncArgsPullDirection(t *testing.T) {
	cfg := &Config{SSH: SSH{User: "u", Host: "h", Port: 22}}
	cat := Category{Local: "/local", Remote: "/remote"}
	opts := RunOptions{Direction: "pull"}

	args, err := buildRsyncArgs(cfg, cat, opts)
	if err != nil {
		t.Fatalf("buildRsyncArgs error: %v", err)
	}
	expectSrc := "u@h:/remote/"
	expectDst := "/local/"
	if len(args) < 2 || args[len(args)-2] != expectSrc || args[len(args)-1] != expectDst {
		t.Fatalf("unexpected src/dst: got %v, want %q %q", args[len(args)-2:], expectSrc, expectDst)
	}
}

func TestBuildRsyncArgsInvalidDirection(t *testing.T) {
	cfg := &Config{SSH: SSH{User: "u", Host: "h", Port: 22}}
	cat := Category{Local: "/local", Remote: "/remote"}
	opts := RunOptions{Direction: "sideways"}

	if _, err := buildRsyncArgs(cfg, cat, opts); err == nil {
		t.Fatalf("expected error for invalid direction")
	}
}

func TestParseArgsValid(t *testing.T) {
	category, direction, err := parseArgs([]string{"Notes", "push"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if category != "Notes" || direction != "push" {
		t.Fatalf("unexpected parse result: %q %q", category, direction)
	}
}

func TestParseArgsTooFew(t *testing.T) {
	if _, _, err := parseArgs([]string{"OnlyOne"}); err == nil {
		t.Fatalf("expected error for missing args")
	}
}

func TestParseArgsTooMany(t *testing.T) {
	if _, _, err := parseArgs([]string{"Notes", "push", "extra"}); err == nil {
		t.Fatalf("expected error for extra args")
	}
}

func TestParseArgsFlagAfterArgs(t *testing.T) {
	if _, _, err := parseArgs([]string{"Notes", "push", "-delete"}); err == nil {
		t.Fatalf("expected error for flag after args")
	}
}

func TestParseArgsInvalidDirection(t *testing.T) {
	if _, _, err := parseArgs([]string{"Notes", "sideways"}); err == nil {
		t.Fatalf("expected error for invalid direction")
	}
}

func boolPtr(v bool) *bool {
	return &v
}

func containsArg(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}
