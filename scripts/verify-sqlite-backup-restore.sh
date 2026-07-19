#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: verify-sqlite-backup-restore.sh --db PATH [--work-dir PATH]

Creates a SQLite online backup and a restore copy under a temporary directory,
then verifies checksums, integrity_check, and foreign_key_check. The source
database is opened read-only by sqlite3 and is never modified.
EOF
}

db_path=""
work_dir=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --db)
      db_path="${2:-}"
      shift 2
      ;;
    --work-dir)
      work_dir="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "$db_path" || ! -f "$db_path" ]]; then
  echo "--db must name an existing SQLite database" >&2
  exit 2
fi
if ! command -v sqlite3 >/dev/null 2>&1; then
  echo "sqlite3 is required" >&2
  exit 127
fi
if command -v sha256sum >/dev/null 2>&1; then
  checksum() { sha256sum "$1" | awk '{print $1}'; }
elif command -v shasum >/dev/null 2>&1; then
  checksum() { shasum -a 256 "$1" | awk '{print $1}'; }
else
  echo "sha256sum or shasum is required" >&2
  exit 127
fi

if [[ -n "$work_dir" ]]; then
  mkdir -p "$work_dir"
  temp_dir="$(mktemp -d "$work_dir/oneday-backup.XXXXXX")"
else
  temp_dir="$(mktemp -d "${TMPDIR:-/tmp}/oneday-backup.XXXXXX")"
fi
trap 'rm -rf "$temp_dir"' EXIT

backup_path="$temp_dir/backup.sqlite"
restore_path="$temp_dir/restore.sqlite"

sql_quote() {
  sed "s/'/''/g" <<<"$1"
}

sqlite_backup() {
  local source="$1"
  local destination="$2"
  local quoted_destination
  quoted_destination="$(sql_quote "$destination")"
  sqlite3 -readonly -cmd '.timeout 5000' "$source" ".backup '$quoted_destination'"
}

verify_database() {
  local path="$1"
  local integrity
  integrity="$(sqlite3 -readonly -cmd '.timeout 5000' "$path" 'PRAGMA integrity_check;')"
  if [[ "$integrity" != "ok" ]]; then
    echo "integrity_check failed for $path: $integrity" >&2
    exit 1
  fi
  local foreign_keys
  foreign_keys="$(sqlite3 -readonly -cmd '.timeout 5000' "$path" 'PRAGMA foreign_key_check;')"
  if [[ -n "$foreign_keys" ]]; then
    echo "foreign_key_check failed for $path:" >&2
    printf '%s\n' "$foreign_keys" >&2
    exit 1
  fi
}

sqlite_backup "$db_path" "$backup_path"
verify_database "$backup_path"
sqlite_backup "$backup_path" "$restore_path"
verify_database "$restore_path"

backup_sum="$(checksum "$backup_path")"
restore_sum="$(checksum "$restore_path")"
if [[ "$backup_sum" != "$restore_sum" ]]; then
  echo "backup and restored checksums differ" >&2
  exit 1
fi

printf 'backup/restore verified: sha256=%s\n' "$backup_sum"
