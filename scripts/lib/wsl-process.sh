#!/bin/sh

wsl_state_root() {
  product="$1"
  printf '%s/%s/services\n' "${XDG_STATE_HOME:-${HOME}/.local/state}" "$product"
}

wsl_is_wsl2_release() {
  case "$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]')" in
    *microsoft*standard*wsl2*) return 0 ;;
    *) return 1 ;;
  esac
}

wsl_is_wsl2() {
  [ "$(uname -s)" = "Linux" ] || return 1
  [ -r /proc/sys/kernel/osrelease ] || return 1
  wsl_is_wsl2_release "$(cat /proc/sys/kernel/osrelease)"
}

wsl_process_stat_field() {
  pid="$1"
  field="$2"
  [ -r "/proc/${pid}/stat" ] || return 1
  IFS= read -r stat <"/proc/${pid}/stat" || return 1
  stat="${stat##*) }"
  set -- $stat
  case "$field" in
    1) printf '%s\n' "$1" ;;
    3) printf '%s\n' "$3" ;;
    4) printf '%s\n' "$4" ;;
    20) printf '%s\n' "${20}" ;;
    *) return 2 ;;
  esac
}

wsl_process_start_time() {
  wsl_process_stat_field "$1" 20
}

wsl_current_process_identity() {
  [ -r /proc/self/stat ] || return 1
  IFS= read -r stat </proc/self/stat || return 1
  WSL_CURRENT_PROCESS_PID="${stat%% *}"
  stat="${stat##*) }"
  set -- $stat
  WSL_CURRENT_PROCESS_START_TIME="${20}"
  case "$WSL_CURRENT_PROCESS_PID" in ''|*[!0-9]*) return 1 ;; esac
  case "$WSL_CURRENT_PROCESS_START_TIME" in ''|*[!0-9]*) return 1 ;; esac
}

wsl_process_run_state() {
  wsl_process_stat_field "$1" 1
}

wsl_process_group_id() {
  wsl_process_stat_field "$1" 3
}

wsl_process_session_id() {
  wsl_process_stat_field "$1" 4
}

wsl_process_has_token() {
  pid="$1"
  token="$2"
  [ -r "/proc/${pid}/environ" ] || return 1
  tr '\000' '\n' <"/proc/${pid}/environ" | grep -Fx -- "WSL_PROCESS_TOKEN=${token}" >/dev/null 2>&1
}

wsl_process_state_file() {
  printf '%s/process.state\n' "$1"
}

wsl_process_read_state() {
  state_dir="$1"
  state_file="$(wsl_process_state_file "$state_dir")"
  [ -r "$state_file" ] || return 1

  WSL_PROCESS_LEADER_PID=""
  WSL_PROCESS_LEADER_START_TIME=""
  WSL_PROCESS_GROUP_ID=""
  WSL_PROCESS_TOKEN=""
  while IFS='=' read -r key value || [ -n "$key" ]; do
    case "$key" in
      leader_pid) WSL_PROCESS_LEADER_PID="$value" ;;
      leader_start_time) WSL_PROCESS_LEADER_START_TIME="$value" ;;
      group_id) WSL_PROCESS_GROUP_ID="$value" ;;
      token) WSL_PROCESS_TOKEN="$value" ;;
    esac
  done <"$state_file"

  case "$WSL_PROCESS_LEADER_PID" in ''|*[!0-9]*) return 1 ;; esac
  case "$WSL_PROCESS_LEADER_START_TIME" in ''|*[!0-9]*) return 1 ;; esac
  case "$WSL_PROCESS_GROUP_ID" in ''|*[!0-9]*) return 1 ;; esac
  case "$WSL_PROCESS_TOKEN" in ''|*[!A-Za-z0-9._-]*) return 1 ;; esac
}

# Returns 0 when the recorded process group contains one or more processes and
# every member carries the recorded per-start token. Returns 1 when the group
# is gone and 2 when ownership cannot be proven.
wsl_process_group_is_owned() {
  state_dir="$1"
  wsl_process_read_state "$state_dir" || return 2

  found=0
  for process_dir in /proc/[0-9]*; do
    pid="${process_dir#/proc/}"
    [ "$(wsl_process_group_id "$pid" 2>/dev/null || true)" = "$WSL_PROCESS_GROUP_ID" ] || continue
    case "$(wsl_process_run_state "$pid" 2>/dev/null || true)" in
      Z|X) continue ;;
    esac
    if ! wsl_process_has_token "$pid" "$WSL_PROCESS_TOKEN"; then
      [ -d "$process_dir" ] || continue
      return 2
    fi
    found=1
  done
  [ "$found" -eq 1 ]
}

wsl_process_token_group_is_owned() {
  group_id="$1"
  token="$2"
  case "$group_id" in ''|*[!0-9]*) return 2 ;; esac

  found=0
  for process_dir in /proc/[0-9]*; do
    pid="${process_dir#/proc/}"
    [ "$(wsl_process_group_id "$pid" 2>/dev/null || true)" = "$group_id" ] || continue
    case "$(wsl_process_run_state "$pid" 2>/dev/null || true)" in
      Z|X) continue ;;
    esac
    if ! wsl_process_has_token "$pid" "$token"; then
      [ -d "$process_dir" ] || continue
      return 2
    fi
    found=1
  done
  [ "$found" -eq 1 ]
}

wsl_process_collect_token_groups() {
  token="$1"
  groups_file="$2"
  : >"$groups_file"
  for process_dir in /proc/[0-9]*; do
    pid="${process_dir#/proc/}"
    case "$(wsl_process_run_state "$pid" 2>/dev/null || true)" in
      Z|X|'') continue ;;
    esac
    wsl_process_has_token "$pid" "$token" || continue
    group_id="$(wsl_process_group_id "$pid" 2>/dev/null || true)"
    case "$group_id" in ''|*[!0-9]*) continue ;; esac
    grep -Fx -- "$group_id" "$groups_file" >/dev/null 2>&1 || printf '%s\n' "$group_id" >>"$groups_file"
  done
}

wsl_process_token_is_owned() {
  state_dir="$1"
  wsl_process_read_state "$state_dir" || return 2

  groups_file="$(mktemp)"
  wsl_process_collect_token_groups "$WSL_PROCESS_TOKEN" "$groups_file"
  found=0
  unsafe=0
  while IFS= read -r group_id; do
    [ -n "$group_id" ] || continue
    if wsl_process_token_group_is_owned "$group_id" "$WSL_PROCESS_TOKEN"; then
      ownership_status=0
    else
      ownership_status=$?
    fi
    case "$ownership_status" in
      0) found=1 ;;
      2) unsafe=1 ;;
    esac
  done <"$groups_file"
  rm -f "$groups_file"

  [ "$found" -eq 1 ] && return 0
  [ "$unsafe" -eq 1 ] && return 2
  return 1
}

wsl_process_token_processes_remain() {
  token="$1"
  for process_dir in /proc/[0-9]*; do
    pid="${process_dir#/proc/}"
    case "$(wsl_process_run_state "$pid" 2>/dev/null || true)" in
      Z|X|'') continue ;;
    esac
    wsl_process_has_token "$pid" "$token" && return 0
  done
  return 1
}

wsl_process_matches() {
  state_dir="$1"
  if wsl_process_group_is_owned "$state_dir"; then
    return 0
  else
    group_status=$?
  fi
  [ "$group_status" -eq 1 ] || return "$group_status"
  wsl_process_token_is_owned "$state_dir"
}

wsl_process_state() {
  state_dir="$1"
  if wsl_process_matches "$state_dir"; then
    printf 'running\n'
  elif [ -e "$(wsl_process_state_file "$state_dir")" ]; then
    printf 'stale\n'
  else
    printf 'stopped\n'
  fi
}

wsl_process_clear_state_locked() {
  state_dir="$1"
  rm -f "$(wsl_process_state_file "$state_dir")"
}

wsl_process_clear_state() {
  state_dir="$1"
  wsl_process_lock "$state_dir" || return 1
  wsl_process_clear_state_locked "$state_dir"
  wsl_process_unlock "$state_dir"
}

wsl_process_lock_owner_is_live() {
  lock_dir="$1"
  owner_file="${lock_dir}/owner.state"
  [ -r "$owner_file" ] || return 1
  owner_pid=""
  owner_start_time=""
  while IFS='=' read -r key value || [ -n "$key" ]; do
    case "$key" in
      pid) owner_pid="$value" ;;
      start_time) owner_start_time="$value" ;;
    esac
  done <"$owner_file"
  case "$owner_pid" in ''|*[!0-9]*) return 1 ;; esac
  case "$owner_start_time" in ''|*[!0-9]*) return 1 ;; esac
  [ "$(wsl_process_start_time "$owner_pid" 2>/dev/null || true)" = "$owner_start_time" ]
}

wsl_process_lock() {
  state_dir="$1"
  lock_dir="${state_dir}/.operation-lock"
  mkdir -p "$state_dir"
  attempts=0
  missing_owner_attempts=0
  while ! mkdir "$lock_dir" 2>/dev/null; do
    if [ -e "${lock_dir}/owner.state" ]; then
      missing_owner_attempts=0
      if ! wsl_process_lock_owner_is_live "$lock_dir"; then
        rm -f "${lock_dir}/owner.state"
        rmdir "$lock_dir" 2>/dev/null || true
      fi
    else
      missing_owner_attempts=$((missing_owner_attempts + 1))
      if [ "$missing_owner_attempts" -ge 10 ]; then
        rmdir "$lock_dir" 2>/dev/null || true
      fi
    fi
    attempts=$((attempts + 1))
    if [ "$attempts" -ge 600 ]; then
      printf 'timed out waiting for service operation lock: %s\n' "$state_dir" >&2
      return 1
    fi
    sleep 0.05
  done

  if ! wsl_current_process_identity; then
    rmdir "$lock_dir" 2>/dev/null || true
    return 1
  fi
  (
    umask 077
    printf 'pid=%s\nstart_time=%s\n' "$WSL_CURRENT_PROCESS_PID" "$WSL_CURRENT_PROCESS_START_TIME" >"${lock_dir}/owner.state.new"
    mv -f "${lock_dir}/owner.state.new" "${lock_dir}/owner.state"
  )
}

wsl_process_unlock() {
  state_dir="$1"
  lock_dir="${state_dir}/.operation-lock"
  rm -f "${lock_dir}/owner.state"
  rmdir "$lock_dir" 2>/dev/null || true
}

wsl_process_write_state_locked() {
  state_dir="$1"
  leader_pid="$2"
  leader_start_time="$3"
  group_id="$4"
  token="$5"
  state_file="$(wsl_process_state_file "$state_dir")"
  temp_file="${state_file}.new.$$"
  (
    umask 077
    printf 'leader_pid=%s\nleader_start_time=%s\ngroup_id=%s\ntoken=%s\n' \
      "$leader_pid" "$leader_start_time" "$group_id" "$token" >"$temp_file"
    mv -f "$temp_file" "$state_file"
  )
}

# Sends a signal only after ownership of every process in the group has been
# rechecked. This avoids signalling a reused PID or process-group identifier.
wsl_process_signal_token_group() {
  group_id="$1"
  token="$2"
  signal="$3"
  wsl_process_token_group_is_owned "$group_id" "$token"
  ownership_status=$?
  if [ "$ownership_status" -ne 0 ]; then
    return "$ownership_status"
  fi
  /bin/kill "-${signal}" -- "-${group_id}" >/dev/null 2>&1 || true
}

wsl_process_stop_token_group() {
  group_id="$1"
  token="$2"
  wsl_process_token_group_is_owned "$group_id" "$token"
  ownership_status=$?
  case "$ownership_status" in
    1) return 0 ;;
    2) return 1 ;;
  esac

  wsl_process_signal_token_group "$group_id" "$token" TERM || return 1
  attempts=0
  while wsl_process_token_group_is_owned "$group_id" "$token" && [ "$attempts" -lt 50 ]; do
    sleep 0.1
    attempts=$((attempts + 1))
  done
  if wsl_process_token_group_is_owned "$group_id" "$token"; then
    if ! wsl_process_signal_token_group "$group_id" "$token" KILL; then
      return 1
    fi
    attempts=0
    while wsl_process_token_group_is_owned "$group_id" "$token" && [ "$attempts" -lt 20 ]; do
      sleep 0.1
      attempts=$((attempts + 1))
    done
  fi
  if wsl_process_token_group_is_owned "$group_id" "$token"; then
    printf 'failed to stop owned process group %s\n' "$group_id" >&2
    return 1
  fi
}

wsl_process_stop_locked() {
  state_dir="$1"
  if ! wsl_process_read_state "$state_dir"; then
    wsl_process_clear_state_locked "$state_dir"
    return 0
  fi

  token="$WSL_PROCESS_TOKEN"
  stop_failed=0
  passes=0
  while wsl_process_token_processes_remain "$token" && [ "$passes" -lt 5 ]; do
    groups_file="$(mktemp)"
    wsl_process_collect_token_groups "$token" "$groups_file"
    while IFS= read -r group_id; do
      [ -n "$group_id" ] || continue
      wsl_process_stop_token_group "$group_id" "$token" || stop_failed=1
    done <"$groups_file"
    rm -f "$groups_file"
    passes=$((passes + 1))
  done

  if wsl_process_token_processes_remain "$token"; then
    printf 'failed to stop all token-owned process groups\n' >&2
    return 1
  fi
  if [ "$stop_failed" -ne 0 ]; then
    printf 'failed to verify token-owned process groups during stop\n' >&2
    return 1
  fi
  wsl_process_clear_state_locked "$state_dir"
}

wsl_process_stop() {
  state_dir="$1"
  wsl_process_lock "$state_dir" || return 1
  if wsl_process_stop_locked "$state_dir"; then
    wsl_process_unlock "$state_dir"
    return 0
  fi
  wsl_process_unlock "$state_dir"
  return 1
}

# Before state is published, only a living leader with its recorded start time
# may be signalled without a token. Never signal a numeric group from this
# path: its original leader may already be gone and the identifier reused.
wsl_process_started_leader_matches() {
  leader_pid="$1"
  leader_start_time="$2"
  case "$leader_pid" in ''|*[!0-9]*) return 2 ;; esac
  case "$leader_start_time" in ''|*[!0-9]*) return 2 ;; esac
  case "$(wsl_process_run_state "$leader_pid" 2>/dev/null || true)" in
    Z|X|'') return 1 ;;
  esac
  [ "$(wsl_process_start_time "$leader_pid" 2>/dev/null || true)" = "$leader_start_time" ]
}

wsl_process_cleanup_started_leader() {
  leader_pid="$1"
  leader_start_time="$2"
  if ! wsl_process_started_leader_matches "$leader_pid" "$leader_start_time"; then
    return 0
  fi
  /bin/kill -TERM "$leader_pid" >/dev/null 2>&1 || true
  attempts=0
  while wsl_process_started_leader_matches "$leader_pid" "$leader_start_time" && [ "$attempts" -lt 50 ]; do
    sleep 0.1
    attempts=$((attempts + 1))
  done
  if wsl_process_started_leader_matches "$leader_pid" "$leader_start_time"; then
    /bin/kill -KILL "$leader_pid" >/dev/null 2>&1 || true
  fi
}

wsl_process_cleanup_token_group() {
  group_id="$1"
  token="$2"
  wsl_process_stop_token_group "$group_id" "$token" || true
}

wsl_process_cleanup_token_groups() {
  token="$1"
  passes=0
  while wsl_process_token_processes_remain "$token" && [ "$passes" -lt 5 ]; do
    groups_file="$(mktemp)"
    wsl_process_collect_token_groups "$token" "$groups_file"
    while IFS= read -r group_id; do
      [ -n "$group_id" ] || continue
      wsl_process_cleanup_token_group "$group_id" "$token"
    done <"$groups_file"
    rm -f "$groups_file"
    passes=$((passes + 1))
  done
  ! wsl_process_token_processes_remain "$token"
}

wsl_process_start_locked() {
  state_dir="$1"
  command_marker="$2"
  work_dir="$3"
  stdout_path="$4"
  stderr_path="$5"
  shift 5

  wsl_process_stop_locked "$state_dir" || return 1
  token_file="$(mktemp "${state_dir}/.process-token.XXXXXX")"
  token="$(basename "$token_file")"
  rm -f "$token_file"

  (
    cd "$work_dir"
    exec nohup setsid env "WSL_PROCESS_TOKEN=${token}" "$@" >"$stdout_path" 2>"$stderr_path" </dev/null
  ) &
  leader_pid=$!
  attempts=0
  leader_start_time=""
  group_id=""
  while [ "$attempts" -lt 20 ]; do
    leader_start_time="$(wsl_process_start_time "$leader_pid" 2>/dev/null || true)"
    group_id="$(wsl_process_group_id "$leader_pid" 2>/dev/null || true)"
    session_id="$(wsl_process_session_id "$leader_pid" 2>/dev/null || true)"
    if [ -n "$leader_start_time" ] && [ -n "$group_id" ] && [ "$group_id" = "$session_id" ] && \
      wsl_process_has_token "$leader_pid" "$token"; then
      break
    fi
    sleep 0.05
    attempts=$((attempts + 1))
  done
  if [ -z "$leader_start_time" ] || [ -z "$group_id" ] || \
    [ "$(wsl_process_session_id "$leader_pid" 2>/dev/null || true)" != "$group_id" ] || \
    ! wsl_process_has_token "$leader_pid" "$token"; then
    wsl_process_cleanup_started_leader "$leader_pid" "$leader_start_time"
    if ! wsl_process_cleanup_token_groups "$token"; then
      printf 'failed to reclaim token-owned process groups after start failure\n' >&2
    fi
    wait "$leader_pid" 2>/dev/null || true
    return 1
  fi

  wsl_process_write_state_locked "$state_dir" "$leader_pid" "$leader_start_time" "$group_id" "$token"
  if ! wsl_process_matches "$state_dir"; then
    wsl_process_cleanup_started_leader "$leader_pid" "$leader_start_time"
    if ! wsl_process_cleanup_token_groups "$token"; then
      printf 'failed to reclaim token-owned process groups after ownership check\n' >&2
      return 1
    fi
    wsl_process_clear_state_locked "$state_dir"
    return 1
  fi
}

wsl_process_start() {
  state_dir="$1"
  stdout_path="$4"
  stderr_path="$5"
  mkdir -p "$state_dir" "$(dirname "$stdout_path")" "$(dirname "$stderr_path")"
  wsl_process_lock "$state_dir" || return 1
  if wsl_process_start_locked "$@"; then
    wsl_process_unlock "$state_dir"
    return 0
  fi
  wsl_process_unlock "$state_dir"
  return 1
}

wsl_process_logs() {
  lines="$1"
  shift
  for log_path in "$@"; do
    if [ -f "$log_path" ]; then
      printf '== %s ==\n' "$log_path"
      tail -n "$lines" "$log_path"
    else
      printf '%s: no log yet\n' "$log_path"
    fi
  done
}
