#!/usr/bin/env python3
"""Compute fairness metrics from an eval CSV file.

Required columns: y_true, y_pred, group
Optional columns: y_score, sample_weight
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

try:
    import pandas as pd
    from fairlearn.metrics import (
        MetricFrame,
        demographic_parity_difference,
        equalized_odds_difference,
        false_positive_rate,
        selection_rate,
        true_positive_rate,
    )
except ModuleNotFoundError as exc:  # pragma: no cover
    print(
        "Missing dependency: {}. Install with `pip install fairlearn pandas`.".format(exc.name),
        file=sys.stderr,
    )
    sys.exit(2)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Compute fairness metrics")
    parser.add_argument("--input", required=True, help="Input CSV path")
    parser.add_argument("--output", required=True, help="Output JSON path")
    return parser.parse_args()


def main() -> int:
    args = parse_args()

    input_path = Path(args.input)
    output_path = Path(args.output)

    if not input_path.exists():
        print(f"Input file not found: {input_path}", file=sys.stderr)
        return 1

    df = pd.read_csv(input_path)

    required = {"y_true", "y_pred", "group"}
    missing = required.difference(df.columns)
    if missing:
        print(f"Missing required columns: {', '.join(sorted(missing))}", file=sys.stderr)
        return 1

    y_true = df["y_true"]
    y_pred = df["y_pred"]
    group = df["group"]

    dp = float(demographic_parity_difference(y_true, y_pred, sensitive_features=group))
    eo = float(equalized_odds_difference(y_true, y_pred, sensitive_features=group))

    metric_frame = MetricFrame(
        metrics={
            "selection_rate": selection_rate,
            "true_positive_rate": true_positive_rate,
            "false_positive_rate": false_positive_rate,
        },
        y_true=y_true,
        y_pred=y_pred,
        sensitive_features=group,
    )

    by_group = metric_frame.by_group.sort_index()
    group_stats = []
    for group_key, row in by_group.iterrows():
        group_name = str(group_key)
        support = int((group == group_key).sum())
        if support == 0:
            support = int((group.astype(str) == group_name).sum())
        group_stats.append(
            {
                "group": group_name,
                "selection_rate": round(float(row["selection_rate"]), 6),
                "true_positive_rate": round(float(row["true_positive_rate"]), 6),
                "false_positive_rate": round(float(row["false_positive_rate"]), 6),
                "support": support,
            }
        )

    payload = {
        "demographic_parity_diff": round(abs(dp), 6),
        "equalized_odds_diff": round(abs(eo), 6),
        "group_stats": group_stats,
    }

    output_path.write_text(json.dumps(payload, indent=2), encoding="utf-8")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
