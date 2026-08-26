// Command ai-config-manager installs agent and skill definitions from a
// source directory into a target directory. See usage() for the full
// description of what it does.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/atfoc/agents/ai-config-manager/internal/installer"
	"github.com/atfoc/agents/ai-config-manager/internal/tui"
)

// errHelp is returned by parseArgs when the user asked for help (-h /
// --help) rather than made a mistake. run() checks for it with errors.Is so
// it can be told apart from a genuine usage error: help goes to stdout with
// exit 0, a usage error goes to stderr with exit 1.
var errHelp = errors.New("help requested")

// nameList collects a repeatable string flag (--install, --uninstall) into
// a slice. stdlib flag has no repeatable-string flag: a plain StringVar
// would let the last occurrence silently overwrite the earlier ones, which
// for --uninstall would mean quietly removing one item when the user asked
// for three.
type nameList []string

func (n *nameList) String() string {
	return strings.Join(*n, ", ")
}

// Set appends v, rejecting an empty name and skipping one already present.
// A repeated name is not an error: these flags are declarative ("these are
// the items I want"), and listing one twice describes the same wish.
// Deduplicating here rather than at use keeps the report from printing the
// same item twice. An empty name, by contrast, cannot match anything, and
// catching it here names the actual problem instead of letting a later
// "no such agents: \"\"" do it badly.
func (n *nameList) Set(v string) error {
	if v == "" {
		return errors.New("name cannot be empty")
	}
	for _, existing := range *n {
		if existing == v {
			return nil
		}
	}
	*n = append(*n, v)
	return nil
}

// usage is hand-written rather than produced by fs.PrintDefaults(). The
// short and long forms of each flag (-s/--source, -t/--target) are
// registered with the flag package as separate entries bound to the same
// variable, which is how stdlib flag does aliasing — but it means
// PrintDefaults would list them as two unrelated options instead of one
// aliased pair. Writing the text out by hand is the only way to present
// them the way they're actually meant to be used.
func usage(w io.Writer) {
	fmt.Fprint(w, `Usage: ai-config-manager -s DIR -t DIR [--dry-run]
       ai-config-manager -t DIR (--agents | --skills) --uninstall NAME [--dry-run]
       ai-config-manager -i -s DIR -t DIR
       ai-config-manager -h | --help

Installs agent and skill definitions from a source directory into a target
directory, and removes ones already installed there:

  <source>/agents/<name>.md   ->  <target>/agents/<name>.md
  <source>/skills/<name>/     ->  <target>/skills/<name>/

Every item the source ships is replaced in the target, with no prompting and
no backup. The tool reads and lists everything in the target's agents/ and
skills/, and removes only what you explicitly name or mark; nothing else in
the target is ever read, written, or removed.

Options:
  -s, --source DIR   Directory holding the agents/ and skills/ to install
                     from. Required, except with --uninstall.
  -t, --target DIR   Directory to install agents/ and skills/ into, and to
                     remove them from. Created if it does not exist.
                     Required.
      --dry-run      Print what would happen but write nothing.
  -i                 Choose what to install and remove in an interactive
                     terminal UI. Only -s and -t may be combined with it.
      --agents       Restrict the run to agents.
      --skills       Restrict the run to skills.
      --install NAME Install only the item with this exact name. Repeatable.
                     Must be used with --agents or --skills. NAME is the
                     bare name: "scout", not "scout.md".
      --uninstall NAME
                     Remove the item with this exact name from the target.
                     Repeatable. Must be used with --agents or --skills, and
                     cannot be combined with -s or --install. Every name must
                     already exist in the target, or nothing is removed.
  -h, --help         Show this help and exit.

Exit codes:
  0   success
  1   usage error or fatal error

Examples:
  ai-config-manager -s ./claude -t ~/.claude
  ai-config-manager --source ./cursor --target ~/.cursor --dry-run
  ai-config-manager -i -s ./claude -t ~/.claude
  ai-config-manager -s ./claude -t ~/.claude --agents --install scout
  ai-config-manager -t ~/.claude --skills --uninstall research --uninstall stale
`)
}

// parseArgs parses args into installer.Options. Requiredness of --source and
// --target is checked after Parse returns, against the resulting values
// rather than fs.NFlag(): either the short or the long form of a flag can
// supply it, so counting how many flags were seen on the command line can't
// tell "given" from "missing" the way checking the value itself can.
func parseArgs(args []string) (installer.Options, error) {
	var opts installer.Options

	fs := flag.NewFlagSet("ai-config-manager", flag.ContinueOnError)
	// The flag package's own error/usage output is not what we want printed:
	// run() decides where errors and usage go (stderr vs stdout, with our
	// own hand-written text), so flag's default reporting is silenced here.
	fs.SetOutput(io.Discard)

	fs.StringVar(&opts.Source, "source", "", "directory holding the agents/ and skills/ to install from")
	fs.StringVar(&opts.Source, "s", "", "shorthand for --source")
	fs.StringVar(&opts.Target, "target", "", "directory to install agents/ and skills/ into")
	fs.StringVar(&opts.Target, "t", "", "shorthand for --target")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "print what would happen but write nothing")
	fs.BoolVar(&opts.OnlyAgents, "agents", false, "install only agents")
	fs.BoolVar(&opts.OnlySkills, "skills", false, "install only skills")
	var install, uninstall nameList
	fs.Var(&install, "install", "install only the items with these exact names")
	fs.Var(&uninstall, "uninstall", "remove the items with these exact names from the target")
	fs.BoolVar(&opts.Interactive, "i", false, "choose what to install in an interactive terminal UI")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return installer.Options{}, errHelp
		}
		return installer.Options{}, fmt.Errorf("%w", err)
	}

	// Unchanged, and still first: flag stops parsing at the first non-flag
	// argument, so e.g. "-s a b -t c" leaves "-t c" unparsed behind the stray
	// "b". Checking requiredness first would misreport that as a missing
	// --target instead of the actual problem, the stray "b".
	if fs.NArg() > 0 {
		return installer.Options{}, fmt.Errorf("unexpected argument: %q", fs.Arg(0))
	}

	opts.Install = install
	opts.Uninstall = uninstall
	installing := len(opts.Install) > 0
	uninstalling := len(opts.Uninstall) > 0

	// Mode conflicts come before requiredness: a run asking for two jobs at
	// once has no single set of required flags to check it against. The
	// symmetry between the two flags is in the vocabulary, not in the run.
	if installing && uninstalling {
		return installer.Options{}, errors.New("--install and --uninstall cannot be combined")
	}

	// -i hands every choice to the interactive picker; combining it with any
	// flag that pre-decides part of that choice would make it unclear which one
	// wins, so -i accepts only -s and -t. Checked in a fixed order so the
	// reported conflict is deterministic regardless of which flags are set.
	if opts.Interactive {
		switch {
		case opts.DryRun:
			return installer.Options{}, errors.New("-i cannot be combined with --dry-run")
		case opts.OnlyAgents:
			return installer.Options{}, errors.New("-i cannot be combined with --agents")
		case opts.OnlySkills:
			return installer.Options{}, errors.New("-i cannot be combined with --skills")
		case installing:
			return installer.Options{}, errors.New("-i cannot be combined with --install")
		case uninstalling:
			return installer.Options{}, errors.New("-i cannot be combined with --uninstall")
		}
	}

	// Removing something from the target needs nothing from the source, so a
	// source given alongside --uninstall is not merely redundant — it means the
	// caller believes the source has a say in what gets deleted, which is
	// exactly the misunderstanding this rule exists to catch.
	if uninstalling && opts.Source != "" {
		return installer.Options{}, errors.New("--uninstall cannot be combined with --source")
	}

	if !uninstalling && opts.Source == "" {
		return installer.Options{}, errors.New("--source is required")
	}
	if opts.Target == "" {
		return installer.Options{}, errors.New("--target is required")
	}

	// --agents and --skills each restrict the run to a single kind; taken
	// together they'd restrict to both at once, which is the same as neither,
	// and would leave --install/--uninstall's target kind ambiguous.
	if opts.OnlyAgents && opts.OnlySkills {
		return installer.Options{}, errors.New("--agents and --skills are mutually exclusive")
	}

	// A bare name is ambiguous between an agent and a skill. For --install that
	// would install the wrong thing; for --uninstall it would delete the wrong
	// thing, and an irreversible operation must not guess.
	if installing && !opts.OnlyAgents && !opts.OnlySkills {
		return installer.Options{}, errors.New("--install requires --agents or --skills")
	}
	if uninstalling && !opts.OnlyAgents && !opts.OnlySkills {
		return installer.Options{}, errors.New("--uninstall requires --agents or --skills")
	}

	return opts, nil
}

// run implements the CLI's behavior with its output writers passed in,
// rather than reaching for os.Stdout/os.Stderr directly, so it can be
// exercised by tests without spawning a process.
func run(stdout, stderr io.Writer, args []string) int {
	opts, err := parseArgs(args)
	if errors.Is(err, errHelp) {
		// The user asked for help: that's success, and it belongs on stdout.
		usage(stdout)
		return 0
	}
	if err != nil {
		// A usage mistake: that's a failure, and it belongs on stderr.
		fmt.Fprintf(stderr, "error: %v\n\n", err)
		usage(stderr)
		return 1
	}

	if opts.Interactive {
		return runInteractive(stdout, stderr, opts)
	}

	if len(opts.Uninstall) > 0 {
		res, err := installer.RunUninstall(opts)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		if err := installer.RenderRemoved(stdout, res, opts.DryRun); err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		return 0
	}

	res, err := installer.Run(opts)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	if err := installer.Render(stdout, res, opts.DryRun); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	return 0
}

// runInteractive drives the -i picker. It deliberately plans and hands the
// plans to the UI rather than calling installer.Run: nothing may be written
// or deleted before the user has had a chance to choose.
func runInteractive(stdout, stderr io.Writer, opts installer.Options) int {
	// The picker reads keystrokes directly from the terminal, so a pipe or
	// redirect on stdin can't drive it. Failing clearly here beats hanging
	// waiting for input that will never come, or rendering garbage into
	// whatever stdin actually is.
	info, err := os.Stdin.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		fmt.Fprintln(stderr, "error: -i needs an interactive terminal (stdin is not a tty)")
		return 1
	}

	res, tgt, err := installer.PlanInteractive(opts)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	out, err := tui.Run(res, tgt)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	if !out.Confirmed {
		// The user quit without applying anything: that's not an error.
		return 0
	}

	// The report is printed only now, after tui.Run has returned: the inline
	// TUI erases its own last frame when it quits, so the report can't be
	// printed while the picker is still the terminal's active frame - it
	// would just be erased along with it.
	if err := tui.RenderReport(stdout, out); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	for _, item := range out.Applied {
		if item.Err != nil {
			return 1
		}
	}
	return 0
}

func main() {
	os.Exit(run(os.Stdout, os.Stderr, os.Args[1:]))
}
