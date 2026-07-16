#!/bin/sh
set -eu

liquid_dir="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
root_dir="$(CDPATH= cd -- "${liquid_dir}/.." && pwd)"
. "${root_dir}/scripts/lib/wsl-process.sh"

runtime_mode="${1:-}"
shift || true
case "$runtime_mode" in
  dev)
    status_label="development"
    api_port="6011"
    web_port_value="6001"
    bin_name="liquid2-dev-api"
    ;;
  release)
    status_label="release"
    api_port="3011"
    web_port_value="3001"
    bin_name="liquid2-release-api"
    ;;
  *)
    printf 'usage: %s <dev|release> <command>\n' "$0" >&2
    exit 2
    ;;
esac

bin_path="${TMPDIR:-/tmp}/${bin_name}"
client_dir="${liquid_dir}/client"
state_root="$(wsl_state_root liquid2)"
api_state_dir="${state_root}/${runtime_mode}-api"
web_state_dir="${state_root}/${runtime_mode}-web"
api_stdout_path="${api_state_dir}/stdout.log"
api_stderr_path="${api_state_dir}/stderr.log"
web_stdout_path="${web_state_dir}/stdout.log"
web_stderr_path="${web_state_dir}/stderr.log"
fallback_api_url="http://127.0.0.1:${api_port}"
fallback_web_url="http://127.0.0.1:${web_port_value}"
operation_state_dir="${state_root}/${runtime_mode}-operation"
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

api_url() { status_field api_url "$fallback_api_url"; }
web_host() { status_field web_host '127.0.0.1'; }
web_port() { status_field web_port "$web_port_value"; }
web_url() { status_field web_url "$fallback_web_url"; }
environment_label() { status_field environment_label ''; }

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
  printf '  Mode    %s\n' "$runtime_mode"
}

build_binary() (
  mkdir -p "$(dirname "$bin_path")"
  tmp="$(mktemp "${bin_path}.new.XXXXXX")"
  cleanup_build() { rm -f "$tmp"; }
  trap cleanup_build EXIT
  trap 'cleanup_build; exit 130' HUP INT TERM
  cd "$liquid_dir"
  go build -o "$tmp" ./cmd/api
  mv -f "$tmp" "$bin_path"
)

build_web() {
  (cd "$client_dir" && flutter build web --release --dart-define="LIQUID2_API_BASE_URL=$(api_url)")
}

wait_ready() {
  url="$1"
  attempts="$2"
  process_state_dir="$3"
  count=0
  while [ "$count" -lt "$attempts" ]; do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    if ! wsl_process_matches "$process_state_dir"; then
      return 1
    fi
    sleep 0.5
    count=$((count + 1))
  done
  return 1
}

http_state() {
  if curl -fsS "$1" >/dev/null 2>&1; then
    printf 'ok'
  else
    printf 'down'
  fi
}

print_status() {
  api="$(api_url)"
  web="$(web_url)"
  runtime_status
  printf '  %-5s %-30s %-7s %s\n' 'Role' 'URL' 'HTTP' 'Service'
  printf '  %-5s %-30s %-7s %s\n' 'API' "$api" "$(http_state "${api}/api/v1/documents")" "$(wsl_process_state "$api_state_dir")"
  printf '  %-5s %-30s %-7s %s\n' 'Web' "$web" "$(http_state "$web")" "$(wsl_process_state "$web_state_dir")"
  printf '  Logs    %s | %s\n' "$api_stdout_path" "$web_stdout_path"
}

start_services() {
  api_started=0
  web_started=0
  cancel_start() {
    [ "$web_started" -eq 0 ] || wsl_process_stop "$web_state_dir"
    [ "$api_started" -eq 0 ] || wsl_process_stop "$api_state_dir"
    finish_operation
    exit 130
  }
  begin_operation
  build_binary
  trap cancel_start HUP INT TERM
  api="$(api_url)"
  wsl_process_start "$api_state_dir" "$bin_path serve" "$liquid_dir" "$api_stdout_path" "$api_stderr_path" \
    env LIQUID2_RUNTIME_MODE="$runtime_mode" "$bin_path" serve
  api_started=1
  if ! wait_ready "${api}/api/v1/documents" 16 "$api_state_dir"; then
    print_status >&2
    wsl_process_stop "$api_state_dir"
    exit 1
  fi

  host="$(web_host)"
  port="$(web_port)"
  label="$(environment_label)"
  if [ "$runtime_mode" = release ]; then
    if ! wsl_process_start "$web_state_dir" "--web-port=${port}" "$client_dir" "$web_stdout_path" "$web_stderr_path" \
      env flutter run --release -d web-server --web-hostname="$host" --web-port="$port" \
      --dart-define="LIQUID2_API_BASE_URL=${api}" --dart-define="LIQUID2_ENVIRONMENT_LABEL=${label}"; then
      wsl_process_stop "$api_state_dir"
      printf 'failed to start Liquid2 Web; see %s\n' "$web_stderr_path" >&2
      exit 1
    fi
    web_started=1
  else
    if ! wsl_process_start "$web_state_dir" "--web-port=${port}" "$client_dir" "$web_stdout_path" "$web_stderr_path" \
      env flutter run -d web-server --web-hostname="$host" --web-port="$port" \
      --dart-define="LIQUID2_API_BASE_URL=${api}" --dart-define="LIQUID2_ENVIRONMENT_LABEL=${label}"; then
      wsl_process_stop "$api_state_dir"
      printf 'failed to start Liquid2 Web; see %s\n' "$web_stderr_path" >&2
      exit 1
    fi
    web_started=1
  fi
  if ! wait_ready "$(web_url)" 240 "$web_state_dir"; then
    print_status >&2
    wsl_process_stop "$web_state_dir"
    wsl_process_stop "$api_state_dir"
    exit 1
  fi
  trap abort_operation HUP INT TERM
  finish_operation
  print_status
}

stop_services() {
  begin_operation
  wsl_process_stop "$web_state_dir"
  wsl_process_stop "$api_state_dir"
  build_binary
  print_status
  finish_operation
}

cmd="${1:-status}"
case "$cmd" in
  build)
    build_binary
    if [ "$runtime_mode" = release ]; then
      build_web
      printf 'Built %s and Flutter release web assets\n' "$bin_path"
    else
      (cd "$client_dir" && flutter pub get)
      printf 'Built %s and prepared Flutter web dependencies\n' "$bin_path"
    fi
    ;;
  install|start|restart) start_services ;;
  stop) stop_services ;;
  status)
    build_binary
    print_status
    ;;
  logs) wsl_process_logs "${2:-80}" "$api_stdout_path" "$api_stderr_path" "$web_stdout_path" "$web_stderr_path" ;;
  -h|--help|help) usage ;;
  *) usage >&2; exit 2 ;;
esac
