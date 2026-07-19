#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  verify-sqlite-backup-restore.sh --db PATH --backup PATH [--restore-dir EMPTY_DIR] [--work-dir PATH]
  verify-sqlite-backup-restore.sh --backup PATH --restore-dir EMPTY_DIR [--work-dir PATH]

Creates or verifies a checksummed SQLite online backup, then optionally restores
it into an existing empty directory. The backup and restored database both pass
integrity_check and foreign_key_check before they are made visible. The source
database is opened read-only and is never modified. A non-empty restore target
is refused so recovery cannot overwrite an existing original.
EOF
}

db_path=""
backup_path=""
restore_dir=""
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
    --backup)
      backup_path="${2:-}"
      shift 2
      ;;
    --restore-dir)
      restore_dir="${2:-}"
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

if [[ -z "$backup_path" ]]; then
  echo "--backup is required" >&2
  exit 2
fi
if [[ -n "$db_path" && ! -f "$db_path" ]]; then
  echo "--db must name an existing SQLite database" >&2
  exit 2
fi
if [[ -z "$db_path" && -z "$restore_dir" ]]; then
  echo "--db is required when not restoring an existing backup" >&2
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
backup_stage_dir=""
restore_stage_dir=""
cleanup() {
  rm -rf -- "$temp_dir" "$backup_stage_dir" "$restore_stage_dir"
}
trap cleanup EXIT

sql_quote() {
  printf '%s\n' "${1//\'/\'\'}"
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
    echo "integrity_check failed" >&2
    exit 1
  fi
  local foreign_keys
  foreign_keys="$(sqlite3 -readonly -cmd '.timeout 5000' "$path" 'PRAGMA foreign_key_check;')"
  if [[ -n "$foreign_keys" ]]; then
    echo "foreign_key_check failed" >&2
    exit 1
  fi
}

write_checksum() {
  local path="$1"
  local published_name="$2"
  local sum
  sum="$(checksum "$path")"
  printf '%s  %s\n' "$sum" "$published_name"
}

verify_checksum() {
  local path="$1"
  local checksum_path="${path}.sha256"
  if [[ ! -f "$checksum_path" ]]; then
    echo "backup checksum is missing" >&2
    exit 1
  fi
  local expected actual
  expected="$(awk 'NF { print $1; exit }' "$checksum_path")"
  actual="$(checksum "$path")"
  if [[ ! "$expected" =~ ^[[:xdigit:]]{64}$ ]] || [[ "$expected" != "$actual" ]]; then
    echo "backup checksum does not match" >&2
    exit 1
  fi
}

# Staging happens beside the destination so ln(1) can atomically create a new
# hard link. Unlike a check followed by mv, ln fails if another process wins the
# name first and never replaces its file. The staged link is consumed only after
# confirming that it is the exact inode published at the destination.
publish_no_clobber() {
  local staged="$1"
  local destination="$2"
  local label="$3"
  if ! ln -- "$staged" "$destination"; then
    echo "$label destination already exists; nothing was overwritten" >&2
    return 1
  fi
  if [[ ! "$staged" -ef "$destination" ]]; then
    echo "$label publication could not be verified; staged data was retained" >&2
    return 1
  fi
}

consume_published_stage() {
  local staged="$1"
  local destination="$2"
  local label="$3"
  if [[ ! "$staged" -ef "$destination" ]]; then
    echo "$label publication identity changed; staged data was retained" >&2
    return 1
  fi
  if ! rm -f -- "$staged" || [[ -e "$staged" ]]; then
    echo "$label staged file was not consumed after publication; destination may contain the newly published file" >&2
    return 1
  fi
  if [[ ! -f "$destination" ]]; then
    echo "$label publication did not create a regular file" >&2
    return 1
  fi
}

if [[ -n "$db_path" ]]; then
  if [[ -e "$backup_path" || -e "${backup_path}.sha256" ]]; then
    echo "backup destination already exists" >&2
    exit 2
  fi
  backup_parent="$(dirname "$backup_path")"
  if [[ ! -d "$backup_parent" ]]; then
    echo "backup parent directory does not exist" >&2
    exit 2
  fi
  backup_stage_dir="$(mktemp -d "$backup_parent/.oneday-backup.XXXXXX")"
  staged_backup="$backup_stage_dir/backup.sqlite"
  staged_checksum="$backup_stage_dir/backup.sqlite.sha256"
  sqlite_backup "$db_path" "$staged_backup"
  verify_database "$staged_backup"
  write_checksum "$staged_backup" "$(basename "$backup_path")" >"$staged_checksum"
  chmod 600 "$staged_backup" "$staged_checksum"
  publish_no_clobber "$staged_backup" "$backup_path" "backup"
  if ! publish_no_clobber "$staged_checksum" "${backup_path}.sha256" "backup checksum"; then
    echo "backup was published but its checksum was not; do not restore from this incomplete backup and clean it up manually only after confirming ownership" >&2
    exit 1
  fi
  consume_published_stage "$staged_checksum" "${backup_path}.sha256" "backup checksum"
  consume_published_stage "$staged_backup" "$backup_path" "backup"
fi

verify_checksum "$backup_path"
verify_database "$backup_path"

backup_sum="$(checksum "$backup_path")"

if [[ -n "$restore_dir" ]]; then
  if [[ ! -d "$restore_dir" ]]; then
    echo "restore target must be an existing empty directory" >&2
    exit 2
  fi
  if [[ -n "$(find "$restore_dir" -mindepth 1 -print -quit)" ]]; then
    echo "restore target is not empty; original was not changed" >&2
    exit 2
  fi
  restore_stage_dir="$(mktemp -d "$restore_dir/.oneday-restore.XXXXXX")"
  staged_restore="$restore_stage_dir/oneday.db"
  sqlite_backup "$backup_path" "$staged_restore"
  verify_database "$staged_restore"
  restore_sum="$(checksum "$staged_restore")"
  if [[ "$backup_sum" != "$restore_sum" ]]; then
    echo "backup and restored checksums differ" >&2
    exit 1
  fi
  chmod 600 "$staged_restore"
  publish_no_clobber "$staged_restore" "$restore_dir/oneday.db" "restore"
  consume_published_stage "$staged_restore" "$restore_dir/oneday.db" "restore"
  printf 'backup/restore verified: sha256=%s\n' "$backup_sum"
  exit 0
fi

printf 'backup verified: sha256=%s\n' "$backup_sum"
