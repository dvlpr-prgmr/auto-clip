#!/usr/bin/env bash
set -euo pipefail

export PATH=/usr/local/go/bin:$PATH

# Assumes the repo is already updated on the VM.
cd /home/dvlpr_prgmr/projects/auto-clip/backend
/usr/local/go/bin/go build -o autoclip-server .

sudo systemctl restart auto-clip
sudo systemctl status auto-clip --no-pager
