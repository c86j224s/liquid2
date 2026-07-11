#!/bin/sh
set -eu

root_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
liquid_script="${root_dir}/liquid2/scripts/release-browser.sh"
plasma_script="${root_dir}/plasma/scripts/release-browser.sh"

usage() {
  cat <<EOF
usage:
  $0 <start|stop|restart|status|logs|build|install>
  $0 <all|liquid2|liquid|plasma> <start|stop|restart|status|logs|build|install>
  $0 <start|stop|restart|status|logs|build|install> <all|liquid2|liquid|plasma>

Controls the local release server stack:
  liquid2  Liquid2 release Flutter web and API, default ports 3001 and 3011
  plasma   Plasma release browser/API, default port 3002

Configuration:
  The root script delegates setting resolution to the product scripts and
  does not synthesize host, port, or URL overrides. Product release scripts
  select release mode only; the apps resolve user config files, defaults, and
  status output.
EOF
}

normalize_scope() {
  case "$1" in
    all) printf 'all\n' ;;
    liquid|liquid2) printf 'liquid2\n' ;;
    plasma) printf 'plasma\n' ;;
    *) return 1 ;;
  esac
}

is_cmd() {
  case "$1" in
    start|stop|restart|status|logs|build|install) return 0 ;;
    *) return 1 ;;
  esac
}

run_liquid() {
  "$liquid_script" "$cmd" "$@"
}

run_plasma() {
  "$plasma_script" "$cmd" "$@"
}

scope="all"
cmd="status"

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ] || [ "${1:-}" = "help" ]; then
  usage
  exit 0
fi

if [ $# -gt 0 ]; then
  if normalized="$(normalize_scope "$1" 2>/dev/null)"; then
    scope="$normalized"
    shift
    cmd="${1:-status}"
    [ $# -eq 0 ] || shift
  elif is_cmd "$1"; then
    cmd="$1"
    shift
    if [ $# -gt 0 ] && normalized="$(normalize_scope "$1" 2>/dev/null)"; then
      scope="$normalized"
      shift
    fi
  else
    usage >&2
    exit 2
  fi
fi

if ! is_cmd "$cmd"; then
  usage >&2
  exit 2
fi

if [ ! -x "$liquid_script" ]; then
  printf 'missing executable: %s\n' "$liquid_script" >&2
  exit 1
fi
if [ ! -x "$plasma_script" ]; then
  printf 'missing executable: %s\n' "$plasma_script" >&2
  exit 1
fi

case "$scope:$cmd" in
  all:stop)
    printf '%s\n' 'Plasma'
    printf '%s\n' '------'
    run_plasma "$@"
    printf '\n%s\n' 'Liquid2'
    printf '%s\n' '-------'
    run_liquid "$@"
    ;;
  all:*)
    printf '%s\n' 'Liquid2'
    printf '%s\n' '-------'
    run_liquid "$@"
    printf '\n%s\n' 'Plasma'
    printf '%s\n' '------'
    run_plasma "$@"
    ;;
  liquid2:*)
    run_liquid "$@"
    ;;
  plasma:*)
    run_plasma "$@"
    ;;
esac
