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
  -h, --help         Show this help and exit.

Exit codes:
  0   success
  1   usage error or fatal error

Examples:
  ai-config-manager -s ./claude -t ~/.claude
  ai-config-manager --source ./cursor --target ~/.cursor --dry-run
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

func main() {
	os.Exit(run(os.Stdout, os.Stderr, os.Args[1:]))
}
