export const SUPPORTED_SOURCES = ["custom", "hf", "wandb", "mlflow"];

export function normalizeSource(value) {
  const candidate = String(value || "")
    .trim()
    .toLowerCase();
  if (candidate === "") {
    return "custom";
  }
  return candidate;
}

function parseWandBModelID(model) {
  const parts = String(model || "")
    .trim()
    .split("/");
  if (parts.length !== 3 || parts.some((part) => part.trim() === "")) {
    return { ok: false, error: "expected format <entity>/<project>/<run_id>" };
  }
  return { ok: true };
}

function parseMLflowModelID(model) {
  const value = String(model || "").trim();
  if (!value.toLowerCase().startsWith("run:")) {
    return { ok: false, error: "expected format run:<run_id>" };
  }
  const runID = value.slice(value.indexOf(":") + 1).trim();
  if (!runID) {
    return { ok: false, error: "expected format run:<run_id>" };
  }
  return { ok: true };
}

export function validateGeneratePayload(payload) {
  const source = normalizeSource(payload?.source);
  const model = String(payload?.model || "").trim();
  const metadataFile = String(payload?.metadataFile || "").trim();

  if (!SUPPORTED_SOURCES.includes(source)) {
    return `unsupported source: ${source}`;
  }

  if (!model) {
    return "--model is required";
  }

  if (source === "custom" && !metadataFile) {
    return "--metadataFile is required for custom source";
  }
  if (source === "custom") {
    return null;
  }

  if (source === "hf") {
    return null;
  }

  if (source === "wandb") {
    const parsed = parseWandBModelID(model);
    if (!parsed.ok) {
      return `invalid --model for wandb source: ${parsed.error}`;
    }
    return null;
  }

  if (source === "mlflow") {
    const parsed = parseMLflowModelID(model);
    if (!parsed.ok) {
      return `invalid --model for mlflow source: ${parsed.error}`;
    }
    return null;
  }

  return `unsupported source: ${source}`;
}
