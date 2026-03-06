import test from "node:test";
import assert from "node:assert/strict";

import { normalizeSource, validateGeneratePayload } from "../lib/sourceValidation.js";

test("normalizeSource defaults empty source to custom", () => {
  assert.equal(normalizeSource(""), "custom");
});

test("normalizeSource preserves unsupported source for explicit validation error", () => {
  assert.equal(normalizeSource("invalid"), "invalid");
});

test("validateGeneratePayload enforces custom metadataFile", () => {
  const error = validateGeneratePayload({
    source: "custom",
    model: "demo-model",
    metadataFile: ""
  });
  assert.equal(error, "--metadataFile is required for custom source");
});

test("validateGeneratePayload accepts custom source with metadataFile", () => {
  const error = validateGeneratePayload({
    source: "custom",
    model: "demo-model",
    metadataFile: "tests/fixtures/custom_metadata.json"
  });
  assert.equal(error, null);
});

test("validateGeneratePayload validates wandb model format", () => {
  const error = validateGeneratePayload({
    source: "wandb",
    model: "acme/support",
    metadataFile: ""
  });
  assert.equal(
    error,
    "invalid --model for wandb source: expected format <entity>/<project>/<run_id>"
  );
});

test("validateGeneratePayload rejects unsupported source", () => {
  const error = validateGeneratePayload({
    source: "invalid",
    model: "demo-model",
    metadataFile: "tests/fixtures/custom_metadata.json"
  });
  assert.equal(error, "unsupported source: invalid");
});

test("validateGeneratePayload validates mlflow model format", () => {
  const error = validateGeneratePayload({
    source: "mlflow",
    model: "abc123",
    metadataFile: ""
  });
  assert.equal(error, "invalid --model for mlflow source: expected format run:<run_id>");
});

test("validateGeneratePayload accepts hf source with non-empty model", () => {
  const error = validateGeneratePayload({
    source: "hf",
    model: "bert-base-uncased",
    metadataFile: ""
  });
  assert.equal(error, null);
});

test("validateGeneratePayload accepts valid wandb and mlflow model formats", () => {
  const wandb = validateGeneratePayload({
    source: "wandb",
    model: "acme/project/run123",
    metadataFile: ""
  });
  const mlflow = validateGeneratePayload({
    source: "mlflow",
    model: "run:abc123",
    metadataFile: ""
  });
  assert.equal(wandb, null);
  assert.equal(mlflow, null);
});
