#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

PUBLIC_DOMAIN="${PUBLIC_DOMAIN:-}"
CERTBOT_EMAIL="${CERTBOT_EMAIL:-}"
SSL_STAGING="${SSL_STAGING:-false}"
SSL_RSA_KEY_SIZE="${SSL_RSA_KEY_SIZE:-4096}"
SSL_RENEW_BEFORE_SECONDS="${SSL_RENEW_BEFORE_SECONDS:-2592000}"
CERTBOT_IMAGE="${CERTBOT_IMAGE:-certbot/certbot:v2.11.0}"

if [[ -z "$PUBLIC_DOMAIN" ]]; then
  echo "PUBLIC_DOMAIN is required" >&2
  exit 1
fi

if [[ "${1:-issue}" != "renew" && -z "$CERTBOT_EMAIL" ]]; then
  echo "CERTBOT_EMAIL is required for initial certificate issue" >&2
  exit 1
fi

compose=(docker compose -f docker-compose.yml -f docker-compose.prod.yml)
letsencrypt_dir="$ROOT_DIR/nginx/certbot/conf"
live_dir="$letsencrypt_dir/live/$PUBLIC_DOMAIN"
renewal_conf="$letsencrypt_dir/renewal/$PUBLIC_DOMAIN.conf"
cert_path="$live_dir/fullchain.pem"

mkdir -p "$letsencrypt_dir"

staging_args=()
if [[ "$SSL_STAGING" == "true" ]]; then
  staging_args=(--staging)
fi

if [[ "$SSL_STAGING" != "true" && -f "$renewal_conf" ]] && grep -qi "staging" "$renewal_conf"; then
  echo "Removing staging certificate lineage for $PUBLIC_DOMAIN before production issue."
  rm -rf \
    "$letsencrypt_dir/live/$PUBLIC_DOMAIN" \
    "$letsencrypt_dir/archive/$PUBLIC_DOMAIN" \
    "$renewal_conf"
fi

if [[ "${1:-issue}" == "renew" ]]; then
  "${compose[@]}" stop nginx || true
  docker run --rm \
    -p 80:80 \
    -v "$letsencrypt_dir:/etc/letsencrypt" \
    "$CERTBOT_IMAGE" renew \
    --standalone \
    --preferred-challenges http \
    "${staging_args[@]}"
  "${compose[@]}" up -d nginx
  exit 0
fi

if [[ -f "$cert_path" ]] && openssl x509 -checkend "$SSL_RENEW_BEFORE_SECONDS" -noout -in "$cert_path" >/dev/null; then
  echo "Existing certificate for $PUBLIC_DOMAIN is still valid; skipping issue."
  "${compose[@]}" up -d nginx
  exit 0
fi

if [[ -f "$renewal_conf" ]]; then
  "${compose[@]}" stop nginx || true
  docker run --rm \
    -p 80:80 \
    -v "$letsencrypt_dir:/etc/letsencrypt" \
    "$CERTBOT_IMAGE" renew \
    --cert-name "$PUBLIC_DOMAIN" \
    --standalone \
    --preferred-challenges http \
    "${staging_args[@]}"
  "${compose[@]}" up -d nginx
  exit 0
fi

"${compose[@]}" stop nginx || true

docker run --rm \
  -p 80:80 \
  -v "$letsencrypt_dir:/etc/letsencrypt" \
  "$CERTBOT_IMAGE" certonly \
  --standalone \
  --preferred-challenges http \
  --email "$CERTBOT_EMAIL" \
  --agree-tos \
  --no-eff-email \
  --keep-until-expiring \
  --rsa-key-size "$SSL_RSA_KEY_SIZE" \
  --cert-name "$PUBLIC_DOMAIN" \
  "${staging_args[@]}" \
  -d "$PUBLIC_DOMAIN"

"${compose[@]}" up -d nginx
