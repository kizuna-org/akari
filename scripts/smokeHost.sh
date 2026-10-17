#!/usr/bin/env bash
set -euo pipefail
export AKARI_ADMIN_ADDR=127.0.0.1:0

binary=${1:?usage: smokeHost.sh /path/to/akari}
scratch=$(mktemp -d)
host_pid=
rejected_pid=
host_log=

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
  local auth_disabled=${1:-false}
  host_log="$scratch/host-$auth_disabled.log"
  : >"$host_log"
  AKARI_DATA_DIR="$scratch/state" AKARI_ADDR=127.0.0.1:0 \
    AKARI_ADMIN_AUTH_DISABLED="$auth_disabled" "$binary" >"$host_log" 2>&1 &
  host_pid=$!
  address=

  for ((attempt = 0; attempt < 100; attempt++)); do
    address=$(sed -n 's/.*foundation host listening.*address=\(127\.0\.0\.1:[0-9]*\).*/\1/p' "$host_log")
    if [[ -n "$address" ]]; then
      break
    fi
    if ! kill -0 "$host_pid" 2>/dev/null; then
      cat "$host_log" >&2
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
admin_address=$(sed -n 's/.*management console listening.*address=\(127\.0\.0\.1:[0-9]*\).*/\1/p' "$host_log")
[[ -n "$admin_address" ]]
unauthorized=$(curl --noproxy '*' --max-time 5 --silent --output /dev/null --write-out '%{http_code}' "http://$admin_address/api/status")
[[ "$unauthorized" == 401 ]]
admin_token=$(cat "$scratch/state/admin.token")
status_json=$(curl --noproxy '*' --max-time 5 --fail --silent -H "Authorization: Bearer $admin_token" "http://$admin_address/api/status")
[[ "$status_json" == *'"permission":"observer"'* ]]
curl --noproxy '*' --max-time 5 --fail --silent -H "Authorization: Bearer $admin_token" "http://$admin_address/api/docs" > "$scratch/docs.json"
[[ -n "$(sed -n '/docs\/06-architecture.md/p' "$scratch/docs.json")" ]]
public_admin=$(curl --noproxy '*' --max-time 5 --silent --output /dev/null --write-out '%{http_code}' "http://$address/api/status")
[[ "$public_admin" == 404 ]]

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

start_host true
[[ ! -e "$scratch/state/admin.token" ]]
admin_address=$(sed -n 's/.*management console listening.*address=\(127\.0\.0\.1:[0-9]*\).*/\1/p' "$host_log")
[[ -n "$admin_address" ]]
status_json=$(curl --noproxy '*' --max-time 5 --fail --silent "http://$admin_address/api/status")
[[ "$status_json" == *'"permission":"observer"'* ]]
curl --noproxy '*' --max-time 5 --fail --silent "http://$admin_address/assets/index.html" > "$scratch/console.html"
[[ "$(sed -n '/data-auth-disabled="true"/p' "$scratch/console.html")" ]]

kill -TERM "$host_pid"
wait "$host_pid"
host_pid=

echo "admin smoke passed: authentication disabled without a token"
