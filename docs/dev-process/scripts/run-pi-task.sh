#!/usr/bin/env bash
# Start a new, non-interactive pi session on one prompt and print its final report.
#
# Usage:
#     run-pi-task.sh "<prompt>"          prompt as arguments
#     run-pi-task.sh < prompt.txt        prompt on stdin, when no arguments are given
#
# The session runs in the current working directory with project files trusted, answers no
# permission prompts (tools run without asking), and can spawn subagents of its own. stdout is
# the session's final message; the exit code is pi's.
set -euo pipefail

if [ "$#" -gt 0 ]; then
    prompt="$*"
else
    prompt="$(cat)"
fi

if [ -z "${prompt//[[:space:]]/}" ]; then
    echo "run-pi-task.sh: no prompt given (pass it as arguments or on stdin)" >&2
    exit 2
fi

exec pi -p --mode text --approve "$prompt" </dev/null
