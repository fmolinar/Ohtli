#!/usr/bin/env bash
# Runs an iperf3 client inside a Containerlab client container against a
# server IP, saving --json output for later inspection (e.g. as a Github
# Actions artifact -- README.md section 9/10).
#
# Usage:
#   run-client.sh -c <client-container> -s <server-ip> [-p udp|tcp]
#                 [-t seconds] [-b bitrate] [-o output-dir]
#
# Defaults match the README.md section 9 MVP example: a 60s UDP flow at
# 100 Mbit/s from client-a to client-b (198.51.100.2).
set -euo pipefail

CLIENT_CONTAINER="clab-isp-lab-client-a"
SERVER_IP="198.51.100.2"
PROTO="udp"
DURATION="60"
BITRATE="100M"
OUTPUT_DIR="$(dirname "${BASH_SOURCE[0]}")/results"

usage() {
  grep '^#' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
  exit 1
}

while getopts "c:s:p:t:b:o:h" opt; do
  case "${opt}" in
    c) CLIENT_CONTAINER="${OPTARG}" ;;
    s) SERVER_IP="${OPTARG}" ;;
    p) PROTO="${OPTARG}" ;;
    t) DURATION="${OPTARG}" ;;
    b) BITRATE="${OPTARG}" ;;
    o) OUTPUT_DIR="${OPTARG}" ;;
    h | *) usage ;;
  esac
done

if [[ "${PROTO}" != "udp" && "${PROTO}" != "tcp" ]]; then
  echo "error: -p must be 'udp' or 'tcp'" >&2
  exit 1
fi

mkdir -p "${OUTPUT_DIR}"
TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
OUTPUT_FILE="${OUTPUT_DIR}/${PROTO}-${TIMESTAMP}.json"

PROTO_ARGS=()
if [[ "${PROTO}" == "udp" ]]; then
  PROTO_ARGS=(--udp --bitrate "${BITRATE}")
fi

echo "Running ${PROTO} test: ${CLIENT_CONTAINER} -> ${SERVER_IP} for ${DURATION}s" >&2
docker exec "${CLIENT_CONTAINER}" iperf3 \
  --client "${SERVER_IP}" \
  "${PROTO_ARGS[@]}" \
  --time "${DURATION}" \
  --interval 1 \
  --json >"${OUTPUT_FILE}"

echo "Result saved to ${OUTPUT_FILE}" >&2

if [[ "${PROTO}" == "udp" ]]; then
  python3 - "${OUTPUT_FILE}" <<'PYEOF'
import json
import sys

with open(sys.argv[1]) as f:
    data = json.load(f)

summary = data["end"]["sum"]
loss_pct = summary.get("lost_percent", 0.0)
print(f"UDP loss: {loss_pct:.3f}%", file=sys.stderr)
if loss_pct > 1.0:
    print("::warning::UDP packet loss exceeds 1%", file=sys.stderr)
PYEOF
fi
