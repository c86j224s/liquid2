#!/bin/sh
set -eu

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
case "$(uname -s)" in
  Darwin) ;;
  Linux)
    . "${script_dir}/../../scripts/lib/wsl-process.sh"
    if wsl_is_wsl2; then
      exec "${script_dir}/browser-wsl.sh" release "$@"
    fi
    printf 'unsupported platform: general Linux support is not available yet\n' >&2
    exit 1
    ;;
  *)
    printf 'unsupported platform: %s\n' "$(uname -s)" >&2
    exit 1
    ;;
esac

liquid_dir="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

runtime_mode="release"
status_label="release"
api_label="release.liquid2.api-3011"
web_label="release.liquid2.web-3001"
bin_path="/tmp/liquid2-release-api"
api_stdout_path="/tmp/liquid2-release-api-3011.log"
api_stderr_path="/tmp/liquid2-release-api-3011.err"
web_stdout_path="/tmp/liquid2-release-web-3001.log"
web_stderr_path="/tmp/liquid2-release-web-3001.err"
api_plist="${HOME}/Library/LaunchAgents/${api_label}.plist"
web_plist="${HOME}/Library/LaunchAgents/${web_label}.plist"
domain="gui/$(id -u)"
client_dir="${liquid_dir}/client"

fallback_api_addr="127.0.0.1:3011"
fallback_api_url="http://127.0.0.1:3011"
fallback_web_host="127.0.0.1"
fallback_web_port="3001"
fallback_web_url="http://127.0.0.1:3001"
fallback_db="${HOME}/Library/Application Support/Liquid2/liquid2.db"

usage() {
  cat <<EOF
usage: $0 <start|stop|restart|status|logs|build|install>

This script only selects Liquid2 release mode and controls launchd/Flutter.
Runtime settings are resolved by the Liquid2 API app from
LIQUID2_RUNTIME_MODE=${runtime_mode}.
EOF
}

xml_escape() {
  printf '%s' "$1" | sed \
    -e 's/&/\&amp;/g' \
    -e 's/</\&lt;/g' \
    -e 's/>/\&gt;/g' \
    -e 's/"/\&quot;/g' \
    -e "s/'/\&apos;/g"
}

status_field() {
  field="$1"
  fallback="$2"
  if [ -x "$bin_path" ]; then
    if value="$(cd "$liquid_dir" && LIQUID2_RUNTIME_MODE="$runtime_mode" "$bin_path" status -field "$field" 2>/dev/null)"; then
      printf '%s\n' "$value"
      return 0
    fi
  fi
  printf '%s\n' "$fallback"
}

api_addr() { status_field api_addr "$fallback_api_addr"; }
api_url() { status_field api_url "$fallback_api_url"; }
web_host() { status_field web_host "$fallback_web_host"; }
web_port() { status_field web_port "$fallback_web_port"; }
web_url() { status_field web_url "$fallback_web_url"; }
db_path() { status_field db_path "$fallback_db"; }
environment_label() { status_field environment_label ""; }

runtime_status() {
  if [ -x "$bin_path" ]; then
    if value="$(cd "$liquid_dir" && LIQUID2_RUNTIME_MODE="$runtime_mode" "$bin_path" status 2>/dev/null)"; then
      printf '%s\n' "$value"
      return 0
    fi
  fi
  printf 'Liquid2 %s\n' "$status_label"
  printf '  API     %s\n' "$fallback_api_url"
  printf '  Web     %s\n' "$fallback_web_url"
  printf '  DB      %s\n' "$fallback_db"
  printf '  Mode    %s\n' "$runtime_mode"
}

is_loaded() {
  label="$1"
  launchctl print "${domain}/${label}" >/dev/null 2>&1
}

build_binary() {
  tmp="${bin_path}.new"
  mkdir -p "$(dirname "$bin_path")"
  (cd "$liquid_dir" && go build -o "$tmp" ./cmd/api)
  mv "$tmp" "$bin_path"
}

build_web() {
  (cd "$client_dir" && flutter build web --release --dart-define="LIQUID2_API_BASE_URL=$(api_url)")
}

write_api_plist() {
  mkdir -p "$(dirname "$api_plist")"
  tmp="${api_plist}.tmp"
  cat >"$tmp" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>$(xml_escape "$api_label")</string>
  <key>ProgramArguments</key>
  <array>
    <string>$(xml_escape "$bin_path")</string>
    <string>serve</string>
  </array>
  <key>WorkingDirectory</key>
  <string>$(xml_escape "$liquid_dir")</string>
  <key>EnvironmentVariables</key>
  <dict>
    <key>HOME</key>
    <string>$(xml_escape "$HOME")</string>
    <key>USER</key>
    <string>$(xml_escape "${USER:-}")</string>
    <key>PATH</key>
    <string>$(xml_escape "$PATH")</string>
    <key>LIQUID2_RUNTIME_MODE</key>
    <string>$(xml_escape "$runtime_mode")</string>
  </dict>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardOutPath</key>
  <string>$(xml_escape "$api_stdout_path")</string>
  <key>StandardErrorPath</key>
  <string>$(xml_escape "$api_stderr_path")</string>
</dict>
</plist>
EOF
  plutil -lint "$tmp" >/dev/null
  mv "$tmp" "$api_plist"
}

write_web_plist() {
  host="$(web_host)"
  port="$(web_port)"
  api="$(api_url)"
  label="$(environment_label)"
  mkdir -p "$(dirname "$web_plist")"
  tmp="${web_plist}.tmp"
  cat >"$tmp" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>$(xml_escape "$web_label")</string>
  <key>ProgramArguments</key>
  <array>
    <string>/usr/bin/env</string>
    <string>flutter</string>
    <string>run</string>
    <string>--release</string>
    <string>-d</string>
    <string>web-server</string>
    <string>--web-hostname=$(xml_escape "$host")</string>
    <string>--web-port=$(xml_escape "$port")</string>
    <string>--dart-define=LIQUID2_API_BASE_URL=$(xml_escape "$api")</string>
    <string>--dart-define=LIQUID2_ENVIRONMENT_LABEL=$(xml_escape "$label")</string>
  </array>
  <key>WorkingDirectory</key>
  <string>$(xml_escape "$client_dir")</string>
  <key>EnvironmentVariables</key>
  <dict>
    <key>HOME</key>
    <string>$(xml_escape "$HOME")</string>
    <key>USER</key>
    <string>$(xml_escape "${USER:-}")</string>
    <key>PATH</key>
    <string>$(xml_escape "$PATH")</string>
  </dict>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardOutPath</key>
  <string>$(xml_escape "$web_stdout_path")</string>
  <key>StandardErrorPath</key>
  <string>$(xml_escape "$web_stderr_path")</string>
</dict>
</plist>
EOF
  plutil -lint "$tmp" >/dev/null
  mv "$tmp" "$web_plist"
}

bootout_service() {
  plist="$1"
  label="$2"
  launchctl bootout "$domain" "$plist" >/dev/null 2>&1 || launchctl remove "$label" >/dev/null 2>&1 || true
}

bootstrap_service() {
  plist="$1"
  label="$2"
  if ! is_loaded "$label"; then
    launchctl bootstrap "$domain" "$plist"
  fi
}

kickstart_service() {
  label="$1"
  launchctl kickstart -k "${domain}/${label}"
}

wait_api_ready() {
  for _ in $(seq 1 80); do
    if curl -fsS "$(api_url)/api/v1/documents" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.1
  done
  return 1
}

wait_web_ready() {
  for _ in $(seq 1 240); do
    if curl -fsS "$(web_url)" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.5
  done
  return 1
}

service_state() {
  label="$1"
  if is_loaded "$label"; then
    printf 'loaded'
  else
    printf 'stopped'
  fi
}

http_state() {
  probe="$1"
  if curl -fsS "$probe" >/dev/null 2>&1; then
    printf 'ok'
  else
    printf 'down'
  fi
}

print_one_status() {
  name="$1"
  label="$2"
  url="$3"
  probe="$4"
  printf '  %-5s %-30s %-7s %s\n' "$name" "$url" "$(http_state "$probe")" "$(service_state "$label")"
}

print_status() {
  api="$(api_url)"
  web="$(web_url)"
  runtime_status
  printf '  %-5s %-30s %-7s %s\n' 'Role' 'URL' 'HTTP' 'Service'
  print_one_status "API" "$api_label" "$api" "${api}/api/v1/documents"
  print_one_status "Web" "$web_label" "$web" "$web"
  printf '  Logs    %s | %s\n' "$api_stdout_path" "$web_stdout_path"
}

start_services() {
  build_binary
  write_api_plist
  bootout_service "$api_plist" "$api_label"
  bootstrap_service "$api_plist" "$api_label"
  kickstart_service "$api_label"
  wait_api_ready || { print_status >&2; exit 1; }

  write_web_plist
  bootout_service "$web_plist" "$web_label"
  bootstrap_service "$web_plist" "$web_label"
  kickstart_service "$web_label"
  wait_web_ready || { print_status >&2; exit 1; }
  print_status
}

stop_services() {
  bootout_service "$web_plist" "$web_label"
  bootout_service "$api_plist" "$api_label"
  build_binary
  print_status
}

print_logs() {
  lines="${1:-80}"
  found=0
  for log_path in "$api_stdout_path" "$api_stderr_path" "$web_stdout_path" "$web_stderr_path"; do
    if [ -f "$log_path" ]; then
      found=1
      printf '== %s ==\n' "$log_path"
      tail -n "$lines" "$log_path"
    else
      printf '%s: no log yet\n' "$log_path"
    fi
  done
  [ "$found" -eq 1 ] || exit 0
}

cmd="${1:-status}"
case "$cmd" in
  build)
    build_binary
    build_web
    printf 'Built %s and Flutter release web assets\n' "$bin_path"
    ;;
  install|start|restart)
    start_services
    ;;
  stop)
    stop_services
    ;;
  status)
    build_binary
    print_status
    ;;
  logs)
    print_logs "${2:-80}"
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    usage >&2
    exit 2
    ;;
esac
