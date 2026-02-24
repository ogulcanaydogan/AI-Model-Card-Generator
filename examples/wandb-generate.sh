#!/usr/bin/env bash
set -euo pipefail

# Required for live W&B extraction
: "${WANDB_API_KEY:?Set WANDB_API_KEY first}"

MODEL_ID="${1:-entity/project/run_id}"
OUT_DIR="${2:-artifacts/wandb}"

# Optional: override W&B API endpoint
# export WANDB_BASE_URL="https://api.wandb.ai"

go run ./cmd/mcg-cli generate \
  --model "${MODEL_ID}" \
  --source wandb \
  --template standard \
  --eval-file examples/eval_sample.csv \
  --formats md,json,html \
  --out-dir "${OUT_DIR}" \
  --lang en \
  --compliance eu-ai-act
