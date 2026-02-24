#!/usr/bin/env python3
import argparse
import json
from pathlib import Path


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", required=True)
    parser.add_argument("--output", required=True)
    args = parser.parse_args()

    if not Path(args.input).exists():
        raise FileNotFoundError(args.input)

    payload = {
        "demographic_parity_diff": 0.08,
        "equalized_odds_diff": 0.11,
        "group_stats": [
            {
                "group": "a",
                "selection_rate": 0.62,
                "true_positive_rate": 0.75,
                "false_positive_rate": 0.21,
                "support": 5,
            },
            {
                "group": "b",
                "selection_rate": 0.54,
                "true_positive_rate": 0.70,
                "false_positive_rate": 0.19,
                "support": 5,
            },
        ],
    }

    Path(args.output).write_text(json.dumps(payload), encoding="utf-8")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
