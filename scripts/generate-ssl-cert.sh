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

if [[ -z "$PUBLIC_DOMAIN" ]]; then
  echo "PUBLIC_DOMAIN is required" >&2
  exit 1
fi

if [[ "${1:-issue}" != "renew" && -z "$CERTBOT_EMAIL" ]]; then
  echo "CERTBOT_EMAIL is required for initial certificate issue" >&2
  exit 1
fi

compose=(docker compose -f docker-compose.yml -f docker-compose.prod.yml)
certbot_webroot="$ROOT_DIR/nginx/certbot/www"
letsencrypt_dir="$ROOT_DIR/nginx/certbot/conf"
live_dir="$letsencrypt_dir/live/$PUBLIC_DOMAIN"
renewal_conf="$letsencrypt_dir/renewal/$PUBLIC_DOMAIN.conf"
cert_path="$live_dir/fullchain.pem"

mkdir -p "$certbot_webroot" "$letsencrypt_dir"

if [[ "${1:-issue}" == "renew" ]]; then
  "${compose[@]}" --profile certbot run --rm certbot renew --webroot -w /var/www/certbot
  "${compose[@]}" exec nginx nginx -s reload
  exit 0
fi

if [[ -f "$cert_path" ]] && openssl x509 -checkend "$SSL_RENEW_BEFORE_SECONDS" -noout -in "$cert_path" >/dev/null; then
  echo "Existing certificate for $PUBLIC_DOMAIN is still valid; skipping issue."
  "${compose[@]}" up -d --no-deps nginx
  exit 0
fi

if [[ -f "$renewal_conf" ]]; then
  "${compose[@]}" up -d --no-deps nginx
  "${compose[@]}" --profile certbot run --rm certbot renew \
    --cert-name "$PUBLIC_DOMAIN" \
    --webroot \
    --webroot-path /var/www/certbot
  "${compose[@]}" exec nginx nginx -s reload
  exit 0
fi

if [[ ! -f "$renewal_conf" ]]; then
  mkdir -p "$live_dir"
  openssl req \
    -x509 \
    -nodes \
    -newkey rsa:2048 \
    -days 1 \
    -keyout "$live_dir/privkey.pem" \
    -out "$live_dir/fullchain.pem" \
    -subj "/CN=$PUBLIC_DOMAIN"
fi

"${compose[@]}" up -d --no-deps nginx

rm -rf \
  "$letsencrypt_dir/live/$PUBLIC_DOMAIN" \
  "$letsencrypt_dir/archive/$PUBLIC_DOMAIN" \
  "$renewal_conf"

staging_args=()
if [[ "$SSL_STAGING" == "true" ]]; then
  staging_args=(--staging)
fi

"${compose[@]}" --profile certbot run --rm certbot certonly \
  --webroot \
  --webroot-path /var/www/certbot \
  --email "$CERTBOT_EMAIL" \
  --agree-tos \
  --no-eff-email \
  --keep-until-expiring \
  --rsa-key-size "$SSL_RSA_KEY_SIZE" \
  --cert-name "$PUBLIC_DOMAIN" \
  "${staging_args[@]}" \
  -d "$PUBLIC_DOMAIN"

"${compose[@]}" exec nginx nginx -s reload
