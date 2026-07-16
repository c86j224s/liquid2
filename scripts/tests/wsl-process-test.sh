#!/bin/sh
set -eu

root_dir="$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)"
. "${root_dir}/scripts/lib/wsl-process.sh"

# These pure marker checks run on every platform. The lifecycle cases below
# need WSL2 /proc semantics, setsid, and Linux process-group signalling.
assert_success_trap_precedes_unlock() {
  awk '
    {
      line = $0
      sub(/^[[:space:]]*/, "", line)
      if (previous == "trap abort_operation HUP INT TERM" && line == "finish_operation") {
        found = 1
      }
      previous = line
    }
    END { exit found ? 0 : 1 }
  ' "$1" || {
    printf 'success trap must precede operation unlock: %s\n' "$1" >&2
    exit 1
  }
}

assert_success_trap_precedes_unlock "${root_dir}/liquid2/scripts/browser-wsl.sh"
assert_success_trap_precedes_unlock "${root_dir}/plasma/scripts/browser-wsl.sh"
wsl_is_wsl2_release '6.6.87.2-microsoft-standard-WSL2'
! wsl_is_wsl2_release '4.4.0-19041-Microsoft'
! wsl_is_wsl2_release '6.8.0-generic'
if ! wsl_is_wsl2; then
  printf 'wsl process helper: skipped (requires WSL2)\n'
  exit 0
fi

tmp_dir="$(mktemp -d)"
dev_state_dir="${tmp_dir}/dev"
release_state_dir="${tmp_dir}/release"
orphan_state_dir="${tmp_dir}/orphan"
tokenless_state_dir="${tmp_dir}/tokenless"
detached_state_dir="${tmp_dir}/detached"
operation_state_dir="${tmp_dir}/operation"
lock_state_dir="${tmp_dir}/lock"
lock_holder=""
old_operation_job=""
replacement_operation_job=""

cleanup() {
  if [ -n "$lock_holder" ]; then
    kill -TERM "$lock_holder" >/dev/null 2>&1 || true
    wait "$lock_holder" 2>/dev/null || true
  fi
  if [ -n "$old_operation_job" ]; then
    kill -TERM "$old_operation_job" >/dev/null 2>&1 || true
    wait "$old_operation_job" 2>/dev/null || true
  fi
  if [ -n "$replacement_operation_job" ]; then
    kill -TERM "$replacement_operation_job" >/dev/null 2>&1 || true
    wait "$replacement_operation_job" 2>/dev/null || true
  fi
  wsl_process_unlock "$lock_state_dir" || true
  wsl_process_stop "$dev_state_dir" || true
  wsl_process_stop "$release_state_dir" || true
  wsl_process_stop "$orphan_state_dir" || true
  wsl_process_stop "$tokenless_state_dir" || true
  wsl_process_stop "$detached_state_dir" || true
  wsl_process_unlock "$operation_state_dir" || true
  rm -rf "$tmp_dir"
}
trap cleanup EXIT HUP INT TERM

assert_state() {
  state_dir="$1"
  expected="$2"
  actual="$(wsl_process_state "$state_dir")"
  [ "$actual" = "$expected" ] || {
    printf 'expected %s state, got %s\n' "$expected" "$actual" >&2
    exit 1
  }
}

assert_no_token_processes() {
  token="$1"
  for process_dir in /proc/[0-9]*; do
    pid="${process_dir#/proc/}"
    wsl_process_has_token "$pid" "$token" && {
      printf 'token process still alive: %s\n' "$pid" >&2
      exit 1
    }
  done
  return 0
}

# A background POSIX shell keeps the parent's $$, so the operation lock must
# record the actual /proc/self PID. Once that child dies, another operation can
# prove the owner stale and reclaim the lock.
lock_ready_file="${tmp_dir}/lock-ready"
(
  wsl_process_lock "$lock_state_dir"
  wsl_current_process_identity
  printf '%s\n' "$WSL_CURRENT_PROCESS_PID" >"${tmp_dir}/lock-child.pid"
  : >"$lock_ready_file"
  while :; do sleep 1; done
) &
lock_holder="$!"
attempts=0
while [ ! -f "$lock_ready_file" ] && [ "$attempts" -lt 40 ]; do
  sleep 0.05
  attempts=$((attempts + 1))
done
[ -f "$lock_ready_file" ]
owner_pid=""
while IFS='=' read -r key value || [ -n "$key" ]; do
  [ "$key" != pid ] || owner_pid="$value"
done <"${lock_state_dir}/.operation-lock/owner.state"
[ "$owner_pid" = "$lock_holder" ]
[ "$owner_pid" = "$(cat "${tmp_dir}/lock-child.pid")" ]
kill -TERM "$lock_holder"
wait "$lock_holder" 2>/dev/null || true
lock_holder=""
! wsl_process_lock_owner_is_live "${lock_state_dir}/.operation-lock"
wsl_process_lock "$lock_state_dir"
wsl_process_unlock "$lock_state_dir"

# Normal start and stop establish and clear an owned state atomically.
wsl_process_start "$dev_state_dir" 'sleep 30' "$tmp_dir" \
  "${tmp_dir}/dev.out" "${tmp_dir}/dev.err" sleep 30
assert_state "$dev_state_dir" running
wsl_process_stop "$dev_state_dir"
assert_state "$dev_state_dir" stopped

# A product operation lock keeps an older caller's cleanup ahead of a later
# replacement. The second caller cannot start until the first cleanup releases
# the shared product/runtime lock.
wsl_process_lock "$operation_state_dir"
wsl_process_start "$dev_state_dir" 'first operation service' "$tmp_dir" \
  "${tmp_dir}/dev.out" "${tmp_dir}/dev.err" sleep 30
second_started_file="${tmp_dir}/second-operation-started"
(
  . "${root_dir}/scripts/lib/wsl-process.sh"
  wsl_process_lock "$operation_state_dir"
  : >"$second_started_file"
  wsl_process_start "$dev_state_dir" 'second operation service' "$tmp_dir" \
    "${tmp_dir}/dev.out" "${tmp_dir}/dev.err" sleep 30
  wsl_process_unlock "$operation_state_dir"
) &
second_operation_pid="$!"
sleep 0.2
[ ! -e "$second_started_file" ]
wsl_process_stop "$dev_state_dir"
wsl_process_unlock "$operation_state_dir"
wait "$second_operation_pid"
assert_state "$dev_state_dir" running
wsl_process_stop "$dev_state_dir"
assert_state "$dev_state_dir" stopped

# Once a successful caller changes to the generic abort trap and releases the
# operation lock, a delayed signal to that caller must not stop the replacement
# service started by the next caller.
old_ready_file="${tmp_dir}/old-operation-ready"
replacement_started_file="${tmp_dir}/replacement-started"
wsl_process_start "$dev_state_dir" 'old successful service' "$tmp_dir" \
  "${tmp_dir}/dev.out" "${tmp_dir}/dev.err" sleep 30
(
  . "${root_dir}/scripts/lib/wsl-process.sh"
  operation_locked=0
  finish_operation() {
    [ "$operation_locked" -eq 0 ] || wsl_process_unlock "$operation_state_dir"
    operation_locked=0
  }
  abort_operation() {
    finish_operation
    exit 130
  }
  cancel_start() {
    wsl_process_stop "$dev_state_dir"
    finish_operation
    exit 130
  }
  trap finish_operation EXIT
  wsl_process_lock "$operation_state_dir"
  operation_locked=1
  wsl_current_process_identity
  printf '%s\n' "$WSL_CURRENT_PROCESS_PID" >"${tmp_dir}/old-operation.pid"
  trap cancel_start HUP INT TERM
  : >"$old_ready_file"
  trap abort_operation HUP INT TERM
  finish_operation
  while :; do sleep 1; done
) &
old_operation_job="$!"
attempts=0
while [ ! -f "$old_ready_file" ] && [ "$attempts" -lt 40 ]; do
  sleep 0.05
  attempts=$((attempts + 1))
done
[ -f "$old_ready_file" ]
(
  . "${root_dir}/scripts/lib/wsl-process.sh"
  wsl_process_lock "$operation_state_dir"
  wsl_process_start "$dev_state_dir" 'replacement after successful unlock' "$tmp_dir" \
    "${tmp_dir}/dev.out" "${tmp_dir}/dev.err" sleep 30
  : >"$replacement_started_file"
  kill -TERM "$(cat "${tmp_dir}/old-operation.pid")"
  wsl_process_unlock "$operation_state_dir"
) &
replacement_operation_job="$!"
wait "$replacement_operation_job"
replacement_operation_job=""
[ -f "$replacement_started_file" ]
wait "$old_operation_job" 2>/dev/null || true
old_operation_job=""
assert_state "$dev_state_dir" running
wsl_process_stop "$dev_state_dir"
assert_state "$dev_state_dir" stopped

# Parallel starts for the same service serialize at the service lock and leave
# one owned process group that can still be stopped.
for number in 1 2 3 4 5 6; do
  (
    . "${root_dir}/scripts/lib/wsl-process.sh"
    wsl_process_start "$dev_state_dir" "sleep 30 #${number}" "$tmp_dir" \
      "${tmp_dir}/dev.out" "${tmp_dir}/dev.err" sleep 30
  ) &
done
wait
assert_state "$dev_state_dir" running
wsl_process_stop "$dev_state_dir"
assert_state "$dev_state_dir" stopped

# A process-group leader may exit before its child. The child keeps the unique
# token, so stop recovers it instead of discarding state as stale.
wsl_process_start "$orphan_state_dir" 'leader exits after child starts' "$tmp_dir" \
  "${tmp_dir}/orphan.out" "${tmp_dir}/orphan.err" \
  sh -c 'sleep 30 & sleep 0.2'
sleep 0.5
assert_state "$orphan_state_dir" running
wsl_process_read_state "$orphan_state_dir"
orphan_token="$WSL_PROCESS_TOKEN"
wsl_process_stop "$orphan_state_dir"
assert_state "$orphan_state_dir" stopped
assert_no_token_processes "$orphan_token"

# A token child may daemonize into a new session and process group. Restart and
# stop must enumerate every token-owned group rather than only the recorded
# leader group.
wsl_process_start "$detached_state_dir" 'setsid token child' "$tmp_dir" \
  "${tmp_dir}/detached.out" "${tmp_dir}/detached.err" \
  sh -c 'setsid sleep 30 & sleep 0.2'
sleep 0.5
wsl_process_read_state "$detached_state_dir"
detached_token="$WSL_PROCESS_TOKEN"
wsl_process_start "$detached_state_dir" 'replacement after setsid token child' "$tmp_dir" \
  "${tmp_dir}/detached.out" "${tmp_dir}/detached.err" sleep 30
assert_state "$detached_state_dir" running
assert_no_token_processes "$detached_token"
wsl_process_stop "$detached_state_dir"
assert_state "$detached_state_dir" stopped

# If ownership confirmation fails after the target clears its inherited token,
# the newly created session is still reclaimed by its PID start-time and
# session/group identity rather than left running without state.
tokenless_pid_file="${tmp_dir}/tokenless.pid"
if wsl_process_start "$tokenless_state_dir" 'clears inherited token' "$tmp_dir" \
  "${tmp_dir}/tokenless.out" "${tmp_dir}/tokenless.err" \
  sh -c 'echo "$$" >"$1"; unset WSL_PROCESS_TOKEN; exec sleep 30' sh "$tokenless_pid_file"; then
  printf 'tokenless process unexpectedly passed ownership confirmation\n' >&2
  exit 1
fi
tokenless_pid="$(cat "$tokenless_pid_file")"
! kill -0 "$tokenless_pid" >/dev/null 2>&1
assert_state "$tokenless_state_dir" stopped

# A state file that points at an unrelated process group lacks its token. It
# is cleared without signalling the caller's own process group.
mkdir -p "$dev_state_dir"
caller_group="$(wsl_process_group_id "$$")"
caller_start_time="$(wsl_process_start_time "$$")"
printf 'leader_pid=%s\nleader_start_time=%s\ngroup_id=%s\ntoken=%s\n' \
  "$$" "$caller_start_time" "$caller_group" 'not-our-process-group' >"${dev_state_dir}/process.state"
wsl_process_stop "$dev_state_dir"
kill -0 "$$"
assert_state "$dev_state_dir" stopped

# Development and release state roots remain independent: stopping one service
# cannot signal the other service group.
wsl_process_start "$dev_state_dir" 'sleep 30 dev' "$tmp_dir" \
  "${tmp_dir}/dev.out" "${tmp_dir}/dev.err" sleep 30
wsl_process_start "$release_state_dir" 'sleep 30 release' "$tmp_dir" \
  "${tmp_dir}/release.out" "${tmp_dir}/release.err" sleep 30
wsl_process_stop "$dev_state_dir"
assert_state "$dev_state_dir" stopped
assert_state "$release_state_dir" running
wsl_process_stop "$release_state_dir"
assert_state "$release_state_dir" stopped

printf 'wsl process helper: ok\n'
