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
mkdir "$restore_dir"
sqlite3 "$source_db" 'PRAGMA foreign_keys = ON; CREATE TABLE checks (id INTEGER PRIMARY KEY, value TEXT NOT NULL); INSERT INTO checks(value) VALUES ("canonical");'
source_checksum="$(checksum "$source_db")"

bash "$verify_script" --db "$source_db" --backup "$backup_db" --restore-dir "$restore_dir" >/dev/null
test "$(sqlite3 "$restore_dir/oneday.db" 'SELECT value FROM checks;')" = "canonical"
test "$(checksum "$source_db")" = "$source_checksum"

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
