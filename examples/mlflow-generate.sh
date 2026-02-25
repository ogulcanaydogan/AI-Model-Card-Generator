#!/usr/bin/env bash
set -euo pipefail

# Required for live MLflow extraction
: "${MLFLOW_TRACKING_URI:?Set MLFLOW_TRACKING_URI first}"

RUN_ID="${1:-abc123}"
OUT_DIR="${2:-artifacts/mlflow}"

# Optional auth
# export MLFLOW_TRACKING_TOKEN="token"
# export MLFLOW_TRACKING_USERNAME="user"
# export MLFLOW_TRACKING_PASSWORD="pass"

go run ./cmd/mcg-cli generate \
  --model "run:${RUN_ID}" \
  --source mlflow \
  --template standard \
  --eval-file examples/eval_sample.csv \
  --formats md,json,html \
  --out-dir "${OUT_DIR}" \
  --lang en \
  --compliance eu-ai-act
