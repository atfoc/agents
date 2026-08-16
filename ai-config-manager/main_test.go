package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atfoc/agents/ai-config-manager/internal/installer"
)

func TestParseArgs_LongForm(t *testing.T) {
	opts, err := parseArgs([]string{"--source", "X", "--target", "Y"})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	want := installer.Options{Source: "X", Target: "Y"}
	if opts != want {
		t.Errorf("opts = %+v, want %+v", opts, want)
	}
}

func TestParseArgs_ShortForm(t *testing.T) {
	opts, err := parseArgs([]string{"-s", "X", "-t", "Y"})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	want := installer.Options{Source: "X", Target: "Y"}
	if opts != want {
		t.Errorf("opts = %+v, want %+v", opts, want)
	}
}

func TestParseArgs_ShortAndLongFormsIdentical(t *testing.T) {
	long, err := parseArgs([]string{"--source", "X", "--target", "Y"})
	if err != nil {
		t.Fatalf("parseArgs (long) returned error: %v", err)
	}
	short, err := parseArgs([]string{"-s", "X", "-t", "Y"})
	if err != nil {
		t.Fatalf("parseArgs (short) returned error: %v", err)
	}
	if long != short {
		t.Errorf("long form %+v != short form %+v", long, short)
	}
}

func TestParseArgs_MixedForm(t *testing.T) {
	opts, err := parseArgs([]string{"-s", "X", "--target", "Y"})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	want := installer.Options{Source: "X", Target: "Y"}
	if opts != want {
		t.Errorf("opts = %+v, want %+v", opts, want)
	}
}

func TestParseArgs_DryRun(t *testing.T) {
	opts, err := parseArgs([]string{"-s", "X", "-t", "Y", "--dry-run"})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	if !opts.DryRun {
		t.Errorf("opts.DryRun = false, want true")
	}
}

func TestParseArgs_DryRunDefaultsFalse(t *testing.T) {
	opts, err := parseArgs([]string{"-s", "X", "-t", "Y"})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	if opts.DryRun {
		t.Errorf("opts.DryRun = true, want false as the default")
	}
}

func TestParseArgs_DryRunHasNoShortForm(t *testing.T) {
	_, err := parseArgs([]string{"-s", "X", "-t", "Y", "-d"})
	if err == nil {
		t.Fatalf("parseArgs(-d) returned nil error, want an error: -d is not a registered flag")
	}
}

func TestParseArgs_MissingSource(t *testing.T) {
	_, err := parseArgs([]string{"-t", "Y"})
	if err == nil {
		t.Fatalf("parseArgs returned nil error, want an error naming --source")
	}
	if !strings.Contains(err.Error(), "source") {
		t.Errorf("error %q does not mention source", err.Error())
	}
}

func TestParseArgs_MissingTarget(t *testing.T) {
	_, err := parseArgs([]string{"-s", "X"})
	if err == nil {
		t.Fatalf("parseArgs returned nil error, want an error naming --target")
	}
	if !strings.Contains(err.Error(), "target") {
		t.Errorf("error %q does not mention target", err.Error())
	}
}

func TestParseArgs_MissingBoth(t *testing.T) {
	_, err := parseArgs(nil)
	if err == nil {
		t.Fatalf("parseArgs returned nil error, want an error")
	}
}

func TestParseArgs_UnknownFlag(t *testing.T) {
	_, err := parseArgs([]string{"-s", "X", "-t", "Y", "--nope"})
	if err == nil {
		t.Fatalf("parseArgs(--nope) returned nil error, want an error")
	}
}

func TestParseArgs_StrayPositionalArgument(t *testing.T) {
	_, err := parseArgs([]string{"-s", "a", "-t", "c", "extra"})
	if err == nil {
		t.Fatalf("parseArgs returned nil error, want an error naming the stray argument")
	}
	if !strings.Contains(err.Error(), "extra") {
		t.Errorf("error %q does not mention the stray argument %q", err.Error(), "extra")
	}
}

func TestParseArgs_StrayPositionalStopsFlagParsing(t *testing.T) {
	// flag stops parsing at the first non-flag argument, so "-t c" here is
	// never consumed as a flag: "b" surfaces as the stray positional.
	_, err := parseArgs([]string{"-s", "a", "b", "-t", "c"})
	if err == nil {
		t.Fatalf("parseArgs returned nil error, want an error naming %q", "b")
	}
	if !strings.Contains(err.Error(), "b") {
		t.Errorf("error %q does not mention the stray argument %q", err.Error(), "b")
	}
}

func TestParseArgs_LastFlagWins(t *testing.T) {
	opts, err := parseArgs([]string{"-s", "first", "--source", "second", "-t", "Y"})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	if opts.Source != "second" {
		t.Errorf("opts.Source = %q, want %q (the later flag should win)", opts.Source, "second")
	}
}

func TestParseArgs_HelpShort(t *testing.T) {
	_, err := parseArgs([]string{"-h"})
	if !errors.Is(err, errHelp) {
		t.Errorf("parseArgs(-h) error = %v, want errors.Is(err, errHelp)", err)
	}
}

func TestParseArgs_HelpLong(t *testing.T) {
	_, err := parseArgs([]string{"--help"})
	if !errors.Is(err, errHelp) {
		t.Errorf("parseArgs(--help) error = %v, want errors.Is(err, errHelp)", err)
	}
}

func TestRun_Help(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(&stdout, &stderr, []string{"--help"})
	if code != 0 {
		t.Errorf("run(--help) code = %d, want 0", code)
	}
	if stdout.Len() == 0 {
		t.Errorf("run(--help) wrote nothing to stdout")
	}
	if stderr.Len() != 0 {
		t.Errorf("run(--help) wrote to stderr: %q", stderr.String())
	}
}

func TestRun_MissingRequiredFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(&stdout, &stderr, nil)
	if code != 1 {
		t.Errorf("run() code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Errorf("run() wrote to stdout: %q", stdout.String())
	}
	if stderr.Len() == 0 {
		t.Fatalf("run() wrote nothing to stderr")
	}
	if !strings.Contains(stderr.String(), "error:") {
		t.Errorf("stderr = %q, want it to contain %q", stderr.String(), "error:")
	}
}

func TestRun_NonexistentSource(t *testing.T) {
	var stdout, stderr bytes.Buffer
	target := filepath.Join(t.TempDir(), "target")
	code := run(&stdout, &stderr, []string{"-s", filepath.Join(t.TempDir(), "does-not-exist"), "-t", target})
	if code != 1 {
		t.Errorf("run() code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "error:") {
		t.Errorf("stderr = %q, want it to contain %q", stderr.String(), "error:")
	}
}

// buildSourceTree creates a minimal, valid installer source tree under dir:
// one agent and one skill.
func buildSourceTree(t *testing.T, dir string) {
	t.Helper()
	agentsDir := filepath.Join(dir, "agents")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", agentsDir, err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "scout.md"), []byte("# scout\n"), 0o644); err != nil {
		t.Fatalf("WriteFile scout.md: %v", err)
	}
	skillDir := filepath.Join(dir, "skills", "research")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", skillDir, err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# research\n"), 0o644); err != nil {
		t.Fatalf("WriteFile SKILL.md: %v", err)
	}
}

func TestRun_Success(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source")
	buildSourceTree(t, source)
	target := filepath.Join(t.TempDir(), "target")

	var stdout, stderr bytes.Buffer
	code := run(&stdout, &stderr, []string{"-s", source, "-t", target})
	if code != 0 {
		t.Fatalf("run() code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("run() wrote to stderr: %q", stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "AGENTS") {
		t.Errorf("stdout %q does not contain %q", out, "AGENTS")
	}
	if !strings.Contains(out, "SKILLS") {
		t.Errorf("stdout %q does not contain %q", out, "SKILLS")
	}

	if _, err := os.Stat(filepath.Join(target, "agents", "scout.md")); err != nil {
		t.Errorf("agent was not installed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "skills", "research", "SKILL.md")); err != nil {
		t.Errorf("skill was not installed: %v", err)
	}
}

func TestRun_DryRun(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source")
	buildSourceTree(t, source)
	target := filepath.Join(t.TempDir(), "target")

	var stdout, stderr bytes.Buffer
	code := run(&stdout, &stderr, []string{"-s", source, "-t", target, "--dry-run"})
	if code != 0 {
		t.Fatalf("run() code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.HasPrefix(stdout.String(), "DRY RUN") {
		t.Errorf("stdout = %q, want it to start with the dry-run banner", stdout.String())
	}

	entries, err := os.ReadDir(target)
	if err == nil && len(entries) != 0 {
		t.Errorf("dry-run wrote to the target: found entries %v", entries)
	} else if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ReadDir(%s): %v", target, err)
	}
}

func TestUsage(t *testing.T) {
	var buf bytes.Buffer
	usage(&buf)
	text := buf.String()

	for _, want := range []string{"-s, --source", "-t, --target", "--dry-run"} {
		if !strings.Contains(text, want) {
			t.Errorf("usage() output does not contain %q\n---\n%s", want, text)
		}
	}
	if strings.Contains(text, "-d,") {
		t.Errorf("usage() output contains a %q short form, but --dry-run must have none", "-d,")
	}
}

func TestUsage_MentionsInteractiveFlags(t *testing.T) {
	var buf bytes.Buffer
	usage(&buf)
	text := buf.String()

	for _, want := range []string{"--agents", "--skills", "--filter", "-i"} {
		if !strings.Contains(text, want) {
			t.Errorf("usage() output does not contain %q\n---\n%s", want, text)
		}
	}
}

func TestParseArgs_Agents(t *testing.T) {
	opts, err := parseArgs([]string{"-s", "X", "-t", "Y", "--agents"})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	if !opts.OnlyAgents {
		t.Errorf("opts.OnlyAgents = false, want true")
	}
}

func TestParseArgs_Skills(t *testing.T) {
	opts, err := parseArgs([]string{"-s", "X", "-t", "Y", "--skills"})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	if !opts.OnlySkills {
		t.Errorf("opts.OnlySkills = false, want true")
	}
}

func TestParseArgs_FilterWithAgents(t *testing.T) {
	opts, err := parseArgs([]string{"-s", "X", "-t", "Y", "--agents", "--filter", "scout"})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	if opts.Filter != "scout" {
		t.Errorf("opts.Filter = %q, want %q", opts.Filter, "scout")
	}
}

func TestParseArgs_Interactive(t *testing.T) {
	opts, err := parseArgs([]string{"-s", "X", "-t", "Y", "-i"})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	if !opts.Interactive {
		t.Errorf("opts.Interactive = false, want true")
	}
	want := installer.Options{Source: "X", Target: "Y", Interactive: true}
	if opts != want {
		t.Errorf("opts = %+v, want %+v", opts, want)
	}
}

func TestParseArgs_AgentsAndSkillsMutuallyExclusive(t *testing.T) {
	_, err := parseArgs([]string{"-s", "X", "-t", "Y", "--agents", "--skills"})
	if err == nil {
		t.Fatalf("parseArgs returned nil error, want an error rejecting --agents with --skills")
	}
	if !strings.Contains(err.Error(), "--agents and --skills are mutually exclusive") {
		t.Errorf("error %q does not match expected message", err.Error())
	}
}

func TestParseArgs_FilterWithoutAgentsOrSkills(t *testing.T) {
	_, err := parseArgs([]string{"-s", "X", "-t", "Y", "--filter", "scout"})
	if err == nil {
		t.Fatalf("parseArgs returned nil error, want an error naming --filter")
	}
	if !strings.Contains(err.Error(), "--filter requires --agents or --skills") {
		t.Errorf("error %q does not match expected message", err.Error())
	}
}

func TestParseArgs_FilterWithSkills(t *testing.T) {
	opts, err := parseArgs([]string{"-s", "X", "-t", "Y", "--skills", "--filter", "research"})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	if opts.Filter != "research" {
		t.Errorf("opts.Filter = %q, want %q", opts.Filter, "research")
	}
}

func TestParseArgs_InteractiveConflicts(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"DryRun", []string{"-s", "X", "-t", "Y", "-i", "--dry-run"}, "-i cannot be combined with --dry-run"},
		{"Agents", []string{"-s", "X", "-t", "Y", "-i", "--agents"}, "-i cannot be combined with --agents"},
		{"Skills", []string{"-s", "X", "-t", "Y", "-i", "--skills"}, "-i cannot be combined with --skills"},
		{"Filter", []string{"-s", "X", "-t", "Y", "-i", "--filter", "scout"}, "-i cannot be combined with --filter"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseArgs(tt.args)
			if err == nil {
				t.Fatalf("parseArgs(%v) returned nil error, want an error naming the conflicting flag", tt.args)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q does not match expected message %q", err.Error(), tt.want)
			}
		})
	}
}

func TestParseArgs_AgentsHasNoShortForm(t *testing.T) {
	_, err := parseArgs([]string{"-s", "X", "-t", "Y", "-a"})
	if err == nil {
		t.Fatalf("parseArgs(-a) returned nil error, want an error: -a is not a registered flag")
	}
}

func TestParseArgs_SkillsHasNoShortForm(t *testing.T) {
	_, err := parseArgs([]string{"-s", "X", "-t", "Y", "-k"})
	if err == nil {
		t.Fatalf("parseArgs(-k) returned nil error, want an error: -k is not a registered flag")
	}
}

func TestParseArgs_FilterHasNoShortForm(t *testing.T) {
	_, err := parseArgs([]string{"-s", "X", "-t", "Y", "-f", "scout"})
	if err == nil {
		t.Fatalf("parseArgs(-f) returned nil error, want an error: -f is not a registered flag")
	}
}

// TestRun_InteractiveRequiresTTY asserts the -i tty guard in run()'s
// interactive branch. Under `go test`, stdin is normally not a character
// device, so this is expected to hold on CI and most dev machines; if it
// ever proves flaky here (e.g. a test runner that attaches a real tty to
// stdin), that would need a skip rather than forcing this to pass.
func TestRun_InteractiveRequiresTTY(t *testing.T) {
	info, err := os.Stdin.Stat()
	if err == nil && info.Mode()&os.ModeCharDevice != 0 {
		t.Skip("stdin is a character device in this environment; the tty guard would not trigger")
	}

	var stdout, stderr bytes.Buffer
	code := run(&stdout, &stderr, []string{"-s", "X", "-t", "Y", "-i"})
	if code != 1 {
		t.Errorf("run(-i) code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "interactive terminal") {
		t.Errorf("stderr = %q, want it to mention the tty requirement", stderr.String())
	}
}
