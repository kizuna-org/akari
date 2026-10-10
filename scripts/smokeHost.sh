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

if AKARI_ADDR=invalid "$binary" >"$scratch/invalid.log" 2>&1; then
  echo "invalid address was accepted" >&2
  exit 1
fi

AKARI_ADDR=127.0.0.1:0 "$binary" >"$scratch/host.log" 2>&1 &
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
  echo "host did not bind within deadline" >&2
  exit 1
fi

response=$(curl --noproxy '*' --max-time 5 --fail --silent "http://$address/healthz")
[[ "$response" == '{"status":"alive","mode":"foundation"}' ]]
status=$(curl --noproxy '*' --max-time 5 --silent --output /dev/null --write-out '%{http_code}' "http://$address/readyz")
[[ "$status" == 404 ]]

if AKARI_ADDR="$address" "$binary" >"$scratch/occupied.log" 2>&1; then
  echo "occupied address was accepted" >&2
  exit 1
fi

kill -TERM "$host_pid"
wait "$host_pid"
host_pid=
echo "host smoke passed: bind failure, liveness, absent readiness, SIGTERM shutdown"
