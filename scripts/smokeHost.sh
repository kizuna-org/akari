#!/usr/bin/env bash
set -euo pipefail

binary=${1:?usage: smokeHost.sh /path/to/akari}
scratch=$(mktemp -d)
host_pid=

cleanup() {
  if [[ -n "$host_pid" ]]; then
    kill -TERM "$host_pid" 2>/dev/null || true
    wait "$host_pid" 2>/dev/null || true
  fi
  rm -rf "$scratch"
}
trap cleanup EXIT

if AKARI_DATA_DIR="$scratch/state" AKARI_ADDR=invalid "$binary" >"$scratch/invalid.log" 2>&1; then
  echo "invalid address was accepted" >&2
  exit 1
fi

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

if AKARI_DATA_DIR="$scratch/state" AKARI_ADDR="$address" "$binary" >"$scratch/occupied.log" 2>&1; then
  echo "occupied address was accepted" >&2
  exit 1
fi

if AKARI_DATA_DIR="$scratch/state" AKARI_ADDR=127.0.0.1:0 "$binary" >"$scratch/lease.log" 2>&1; then
  echo "second host acquired an active installation" >&2
  exit 1
fi

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
if AKARI_DATA_DIR="$scratch/corrupt" AKARI_ADDR=127.0.0.1:0 "$binary" >"$scratch/corrupt.log" 2>&1; then
  echo "corrupt saved state was accepted" >&2
  exit 1
fi
[[ -z "$(sed -n '/foundation host listening/p' "$scratch/corrupt.log")" ]]

echo "host smoke passed: restore, exclusive ownership, SIGTERM save, SIGKILL restart, corrupt-state rejection, liveness"
