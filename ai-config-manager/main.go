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

	"github.com/atfoc/agents/ai-config-manager/internal/installer"
	"github.com/atfoc/agents/ai-config-manager/internal/tui"
)

// errHelp is returned by parseArgs when the user asked for help (-h /
// --help) rather than made a mistake. run() checks for it with errors.Is so
// it can be told apart from a genuine usage error: help goes to stdout with
// exit 0, a usage error goes to stderr with exit 1.
var errHelp = errors.New("help requested")

// usage is hand-written rather than produced by fs.PrintDefaults(). The
// short and long forms of each flag (-s/--source, -t/--target) are
// registered with the flag package as separate entries bound to the same
// variable, which is how stdlib flag does aliasing — but it means
// PrintDefaults would list them as two unrelated options instead of one
// aliased pair. Writing the text out by hand is the only way to present
// them the way they're actually meant to be used.
func usage(w io.Writer) {
	fmt.Fprint(w, `Usage: ai-config-manager -s DIR -t DIR [--dry-run]
       ai-config-manager -i -s DIR -t DIR
       ai-config-manager -h | --help

Installs agent and skill definitions from a source directory into a target
directory:

  <source>/agents/<name>.md   ->  <target>/agents/<name>.md
  <source>/skills/<name>/     ->  <target>/skills/<name>/

Every item the source ships is replaced in the target, with no prompting and
no backup. Anything else already present in the target's agents/ and skills/
is never read, written, or removed.

Options:
  -s, --source DIR   Directory holding the agents/ and skills/ to install
                     from. Required.
  -t, --target DIR   Directory to install agents/ and skills/ into. Created
                     if it does not exist. Required.
      --dry-run      Print what would happen but write nothing.
  -i                 Choose what to install in an interactive terminal
                     UI. Only -s and -t may be combined with it.
      --agents       Install only agents.
      --skills       Install only skills.
      --filter NAME  Install only the item with this exact name. Must
                     be used with --agents or --skills. NAME is the
                     bare name: "scout", not "scout.md".
  -h, --help         Show this help and exit.

Exit codes:
  0   success
  1   usage error or fatal error

Examples:
  ai-config-manager -s ./claude -t ~/.claude
  ai-config-manager --source ./cursor --target ~/.cursor --dry-run
  ai-config-manager -i -s ./claude -t ~/.claude
  ai-config-manager -s ./claude -t ~/.claude --agents --filter scout
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
	fs.StringVar(&opts.Filter, "filter", "", "install only the item with this exact name")
	fs.BoolVar(&opts.Interactive, "i", false, "choose what to install in an interactive terminal UI")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return installer.Options{}, errHelp
		}
		return installer.Options{}, fmt.Errorf("%w", err)
	}

	// Checked before requiredness: flag stops parsing at the first non-flag
	// argument, so e.g. "-s a b -t c" leaves "-t c" sitting unparsed in
	// fs.Args() behind the stray "b". Checking requiredness first would
	// misreport that as a missing --target instead of the actual problem,
	// the stray "b".
	if fs.NArg() > 0 {
		return installer.Options{}, fmt.Errorf("unexpected argument: %q", fs.Arg(0))
	}
	if opts.Source == "" {
		return installer.Options{}, errors.New("--source is required")
	}
	if opts.Target == "" {
		return installer.Options{}, errors.New("--target is required")
	}

	// --agents and --skills each restrict the run to a single kind; taken
	// together they'd restrict to both kinds at once, which is the same as
	// neither, and would also leave --filter's target kind ambiguous.
	if opts.OnlyAgents && opts.OnlySkills {
		return installer.Options{}, errors.New("--agents and --skills are mutually exclusive")
	}
	// --filter narrows within a kind; without --agents or --skills there is
	// no kind to narrow within. Skipped when -i is set: in that case --filter
	// is invalid regardless of --agents/--skills, and the check below names
	// -i as the actual conflict instead of this more generic one.
	if opts.Filter != "" && !opts.OnlyAgents && !opts.OnlySkills && !opts.Interactive {
		return installer.Options{}, errors.New("--filter requires --agents or --skills")
	}
	// -i hands every choice to the interactive picker; combining it with any
	// flag that pre-decides part of that choice would make it unclear which
	// one wins, so -i accepts only -s and -t. Checked in a fixed order so the
	// reported conflict is deterministic regardless of which flags are set.
	if opts.Interactive {
		switch {
		case opts.DryRun:
			return installer.Options{}, errors.New("-i cannot be combined with --dry-run")
		case opts.OnlyAgents:
			return installer.Options{}, errors.New("-i cannot be combined with --agents")
		case opts.OnlySkills:
			return installer.Options{}, errors.New("-i cannot be combined with --skills")
		case opts.Filter != "":
			return installer.Options{}, errors.New("-i cannot be combined with --filter")
		}
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
// plan to the UI rather than calling installer.Run: nothing may be written
// before the user has had a chance to choose what to install.
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

	res, err := installer.Plan(opts)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	out, err := tui.Run(res)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	if !out.Confirmed {
		// The user quit without installing anything: that's not an error.
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
