#!/usr/bin/env bash
set -euo pipefail

# Restore a gzipped pg_dump (plain SQL) into the PostgreSQL database running
# inside the Docker container. Counterpart of backup-telegram.sh.
#
# Usage:
#   scripts/restore-backup.sh [backup-file.sql.gz] [-y|--yes]
#
# Without a file argument the newest backup/${POSTGRES_DB}_*.sql.gz is used.
#
# WARNING: this DROPs the `public` schema and recreates it from the dump,
# so ALL current data in the database is replaced. The whole load runs in a
# single transaction (ON_ERROR_STOP) — on any error nothing is changed.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-gravel_postgres}"
POSTGRES_DB="${POSTGRES_DB:-gravel_bot}"
POSTGRES_USER="${POSTGRES_USER:-gravel}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-}"
BACKUP_DIR="${BACKUP_DIR:-$ROOT_DIR/backup}"

log() { echo "[$(date '+%F %T')] $*"; }
die() { echo "[$(date '+%F %T')] ERROR: $*" >&2; exit 1; }

# --- parse args -------------------------------------------------------------
ASSUME_YES=0
backup_file=""
for arg in "$@"; do
  case "$arg" in
    -y|--yes) ASSUME_YES=1 ;;
    -h|--help)
      grep -E '^#( |$)' "$0" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    -*) die "unknown option: $arg" ;;
    *)  backup_file="$arg" ;;
  esac
done

# Default to the newest local dump for this database.
if [[ -z "$backup_file" ]]; then
  backup_file="$(ls -1t "$BACKUP_DIR/${POSTGRES_DB}"_*.sql.gz 2>/dev/null | head -n1 || true)"
  [[ -n "$backup_file" ]] || die "no backup file given and none found in $BACKUP_DIR"
fi

# --- preflight --------------------------------------------------------------
command -v docker >/dev/null 2>&1 || die "docker is not installed"
[[ -f "$backup_file" ]] || die "backup file not found: $backup_file"

if ! docker ps --format '{{.Names}}' | grep -qx "$POSTGRES_CONTAINER"; then
  die "postgres container '$POSTGRES_CONTAINER' is not running"
fi

# Make sure the archive is a readable gzip before we touch the database.
gzip -t "$backup_file" 2>/dev/null || die "not a valid gzip file: $backup_file"

size_human="$(ls -lh "$backup_file" | awk '{print $5}')"
log "Backup file : $backup_file ($size_human)"
log "Container   : $POSTGRES_CONTAINER"
log "Database    : $POSTGRES_DB (user: $POSTGRES_USER)"

if [[ "$ASSUME_YES" -ne 1 ]]; then
  echo
  echo "!! This will WIPE the current '$POSTGRES_DB' database and replace it"
  echo "!! with the dump above. This cannot be undone."
  read -r -p "Type 'yes' to continue: " reply
  [[ "$reply" == "yes" ]] || die "aborted by user"
fi

dexec() {
  docker exec -i -e PGPASSWORD="$POSTGRES_PASSWORD" "$POSTGRES_CONTAINER" "$@"
}

# Drop other client connections so DROP SCHEMA isn't blocked by locks
# (e.g. backend-api / backend-bot holding open sessions). Best-effort.
log "Terminating other connections to '$POSTGRES_DB'..."
dexec psql -v ON_ERROR_STOP=0 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c \
  "SELECT pg_terminate_backend(pid) FROM pg_stat_activity
   WHERE datname = current_database() AND pid <> pg_backend_pid();" \
  >/dev/null 2>&1 || log "  (could not terminate connections, continuing)"

# Reset the schema and load the dump in ONE transaction:
# any error rolls everything back, leaving the DB untouched.
log "Restoring (single transaction, rollback on error)..."
{
  printf 'DROP SCHEMA IF EXISTS public CASCADE;\nCREATE SCHEMA public;\n'
  gzip -dc "$backup_file"
} | dexec psql -v ON_ERROR_STOP=1 --single-transaction \
      -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  || die "restore failed — database left unchanged (transaction rolled back)"

log "Restore complete."
log "Tip: restart the app so it picks up the restored data:"
log "  docker restart gravel_bot gravel_api 2>/dev/null || true"
