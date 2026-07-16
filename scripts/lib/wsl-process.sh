#!/bin/sh

wsl_state_root() {
  wsl_process_product="$1"
  printf '%s/%s/services\n' "${XDG_STATE_HOME:-${HOME}/.local/state}" "$wsl_process_product"
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
  wsl_process_pid="$1"
  wsl_process_field="$2"
  [ -r "/proc/${wsl_process_pid}/stat" ] || return 1
  IFS= read -r wsl_process_stat <"/proc/${wsl_process_pid}/stat" || return 1
  wsl_process_stat="${wsl_process_stat##*) }"
  set -- $wsl_process_stat
  case "$wsl_process_field" in
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
  IFS= read -r wsl_process_stat </proc/self/stat || return 1
  WSL_CURRENT_PROCESS_PID="${wsl_process_stat%% *}"
  wsl_process_stat="${wsl_process_stat##*) }"
  set -- $wsl_process_stat
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
  wsl_process_pid="$1"
  wsl_process_token_value="$2"
  [ -r "/proc/${wsl_process_pid}/environ" ] || return 1
  tr '\000' '\n' <"/proc/${wsl_process_pid}/environ" | grep -Fx -- "WSL_PROCESS_TOKEN=${wsl_process_token_value}" >/dev/null 2>&1
}

wsl_process_state_file() {
  printf '%s/process.state\n' "$1"
}

wsl_process_read_state() {
  wsl_process_state_dir="$1"
  wsl_process_state_file_path="$(wsl_process_state_file "$wsl_process_state_dir")"
  [ -r "$wsl_process_state_file_path" ] || return 1

  WSL_PROCESS_LEADER_PID=""
  WSL_PROCESS_LEADER_START_TIME=""
  WSL_PROCESS_GROUP_ID=""
  WSL_PROCESS_TOKEN=""
  while IFS='=' read -r wsl_process_key wsl_process_value || [ -n "$wsl_process_key" ]; do
    case "$wsl_process_key" in
      leader_pid) WSL_PROCESS_LEADER_PID="$wsl_process_value" ;;
      leader_start_time) WSL_PROCESS_LEADER_START_TIME="$wsl_process_value" ;;
      group_id) WSL_PROCESS_GROUP_ID="$wsl_process_value" ;;
      token) WSL_PROCESS_TOKEN="$wsl_process_value" ;;
    esac
  done <"$wsl_process_state_file_path"

  case "$WSL_PROCESS_LEADER_PID" in ''|*[!0-9]*) return 1 ;; esac
  case "$WSL_PROCESS_LEADER_START_TIME" in ''|*[!0-9]*) return 1 ;; esac
  case "$WSL_PROCESS_GROUP_ID" in ''|*[!0-9]*) return 1 ;; esac
  case "$WSL_PROCESS_TOKEN" in ''|*[!A-Za-z0-9._-]*) return 1 ;; esac
}

# Returns 0 when the recorded process group contains one or more processes and
# every member carries the recorded per-start token. Returns 1 when the group
# is gone and 2 when ownership cannot be proven.
wsl_process_group_is_owned() {
  wsl_process_state_dir="$1"
  wsl_process_read_state "$wsl_process_state_dir" || return 2

  wsl_process_found=0
  for wsl_process_proc_dir in /proc/[0-9]*; do
    wsl_process_pid="${wsl_process_proc_dir#/proc/}"
    [ "$(wsl_process_group_id "$wsl_process_pid" 2>/dev/null || true)" = "$WSL_PROCESS_GROUP_ID" ] || continue
    case "$(wsl_process_run_state "$wsl_process_pid" 2>/dev/null || true)" in
      Z|X) continue ;;
    esac
    if ! wsl_process_has_token "$wsl_process_pid" "$WSL_PROCESS_TOKEN"; then
      [ -d "$wsl_process_proc_dir" ] || continue
      return 2
    fi
    wsl_process_found=1
  done
  [ "$wsl_process_found" -eq 1 ]
}

wsl_process_token_group_is_owned() {
  wsl_process_group_id_value="$1"
  wsl_process_token_value="$2"
  case "$wsl_process_group_id_value" in ''|*[!0-9]*) return 2 ;; esac

  wsl_process_found=0
  for wsl_process_proc_dir in /proc/[0-9]*; do
    wsl_process_pid="${wsl_process_proc_dir#/proc/}"
    [ "$(wsl_process_group_id "$wsl_process_pid" 2>/dev/null || true)" = "$wsl_process_group_id_value" ] || continue
    case "$(wsl_process_run_state "$wsl_process_pid" 2>/dev/null || true)" in
      Z|X) continue ;;
    esac
    if ! wsl_process_has_token "$wsl_process_pid" "$wsl_process_token_value"; then
      [ -d "$wsl_process_proc_dir" ] || continue
      return 2
    fi
    wsl_process_found=1
  done
  [ "$wsl_process_found" -eq 1 ]
}

wsl_process_collect_token_groups() {
  wsl_process_token_value="$1"
  wsl_process_groups_file="$2"
  : >"$wsl_process_groups_file"
  for wsl_process_proc_dir in /proc/[0-9]*; do
    wsl_process_pid="${wsl_process_proc_dir#/proc/}"
    case "$(wsl_process_run_state "$wsl_process_pid" 2>/dev/null || true)" in
      Z|X|'') continue ;;
    esac
    wsl_process_has_token "$wsl_process_pid" "$wsl_process_token_value" || continue
    wsl_process_group_id_value="$(wsl_process_group_id "$wsl_process_pid" 2>/dev/null || true)"
    case "$wsl_process_group_id_value" in ''|*[!0-9]*) continue ;; esac
    grep -Fx -- "$wsl_process_group_id_value" "$wsl_process_groups_file" >/dev/null 2>&1 || printf '%s\n' "$wsl_process_group_id_value" >>"$wsl_process_groups_file"
  done
}

wsl_process_token_is_owned() {
  wsl_process_state_dir="$1"
  wsl_process_read_state "$wsl_process_state_dir" || return 2

  wsl_process_groups_file="$(mktemp)"
  wsl_process_collect_token_groups "$WSL_PROCESS_TOKEN" "$wsl_process_groups_file"
  wsl_process_found=0
  wsl_process_unsafe=0
  while IFS= read -r wsl_process_group_id_value; do
    [ -n "$wsl_process_group_id_value" ] || continue
    if wsl_process_token_group_is_owned "$wsl_process_group_id_value" "$WSL_PROCESS_TOKEN"; then
      wsl_process_ownership_status=0
    else
      wsl_process_ownership_status=$?
    fi
    case "$wsl_process_ownership_status" in
      0) wsl_process_found=1 ;;
      2) wsl_process_unsafe=1 ;;
    esac
  done <"$wsl_process_groups_file"
  rm -f "$wsl_process_groups_file"

  [ "$wsl_process_found" -eq 1 ] && return 0
  [ "$wsl_process_unsafe" -eq 1 ] && return 2
  return 1
}

wsl_process_token_processes_remain() {
  wsl_process_token_value="$1"
  for wsl_process_proc_dir in /proc/[0-9]*; do
    wsl_process_pid="${wsl_process_proc_dir#/proc/}"
    case "$(wsl_process_run_state "$wsl_process_pid" 2>/dev/null || true)" in
      Z|X|'') continue ;;
    esac
    wsl_process_has_token "$wsl_process_pid" "$wsl_process_token_value" && return 0
  done
  return 1
}

wsl_process_matches() {
  wsl_process_state_dir="$1"
  if wsl_process_group_is_owned "$wsl_process_state_dir"; then
    return 0
  else
    wsl_process_group_status=$?
  fi
  [ "$wsl_process_group_status" -eq 1 ] || return "$wsl_process_group_status"
  wsl_process_token_is_owned "$wsl_process_state_dir"
}

wsl_process_state() {
  wsl_process_state_dir="$1"
  if wsl_process_matches "$wsl_process_state_dir"; then
    printf 'running\n'
  elif [ -e "$(wsl_process_state_file "$wsl_process_state_dir")" ]; then
    printf 'stale\n'
  else
    printf 'stopped\n'
  fi
}

wsl_process_clear_state_locked() {
  wsl_process_state_dir="$1"
  rm -f "$(wsl_process_state_file "$wsl_process_state_dir")"
}

wsl_process_clear_state() {
  wsl_process_state_dir="$1"
  wsl_process_lock "$wsl_process_state_dir" || return 1
  wsl_process_clear_state_locked "$wsl_process_state_dir"
  wsl_process_unlock "$wsl_process_state_dir"
}

wsl_process_lock_owner_is_live() {
  wsl_process_lock_dir="$1"
  wsl_process_owner_file="${wsl_process_lock_dir}/owner.state"
  [ -r "$wsl_process_owner_file" ] || return 1
  wsl_process_owner_pid=""
  wsl_process_owner_start_time=""
  while IFS='=' read -r wsl_process_key wsl_process_value || [ -n "$wsl_process_key" ]; do
    case "$wsl_process_key" in
      pid) wsl_process_owner_pid="$wsl_process_value" ;;
      start_time) wsl_process_owner_start_time="$wsl_process_value" ;;
    esac
  done <"$wsl_process_owner_file"
  case "$wsl_process_owner_pid" in ''|*[!0-9]*) return 1 ;; esac
  case "$wsl_process_owner_start_time" in ''|*[!0-9]*) return 1 ;; esac
  [ "$(wsl_process_start_time "$wsl_process_owner_pid" 2>/dev/null || true)" = "$wsl_process_owner_start_time" ]
}

wsl_process_lock() {
  wsl_process_state_dir="$1"
  wsl_process_lock_dir="${wsl_process_state_dir}/.operation-lock"
  mkdir -p "$wsl_process_state_dir"
  wsl_process_attempts=0
  wsl_process_missing_owner_attempts=0
  while ! mkdir "$wsl_process_lock_dir" 2>/dev/null; do
    if [ -e "${wsl_process_lock_dir}/owner.state" ]; then
      wsl_process_missing_owner_attempts=0
      if ! wsl_process_lock_owner_is_live "$wsl_process_lock_dir"; then
        rm -f "${wsl_process_lock_dir}/owner.state"
        rmdir "$wsl_process_lock_dir" 2>/dev/null || true
      fi
    else
      wsl_process_missing_owner_attempts=$((wsl_process_missing_owner_attempts + 1))
      if [ "$wsl_process_missing_owner_attempts" -ge 10 ]; then
        rmdir "$wsl_process_lock_dir" 2>/dev/null || true
      fi
    fi
    wsl_process_attempts=$((wsl_process_attempts + 1))
    if [ "$wsl_process_attempts" -ge 600 ]; then
      printf 'timed out waiting for service operation lock: %s\n' "$wsl_process_state_dir" >&2
      return 1
    fi
    sleep 0.05
  done

  if ! wsl_current_process_identity; then
    rmdir "$wsl_process_lock_dir" 2>/dev/null || true
    return 1
  fi
  (
    umask 077
    printf 'pid=%s\nstart_time=%s\n' "$WSL_CURRENT_PROCESS_PID" "$WSL_CURRENT_PROCESS_START_TIME" >"${wsl_process_lock_dir}/owner.state.new"
    mv -f "${wsl_process_lock_dir}/owner.state.new" "${wsl_process_lock_dir}/owner.state"
  )
}

wsl_process_unlock() {
  wsl_process_state_dir="$1"
  wsl_process_lock_dir="${wsl_process_state_dir}/.operation-lock"
  rm -f "${wsl_process_lock_dir}/owner.state"
  rmdir "$wsl_process_lock_dir" 2>/dev/null || true
}

wsl_process_write_state_locked() {
  wsl_process_state_dir="$1"
  wsl_process_leader_pid="$2"
  wsl_process_leader_start_time="$3"
  wsl_process_group_id_value="$4"
  wsl_process_token_value="$5"
  wsl_process_state_file_path="$(wsl_process_state_file "$wsl_process_state_dir")"
  wsl_process_temp_file="${wsl_process_state_file_path}.new.$$"
  (
    umask 077
    printf 'leader_pid=%s\nleader_start_time=%s\ngroup_id=%s\ntoken=%s\n' \
      "$wsl_process_leader_pid" "$wsl_process_leader_start_time" "$wsl_process_group_id_value" "$wsl_process_token_value" >"$wsl_process_temp_file"
    mv -f "$wsl_process_temp_file" "$wsl_process_state_file_path"
  )
}

# Sends a signal only after ownership of every process in the group has been
# rechecked. This avoids signalling a reused PID or process-group identifier.
wsl_process_signal_token_group() {
  wsl_process_group_id_value="$1"
  wsl_process_token_value="$2"
  wsl_process_signal_name="$3"
  wsl_process_token_group_is_owned "$wsl_process_group_id_value" "$wsl_process_token_value"
  wsl_process_ownership_status=$?
  if [ "$wsl_process_ownership_status" -ne 0 ]; then
    return "$wsl_process_ownership_status"
  fi
  /bin/kill "-${wsl_process_signal_name}" -- "-${wsl_process_group_id_value}" >/dev/null 2>&1 || true
}

wsl_process_stop_token_group() {
  wsl_process_group_id_value="$1"
  wsl_process_token_value="$2"
  wsl_process_token_group_is_owned "$wsl_process_group_id_value" "$wsl_process_token_value"
  wsl_process_ownership_status=$?
  case "$wsl_process_ownership_status" in
    1) return 0 ;;
    2) return 1 ;;
  esac

  wsl_process_signal_token_group "$wsl_process_group_id_value" "$wsl_process_token_value" TERM || return 1
  wsl_process_attempts=0
  while wsl_process_token_group_is_owned "$wsl_process_group_id_value" "$wsl_process_token_value" && [ "$wsl_process_attempts" -lt 50 ]; do
    sleep 0.1
    wsl_process_attempts=$((wsl_process_attempts + 1))
  done
  if wsl_process_token_group_is_owned "$wsl_process_group_id_value" "$wsl_process_token_value"; then
    if ! wsl_process_signal_token_group "$wsl_process_group_id_value" "$wsl_process_token_value" KILL; then
      return 1
    fi
    wsl_process_attempts=0
    while wsl_process_token_group_is_owned "$wsl_process_group_id_value" "$wsl_process_token_value" && [ "$wsl_process_attempts" -lt 20 ]; do
      sleep 0.1
      wsl_process_attempts=$((wsl_process_attempts + 1))
    done
  fi
  if wsl_process_token_group_is_owned "$wsl_process_group_id_value" "$wsl_process_token_value"; then
    printf 'failed to stop owned process group %s\n' "$wsl_process_group_id_value" >&2
    return 1
  fi
}

wsl_process_stop_locked() {
  wsl_process_state_dir="$1"
  if ! wsl_process_read_state "$wsl_process_state_dir"; then
    wsl_process_clear_state_locked "$wsl_process_state_dir"
    return 0
  fi

  wsl_process_token_value="$WSL_PROCESS_TOKEN"
  wsl_process_stop_failed=0
  wsl_process_passes=0
  while wsl_process_token_processes_remain "$wsl_process_token_value" && [ "$wsl_process_passes" -lt 5 ]; do
    wsl_process_groups_file="$(mktemp)"
    wsl_process_collect_token_groups "$wsl_process_token_value" "$wsl_process_groups_file"
    while IFS= read -r wsl_process_group_id_value; do
      [ -n "$wsl_process_group_id_value" ] || continue
      wsl_process_stop_token_group "$wsl_process_group_id_value" "$wsl_process_token_value" || wsl_process_stop_failed=1
    done <"$wsl_process_groups_file"
    rm -f "$wsl_process_groups_file"
    wsl_process_passes=$((wsl_process_passes + 1))
  done

  if wsl_process_token_processes_remain "$wsl_process_token_value"; then
    printf 'failed to stop all token-owned process groups\n' >&2
    return 1
  fi
  if [ "$wsl_process_stop_failed" -ne 0 ]; then
    printf 'failed to verify token-owned process groups during stop\n' >&2
    return 1
  fi
  wsl_process_clear_state_locked "$wsl_process_state_dir"
}

wsl_process_stop() {
  wsl_process_state_dir="$1"
  wsl_process_lock "$wsl_process_state_dir" || return 1
  if wsl_process_stop_locked "$wsl_process_state_dir"; then
    wsl_process_unlock "$wsl_process_state_dir"
    return 0
  fi
  wsl_process_unlock "$wsl_process_state_dir"
  return 1
}

# Before state is published, only a living leader with its recorded start time
# may be signalled without a token. Never signal a numeric group from this
# path: its original leader may already be gone and the identifier reused.
wsl_process_started_leader_matches() {
  wsl_process_leader_pid="$1"
  wsl_process_leader_start_time="$2"
  case "$wsl_process_leader_pid" in ''|*[!0-9]*) return 2 ;; esac
  case "$wsl_process_leader_start_time" in ''|*[!0-9]*) return 2 ;; esac
  case "$(wsl_process_run_state "$wsl_process_leader_pid" 2>/dev/null || true)" in
    Z|X|'') return 1 ;;
  esac
  [ "$(wsl_process_start_time "$wsl_process_leader_pid" 2>/dev/null || true)" = "$wsl_process_leader_start_time" ]
}

wsl_process_cleanup_started_leader() {
  wsl_process_leader_pid="$1"
  wsl_process_leader_start_time="$2"
  if ! wsl_process_started_leader_matches "$wsl_process_leader_pid" "$wsl_process_leader_start_time"; then
    return 0
  fi
  /bin/kill -TERM "$wsl_process_leader_pid" >/dev/null 2>&1 || true
  wsl_process_attempts=0
  while wsl_process_started_leader_matches "$wsl_process_leader_pid" "$wsl_process_leader_start_time" && [ "$wsl_process_attempts" -lt 50 ]; do
    sleep 0.1
    wsl_process_attempts=$((wsl_process_attempts + 1))
  done
  if wsl_process_started_leader_matches "$wsl_process_leader_pid" "$wsl_process_leader_start_time"; then
    /bin/kill -KILL "$wsl_process_leader_pid" >/dev/null 2>&1 || true
  fi
}

wsl_process_cleanup_token_group() {
  wsl_process_group_id_value="$1"
  wsl_process_token_value="$2"
  wsl_process_stop_token_group "$wsl_process_group_id_value" "$wsl_process_token_value" || true
}

wsl_process_cleanup_token_groups() {
  wsl_process_token_value="$1"
  wsl_process_passes=0
  while wsl_process_token_processes_remain "$wsl_process_token_value" && [ "$wsl_process_passes" -lt 5 ]; do
    wsl_process_groups_file="$(mktemp)"
    wsl_process_collect_token_groups "$wsl_process_token_value" "$wsl_process_groups_file"
    while IFS= read -r wsl_process_group_id_value; do
      [ -n "$wsl_process_group_id_value" ] || continue
      wsl_process_cleanup_token_group "$wsl_process_group_id_value" "$wsl_process_token_value"
    done <"$wsl_process_groups_file"
    rm -f "$wsl_process_groups_file"
    wsl_process_passes=$((wsl_process_passes + 1))
  done
  ! wsl_process_token_processes_remain "$wsl_process_token_value"
}

wsl_process_start_locked() {
  wsl_process_state_dir="$1"
  wsl_process_command_marker="$2"
  wsl_process_work_dir="$3"
  wsl_process_stdout_path="$4"
  wsl_process_stderr_path="$5"
  shift 5

  wsl_process_stop_locked "$wsl_process_state_dir" || return 1
  wsl_process_token_file="$(mktemp "${wsl_process_state_dir}/.process-token.XXXXXX")"
  wsl_process_token_value="$(basename "$wsl_process_token_file")"
  rm -f "$wsl_process_token_file"

  (
    cd "$wsl_process_work_dir"
    exec nohup setsid env "WSL_PROCESS_TOKEN=${wsl_process_token_value}" "$@" >"$wsl_process_stdout_path" 2>"$wsl_process_stderr_path" </dev/null
  ) &
  wsl_process_leader_pid=$!
  wsl_process_attempts=0
  wsl_process_leader_start_time=""
  wsl_process_group_id_value=""
  while [ "$wsl_process_attempts" -lt 20 ]; do
    wsl_process_leader_start_time="$(wsl_process_start_time "$wsl_process_leader_pid" 2>/dev/null || true)"
    wsl_process_group_id_value="$(wsl_process_group_id "$wsl_process_leader_pid" 2>/dev/null || true)"
    wsl_process_session_id_value="$(wsl_process_session_id "$wsl_process_leader_pid" 2>/dev/null || true)"
    if [ -n "$wsl_process_leader_start_time" ] && [ -n "$wsl_process_group_id_value" ] && [ "$wsl_process_group_id_value" = "$wsl_process_session_id_value" ] && \
      wsl_process_has_token "$wsl_process_leader_pid" "$wsl_process_token_value"; then
      break
    fi
    sleep 0.05
    wsl_process_attempts=$((wsl_process_attempts + 1))
  done
  if [ -z "$wsl_process_leader_start_time" ] || [ -z "$wsl_process_group_id_value" ] || \
    [ "$(wsl_process_session_id "$wsl_process_leader_pid" 2>/dev/null || true)" != "$wsl_process_group_id_value" ] || \
    ! wsl_process_has_token "$wsl_process_leader_pid" "$wsl_process_token_value"; then
    wsl_process_cleanup_started_leader "$wsl_process_leader_pid" "$wsl_process_leader_start_time"
    if ! wsl_process_cleanup_token_groups "$wsl_process_token_value"; then
      printf 'failed to reclaim token-owned process groups after start failure\n' >&2
    fi
    wait "$wsl_process_leader_pid" 2>/dev/null || true
    return 1
  fi

  wsl_process_write_state_locked "$wsl_process_state_dir" "$wsl_process_leader_pid" "$wsl_process_leader_start_time" "$wsl_process_group_id_value" "$wsl_process_token_value"
  if ! wsl_process_matches "$wsl_process_state_dir"; then
    wsl_process_cleanup_started_leader "$wsl_process_leader_pid" "$wsl_process_leader_start_time"
    if ! wsl_process_cleanup_token_groups "$wsl_process_token_value"; then
      printf 'failed to reclaim token-owned process groups after ownership check\n' >&2
      return 1
    fi
    wsl_process_clear_state_locked "$wsl_process_state_dir"
    return 1
  fi
}

wsl_process_start() {
  wsl_process_state_dir="$1"
  wsl_process_stdout_path="$4"
  wsl_process_stderr_path="$5"
  mkdir -p "$wsl_process_state_dir" "$(dirname "$wsl_process_stdout_path")" "$(dirname "$wsl_process_stderr_path")"
  wsl_process_lock "$wsl_process_state_dir" || return 1
  if wsl_process_start_locked "$@"; then
    wsl_process_unlock "$wsl_process_state_dir"
    return 0
  fi
  wsl_process_unlock "$wsl_process_state_dir"
  return 1
}

wsl_process_logs() {
  wsl_process_lines="$1"
  shift
  for wsl_process_log_path in "$@"; do
    if [ -f "$wsl_process_log_path" ]; then
      printf '== %s ==\n' "$wsl_process_log_path"
      tail -n "$wsl_process_lines" "$wsl_process_log_path"
    else
      printf '%s: no log yet\n' "$wsl_process_log_path"
    fi
  done
}
