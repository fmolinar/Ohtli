#!/usr/bin/env bash
# Orchestrates the README.md section 9 MVP baseline test end to end:
# starts a one-off iperf3 server on client-b in the background, runs the
# client from client-a, and waits for the server to exit.
#
# Usage: run-baseline.sh [-p udp|tcp] [-t seconds] [-b bitrate]
#
# Intended for CI (README.md section 10's `traffic-baseline` job) and for
# manual smoke tests against a running `containerlab deploy` topology.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVER_CONTAINER="clab-isp-lab-client-b"
CLIENT_CONTAINER="clab-isp-lab-client-a"
SERVER_IP="198.51.100.2"

"${SCRIPT_DIR}/run-server.sh" "${SERVER_CONTAINER}" &
SERVER_PID=$!

# Give the server a moment to bind before the client connects.
sleep 1

"${SCRIPT_DIR}/run-client.sh" -c "${CLIENT_CONTAINER}" -s "${SERVER_IP}" "$@"

wait "${SERVER_PID}"
