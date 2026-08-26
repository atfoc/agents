package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
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
	if !reflect.DeepEqual(opts, want) {
		t.Errorf("opts = %+v, want %+v", opts, want)
	}
}

func TestParseArgs_ShortForm(t *testing.T) {
	opts, err := parseArgs([]string{"-s", "X", "-t", "Y"})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	want := installer.Options{Source: "X", Target: "Y"}
	if !reflect.DeepEqual(opts, want) {
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
	if !reflect.DeepEqual(long, short) {
		t.Errorf("long form %+v != short form %+v", long, short)
	}
}

func TestParseArgs_MixedForm(t *testing.T) {
	opts, err := parseArgs([]string{"-s", "X", "--target", "Y"})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	want := installer.Options{Source: "X", Target: "Y"}
	if !reflect.DeepEqual(opts, want) {
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

	for _, want := range []string{"--agents", "--skills", "--install", "-i"} {
		if !strings.Contains(text, want) {
			t.Errorf("usage() output does not contain %q\n---\n%s", want, text)
		}
	}
	// --filter is gone outright, not kept as an alias: the help text must not
	// keep teaching a name the tool no longer accepts.
	if strings.Contains(text, "--filter") {
		t.Errorf("usage() output still mentions %q, but the flag was renamed to --install\n---\n%s", "--filter", text)
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

func TestParseArgs_InstallWithAgents(t *testing.T) {
	opts, err := parseArgs([]string{"-s", "X", "-t", "Y", "--agents", "--install", "scout"})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	want := []string{"scout"}
	if !reflect.DeepEqual(opts.Install, want) {
		t.Errorf("opts.Install = %v, want %v", opts.Install, want)
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
	if !reflect.DeepEqual(opts, want) {
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

func TestParseArgs_InstallWithoutAgentsOrSkills(t *testing.T) {
	_, err := parseArgs([]string{"-s", "X", "-t", "Y", "--install", "scout"})
	if err == nil {
		t.Fatalf("parseArgs returned nil error, want an error naming --install")
	}
	if !strings.Contains(err.Error(), "--install requires --agents or --skills") {
		t.Errorf("error %q does not match expected message", err.Error())
	}
}

func TestParseArgs_InstallWithSkills(t *testing.T) {
	opts, err := parseArgs([]string{"-s", "X", "-t", "Y", "--skills", "--install", "research"})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	want := []string{"research"}
	if !reflect.DeepEqual(opts.Install, want) {
		t.Errorf("opts.Install = %v, want %v", opts.Install, want)
	}
}

// TestParseArgs_FilterFlagRemoved pins the deliberate break: --filter is not
// kept as an alias for --install, so an old invocation must fail loudly
// rather than quietly doing something else.
func TestParseArgs_FilterFlagRemoved(t *testing.T) {
	_, err := parseArgs([]string{"-s", "X", "-t", "Y", "--agents", "--filter", "scout"})
	if err == nil {
		t.Fatalf("parseArgs(--filter) returned nil error, want an error: --filter is no longer a registered flag")
	}
	if !strings.Contains(err.Error(), "flag provided but not defined") {
		t.Errorf("error %q does not report --filter as an unknown flag", err.Error())
	}
}

func TestParseArgs_InstallRepeatable(t *testing.T) {
	opts, err := parseArgs([]string{"-s", "X", "-t", "Y", "--agents", "--install", "scout", "--install", "writer"})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	// Order is the user's, preserved: it is what the error message for a
	// mistyped name reads back.
	want := []string{"scout", "writer"}
	if !reflect.DeepEqual(opts.Install, want) {
		t.Errorf("opts.Install = %v, want %v", opts.Install, want)
	}
}

func TestParseArgs_InstallDeduplicates(t *testing.T) {
	opts, err := parseArgs([]string{"-s", "X", "-t", "Y", "--agents", "--install", "scout", "--install", "scout"})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	want := []string{"scout"}
	if !reflect.DeepEqual(opts.Install, want) {
		t.Errorf("opts.Install = %v, want %v: naming an item twice describes the same wish", opts.Install, want)
	}
}

func TestParseArgs_InstallEmptyNameRejected(t *testing.T) {
	_, err := parseArgs([]string{"-s", "X", "-t", "Y", "--agents", "--install", ""})
	if err == nil {
		t.Fatalf("parseArgs(--install \"\") returned nil error, want an error naming the empty name")
	}
	if !strings.Contains(err.Error(), "name cannot be empty") {
		t.Errorf("error %q does not name the empty --install value as the problem", err.Error())
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
		{"Install", []string{"-s", "X", "-t", "Y", "-i", "--install", "scout"}, "-i cannot be combined with --install"},
		{"Uninstall", []string{"-t", "Y", "-i", "--uninstall", "stale"}, "-i cannot be combined with --uninstall"},
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

func TestParseArgs_InstallHasNoShortForm(t *testing.T) {
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

func TestParseArgs_UninstallRepeatable(t *testing.T) {
	opts, err := parseArgs([]string{"-t", "Y", "--skills", "--uninstall", "research", "--uninstall", "stale"})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	// Order is the user's, preserved: it is what the error message for a
	// mistyped name reads back.
	want := []string{"research", "stale"}
	if !reflect.DeepEqual(opts.Uninstall, want) {
		t.Errorf("opts.Uninstall = %v, want %v", opts.Uninstall, want)
	}
}

func TestParseArgs_UninstallWithoutAgentsOrSkills(t *testing.T) {
	_, err := parseArgs([]string{"-t", "Y", "--uninstall", "research"})
	if err == nil {
		t.Fatalf("parseArgs returned nil error, want an error naming --uninstall")
	}
	if !strings.Contains(err.Error(), "--uninstall requires --agents or --skills") {
		t.Errorf("error %q does not match expected message", err.Error())
	}
}

func TestParseArgs_UninstallHasNoShortForm(t *testing.T) {
	_, err := parseArgs([]string{"-t", "Y", "--skills", "-u", "research"})
	if err == nil {
		t.Fatalf("parseArgs(-u) returned nil error, want an error: -u is not a registered flag")
	}
}

// TestParseArgs_UninstallRejectsSource pins the rule that a source given
// alongside --uninstall is a misunderstanding, not a harmless extra: the
// source has no say in what gets deleted.
func TestParseArgs_UninstallRejectsSource(t *testing.T) {
	_, err := parseArgs([]string{"-s", "X", "-t", "Y", "--skills", "--uninstall", "research"})
	if err == nil {
		t.Fatalf("parseArgs returned nil error, want an error rejecting --source with --uninstall")
	}
	if !strings.Contains(err.Error(), "--uninstall cannot be combined with --source") {
		t.Errorf("error %q does not match expected message", err.Error())
	}
}

func TestParseArgs_UninstallDoesNotRequireSource(t *testing.T) {
	opts, err := parseArgs([]string{"-t", "Y", "--agents", "--uninstall", "scout"})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	if opts.Source != "" {
		t.Errorf("opts.Source = %q, want %q", opts.Source, "")
	}
	want := installer.Options{Target: "Y", OnlyAgents: true, Uninstall: []string{"scout"}}
	if !reflect.DeepEqual(opts, want) {
		t.Errorf("opts = %+v, want %+v", opts, want)
	}
}

func TestParseArgs_UninstallStillRequiresTarget(t *testing.T) {
	_, err := parseArgs([]string{"--agents", "--uninstall", "scout"})
	if err == nil {
		t.Fatalf("parseArgs returned nil error, want an error naming --target")
	}
	if !strings.Contains(err.Error(), "--target is required") {
		t.Errorf("error %q does not match expected message", err.Error())
	}
}

func TestParseArgs_UninstallWithDryRun(t *testing.T) {
	opts, err := parseArgs([]string{"-t", "Y", "--skills", "--uninstall", "research", "--dry-run"})
	if err != nil {
		t.Fatalf("parseArgs returned error: %v", err)
	}
	if !opts.DryRun {
		t.Errorf("opts.DryRun = false, want true: --dry-run applies to a removal run too")
	}
}

func TestParseArgs_InstallAndUninstallMutuallyExclusive(t *testing.T) {
	_, err := parseArgs([]string{"-t", "Y", "--agents", "--install", "scout", "--uninstall", "stale"})
	if err == nil {
		t.Fatalf("parseArgs returned nil error, want an error rejecting --install with --uninstall")
	}
	if !strings.Contains(err.Error(), "--install and --uninstall cannot be combined") {
		t.Errorf("error %q does not match expected message", err.Error())
	}
}

// TestParseArgs_ErrorPrecedence pins which message wins when a command line
// makes several mistakes at once. The order is load-bearing, not incidental:
// a run asking for two jobs at once has no single set of required flags to
// check it against, so mode conflicts must be reported before requiredness
// and before scoping.
func TestParseArgs_ErrorPrecedence(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			"ModeConflictBeatsInteractiveConflict",
			[]string{"-i", "--install", "a", "--uninstall", "b"},
			"--install and --uninstall cannot be combined",
		},
		{
			"SourceConflictBeatsMissingScope",
			[]string{"--uninstall", "a", "-s", "S", "-t", "T"},
			"--uninstall cannot be combined with --source",
		},
		{
			"MissingTargetBeatsMissingScope",
			[]string{"--uninstall", "a", "--agents"},
			"--target is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseArgs(tt.args)
			if err == nil {
				t.Fatalf("parseArgs(%v) returned nil error, want %q", tt.args, tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q, want it to contain %q", err.Error(), tt.want)
			}
		})
	}
}

// buildTargetTree creates a target directory holding one installed agent and
// one installed skill.
func buildTargetTree(t *testing.T, dir string) {
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

func TestRun_UninstallSuccess(t *testing.T) {
	target := filepath.Join(t.TempDir(), "target")
	buildTargetTree(t, target)

	var stdout, stderr bytes.Buffer
	code := run(&stdout, &stderr, []string{"-t", target, "--skills", "--uninstall", "research"})
	if code != 0 {
		t.Fatalf("run() code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("run() wrote to stderr: %q", stderr.String())
	}

	want := "AGENTS\n------\n(none)\n\nSKILLS\n------\nremoved    research\n\n1 removed\n"
	if got := stdout.String(); got != want {
		t.Fatalf("stdout mismatch.\ngot:\n%s\nwant:\n%s", got, want)
	}

	if _, err := os.Stat(filepath.Join(target, "skills", "research")); !os.IsNotExist(err) {
		t.Errorf("skill research was not removed (err %v)", err)
	}
	// The empty skills/ directory is left in place.
	if info, err := os.Stat(filepath.Join(target, "skills")); err != nil || !info.IsDir() {
		t.Errorf("skills/ should still exist after removing its last item (err %v)", err)
	}
	// The agent was never named and is untouched.
	if _, err := os.Stat(filepath.Join(target, "agents", "scout.md")); err != nil {
		t.Errorf("agent scout.md was removed by a --skills-scoped run: %v", err)
	}
}

func TestRun_UninstallDryRun(t *testing.T) {
	target := filepath.Join(t.TempDir(), "target")
	buildTargetTree(t, target)

	var stdout, stderr bytes.Buffer
	code := run(&stdout, &stderr, []string{"-t", target, "--skills", "--uninstall", "research", "--dry-run"})
	if code != 0 {
		t.Fatalf("run() code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if !strings.HasPrefix(stdout.String(), "DRY RUN") {
		t.Errorf("stdout = %q, want it to start with the dry-run banner", stdout.String())
	}
	if !strings.Contains(stdout.String(), "removed    research") {
		t.Errorf("stdout = %q, want it to list the item that would be removed", stdout.String())
	}

	if _, err := os.Stat(filepath.Join(target, "skills", "research", "SKILL.md")); err != nil {
		t.Errorf("dry run removed the skill: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "agents", "scout.md")); err != nil {
		t.Errorf("dry run removed the agent: %v", err)
	}
}

func TestRun_UninstallNameNotFound(t *testing.T) {
	target := filepath.Join(t.TempDir(), "target")
	buildTargetTree(t, target)

	var stdout, stderr bytes.Buffer
	code := run(&stdout, &stderr, []string{
		"-t", target, "--skills", "--uninstall", "research", "--uninstall", "typo",
	})
	if code != 1 {
		t.Fatalf("run() code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Errorf("run() wrote to stdout: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "error:") || !strings.Contains(stderr.String(), "typo") {
		t.Errorf("stderr = %q, want it to report the mistyped name", stderr.String())
	}

	// Neither the good name nor anything else was removed.
	if _, err := os.Stat(filepath.Join(target, "skills", "research", "SKILL.md")); err != nil {
		t.Errorf("a run that failed name verification still removed the skill: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "agents", "scout.md")); err != nil {
		t.Errorf("a run that failed name verification still removed the agent: %v", err)
	}
}

func TestUsage_MentionsInstallAndUninstall(t *testing.T) {
	var buf bytes.Buffer
	usage(&buf)
	text := buf.String()

	for _, want := range []string{"--install", "--uninstall"} {
		if !strings.Contains(text, want) {
			t.Errorf("usage() output does not contain %q\n---\n%s", want, text)
		}
	}
	// --filter is gone outright, not kept as an alias: the help text must not
	// keep teaching a name the tool no longer accepts.
	if strings.Contains(text, "--filter") {
		t.Errorf("usage() output still mentions %q\n---\n%s", "--filter", text)
	}
}
