#!/bin/sh
set -eu

label="${DEMO_API_LABEL:-com.c86j224s.liquid2.demo-api}"
bin_path="${DEMO_API_BIN:-/tmp/liquid2-api}"
port="${LIQUID2_INTERNAL_PORT:-8080}"
repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
db_path="${LIQUID2_DB_PATH:-${repo_root}/liquid2.db}"
translation_provider="${LIQUID2_TRANSLATION_PROVIDER:-codex}"

internal_ip_from_interfaces() {
  ifconfig 2>/dev/null | awk '/inet 100\\./ { print $2; exit }'
}

codex_command() {
  if [ -n "${LIQUID2_CODEX_COMMAND:-}" ]; then
    printf '%s\n' "$LIQUID2_CODEX_COMMAND"
    return 0
  fi
  command -v codex 2>/dev/null || true
}

detect_addr() {
  if [ -n "${LIQUID2_INTERNAL_ADDR:-}" ]; then
    printf '%s\n' "$LIQUID2_INTERNAL_ADDR"
    return 0
  fi
  if [ -n "${LIQUID2_INTERNAL_IP:-}" ]; then
    printf '%s:%s\n' "$LIQUID2_INTERNAL_IP" "$port"
    return 0
  fi

  ip="$(internal_ip_from_interfaces)"
  if [ -z "$ip" ]; then
    echo "internal IPv4 address not found. Set LIQUID2_INTERNAL_ADDR or LIQUID2_INTERNAL_IP." >&2
    return 1
  fi
  printf '%s:%s\n' "$ip" "$port"
}

addr="$(detect_addr)"
base_url="http://${addr}"
codex_bin="$(codex_command)"
if [ "$translation_provider" = "codex" ] && [ -z "$codex_bin" ]; then
  echo "codex CLI not found. Install Codex or set LIQUID2_CODEX_COMMAND." >&2
  exit 1
fi

go build -o "$bin_path" ./cmd/api
launchctl remove "$label" >/dev/null 2>&1 || true
launchctl submit -l "$label" -- /usr/bin/env \
  HOME="$HOME" \
  USER="${USER:-allthatcode}" \
  PATH="$PATH" \
  LIQUID2_ADDR="$addr" \
  LIQUID2_LOG_FORMAT="${LIQUID2_LOG_FORMAT:-text}" \
  LIQUID2_DB_PATH="$db_path" \
  LIQUID2_JOBS_ENABLED=1 \
  LIQUID2_TRANSLATION_PROVIDER="$translation_provider" \
  LIQUID2_CODEX_COMMAND="$codex_bin" \
  "$bin_path"

for _ in $(seq 1 80); do
  if curl -fsS "${base_url}/api/v1/documents" >/dev/null 2>&1; then
    printf 'Liquid2 API listening on internal address: %s\n' "$base_url"
    printf 'Database: %s\n' "$db_path"
    printf 'Translation provider: %s\n' "$translation_provider"
    printf 'Stop with: make demo-stop\n'
    exit 0
  fi
  sleep 0.1
done

echo "Liquid2 API did not become ready at ${base_url}" >&2
launchctl list | grep "$label" >&2 || true
exit 1
