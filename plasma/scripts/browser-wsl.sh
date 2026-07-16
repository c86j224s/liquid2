#!/bin/sh
set -eu

plasma_dir="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
root_dir="$(CDPATH= cd -- "${plasma_dir}/.." && pwd)"
. "${root_dir}/scripts/lib/wsl-process.sh"

runtime_mode="${1:-}"
shift || true
case "$runtime_mode" in
  dev)
    status_label="development"
    port="6002"
    bin_name="plasma-browser-server"
    ;;
  release)
    status_label="release"
    port="3002"
    bin_name="plasma-release-browser-server"
    ;;
  *)
    printf 'usage: %s <dev|release> <command>\n' "$0" >&2
    exit 2
    ;;
esac

bin_path="${TMPDIR:-/tmp}/${bin_name}"
state_dir="$(wsl_state_root plasma)/${runtime_mode}"
stdout_path="${state_dir}/stdout.log"
stderr_path="${state_dir}/stderr.log"
fallback_url="http://127.0.0.1:${port}"
operation_state_dir="$(wsl_state_root plasma)/${runtime_mode}-operation"
operation_locked=0

begin_operation() {
  wsl_process_lock "$operation_state_dir"
  operation_locked=1
}

finish_operation() {
  [ "$operation_locked" -eq 0 ] || wsl_process_unlock "$operation_state_dir"
  operation_locked=0
}

abort_operation() {
  finish_operation
  exit 130
}

trap finish_operation EXIT
trap abort_operation HUP INT TERM

usage() {
  printf 'usage: %s <start|stop|restart|status|logs|build|install>\n' "$0"
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

build_binary() (
  mkdir -p "$(dirname "$bin_path")"
  tmp="$(mktemp "${bin_path}.new.XXXXXX")"
  cleanup_build() { rm -f "$tmp"; }
  trap cleanup_build EXIT
  trap 'cleanup_build; exit 130' HUP INT TERM
  cd "$plasma_dir"
  go build -o "$tmp" ./cmd/plasma
  mv -f "$tmp" "$bin_path"
)

wait_ready() {
  count=0
  while [ "$count" -lt 16 ]; do
    if curl -fsS "$(server_url)/api/missions" >/dev/null 2>&1; then
      return 0
    fi
    if ! wsl_process_matches "$state_dir"; then
      return 1
    fi
    sleep 0.5
    count=$((count + 1))
  done
  return 1
}

print_status() {
  base_url="$(server_url)"
  runtime_status
  printf '  Service %s\n' "$(wsl_process_state "$state_dir")"
  if curl -fsS "${base_url}/api/missions" >/dev/null 2>&1; then
    printf '  HTTP    ok\n'
  else
    printf '  HTTP    down\n'
  fi
  printf '  Logs    %s | %s\n' "$stdout_path" "$stderr_path"
}

start_service() {
  service_started=0
  cancel_start() {
    [ "$service_started" -eq 0 ] || wsl_process_stop "$state_dir"
    finish_operation
    exit 130
  }
  begin_operation
  build_binary
  trap cancel_start HUP INT TERM
  wsl_process_start "$state_dir" "$bin_path serve" "$plasma_dir" "$stdout_path" "$stderr_path" \
    env PLASMA_RUNTIME_MODE="$runtime_mode" "$bin_path" serve
  service_started=1
  if ! wait_ready; then
    print_status >&2
    wsl_process_stop "$state_dir"
    exit 1
  fi
  trap abort_operation HUP INT TERM
  finish_operation
  print_status
}

cmd="${1:-status}"
case "$cmd" in
  build)
    build_binary
    printf 'Built %s\n' "$bin_path"
    ;;
  install|start|restart) start_service ;;
  stop)
    begin_operation
    wsl_process_stop "$state_dir"
    build_binary
    print_status
    finish_operation
    ;;
  status)
    build_binary
    print_status
    ;;
  logs) wsl_process_logs "${2:-80}" "$stdout_path" "$stderr_path" ;;
  -h|--help|help) usage ;;
  *) usage >&2; exit 2 ;;
esac
