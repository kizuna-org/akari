#!/usr/bin/env bash
set -euo pipefail

binary=${1:?usage: smokeHost.sh /path/to/akari}
scratch=$(mktemp -d)
host_pid=
rejected_pid=

cleanup() {
  if [[ -n "$rejected_pid" ]]; then
    kill -KILL "$rejected_pid" 2>/dev/null || true
    wait "$rejected_pid" 2>/dev/null || true
  fi
  if [[ -n "$host_pid" ]]; then
    kill -TERM "$host_pid" 2>/dev/null || true
    wait "$host_pid" 2>/dev/null || true
  fi
  rm -rf "$scratch"
}
trap cleanup EXIT

reject_within_deadline() {
  local log=$1
  local description=$2
  shift 2
  "$@" >"$log" 2>&1 &
  rejected_pid=$!
  for ((attempt = 0; attempt < 100; attempt++)); do
    if ! kill -0 "$rejected_pid" 2>/dev/null; then
      if wait "$rejected_pid"; then
        rejected_pid=
        echo "$description was accepted" >&2
        exit 1
      fi
      rejected_pid=
      return
    fi
    sleep 0.05
  done
  kill -KILL "$rejected_pid" 2>/dev/null || true
  wait "$rejected_pid" 2>/dev/null || true
  rejected_pid=
  echo "$description did not fail within deadline" >&2
  exit 1
}

reject_within_deadline "$scratch/invalid.log" "invalid address" env AKARI_DATA_DIR="$scratch/state" AKARI_ADDR=invalid "$binary"

start_host() {
  AKARI_DATA_DIR="$scratch/state" AKARI_ADDR=127.0.0.1:0 "$binary" >"$scratch/host.log" 2>&1 &
  host_pid=$!
  address=

  for ((attempt = 0; attempt < 100; attempt++)); do
    address=$(sed -n 's/.*address=\(127\.0\.0\.1:[0-9]*\).*/\1/p' "$scratch/host.log")
    if [[ -n "$address" ]]; then
      break
    fi
    if ! kill -0 "$host_pid" 2>/dev/null; then
      cat "$scratch/host.log" >&2
      exit 1
    fi
    sleep 0.05
  done

  if [[ -z "$address" ]]; then
    echo "host did not restore and bind within deadline" >&2
    exit 1
  fi
}

saved_sequence() {
  sed -n 's/.*"sequence":\([0-9]*\).*/\1/p' "$scratch/state/state.json"
}

start_host
initial_sequence=$(saved_sequence)
[[ "$initial_sequence" == 1 ]]

response=$(curl --noproxy '*' --max-time 5 --fail --silent "http://$address/healthz")
[[ "$response" == '{"status":"alive","mode":"foundation"}' ]]
status=$(curl --noproxy '*' --max-time 5 --silent --output /dev/null --write-out '%{http_code}' "http://$address/readyz")
[[ "$status" == 404 ]]

reject_within_deadline "$scratch/occupied.log" "occupied address" env AKARI_DATA_DIR="$scratch/state" AKARI_ADDR="$address" "$binary"

reject_within_deadline "$scratch/lease.log" "second host with active installation" env AKARI_DATA_DIR="$scratch/state" AKARI_ADDR=127.0.0.1:0 "$binary"

kill -TERM "$host_pid"
wait "$host_pid"
host_pid=
[[ "$(saved_sequence)" == $((initial_sequence + 1)) ]]

start_host
before_crash=$(saved_sequence)
kill -KILL "$host_pid"
wait "$host_pid" 2>/dev/null || true
host_pid=
start_host
[[ "$(saved_sequence)" == "$before_crash" ]]
kill -TERM "$host_pid"
wait "$host_pid"
host_pid=
[[ "$(saved_sequence)" == $((before_crash + 1)) ]]

cp -R "$scratch/state" "$scratch/corrupt"
printf '{' >"$scratch/corrupt/state.json"
reject_within_deadline "$scratch/corrupt.log" "corrupt saved state" env AKARI_DATA_DIR="$scratch/corrupt" AKARI_ADDR=127.0.0.1:0 "$binary"
[[ -z "$(sed -n '/foundation host listening/p' "$scratch/corrupt.log")" ]]

echo "host smoke passed: restore, exclusive ownership, SIGTERM save, SIGKILL restart, corrupt-state rejection, liveness"
