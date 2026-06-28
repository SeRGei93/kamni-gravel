#!/usr/bin/env bash
set -euo pipefail

# Dump the PostgreSQL database from the running Docker container, gzip it,
# and upload the archive to a Telegram chat via the Bot API (sendDocument).
# Meant to run hourly from cron. See DEPLOY.md ("Бэкап базы в Telegram").

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

BOT_TOKEN="${BOT_TOKEN:-}"
BACKUP_TELEGRAM_ID="${BACKUP_TELEGRAM_ID:-}"
POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-gravel_postgres}"
POSTGRES_DB="${POSTGRES_DB:-gravel_bot}"
POSTGRES_USER="${POSTGRES_USER:-gravel}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-}"
BACKUP_DIR="${BACKUP_DIR:-$ROOT_DIR/backup}"
# How many local dumps to keep (hourly => 48 == last 2 days). 0 disables pruning.
BACKUP_KEEP="${BACKUP_KEEP:-48}"

log() { echo "[$(date '+%F %T')] $*"; }
die() { echo "[$(date '+%F %T')] ERROR: $*" >&2; exit 1; }

[[ -n "$BOT_TOKEN" ]] || die "BOT_TOKEN is required (set it in .env)"
[[ -n "$BACKUP_TELEGRAM_ID" ]] || die "BACKUP_TELEGRAM_ID is required (set it in .env)"
command -v docker >/dev/null 2>&1 || die "docker is not installed"
command -v curl >/dev/null 2>&1 || die "curl is not installed"

if ! docker ps --format '{{.Names}}' | grep -qx "$POSTGRES_CONTAINER"; then
  die "postgres container '$POSTGRES_CONTAINER' is not running"
fi

mkdir -p "$BACKUP_DIR"

timestamp="$(date +%Y%m%d_%H%M%S)"
backup_name="${POSTGRES_DB}_${timestamp}.sql.gz"
backup_file="$BACKUP_DIR/$backup_name"

log "Dumping '$POSTGRES_DB' from container '$POSTGRES_CONTAINER'..."
if ! docker exec -e PGPASSWORD="$POSTGRES_PASSWORD" "$POSTGRES_CONTAINER" \
      pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" | gzip -9 > "$backup_file"; then
  rm -f "$backup_file"
  die "pg_dump failed"
fi

size_bytes="$(wc -c < "$backup_file" | tr -d ' ')"
size_human="$(awk -v b="$size_bytes" 'BEGIN{split("B KB MB GB TB",u); s=b; i=1; while(s>=1024 && i<5){s/=1024;i++} printf "%.1f %s", s, u[i]}')"
log "Created $backup_name ($size_human)"

if [[ "$size_bytes" -gt $((50 * 1024 * 1024)) ]]; then
  log "WARNING: backup is $size_human, above Telegram's 50 MB bot upload limit; the upload will likely fail."
fi

caption="$(printf '🗄 DB backup %s\nhost: %s\ntime: %s\nsize: %s' \
  "$POSTGRES_DB" "$(hostname)" "$(date '+%F %T %Z')" "$size_human")"

log "Uploading to Telegram chat $BACKUP_TELEGRAM_ID..."
response="$(curl -sS --max-time 300 \
  -F "chat_id=${BACKUP_TELEGRAM_ID}" \
  -F "document=@${backup_file};filename=${backup_name}" \
  -F "caption=${caption}" \
  "https://api.telegram.org/bot${BOT_TOKEN}/sendDocument")" || die "curl request failed"

case "$response" in
  *'"ok":true'*) log "Uploaded $backup_name to Telegram." ;;
  *) die "Telegram rejected the upload: $response" ;;
esac

# Prune old local dumps, keeping the newest $BACKUP_KEEP.
if [[ "$BACKUP_KEEP" =~ ^[0-9]+$ && "$BACKUP_KEEP" -gt 0 ]]; then
  i=0
  for f in $(ls -1t "$BACKUP_DIR/${POSTGRES_DB}"_*.sql.gz 2>/dev/null); do
    i=$((i + 1))
    if [[ "$i" -gt "$BACKUP_KEEP" ]]; then
      rm -f "$f"
      log "Pruned old backup $(basename "$f")"
    fi
  done
fi
