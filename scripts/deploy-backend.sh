#!/usr/bin/env bash
set -euo pipefail

export PATH=/usr/local/go/bin:$PATH

# Assumes the repo is already updated on the VM.
cd /home/dvlpr_prgmr/projects/auto-clip/backend
/usr/local/go/bin/go build -o autoclip-server .

SYSTEMCTL_BIN="$(command -v systemctl)"
if [[ -z "$SYSTEMCTL_BIN" ]]; then
  echo "systemctl not found in PATH"
  exit 1
fi

sudo -n "$SYSTEMCTL_BIN" restart auto-clip
sudo -n "$SYSTEMCTL_BIN" status auto-clip --no-pager
