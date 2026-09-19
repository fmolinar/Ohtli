#!/usr/bin/env bash
# Starts a one-off iperf3 server inside a Containerlab client container.
# Usage: run-server.sh [container-name]
set -euo pipefail

CONTAINER="${1:-clab-isp-lab-client-b}"

echo "Starting one-off iperf3 server in ${CONTAINER}..." >&2
exec docker exec "${CONTAINER}" iperf3 --server --one-off
