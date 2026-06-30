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
# WARNING: this drops every table/sequence/view in the `public` schema and
# reloads them from the dump, so ALL current data is replaced. Extensions
# (e.g. pgvector) and the schema itself are kept. The whole load runs in a
# single transaction (ON_ERROR_STOP) — on any error nothing is changed.
#
# The dump bakes in its original owner role (e.g. OWNER TO gravel). If that
# role differs from $POSTGRES_USER it is remapped on the fly so objects end up
# owned by the connecting user. Override detection with DUMP_OWNER=<role>.

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

# Owner role baked into the dump (the role objects are ALTERed to own).
# Used to remap ownership onto $POSTGRES_USER when they differ.
DUMP_OWNER="${DUMP_OWNER:-$(gzip -dc "$backup_file" | grep -m1 -oE 'OWNER TO [A-Za-z0-9_]+' | awk '{print $3}')}"

size_human="$(ls -lh "$backup_file" | awk '{print $5}')"
log "Backup file : $backup_file ($size_human)"
log "Container   : $POSTGRES_CONTAINER"
log "Database    : $POSTGRES_DB (user: $POSTGRES_USER)"
if [[ -n "$DUMP_OWNER" && "$DUMP_OWNER" != "$POSTGRES_USER" ]]; then
  log "Owner remap : $DUMP_OWNER -> $POSTGRES_USER"
fi

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

# Rewrite the dump's ownership to the connecting user when the baked-in owner
# role doesn't exist here. Only ownership/privilege lines are touched — never
# COPY data — so this is safe to run over the whole stream.
remap_owner() {
  if [[ -n "$DUMP_OWNER" && "$DUMP_OWNER" != "$POSTGRES_USER" ]]; then
    sed -E \
      -e "s/ OWNER TO ${DUMP_OWNER};$/ OWNER TO ${POSTGRES_USER};/" \
      -e "s/^(GRANT .* TO )${DUMP_OWNER}( WITH GRANT OPTION)?;$/\1${POSTGRES_USER}\2;/" \
      -e "s/^(REVOKE .* FROM )${DUMP_OWNER};$/\1${POSTGRES_USER};/"
  else
    cat
  fi
}

# Drop everything the dump recreates, but keep extensions (pgvector) and the
# schema intact. DROP TABLE ... CASCADE also removes owned sequences and FKs.
reset_sql() {
  cat <<'SQL'
DO $$
DECLARE r record;
BEGIN
  FOR r IN SELECT viewname     FROM pg_views     WHERE schemaname = 'public' LOOP
    EXECUTE format('DROP VIEW IF EXISTS public.%I CASCADE', r.viewname);
  END LOOP;
  FOR r IN SELECT tablename    FROM pg_tables    WHERE schemaname = 'public' LOOP
    EXECUTE format('DROP TABLE IF EXISTS public.%I CASCADE', r.tablename);
  END LOOP;
  FOR r IN SELECT sequencename FROM pg_sequences WHERE schemaname = 'public' LOOP
    EXECUTE format('DROP SEQUENCE IF EXISTS public.%I CASCADE', r.sequencename);
  END LOOP;
END $$;
SQL
}

# Drop other client connections so the table drops aren't blocked by locks
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
  reset_sql
  gzip -dc "$backup_file" | remap_owner
} | dexec psql -v ON_ERROR_STOP=1 --single-transaction \
      -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  || die "restore failed — database left unchanged (transaction rolled back)"

log "Restore complete."
log "Tip: restart the app so it picks up the restored data:"
log "  docker restart gravel_bot gravel_api 2>/dev/null || true"
