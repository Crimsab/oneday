#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
verify_script="$repo_root/scripts/verify-sqlite-backup-restore.sh"
test_dir="$(mktemp -d)"
trap 'rm -rf "$test_dir"' EXIT

if command -v sha256sum >/dev/null 2>&1; then
  checksum() { sha256sum "$1" | awk '{print $1}'; }
elif command -v shasum >/dev/null 2>&1; then
  checksum() { shasum -a 256 "$1" | awk '{print $1}'; }
else
  echo "sha256sum or shasum is required" >&2
  exit 127
fi

source_db="$test_dir/source.sqlite"
backup_db="$test_dir/backup.sqlite"
restore_dir="$test_dir/recovery"
fake_bin="$test_dir/fake-bin"
real_ln="$(command -v ln)"
mkdir "$fake_bin"
# shellcheck disable=SC2016 # The shim must receive these as literal shell code.
printf '%s\n' \
  '#!/usr/bin/env bash' \
  'destination="${!#}"' \
  'if [[ -n "${ONEDAY_TEST_LN_COLLISION_DEST:-}" && "$destination" == "$ONEDAY_TEST_LN_COLLISION_DEST" ]]; then' \
  '  printf "concurrent-owner" > "$destination"' \
  'fi' \
  'exec "$ONEDAY_TEST_REAL_LN" "$@"' > "$fake_bin/ln"
chmod +x "$fake_bin/ln"
mkdir "$restore_dir"
sqlite3 "$source_db" 'PRAGMA foreign_keys = ON; CREATE TABLE checks (id INTEGER PRIMARY KEY, value TEXT NOT NULL); INSERT INTO checks(value) VALUES ("canonical");'
source_checksum="$(checksum "$source_db")"

bash "$verify_script" --db "$source_db" --backup "$backup_db" --restore-dir "$restore_dir" >/dev/null
test "$(sqlite3 "$restore_dir/oneday.db" 'SELECT value FROM checks;')" = "canonical"
test "$(checksum "$source_db")" = "$source_checksum"

assert_concurrent_destination_wins() {
  local destination="$1"
  shift
  if PATH="$fake_bin:$PATH" ONEDAY_TEST_LN_COLLISION_DEST="$destination" ONEDAY_TEST_REAL_LN="$real_ln" bash "$verify_script" "$@" >/dev/null 2>&1; then
    echo "concurrent destination was accepted" >&2
    exit 1
  fi
  test "$(cat "$destination")" = "concurrent-owner"
}

preexisting_backup="$test_dir/preexisting.sqlite"
printf 'existing-owner' > "$preexisting_backup"
if bash "$verify_script" --db "$source_db" --backup "$preexisting_backup" >/dev/null 2>&1; then
  echo "pre-existing backup destination was accepted" >&2
  exit 1
fi
test "$(cat "$preexisting_backup")" = "existing-owner"

race_backup="$test_dir/race-backup.sqlite"
assert_concurrent_destination_wins "$race_backup" --db "$source_db" --backup "$race_backup"

race_checksum="$test_dir/race-checksum.sqlite"
assert_concurrent_destination_wins "${race_checksum}.sha256" --db "$source_db" --backup "$race_checksum"
test -f "$race_checksum"
test "$(sqlite3 -readonly "$race_checksum" 'PRAGMA integrity_check;')" = "ok"
test -z "$(sqlite3 -readonly "$race_checksum" 'PRAGMA foreign_key_check;')"
test "$(sqlite3 -readonly "$race_checksum" 'SELECT value FROM checks;')" = "canonical"

race_restore="$test_dir/race-restore"
mkdir "$race_restore"
assert_concurrent_destination_wins "$race_restore/oneday.db" --backup "$backup_db" --restore-dir "$race_restore"

# A failed migration in the isolated recovery target cannot modify the source.
if sqlite3 "$restore_dir/oneday.db" 'ALTER TABLE missing_table ADD COLUMN ignored TEXT;' 2>/dev/null; then
  echo "migration failure fixture unexpectedly succeeded" >&2
  exit 1
fi
test "$(checksum "$source_db")" = "$source_checksum"

protected_dir="$test_dir/protected"
mkdir "$protected_dir"
printf 'original' > "$protected_dir/marker"
if bash "$verify_script" --backup "$backup_db" --restore-dir "$protected_dir" >/dev/null 2>&1; then
  echo "non-empty restore target was accepted" >&2
  exit 1
fi
test "$(cat "$protected_dir/marker")" = "original"

printf 'corrupt' >> "$backup_db"
empty_dir="$test_dir/empty"
mkdir "$empty_dir"
if bash "$verify_script" --backup "$backup_db" --restore-dir "$empty_dir" >/dev/null 2>&1; then
  echo "corrupted backup was accepted" >&2
  exit 1
fi
test -z "$(find "$empty_dir" -mindepth 1 -print -quit)"

printf 'SQLite backup/restore recovery tests passed\n'
