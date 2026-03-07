export const SUPPORTED_TEMPLATE_BASES = ["standard", "minimal", "eu-ai-act"];
export const SUPPORTED_TEMPLATE_SOURCES = ["built-in", "template-file"];

export function normalizeTemplateSource(value) {
  const candidate = String(value || "")
    .trim()
    .toLowerCase();
  if (SUPPORTED_TEMPLATE_SOURCES.includes(candidate)) {
    return candidate;
  }
  return "built-in";
}

export function validateTemplateSelection(payload) {
  const templateSource = normalizeTemplateSource(payload?.templateSource);
  const template = String(payload?.template || "")
    .trim()
    .toLowerCase();
  const templateFile = String(payload?.templateFile || "").trim();

  if (templateSource === "template-file") {
    if (!templateFile) {
      return "--templateFile is required when template source is template-file";
    }
    return null;
  }

  if (!SUPPORTED_TEMPLATE_BASES.includes(template || "standard")) {
    return `unsupported template base: ${template}`;
  }
  return null;
}

export function validateTemplateInitPayload(payload) {
  const name = String(payload?.name || "").trim();
  const out = String(payload?.out || "").trim();
  const base = String(payload?.base || "")
    .trim()
    .toLowerCase();

  if (!name) {
    return "--name is required";
  }
  if (!out) {
    return "--out is required";
  }
  if (!SUPPORTED_TEMPLATE_BASES.includes(base || "standard")) {
    return `unsupported base template: ${base}`;
  }
  return null;
}

export function validateTemplateValidatePayload(payload) {
  const input = String(payload?.input || "").trim();
  if (!input) {
    return "--input is required";
  }
  return null;
}

export function validateTemplatePreviewPayload(payload) {
  const input = String(payload?.input || "").trim();
  const card = String(payload?.card || "").trim();
  const out = String(payload?.out || "").trim();
  if (!input) {
    return "--input is required";
  }
  if (!card) {
    return "--card is required";
  }
  if (!out) {
    return "--out is required";
  }
  return null;
}
