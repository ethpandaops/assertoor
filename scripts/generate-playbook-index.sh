#!/usr/bin/env bash
#
# Generate the playbook index file used by the assertoor UI's remote
# library tab.
#
# Usage:
#   ./scripts/generate-playbook-index.sh [PLAYBOOKS_DIR]
#
# PLAYBOOKS_DIR defaults to ./playbooks (relative to the repo root).

set -euo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
REPO_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

PLAYBOOKS_DIR="${1:-$REPO_ROOT/playbooks}"

cd "$REPO_ROOT"
go run ./scripts/generate-playbook-index "$PLAYBOOKS_DIR"
