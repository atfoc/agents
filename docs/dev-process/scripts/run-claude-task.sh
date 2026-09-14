#!/usr/bin/env bash
# Start a new, non-interactive Claude Code session on one prompt and print its final report.
#
# Usage:
#     run-claude-task.sh "<prompt>"          prompt as arguments
#     run-claude-task.sh < prompt.txt        prompt on stdin, when no arguments are given
#
# The session runs in the current working directory, answers no permission prompts (auto mode), and
# can spawn subagents of its own. stdout is the session's final message; the exit code is Claude's.
set -euo pipefail

if [ "$#" -gt 0 ]; then
    prompt="$*"
else
    prompt="$(cat)"
fi

if [ -z "${prompt//[[:space:]]/}" ]; then
    echo "run-claude-task.sh: no prompt given (pass it as arguments or on stdin)" >&2
    exit 2
fi

exec claude -p --permission-mode auto --output-format text "$prompt"
