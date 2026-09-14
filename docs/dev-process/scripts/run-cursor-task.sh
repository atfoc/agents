#!/usr/bin/env bash
# Start a new, non-interactive Cursor agent session on one prompt and print its final report.
#
# Usage:
#     run-cursor-task.sh "<prompt>"          prompt as arguments
#     run-cursor-task.sh < prompt.txt        prompt on stdin, when no arguments are given
#
# The session runs in the current working directory as its trusted workspace, answers no permission
# prompts (every command is force-allowed unless explicitly denied), and can spawn subagents of its
# own. stdout is the session's final message; the exit code is the agent's.
set -euo pipefail

if [ "$#" -gt 0 ]; then
    prompt="$*"
else
    prompt="$(cat)"
fi

if [ -z "${prompt//[[:space:]]/}" ]; then
    echo "run-cursor-task.sh: no prompt given (pass it as arguments or on stdin)" >&2
    exit 2
fi

exec agent -p --force --trust --output-format text "$prompt"
