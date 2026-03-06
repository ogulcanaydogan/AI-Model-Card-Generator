#!/usr/bin/env python3
"""Normalize carbon footprint evidence to a deterministic JSON contract.

Output contract:
{
  "estimated_kg_co2e": <float>,
  "method": "<string>"
}

Priority:
1) MCG_CARBON_KG_CO2E environment value (manual override)
2) MCG_CARBON_EMISSIONS_FILE pointing to a CodeCarbon-like CSV (emissions column)
3) unavailable fallback (non-failing)
"""

from __future__ import annotations

import argparse
import csv
import json
import os
import sys
from pathlib import Path


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Normalize carbon metrics")
    parser.add_argument("--input", required=True, help="Input CSV path (eval file)")
    parser.add_argument("--output", required=True, help="Output JSON path")
    return parser.parse_args()


def parse_float(value: str) -> float | None:
    try:
        return float(value.strip())
    except (TypeError, ValueError):
        return None


def load_from_manual_env() -> tuple[float, str] | None:
    raw = os.getenv("MCG_CARBON_KG_CO2E", "").strip()
    if not raw:
        return None
    value = parse_float(raw)
    if value is None:
        return None
    return (max(0.0, value), "manual_env")


def load_from_emissions_csv() -> tuple[float, str] | None:
    path = os.getenv("MCG_CARBON_EMISSIONS_FILE", "").strip()
    if not path:
        return None

    csv_path = Path(path)
    if not csv_path.exists():
        return None

    last_value: float | None = None
    with csv_path.open(newline="", encoding="utf-8") as handle:
        reader = csv.DictReader(handle)
        for row in reader:
            value = parse_float(str(row.get("emissions", "")))
            if value is not None:
                last_value = value

    if last_value is None:
        return None
    return (max(0.0, last_value), "codecarbon_csv")


def main() -> int:
    args = parse_args()
    input_path = Path(args.input)
    output_path = Path(args.output)

    if not input_path.exists():
        print(f"Input file not found: {input_path}", file=sys.stderr)
        return 1

    payload = {"estimated_kg_co2e": 0.0, "method": "unavailable"}
    manual = load_from_manual_env()
    if manual is not None:
        payload["estimated_kg_co2e"] = round(manual[0], 9)
        payload["method"] = manual[1]
    else:
        csv_based = load_from_emissions_csv()
        if csv_based is not None:
            payload["estimated_kg_co2e"] = round(csv_based[0], 9)
            payload["method"] = csv_based[1]

    output_path.write_text(json.dumps(payload, indent=2), encoding="utf-8")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
