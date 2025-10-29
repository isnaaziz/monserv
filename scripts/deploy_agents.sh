#!/usr/bin/env bash
set -euo pipefail

# Helper to copy agent binary and run it remotely via SSH.
# Requires: ssh/scp; optional: sshpass (if using password auth).
# Reads hosts from SERVERS or from HOSTS array; uses SSH_PORT/SSH_USER/SSH_PASSWORD.

if [[ -f .env ]]; then
  # shellcheck disable=SC2046
  export $(grep -v '^#' .env | xargs -0 -I {} bash -c 'echo {}' 2>/dev/null || true)
fi

AGENT_BIN=${AGENT_BIN:-bin/agent}
SSH_PORT=${SSH_PORT:-22}
SSH_USER=${SSH_USER:-$USER}
SSH_PASSWORD=${SSH_PASSWORD:-}

if [[ ! -x "$AGENT_BIN" ]]; then
  echo "Agent binary not found at $AGENT_BIN; build first (go build -o bin/agent ./cmd/agent)" >&2
  exit 1
fi

# Derive host list from SERVERS env (http://IP:port,...)
HOSTS=()
if [[ -n "${SERVERS:-}" ]]; then
  IFS=',' read -ra URLS <<< "$SERVERS"
  for u in "${URLS[@]}"; do
    h=$(echo "$u" | sed -E 's#^https?://([^:/]+).*$#\1#')
    [[ -n "$h" ]] && HOSTS+=("$h")
  done
fi

if [[ ${#HOSTS[@]} -eq 0 ]]; then
  echo "No hosts derived from SERVERS; set SERVERS or edit this script." >&2
  exit 1
fi

copy_and_run() {
  local host=$1
  echo "==> $host"
  if command -v sshpass >/dev/null 2>&1 && [[ -n "$SSH_PASSWORD" ]]; then
    sshpass -p "$SSH_PASSWORD" scp -P "$SSH_PORT" "$AGENT_BIN" "$SSH_USER@$host:/tmp/agent"
    sshpass -p "$SSH_PASSWORD" ssh -p "$SSH_PORT" "$SSH_USER@$host" 'AGENT_PORT=9123 nohup /tmp/agent >/tmp/agent.log 2>&1 & disown || true'
  else
    scp -P "$SSH_PORT" "$AGENT_BIN" "$SSH_USER@$host:/tmp/agent"
    ssh -p "$SSH_PORT" "$SSH_USER@$host" 'AGENT_PORT=9123 nohup /tmp/agent >/tmp/agent.log 2>&1 & disown || true'
  fi
}

for h in "${HOSTS[@]}"; do
  copy_and_run "$h" || echo "Failed on $h" >&2
done

echo "Done. Verify agents: curl http://<host>:9123/metrics"
