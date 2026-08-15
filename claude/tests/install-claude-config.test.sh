#!/usr/bin/env bash
#
# install-claude-config.test.sh
#
# Dependency-free bash unit-test suite for install-claude-config.sh. No bats,
# no shellcheck -- plain bash, written to also run under bash 3.2 (macOS
# /bin/bash): no associative arrays, no mapfile/readarray, no ${var,,}, no
# declare -A.
#
# Deliberately does NOT `set -e`: several tests exercise commands that are
# expected to exit non-zero, and letting those kill the harness would be
# exactly the trap the spec warns about. Every command whose status matters
# is checked explicitly instead.

TEST_DIR="$(cd "$(dirname "$0")" && pwd)"
REAL_SCRIPT="$TEST_DIR/../install-claude-config.sh"

BASE_TMP="$(mktemp -d "${TMPDIR:-/tmp}/install-claude-config-test.XXXXXX")"
trap 'rm -rf "$BASE_TMP"' EXIT

TESTS_RUN=0
TESTS_FAILED=0
CURRENT_TEST_FAILED=0

LABELS="agents/scout.md agents/thinker.md agents/worker.md skills/make-plan skills/implement-plan skills/research"

# ---------------------------------------------------------------------------
# Assertion helpers
# ---------------------------------------------------------------------------

fail() {
  CURRENT_TEST_FAILED=1
  echo "    FAIL: $1" >&2
}

assert_eq() {
  local expected="$1" actual="$2" msg="${3:-}"
  if [ "$expected" != "$actual" ]; then
    fail "expected [$expected] but got [$actual] -- $msg"
  fi
}

assert_file() {
  local path="$1" msg="${2:-}"
  if [ -L "$path" ]; then
    fail "expected regular file (not a symlink): $path -- $msg"
  elif [ ! -f "$path" ]; then
    fail "expected regular file to exist: $path -- $msg"
  fi
}

assert_dir() {
  local path="$1" msg="${2:-}"
  if [ -L "$path" ]; then
    fail "expected real directory (not a symlink): $path -- $msg"
  elif [ ! -d "$path" ]; then
    fail "expected directory to exist: $path -- $msg"
  fi
}

assert_symlink() {
  local path="$1" msg="${2:-}"
  if [ ! -L "$path" ]; then
    fail "expected symlink: $path -- $msg"
  fi
}

assert_not_symlink() {
  local path="$1" msg="${2:-}"
  if [ -L "$path" ]; then
    fail "expected not to be a symlink: $path -- $msg"
  elif [ ! -e "$path" ]; then
    fail "expected to exist (and not be a symlink): $path -- $msg"
  fi
}

assert_missing() {
  local path="$1" msg="${2:-}"
  if [ -e "$path" ] || [ -L "$path" ]; then
    fail "expected path to be absent: $path -- $msg"
  fi
}

assert_contains() {
  local haystack="$1" needle="$2" msg="${3:-}"
  case "$haystack" in
    *"$needle"*) ;;
    *) fail "expected output to contain [$needle] -- $msg" ;;
  esac
}

assert_files_equal() {
  local a="$1" b="$2" msg="${3:-}"
  if ! cmp -s "$a" "$b"; then
    fail "expected file contents to match: $a vs $b -- $msg"
  fi
}

# ---------------------------------------------------------------------------
# Fixture builder
#
# Reproduces the real repo's two-level shape:
#   $FIXTURE/                          <- acts as REPO_ROOT
#   `- claude/                         <- acts as SRC_ROOT
#      |- install-claude-config.sh        (copy of the real script, +x)
#      |- agents/{scout,thinker,worker}.md
#      `- skills/{make-plan,implement-plan,research}/SKILL.md
#
# Unlike the cursor sibling's fixture, these skills are real directories, not
# symlinks -- only test_02 below builds a symlinked source skill.
#
# The project/home target directories are SEPARATE mktemp trees so they never
# resolve inside the fixture and never trip the self-install guard.
# ---------------------------------------------------------------------------

new_fixture_dir() {
  mktemp -d "$BASE_TMP/fixture.XXXXXX"
}

new_project_dir() {
  mktemp -d "$BASE_TMP/project.XXXXXX"
}

new_home_dir() {
  mktemp -d "$BASE_TMP/home.XXXXXX"
}

make_fixture() {
  local fixture="$1"

  mkdir -p "$fixture/claude/agents"
  mkdir -p "$fixture/claude/skills/make-plan"
  mkdir -p "$fixture/claude/skills/implement-plan"
  mkdir -p "$fixture/claude/skills/research"

  printf '# Scout Agent (fixture)\n' > "$fixture/claude/agents/scout.md"
  printf '# Thinker Agent (fixture)\n' > "$fixture/claude/agents/thinker.md"
  printf '# Worker Agent (fixture)\n' > "$fixture/claude/agents/worker.md"

  printf '# Make Plan Skill (fixture)\n' > "$fixture/claude/skills/make-plan/SKILL.md"
  printf '# Implement Plan Skill (fixture)\n' > "$fixture/claude/skills/implement-plan/SKILL.md"
  printf '# Research Skill (fixture)\n' > "$fixture/claude/skills/research/SKILL.md"

  cp "$REAL_SCRIPT" "$fixture/claude/install-claude-config.sh"
  chmod +x "$fixture/claude/install-claude-config.sh"
}

# run_installer SCRIPT [ARGS...]
# Sets INSTALLER_OUT, INSTALLER_ERR, INSTALLER_STATUS.
run_installer() {
  local script="$1"
  shift
  local errfile
  errfile="$(mktemp "$BASE_TMP/stderr.XXXXXX")"
  INSTALLER_OUT="$("$script" "$@" 2>"$errfile")"
  INSTALLER_STATUS=$?
  INSTALLER_ERR="$(cat "$errfile")"
  rm -f "$errfile"
}

# run_installer_in DIR SCRIPT [ARGS...]
# Like run_installer, but runs with the working directory changed to DIR
# first. Needed for the default-target (no --target/--target-home) test.
run_installer_in() {
  local dir="$1" script="$2"
  shift 2
  local errfile
  errfile="$(mktemp "$BASE_TMP/stderr.XXXXXX")"
  INSTALLER_OUT="$(cd "$dir" && "$script" "$@" 2>"$errfile")"
  INSTALLER_STATUS=$?
  INSTALLER_ERR="$(cat "$errfile")"
  rm -f "$errfile"
}

# run_installer_home HOMEDIR SCRIPT [ARGS...]
# Like run_installer, but runs with HOME overridden to HOMEDIR. Needed for
# the --target-home test, since the script reads the HOME environment
# variable rather than $HOME expanded ahead of time.
run_installer_home() {
  local home="$1" script="$2"
  shift 2
  local errfile
  errfile="$(mktemp "$BASE_TMP/stderr.XXXXXX")"
  INSTALLER_OUT="$(HOME="$home" "$script" "$@" 2>"$errfile")"
  INSTALLER_STATUS=$?
  INSTALLER_ERR="$(cat "$errfile")"
  rm -f "$errfile"
}

plant_foreign_entries() {
  local project="$1"
  mkdir -p "$project/.claude/agents"
  mkdir -p "$project/.claude/skills/my-own-skill"
  printf 'my own custom agent, do not touch\n' > "$project/.claude/agents/my-own-agent.md"
  printf 'my own custom skill, do not touch\n' > "$project/.claude/skills/my-own-skill/SKILL.md"
}

# assert_label_installed BASE LABEL [MSG]
# LABEL is one of the entries in $LABELS ("agents/scout.md", "skills/make-plan",
# ...). Asserts BASE/LABEL exists as the right kind of real (non-symlink)
# entry: a regular file for agents/*, a real directory for skills/*.
assert_label_installed() {
  local base="$1" label="$2" msg="${3:-}"
  case "$label" in
    agents/*)
      assert_file "$base/$label" "$msg"
      assert_not_symlink "$base/$label" "$msg"
      ;;
    skills/*)
      assert_dir "$base/$label" "$msg"
      assert_not_symlink "$base/$label" "$msg"
      ;;
  esac
}

# ---------------------------------------------------------------------------
# Test runner
# ---------------------------------------------------------------------------

run_test() {
  local name="$1"
  TESTS_RUN=$((TESTS_RUN + 1))
  CURRENT_TEST_FAILED=0

  "$name"

  if [ "$CURRENT_TEST_FAILED" -eq 0 ]; then
    echo "PASS: $name"
  else
    echo "FAIL: $name"
    TESTS_FAILED=$((TESTS_FAILED + 1))
  fi
}

# ---------------------------------------------------------------------------
# Test cases
# ---------------------------------------------------------------------------

test_01_fresh_install_copies_real_files() {
  local fixture project label
  fixture="$(new_fixture_dir)"
  project="$(new_project_dir)"
  make_fixture "$fixture"

  run_installer "$fixture/claude/install-claude-config.sh" --target "$project"
  assert_eq "0" "$INSTALLER_STATUS" "exit status"

  for label in $LABELS; do
    assert_contains "$INSTALLER_OUT" "install $label" "missing install line for $label"
    assert_label_installed "$project/.claude" "$label"
  done
  assert_contains "$INSTALLER_OUT" "6 install, 0 updated, 0 unchanged, 0 removed, 0 absent" "summary line"
}

test_02_symlinked_source_skill_is_flattened() {
  local fixture project
  fixture="$(new_fixture_dir)"
  project="$(new_project_dir)"
  make_fixture "$fixture"

  mkdir -p "$fixture/real-skills/make-plan"
  printf '# Make Plan Skill (real symlink target)\n' > "$fixture/real-skills/make-plan/SKILL.md"
  rm -rf "$fixture/claude/skills/make-plan"
  ( cd "$fixture/claude/skills" && ln -s ../../real-skills/make-plan make-plan )

  run_installer "$fixture/claude/install-claude-config.sh" --target "$project"
  assert_eq "0" "$INSTALLER_STATUS" "exit status"

  assert_dir "$project/.claude/skills/make-plan" "flattened skill dir"
  assert_not_symlink "$project/.claude/skills/make-plan"
  assert_files_equal "$fixture/real-skills/make-plan/SKILL.md" "$project/.claude/skills/make-plan/SKILL.md" "flattened SKILL.md content"
}

test_03_idempotent_rerun() {
  local fixture project label
  fixture="$(new_fixture_dir)"
  project="$(new_project_dir)"
  make_fixture "$fixture"

  run_installer "$fixture/claude/install-claude-config.sh" --target "$project"
  assert_eq "0" "$INSTALLER_STATUS" "first run exit status"

  run_installer "$fixture/claude/install-claude-config.sh" --target "$project"
  assert_eq "0" "$INSTALLER_STATUS" "second run exit status"

  for label in $LABELS; do
    assert_contains "$INSTALLER_OUT" "unchanged $label" "missing unchanged line for $label"
  done
  assert_contains "$INSTALLER_OUT" "0 install, 0 updated, 6 unchanged, 0 removed, 0 absent" "summary line"
}

test_04_edited_source_resyncs() {
  local fixture project
  fixture="$(new_fixture_dir)"
  project="$(new_project_dir)"
  make_fixture "$fixture"

  run_installer "$fixture/claude/install-claude-config.sh" --target "$project"
  assert_eq "0" "$INSTALLER_STATUS" "first run exit status"

  printf '# Make Plan Skill (EDITED)\n' > "$fixture/claude/skills/make-plan/SKILL.md"

  run_installer "$fixture/claude/install-claude-config.sh" --target "$project"
  assert_eq "0" "$INSTALLER_STATUS" "second run exit status"

  assert_contains "$INSTALLER_OUT" "updated skills/make-plan" "expected updated status for edited skill"
  assert_contains "$INSTALLER_OUT" "unchanged agents/scout.md" "unrelated file should be unchanged"
  assert_files_equal "$fixture/claude/skills/make-plan/SKILL.md" "$project/.claude/skills/make-plan/SKILL.md" "resynced content"
}

test_05_stale_file_removed_no_nested_duplicate() {
  local fixture project
  fixture="$(new_fixture_dir)"
  project="$(new_project_dir)"
  make_fixture "$fixture"

  run_installer "$fixture/claude/install-claude-config.sh" --target "$project"
  assert_eq "0" "$INSTALLER_STATUS" "first run exit status"

  printf 'stale leftover\n' > "$project/.claude/skills/make-plan/stale.txt"

  run_installer "$fixture/claude/install-claude-config.sh" --target "$project"
  assert_eq "0" "$INSTALLER_STATUS" "second run exit status"

  assert_missing "$project/.claude/skills/make-plan/stale.txt" "stale file must be gone"
  assert_missing "$project/.claude/skills/make-plan/make-plan" "must not nest a duplicate copy (BSD cp -R hazard)"
  assert_file "$project/.claude/skills/make-plan/SKILL.md"
}

test_06_own_items_overwritten_unconditionally() {
  local fixture project
  fixture="$(new_fixture_dir)"
  project="$(new_project_dir)"
  make_fixture "$fixture"

  mkdir -p "$project/.claude/agents"
  mkdir -p "$project/.claude/skills/make-plan"
  printf 'hand-written, not the fixture content\n' > "$project/.claude/agents/scout.md"
  printf 'hand-written, not the fixture content\n' > "$project/.claude/skills/make-plan/SKILL.md"

  run_installer "$fixture/claude/install-claude-config.sh" --target "$project"
  assert_eq "0" "$INSTALLER_STATUS" "exit status"

  assert_contains "$INSTALLER_OUT" "updated agents/scout.md" "hand-written agent file should register as updated"
  assert_contains "$INSTALLER_OUT" "updated skills/make-plan" "hand-written skill dir should register as updated"
  assert_files_equal "$fixture/claude/agents/scout.md" "$project/.claude/agents/scout.md" "should be replaced by fixture content"
  assert_files_equal "$fixture/claude/skills/make-plan/SKILL.md" "$project/.claude/skills/make-plan/SKILL.md" "should be replaced by fixture content"
}

test_07_foreign_entries_survive_byte_identical() {
  local fixture project before_agent before_skill after_agent after_skill
  fixture="$(new_fixture_dir)"
  project="$(new_project_dir)"
  make_fixture "$fixture"
  plant_foreign_entries "$project"

  before_agent="$(cksum "$project/.claude/agents/my-own-agent.md")"
  before_skill="$(cksum "$project/.claude/skills/my-own-skill/SKILL.md")"

  run_installer "$fixture/claude/install-claude-config.sh" --target "$project"
  assert_eq "0" "$INSTALLER_STATUS" "exit status"

  assert_file "$project/.claude/agents/my-own-agent.md"
  assert_dir "$project/.claude/skills/my-own-skill"
  assert_file "$project/.claude/skills/my-own-skill/SKILL.md"

  after_agent="$(cksum "$project/.claude/agents/my-own-agent.md")"
  after_skill="$(cksum "$project/.claude/skills/my-own-skill/SKILL.md")"

  assert_eq "$before_agent" "$after_agent" "foreign agent file must be byte-identical"
  assert_eq "$before_skill" "$after_skill" "foreign skill file must be byte-identical"
}

test_08_dry_run_changes_nothing() {
  local fixture project label
  fixture="$(new_fixture_dir)"
  project="$(new_project_dir)"
  make_fixture "$fixture"

  run_installer "$fixture/claude/install-claude-config.sh" --target "$project" --dry-run
  assert_eq "0" "$INSTALLER_STATUS" "exit status"

  for label in $LABELS; do
    assert_contains "$INSTALLER_OUT" "[dry-run] install $label" "missing dry-run install line for $label"
  done
  assert_contains "$INSTALLER_OUT" "[dry-run] 6 install, 0 updated, 0 unchanged, 0 removed, 0 absent" "dry-run summary line"

  assert_missing "$project/.claude" ".claude must not be created by --dry-run"
}

test_09_uninstall_removes_ours_keeps_foreign() {
  local fixture project label before_agent before_skill after_agent after_skill
  fixture="$(new_fixture_dir)"
  project="$(new_project_dir)"
  make_fixture "$fixture"

  run_installer "$fixture/claude/install-claude-config.sh" --target "$project"
  assert_eq "0" "$INSTALLER_STATUS" "install run exit status"

  plant_foreign_entries "$project"
  before_agent="$(cksum "$project/.claude/agents/my-own-agent.md")"
  before_skill="$(cksum "$project/.claude/skills/my-own-skill/SKILL.md")"

  run_installer "$fixture/claude/install-claude-config.sh" --target "$project" --uninstall
  assert_eq "0" "$INSTALLER_STATUS" "uninstall run exit status"

  for label in $LABELS; do
    assert_contains "$INSTALLER_OUT" "removed $label" "missing removed line for $label"
  done
  assert_contains "$INSTALLER_OUT" "0 install, 0 updated, 0 unchanged, 6 removed, 0 absent" "uninstall summary line"

  for label in $LABELS; do
    assert_missing "$project/.claude/$label"
  done

  assert_file "$project/.claude/agents/my-own-agent.md" "foreign agent must survive uninstall"
  assert_dir "$project/.claude/skills/my-own-skill" "foreign skill dir must survive uninstall"
  assert_file "$project/.claude/skills/my-own-skill/SKILL.md" "foreign skill must survive uninstall"

  assert_dir "$project/.claude/agents" "agents dir must survive since it still has foreign entries"
  assert_dir "$project/.claude/skills" "skills dir must survive since it still has foreign entries"

  after_agent="$(cksum "$project/.claude/agents/my-own-agent.md")"
  after_skill="$(cksum "$project/.claude/skills/my-own-skill/SKILL.md")"
  assert_eq "$before_agent" "$after_agent" "foreign agent file must be byte-identical after uninstall"
  assert_eq "$before_skill" "$after_skill" "foreign skill file must be byte-identical after uninstall"
}

test_10_uninstall_on_clean_project_reports_absent() {
  local fixture project label
  fixture="$(new_fixture_dir)"
  project="$(new_project_dir)"
  make_fixture "$fixture"

  run_installer "$fixture/claude/install-claude-config.sh" --target "$project" --uninstall
  assert_eq "0" "$INSTALLER_STATUS" "exit status"

  for label in $LABELS; do
    assert_contains "$INSTALLER_OUT" "absent $label" "missing absent line for $label"
  done
  assert_contains "$INSTALLER_OUT" "0 install, 0 updated, 0 unchanged, 0 removed, 6 absent" "summary line"
}

test_11_default_target_is_cwd() {
  local fixture project label
  fixture="$(new_fixture_dir)"
  project="$(new_project_dir)"
  make_fixture "$fixture"

  run_installer_in "$project" "$fixture/claude/install-claude-config.sh"
  assert_eq "0" "$INSTALLER_STATUS" "exit status"

  for label in $LABELS; do
    assert_contains "$INSTALLER_OUT" "install $label" "missing install line for $label"
    assert_label_installed "$project/.claude" "$label"
  done
}

test_12_target_home_installs_into_home() {
  local fixture project home label
  fixture="$(new_fixture_dir)"
  project="$(new_project_dir)"
  home="$(new_home_dir)"
  make_fixture "$fixture"

  run_installer_home "$home" "$fixture/claude/install-claude-config.sh" --target-home
  assert_eq "0" "$INSTALLER_STATUS" "exit status"

  for label in $LABELS; do
    assert_contains "$INSTALLER_OUT" "install $label" "missing install line for $label"
    assert_label_installed "$home/.claude" "$label"
  done

  assert_missing "$project/.claude" "--target-home must redirect, not additionally install into an unrelated project"
}

test_13_target_and_target_home_are_mutually_exclusive() {
  local fixture project home
  fixture="$(new_fixture_dir)"
  project="$(new_project_dir)"
  home="$(new_home_dir)"
  make_fixture "$fixture"

  run_installer "$fixture/claude/install-claude-config.sh" --target "$project" --target-home
  assert_eq "1" "$INSTALLER_STATUS" "--target then --target-home must exit 1"
  assert_contains "$INSTALLER_ERR" "mutually exclusive" "--target then --target-home stderr"
  assert_missing "$project/.claude"
  assert_missing "$home/.claude"

  run_installer "$fixture/claude/install-claude-config.sh" --target-home --target "$project"
  assert_eq "1" "$INSTALLER_STATUS" "--target-home then --target must exit 1"
  assert_contains "$INSTALLER_ERR" "mutually exclusive" "--target-home then --target stderr"
  assert_missing "$project/.claude"
  assert_missing "$home/.claude"
}

test_14_self_install_guard() {
  local fixture
  fixture="$(new_fixture_dir)"
  make_fixture "$fixture"

  run_installer "$fixture/claude/install-claude-config.sh" --target "$fixture"
  assert_eq "1" "$INSTALLER_STATUS" "targeting the repo root itself must exit 1"

  run_installer "$fixture/claude/install-claude-config.sh" --target "$fixture/claude"
  assert_eq "1" "$INSTALLER_STATUS" "targeting a subdirectory of the repo must exit 1"
}

test_15_usage_errors() {
  local fixture project
  fixture="$(new_fixture_dir)"
  project="$(new_project_dir)"
  make_fixture "$fixture"

  run_installer "$fixture/claude/install-claude-config.sh" --target "$project" --bogus-flag
  assert_eq "1" "$INSTALLER_STATUS" "unknown flag must exit 1"

  run_installer "$fixture/claude/install-claude-config.sh" --target
  assert_eq "1" "$INSTALLER_STATUS" "--target with no argument must exit 1"

  run_installer "$fixture/claude/install-claude-config.sh" --help
  assert_eq "0" "$INSTALLER_STATUS" "--help must exit 0"
  assert_contains "$INSTALLER_OUT" "--target-home" "help output must mention --target-home"

  run_installer "$fixture/claude/install-claude-config.sh" -h
  assert_eq "0" "$INSTALLER_STATUS" "-h must exit 0"
  assert_contains "$INSTALLER_OUT" "--target-home" "help output must mention --target-home"
}

test_16_syntax_gate() {
  if ! bash -n "$REAL_SCRIPT" 2>"$BASE_TMP/syntax-err"; then
    fail "bash -n failed on $REAL_SCRIPT: $(cat "$BASE_TMP/syntax-err")"
  fi
}

test_17_symlink_at_destination_is_replaced() {
  local fixture project before_agent before_skill after_agent after_skill
  fixture="$(new_fixture_dir)"
  project="$(new_project_dir)"
  make_fixture "$fixture"

  mkdir -p "$project/.claude/agents"
  mkdir -p "$project/.claude/skills"

  cp "$fixture/claude/agents/scout.md" "$project/link-target-agent.md"
  ln -s "$project/link-target-agent.md" "$project/.claude/agents/scout.md"

  mkdir -p "$project/link-target-skill"
  cp "$fixture/claude/skills/make-plan/SKILL.md" "$project/link-target-skill/SKILL.md"
  ln -s "$project/link-target-skill" "$project/.claude/skills/make-plan"

  before_agent="$(cksum "$project/link-target-agent.md")"
  before_skill="$(cksum "$project/link-target-skill/SKILL.md")"

  run_installer "$fixture/claude/install-claude-config.sh" --target "$project"
  assert_eq "0" "$INSTALLER_STATUS" "exit status"

  assert_file "$project/.claude/agents/scout.md" "destination must become a real file"
  assert_not_symlink "$project/.claude/agents/scout.md"
  assert_dir "$project/.claude/skills/make-plan" "destination must become a real directory"
  assert_not_symlink "$project/.claude/skills/make-plan"

  assert_contains "$INSTALLER_OUT" "updated agents/scout.md" "symlinked destination must report updated, not unchanged"
  assert_contains "$INSTALLER_OUT" "updated skills/make-plan" "symlinked destination must report updated, not unchanged"

  after_agent="$(cksum "$project/link-target-agent.md")"
  after_skill="$(cksum "$project/link-target-skill/SKILL.md")"
  assert_eq "$before_agent" "$after_agent" "the file the old symlink pointed to must be untouched"
  assert_eq "$before_skill" "$after_skill" "the dir the old symlink pointed to must be untouched"
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

run_test test_01_fresh_install_copies_real_files
run_test test_02_symlinked_source_skill_is_flattened
run_test test_03_idempotent_rerun
run_test test_04_edited_source_resyncs
run_test test_05_stale_file_removed_no_nested_duplicate
run_test test_06_own_items_overwritten_unconditionally
run_test test_07_foreign_entries_survive_byte_identical
run_test test_08_dry_run_changes_nothing
run_test test_09_uninstall_removes_ours_keeps_foreign
run_test test_10_uninstall_on_clean_project_reports_absent
run_test test_11_default_target_is_cwd
run_test test_12_target_home_installs_into_home
run_test test_13_target_and_target_home_are_mutually_exclusive
run_test test_14_self_install_guard
run_test test_15_usage_errors
run_test test_16_syntax_gate
run_test test_17_symlink_at_destination_is_replaced

echo
echo "-----------------------------------------------"
echo "$TESTS_RUN run, $((TESTS_RUN - TESTS_FAILED)) passed, $TESTS_FAILED failed"

if [ "$TESTS_FAILED" -gt 0 ]; then
  exit 1
fi
exit 0
