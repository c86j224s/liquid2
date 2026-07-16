#!/bin/sh
set -eu

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
case "$(uname -s)" in
  Darwin) ;;
  Linux)
    . "${script_dir}/../../scripts/lib/wsl-process.sh"
    if wsl_is_wsl2; then
      exec "${script_dir}/browser-wsl.sh" dev "$@"
    fi
    printf 'unsupported platform: general Linux support is not available yet\n' >&2
    exit 1
    ;;
  *)
    printf 'unsupported platform: %s\n' "$(uname -s)" >&2
    exit 1
    ;;
esac

plasma_dir="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

runtime_mode="dev"
status_label="development"
label="dev.plasma.browser-6002"
bin_path="/tmp/plasma-browser-server"
fallback_url="http://127.0.0.1:6002"
stdout_path="/tmp/plasma-browser-6002.log"
stderr_path="/tmp/plasma-browser-6002.err"
plist="${HOME}/Library/LaunchAgents/${label}.plist"
domain="gui/$(id -u)"

usage() {
  cat <<EOF
usage: $0 <start|stop|restart|status|logs|build|install>

This script only selects Plasma development mode and controls launchd. Runtime
settings are resolved by the Plasma app from PLASMA_RUNTIME_MODE=${runtime_mode}.
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

server_url() {
  if [ -x "$bin_path" ]; then
    if value="$(cd "$plasma_dir" && PLASMA_RUNTIME_MODE="$runtime_mode" "$bin_path" status -url 2>/dev/null)"; then
      printf '%s\n' "$value"
      return 0
    fi
  fi
  printf '%s\n' "$fallback_url"
}

runtime_status() {
  if [ -x "$bin_path" ]; then
    if value="$(cd "$plasma_dir" && PLASMA_RUNTIME_MODE="$runtime_mode" "$bin_path" status 2>/dev/null)"; then
      printf '%s\n' "$value"
      return 0
    fi
  fi
  printf 'Plasma %s\n' "$status_label"
  printf '  URL     %s\n' "$fallback_url"
  printf '  Mode    %s\n' "$runtime_mode"
}

is_loaded() {
  launchctl print "${domain}/${label}" >/dev/null 2>&1
}

build_binary() {
  tmp="${bin_path}.new"
  mkdir -p "$(dirname "$bin_path")"
  (cd "$plasma_dir" && go build -o "$tmp" ./cmd/plasma)
  mv "$tmp" "$bin_path"
}

write_plist() {
  mkdir -p "$(dirname "$plist")"
  tmp="${plist}.tmp"
  cat >"$tmp" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>$(xml_escape "$label")</string>
  <key>ProgramArguments</key>
  <array>
    <string>$(xml_escape "$bin_path")</string>
    <string>serve</string>
  </array>
  <key>WorkingDirectory</key>
  <string>$(xml_escape "$plasma_dir")</string>
  <key>EnvironmentVariables</key>
  <dict>
    <key>HOME</key>
    <string>$(xml_escape "$HOME")</string>
    <key>USER</key>
    <string>$(xml_escape "${USER:-}")</string>
    <key>PATH</key>
    <string>$(xml_escape "$PATH")</string>
    <key>PLASMA_RUNTIME_MODE</key>
    <string>$(xml_escape "$runtime_mode")</string>
  </dict>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardOutPath</key>
  <string>$(xml_escape "$stdout_path")</string>
  <key>StandardErrorPath</key>
  <string>$(xml_escape "$stderr_path")</string>
</dict>
</plist>
EOF
  plutil -lint "$tmp" >/dev/null
  mv "$tmp" "$plist"
}

bootout_service() {
  launchctl bootout "$domain" "$plist" >/dev/null 2>&1 || launchctl remove "$label" >/dev/null 2>&1 || true
}

bootstrap_service() {
  if ! is_loaded; then
    launchctl bootstrap "$domain" "$plist"
  fi
}

kickstart_service() {
  launchctl kickstart -k "${domain}/${label}"
}

wait_ready() {
  for _ in $(seq 1 80); do
    if curl -fsS "$(server_url)/api/missions" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.1
  done
  return 1
}

print_status() {
  base_url="$(server_url)"
  runtime_status
  if is_loaded; then
    printf '  Service loaded\n'
  else
    printf '  Service stopped\n'
  fi
  if curl -fsS "${base_url}/api/missions" >/dev/null 2>&1; then
    printf '  HTTP    ok\n'
  else
    printf '  HTTP    down\n'
  fi
  printf '  Logs    %s | %s\n' "$stdout_path" "$stderr_path"
}

cmd="${1:-status}"
case "$cmd" in
  build)
    build_binary
    printf 'Built %s\n' "$bin_path"
    ;;
  install|start)
    build_binary
    write_plist
    bootout_service
    bootstrap_service
    kickstart_service
    wait_ready || { print_status >&2; exit 1; }
    print_status
    ;;
  restart)
    build_binary
    write_plist
    bootout_service
    bootstrap_service
    kickstart_service
    wait_ready || { print_status >&2; exit 1; }
    print_status
    ;;
  stop)
    bootout_service
    build_binary
    print_status
    ;;
  status)
    build_binary
    print_status
    ;;
  logs)
    lines="${2:-80}"
    found=0
    for log_path in "$stdout_path" "$stderr_path"; do
      if [ -f "$log_path" ]; then
        found=1
        tail -n "$lines" "$log_path"
      else
        printf '%s: no log yet\n' "$log_path"
      fi
    done
    [ "$found" -eq 1 ] || exit 0
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    usage >&2
    exit 2
    ;;
esac
