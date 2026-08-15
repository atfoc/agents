#!/usr/bin/env bash
#
# install-claude-config.sh
#
# Copies this repo's Claude agent and skill definitions into a target
# project's .claude/ directory. Every item this repo ships is
# unconditionally replaced in the target; everything else already living in
# the target's .claude/ is left completely untouched.
#
# Run with -h/--help for full usage.

set -euo pipefail

# ---------------------------------------------------------------------------
# Usage
# ---------------------------------------------------------------------------

SCRIPT_NAME="$(basename "$0")"

print_usage() {
  cat <<EOF
Usage: ${SCRIPT_NAME} [--target DIR | --target-home] [--dry-run] [--uninstall]
       ${SCRIPT_NAME} -h | --help

Installs this repo's Claude agent and skill definitions into a target
project's .claude/ directory:

  <target>/.claude/agents/scout.md
  <target>/.claude/agents/thinker.md
  <target>/.claude/agents/worker.md
  <target>/.claude/skills/make-plan/SKILL.md
  <target>/.claude/skills/implement-plan/SKILL.md
  <target>/.claude/skills/research/SKILL.md

Every item this repo ships is ALWAYS replaced in the target -- whatever is
currently there (a real file, a real directory, or a symlink) -- with no
prompting and no backup. Anything else already present under the target's
.claude/ (your own agents and skills, plus Claude Code's own sessions/,
projects/, history.jsonl, settings.json, etc.) is never read or written.

Options:
  --target DIR   Install into DIR/.claude instead of \$PWD/.claude. DIR does
                 not need to exist yet; it (and .claude/agents, .claude/skills)
                 will be created as needed. Refuses to run if DIR resolves to
                 a path inside this repo (that would let the repo install
                 into itself). Cannot be combined with --target-home.

  --target-home  Install into \$HOME/.claude, the personal location that
                 applies to all your projects. Cannot be combined with
                 --target.

  --dry-run      Print what would happen (install/updated/unchanged/removed/
                 absent, one line per item, prefixed with "[dry-run] ") but
                 do not create, modify, or remove anything on disk.

  --uninstall    Remove exactly this repo's six items from the target's
                 .claude/agents and .claude/skills, then try to rmdir those
                 two directories (silently left in place if the project has
                 its own entries in them). Nothing else under .claude/ is
                 touched.

  -h, --help     Show this help and exit.

Exit codes:
  0   success
  1   fatal error or usage error (unknown flag, missing --target argument,
      both --target and --target-home given, target resolves inside this
      repo, this repo's own agents/ missing)

Examples:
  # Install (copy) into the current project
  ${SCRIPT_NAME}

  # Install into another project by path
  ${SCRIPT_NAME} --target ~/code/some-project

  # Install into your personal ~/.claude, used across all projects
  ${SCRIPT_NAME} --target-home

  # See what would change without touching anything
  ${SCRIPT_NAME} --target ~/code/some-project --dry-run

  # Remove this repo's items from a project, leaving its own entries alone
  ${SCRIPT_NAME} --target ~/code/some-project --uninstall
EOF
}

usage_error() {
  echo "error: $1" >&2
  echo >&2
  print_usage >&2
  exit 1
}

die() {
  echo "error: $1" >&2
  exit 1
}

# ---------------------------------------------------------------------------
# Path resolution helpers (BSD/macOS + bash 3.2 portable; no `readlink -f`,
# no GNU-only flags). /bin/realpath is present on this platform and is
# preferred; the loop below is the portable fallback and also documents what
# realpath is doing.
# ---------------------------------------------------------------------------

# resolve PATH
# Print the absolute, symlink-resolved form of an EXISTING file or directory.
resolve() {
  local target="$1"

  if command -v realpath >/dev/null 2>&1; then
    realpath "$target"
    return
  fi

  while [ -L "$target" ]; do
    local link
    link="$(readlink "$target")"
    case "$link" in
      /*) target="$link" ;;
      *)  target="$(dirname "$target")/$link" ;;
    esac
  done

  local dir base
  dir="$(cd -P "$(dirname "$target")" && pwd)"
  base="$(basename "$target")"
  printf '%s/%s\n' "$dir" "$base"
}

# resolve_target PATH
# Like resolve(), but PATH may not exist yet (e.g. a --target that hasn't
# been created). Resolves the longest existing prefix through symlinks and
# reattaches the remaining, not-yet-existing tail as plain text.
resolve_target() {
  local input="$1"
  local cur="$input"
  local suffix=""

  while [ ! -d "$cur" ]; do
    local b parent
    b="$(basename "$cur")"
    if [ -z "$suffix" ]; then
      suffix="$b"
    else
      suffix="$b/$suffix"
    fi
    parent="$(dirname "$cur")"
    if [ "$parent" = "$cur" ]; then
      break
    fi
    cur="$parent"
  done

  local resolved
  resolved="$(resolve "$cur")"
  if [ -n "$suffix" ]; then
    printf '%s/%s\n' "$resolved" "$suffix"
  else
    printf '%s\n' "$resolved"
  fi
}

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------

TARGET="$PWD"
TARGET_SET=0
HOME_SET=0
DRY_RUN=0
UNINSTALL=0

while [ $# -gt 0 ]; do
  case "$1" in
    --target)
      if [ $# -lt 2 ]; then
        usage_error "--target requires an argument"
      fi
      TARGET="$2"
      TARGET_SET=1
      shift 2
      ;;
    --target-home)
      HOME_SET=1
      shift
      ;;
    --dry-run)
      DRY_RUN=1
      shift
      ;;
    --uninstall)
      UNINSTALL=1
      shift
      ;;
    -h|--help)
      print_usage
      exit 0
      ;;
    *)
      usage_error "unknown argument: $1"
      ;;
  esac
done

if [ "$TARGET_SET" -eq 1 ] && [ "$HOME_SET" -eq 1 ]; then
  usage_error "--target and --target-home are mutually exclusive"
fi

if [ "$HOME_SET" -eq 1 ]; then
  [ -n "${HOME:-}" ] || die "--target-home given but \$HOME is not set"
  TARGET="$HOME"
fi

if [ -z "$TARGET" ]; then
  usage_error "--target requires a non-empty argument"
fi

# ---------------------------------------------------------------------------
# Locate this repo relative to the running script (works even if this
# script is itself symlinked, e.g. onto \$PATH).
# ---------------------------------------------------------------------------

SRC_ROOT="$(dirname "$(resolve "$0")")"
REPO_ROOT="$(dirname "$SRC_ROOT")"

[ -d "$SRC_ROOT/agents" ] || die "expected '$SRC_ROOT/agents' to exist but it does not; this script must stay in place alongside the repo's agents/ and skills/ directories"

DEST="$TARGET/.claude"

# Guard: refuse to install the repo into itself.
TARGET_RESOLVED="$(resolve_target "$TARGET")"
case "$TARGET_RESOLVED" in
  "$REPO_ROOT"|"$REPO_ROOT"/*)
    die "target '$TARGET' resolves to '$TARGET_RESOLVED', which is inside this repo ('$REPO_ROOT'); installing the repo into itself would recurse"
    ;;
esac

DRY_PREFIX=""
[ "$DRY_RUN" -eq 1 ] && DRY_PREFIX="[dry-run] "

# ---------------------------------------------------------------------------
# Status bookkeeping
# ---------------------------------------------------------------------------

count_install=0
count_updated=0
count_unchanged=0
count_removed=0
count_absent=0

status_line() {
  local status="$1" label="$2"
  printf '%s%s %s\n' "$DRY_PREFIX" "$status" "$label"
  case "$status" in
    install)   count_install=$((count_install + 1)) ;;
    updated)   count_updated=$((count_updated + 1)) ;;
    unchanged) count_unchanged=$((count_unchanged + 1)) ;;
    removed)   count_removed=$((count_removed + 1)) ;;
    absent)    count_absent=$((count_absent + 1)) ;;
  esac
}

# install_item KIND SRC_REAL DEST LABEL
# KIND is "file" or "dir". SRC_REAL must already be symlink-resolved (see
# the cp -R hazard note below). Always leaves DEST holding a fresh copy of
# SRC_REAL, replacing whatever was there.
install_item() {
  local kind="$1" src_real="$2" dest="$3" label="$4"
  local status

  if [ -L "$dest" ]; then
    # Any symlink here is replaced by a real file/dir, so this is always a
    # change -- including a link whose target matches byte-for-byte. Tested
    # before -e, which follows the link and would report "unchanged".
    # This also covers a dangling link.
    status="updated"
  elif [ -e "$dest" ]; then
    if [ "$kind" = "file" ]; then
      if cmp -s "$src_real" "$dest" 2>/dev/null; then
        status="unchanged"
      else
        status="updated"
      fi
    else
      if diff -rq "$src_real" "$dest" >/dev/null 2>&1; then
        status="unchanged"
      else
        status="updated"
      fi
    fi
  else
    status="install"
  fi

  if [ "$DRY_RUN" -eq 0 ]; then
    # Always remove first: `cp -R src existing_dir` nests a duplicate copy
    # inside existing_dir instead of replacing it, and leaves stale files
    # behind. Removing dest first makes cp (re)create it fresh every time,
    # for both files and directories. It also means a symlink at dest is
    # replaced, rather than written through -- `cp` over a symlink modifies
    # the link's target file, which would corrupt a file we were never asked
    # to touch.
    rm -rf "$dest"
    if [ "$kind" = "file" ]; then
      cp "$src_real" "$dest"
    else
      cp -R "$src_real" "$dest"
    fi
  fi

  status_line "$status" "$label"
}

# uninstall_item DEST LABEL
uninstall_item() {
  local dest="$1" label="$2"
  if [ -e "$dest" ] || [ -L "$dest" ]; then
    if [ "$DRY_RUN" -eq 0 ]; then
      rm -rf "$dest"
    fi
    status_line "removed" "$label"
  else
    status_line "absent" "$label"
  fi
}

# ---------------------------------------------------------------------------
# Main flow: agents (*.md files), then skills (directories).
# ---------------------------------------------------------------------------

for kind in agents skills; do
  src_kind_dir="$SRC_ROOT/$kind"
  dest_kind_dir="$DEST/$kind"

  if [ "$UNINSTALL" -eq 0 ] && [ "$DRY_RUN" -eq 0 ]; then
    mkdir -p "$dest_kind_dir"
  fi

  for src in "$src_kind_dir"/*; do
    [ -e "$src" ] || continue

    name="$(basename "$src")"

    if [ "$kind" = "agents" ]; then
      case "$name" in
        *.md) ;;
        *) continue ;;
      esac
      item_kind="file"
    else
      # Only directories, judged after following symlinks (`-d` already
      # follows symlinks, so a symlink-to-directory passes this test).
      [ -d "$src" ] || continue
      item_kind="dir"
    fi

    dest="$dest_kind_dir/$name"
    label="$kind/$name"

    if [ "$UNINSTALL" -eq 1 ]; then
      uninstall_item "$dest" "$label"
      continue
    fi

    # Hazard: `cp -R` on a symlink argument copies the symlink itself, not
    # its content, which would leave a dangling link in the target once the
    # source repo moves or is deleted. Resolve to the real path first.
    src_real="$(resolve "$src")"

    install_item "$item_kind" "$src_real" "$dest" "$label"
  done
done

if [ "$UNINSTALL" -eq 1 ] && [ "$DRY_RUN" -eq 0 ]; then
  rmdir "$DEST/agents" 2>/dev/null || true
  rmdir "$DEST/skills" 2>/dev/null || true
fi

printf '%s%d install, %d updated, %d unchanged, %d removed, %d absent\n' \
  "$DRY_PREFIX" "$count_install" "$count_updated" "$count_unchanged" "$count_removed" "$count_absent"

exit 0
