#!/usr/bin/env bash
#
# install-cursor-config.test.sh
#
# Dependency-free bash unit-test suite for install-cursor-config.sh. No bats,
# no shellcheck -- plain bash, written to also run under bash 3.2 (macOS
# /bin/bash): no associative arrays, no mapfile/readarray, no ${var,,}, no
# declare -A.
#
# Deliberately does NOT `set -e`: several tests exercise commands that are
# expected to exit non-zero, and letting those kill the harness would be
# exactly the trap the spec warns about. Every command whose status matters
# is checked explicitly instead.

TEST_DIR="$(cd "$(dirname "$0")" && pwd)"
REAL_SCRIPT="$TEST_DIR/../install-cursor-config.sh"

BASE_TMP="$(mktemp -d "${TMPDIR:-/tmp}/install-cursor-config-test.XXXXXX")"
trap 'rm -rf "$BASE_TMP"' EXIT

TESTS_RUN=0
TESTS_FAILED=0
CURRENT_TEST_FAILED=0

LABELS="agents/scout.md agents/thinker.md agents/worker.md skills/make-plan skills/implement-plan"

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
#   $FIXTURE/                     <- acts as REPO_ROOT
#   |- claude/skills/make-plan/SKILL.md
#   |- claude/skills/implement-plan/SKILL.md
#   `- cursor/                    <- acts as SRC_ROOT
#      |- install-cursor-config.sh   (copy of the real script)
#      |- agents/{scout,thinker,worker}.md
#      `- skills/{make-plan,implement-plan} -> relative symlinks into claude/skills
#
# The target project directory is a SEPARATE mktemp tree so it never resolves
# inside the fixture repo root.
# ---------------------------------------------------------------------------

new_fixture_dir() {
  mktemp -d "$BASE_TMP/fixture.XXXXXX"
}

new_project_dir() {
  mktemp -d "$BASE_TMP/project.XXXXXX"
}

make_fixture() {
  local fixture="$1"

  mkdir -p "$fixture/claude/skills/make-plan"
  mkdir -p "$fixture/claude/skills/implement-plan"
  printf '# Make Plan Skill (fixture)\n' > "$fixture/claude/skills/make-plan/SKILL.md"
  printf '# Implement Plan Skill (fixture)\n' > "$fixture/claude/skills/implement-plan/SKILL.md"

  mkdir -p "$fixture/cursor/agents"
  mkdir -p "$fixture/cursor/skills"

  cp "$REAL_SCRIPT" "$fixture/cursor/install-cursor-config.sh"
  chmod +x "$fixture/cursor/install-cursor-config.sh"

  printf '# Scout Agent (fixture)\n' > "$fixture/cursor/agents/scout.md"
  printf '# Thinker Agent (fixture)\n' > "$fixture/cursor/agents/thinker.md"
  printf '# Worker Agent (fixture)\n' > "$fixture/cursor/agents/worker.md"

  ( cd "$fixture/cursor/skills" && ln -s ../../claude/skills/make-plan make-plan )
  ( cd "$fixture/cursor/skills" && ln -s ../../claude/skills/implement-plan implement-plan )
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

plant_foreign_entries() {
  local project="$1"
  mkdir -p "$project/.cursor/agents"
  mkdir -p "$project/.cursor/skills/mine"
  printf 'my own custom agent, do not touch\n' > "$project/.cursor/agents/custom.md"
  printf 'my own custom skill, do not touch\n' > "$project/.cursor/skills/mine/SKILL.md"
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

  run_installer "$fixture/cursor/install-cursor-config.sh" --target "$project"
  assert_eq "0" "$INSTALLER_STATUS" "exit status"

  for label in $LABELS; do
    assert_contains "$INSTALLER_OUT" "install $label" "missing install line for $label"
  done
  assert_contains "$INSTALLER_OUT" "5 install, 0 updated, 0 unchanged, 0 removed, 0 absent" "summary line"

  assert_file "$project/.cursor/agents/scout.md"
  assert_files_equal "$fixture/cursor/agents/scout.md" "$project/.cursor/agents/scout.md" "scout.md content"
}

test_02_symlinked_source_skill_is_flattened() {
  local fixture project
  fixture="$(new_fixture_dir)"
  project="$(new_project_dir)"
  make_fixture "$fixture"

  run_installer "$fixture/cursor/install-cursor-config.sh" --target "$project"
  assert_eq "0" "$INSTALLER_STATUS" "exit status"

  assert_dir "$project/.cursor/skills/make-plan" "flattened skill dir"
  assert_file "$project/.cursor/skills/make-plan/SKILL.md"
  assert_files_equal "$fixture/claude/skills/make-plan/SKILL.md" "$project/.cursor/skills/make-plan/SKILL.md" "flattened SKILL.md content"
}

test_03_idempotent_rerun() {
  local fixture project label
  fixture="$(new_fixture_dir)"
  project="$(new_project_dir)"
  make_fixture "$fixture"

  run_installer "$fixture/cursor/install-cursor-config.sh" --target "$project"
  assert_eq "0" "$INSTALLER_STATUS" "first run exit status"

  run_installer "$fixture/cursor/install-cursor-config.sh" --target "$project"
  assert_eq "0" "$INSTALLER_STATUS" "second run exit status"

  for label in $LABELS; do
    assert_contains "$INSTALLER_OUT" "unchanged $label" "missing unchanged line for $label"
  done
  assert_contains "$INSTALLER_OUT" "0 install, 0 updated, 5 unchanged, 0 removed, 0 absent" "summary line"
}

test_04_edited_source_resyncs() {
  local fixture project
  fixture="$(new_fixture_dir)"
  project="$(new_project_dir)"
  make_fixture "$fixture"

  run_installer "$fixture/cursor/install-cursor-config.sh" --target "$project"
  assert_eq "0" "$INSTALLER_STATUS" "first run exit status"

  printf '# Scout Agent (EDITED)\n' > "$fixture/cursor/agents/scout.md"

  run_installer "$fixture/cursor/install-cursor-config.sh" --target "$project"
  assert_eq "0" "$INSTALLER_STATUS" "second run exit status"

  assert_contains "$INSTALLER_OUT" "updated agents/scout.md" "expected updated status for edited file"
  assert_contains "$INSTALLER_OUT" "unchanged agents/thinker.md" "unrelated file should be unchanged"
  assert_files_equal "$fixture/cursor/agents/scout.md" "$project/.cursor/agents/scout.md" "resynced content"
}

test_05_stale_file_removed_no_nested_duplicate() {
  local fixture project
  fixture="$(new_fixture_dir)"
  project="$(new_project_dir)"
  make_fixture "$fixture"

  run_installer "$fixture/cursor/install-cursor-config.sh" --target "$project"
  assert_eq "0" "$INSTALLER_STATUS" "first run exit status"

  printf 'stale leftover\n' > "$project/.cursor/skills/make-plan/STALE.md"

  run_installer "$fixture/cursor/install-cursor-config.sh" --target "$project"
  assert_eq "0" "$INSTALLER_STATUS" "second run exit status"

  assert_missing "$project/.cursor/skills/make-plan/STALE.md" "stale file must be gone"
  assert_missing "$project/.cursor/skills/make-plan/make-plan" "must not nest a duplicate copy (BSD cp -R hazard)"
  assert_file "$project/.cursor/skills/make-plan/SKILL.md"
}

test_06_own_items_overwritten_unconditionally() {
  local fixture project
  fixture="$(new_fixture_dir)"
  project="$(new_project_dir)"
  make_fixture "$fixture"

  mkdir -p "$project/.cursor/agents"
  printf 'hand-written, not the fixture content\n' > "$project/.cursor/agents/scout.md"

  run_installer "$fixture/cursor/install-cursor-config.sh" --target "$project"
  assert_eq "0" "$INSTALLER_STATUS" "exit status"

  assert_contains "$INSTALLER_OUT" "updated agents/scout.md" "hand-written file should register as updated"
  assert_files_equal "$fixture/cursor/agents/scout.md" "$project/.cursor/agents/scout.md" "should be replaced by fixture content"
}

test_07_foreign_entries_survive_byte_identical() {
  local fixture project before_custom before_mine after_custom after_mine
  fixture="$(new_fixture_dir)"
  project="$(new_project_dir)"
  make_fixture "$fixture"
  plant_foreign_entries "$project"

  before_custom="$(cksum "$project/.cursor/agents/custom.md")"
  before_mine="$(cksum "$project/.cursor/skills/mine/SKILL.md")"

  run_installer "$fixture/cursor/install-cursor-config.sh" --target "$project"
  assert_eq "0" "$INSTALLER_STATUS" "exit status"

  assert_file "$project/.cursor/agents/custom.md"
  assert_dir "$project/.cursor/skills/mine"
  assert_file "$project/.cursor/skills/mine/SKILL.md"

  after_custom="$(cksum "$project/.cursor/agents/custom.md")"
  after_mine="$(cksum "$project/.cursor/skills/mine/SKILL.md")"

  assert_eq "$before_custom" "$after_custom" "foreign agent file must be byte-identical"
  assert_eq "$before_mine" "$after_mine" "foreign skill file must be byte-identical"
}

test_08_dry_run_changes_nothing() {
  local fixture project label
  fixture="$(new_fixture_dir)"
  project="$(new_project_dir)"
  make_fixture "$fixture"

  run_installer "$fixture/cursor/install-cursor-config.sh" --target "$project" --dry-run
  assert_eq "0" "$INSTALLER_STATUS" "exit status"

  for label in $LABELS; do
    assert_contains "$INSTALLER_OUT" "[dry-run] install $label" "missing dry-run install line for $label"
  done
  assert_contains "$INSTALLER_OUT" "[dry-run] 5 install, 0 updated, 0 unchanged, 0 removed, 0 absent" "dry-run summary line"

  assert_missing "$project/.cursor" ".cursor must not be created by --dry-run"
}

test_09_uninstall_removes_ours_keeps_foreign() {
  local fixture project label before_custom before_mine after_custom after_mine
  fixture="$(new_fixture_dir)"
  project="$(new_project_dir)"
  make_fixture "$fixture"

  run_installer "$fixture/cursor/install-cursor-config.sh" --target "$project"
  assert_eq "0" "$INSTALLER_STATUS" "install run exit status"

  plant_foreign_entries "$project"
  before_custom="$(cksum "$project/.cursor/agents/custom.md")"
  before_mine="$(cksum "$project/.cursor/skills/mine/SKILL.md")"

  run_installer "$fixture/cursor/install-cursor-config.sh" --target "$project" --uninstall
  assert_eq "0" "$INSTALLER_STATUS" "uninstall run exit status"

  for label in $LABELS; do
    assert_contains "$INSTALLER_OUT" "removed $label" "missing removed line for $label"
  done
  assert_contains "$INSTALLER_OUT" "0 install, 0 updated, 0 unchanged, 5 removed, 0 absent" "uninstall summary line"

  assert_missing "$project/.cursor/agents/scout.md"
  assert_missing "$project/.cursor/agents/thinker.md"
  assert_missing "$project/.cursor/agents/worker.md"
  assert_missing "$project/.cursor/skills/make-plan"
  assert_missing "$project/.cursor/skills/implement-plan"

  assert_file "$project/.cursor/agents/custom.md" "foreign agent must survive uninstall"
  assert_file "$project/.cursor/skills/mine/SKILL.md" "foreign skill must survive uninstall"

  after_custom="$(cksum "$project/.cursor/agents/custom.md")"
  after_mine="$(cksum "$project/.cursor/skills/mine/SKILL.md")"
  assert_eq "$before_custom" "$after_custom" "foreign agent file must be byte-identical after uninstall"
  assert_eq "$before_mine" "$after_mine" "foreign skill file must be byte-identical after uninstall"
}

test_10_uninstall_on_clean_project_reports_absent() {
  local fixture project label
  fixture="$(new_fixture_dir)"
  project="$(new_project_dir)"
  make_fixture "$fixture"

  run_installer "$fixture/cursor/install-cursor-config.sh" --target "$project" --uninstall
  assert_eq "0" "$INSTALLER_STATUS" "exit status"

  for label in $LABELS; do
    assert_contains "$INSTALLER_OUT" "absent $label" "missing absent line for $label"
  done
  assert_contains "$INSTALLER_OUT" "0 install, 0 updated, 0 unchanged, 0 removed, 5 absent" "summary line"
}

test_11_symlink_mode_produces_symlinks() {
  local fixture project
  fixture="$(new_fixture_dir)"
  project="$(new_project_dir)"
  make_fixture "$fixture"

  run_installer "$fixture/cursor/install-cursor-config.sh" --target "$project" --symlink
  assert_eq "0" "$INSTALLER_STATUS" "exit status"

  assert_symlink "$project/.cursor/agents/scout.md"
  assert_files_equal "$fixture/cursor/agents/scout.md" "$project/.cursor/agents/scout.md" "symlink must resolve to fixture content"

  # Bonus: skills symlinks must resolve past the repo's own indirection into
  # claude/skills, not just point back at cursor/skills/make-plan.
  assert_symlink "$project/.cursor/skills/make-plan"
  assert_files_equal "$fixture/claude/skills/make-plan/SKILL.md" "$project/.cursor/skills/make-plan/SKILL.md" "flattened symlink must resolve to claude/skills content"
}

test_12_self_install_guard() {
  local fixture project
  fixture="$(new_fixture_dir)"
  project="$(new_project_dir)"
  make_fixture "$fixture"

  run_installer "$fixture/cursor/install-cursor-config.sh" --target "$fixture"
  assert_eq "1" "$INSTALLER_STATUS" "targeting the repo root itself must exit 1"

  run_installer "$fixture/cursor/install-cursor-config.sh" --target "$fixture/claude"
  assert_eq "1" "$INSTALLER_STATUS" "targeting a subdirectory of the repo must exit 1"
}

test_13_usage_errors() {
  local fixture project
  fixture="$(new_fixture_dir)"
  project="$(new_project_dir)"
  make_fixture "$fixture"

  run_installer "$fixture/cursor/install-cursor-config.sh" --target "$project" --bogus-flag
  assert_eq "1" "$INSTALLER_STATUS" "unknown flag must exit 1"

  run_installer "$fixture/cursor/install-cursor-config.sh" --target
  assert_eq "1" "$INSTALLER_STATUS" "--target with no argument must exit 1"

  run_installer "$fixture/cursor/install-cursor-config.sh" --help
  assert_eq "0" "$INSTALLER_STATUS" "--help must exit 0"
  assert_contains "$INSTALLER_OUT" "--symlink" "help output must mention --symlink"

  run_installer "$fixture/cursor/install-cursor-config.sh" -h
  assert_eq "0" "$INSTALLER_STATUS" "-h must exit 0"
  assert_contains "$INSTALLER_OUT" "--symlink" "help output must mention --symlink"
}

test_14_syntax_gate() {
  if ! bash -n "$REAL_SCRIPT" 2>"$BASE_TMP/syntax-err"; then
    fail "bash -n failed on $REAL_SCRIPT: $(cat "$BASE_TMP/syntax-err")"
  fi
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
run_test test_11_symlink_mode_produces_symlinks
run_test test_12_self_install_guard
run_test test_13_usage_errors
run_test test_14_syntax_gate

echo
echo "-----------------------------------------------"
echo "$TESTS_RUN run, $((TESTS_RUN - TESTS_FAILED)) passed, $TESTS_FAILED failed"

if [ "$TESTS_FAILED" -gt 0 ]; then
  exit 1
fi
exit 0
