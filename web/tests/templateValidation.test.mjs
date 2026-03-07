import test from "node:test";
import assert from "node:assert/strict";

import {
  normalizeTemplateSource,
  validateTemplateInitPayload,
  validateTemplatePreviewPayload,
  validateTemplateSelection,
  validateTemplateValidatePayload
} from "../lib/templateValidation.js";

test("normalizeTemplateSource falls back to built-in", () => {
  assert.equal(normalizeTemplateSource(""), "built-in");
  assert.equal(normalizeTemplateSource("unknown"), "built-in");
});

test("validateTemplateSelection requires templateFile in template-file mode", () => {
  const error = validateTemplateSelection({
    templateSource: "template-file",
    templateFile: ""
  });
  assert.equal(error, "--templateFile is required when template source is template-file");
});

test("validateTemplateSelection accepts built-in and template-file modes", () => {
  const builtIn = validateTemplateSelection({
    templateSource: "built-in",
    template: "standard"
  });
  const fileMode = validateTemplateSelection({
    templateSource: "template-file",
    templateFile: "templates/custom.tmpl"
  });
  assert.equal(builtIn, null);
  assert.equal(fileMode, null);
});

test("validateTemplateInitPayload validates required fields and base", () => {
  assert.equal(validateTemplateInitPayload({}), "--name is required");
  assert.equal(
    validateTemplateInitPayload({ name: "X", out: "a.tmpl", base: "unknown" }),
    "unsupported base template: unknown"
  );
  assert.equal(
    validateTemplateInitPayload({ name: "X", out: "a.tmpl", base: "minimal" }),
    null
  );
});

test("validateTemplateValidatePayload and preview payload enforce fields", () => {
  assert.equal(validateTemplateValidatePayload({}), "--input is required");
  assert.equal(validateTemplateValidatePayload({ input: "x.tmpl" }), null);

  assert.equal(validateTemplatePreviewPayload({ input: "", card: "", out: "" }), "--input is required");
  assert.equal(
    validateTemplatePreviewPayload({ input: "a.tmpl", card: "", out: "" }),
    "--card is required"
  );
  assert.equal(
    validateTemplatePreviewPayload({ input: "a.tmpl", card: "card.json", out: "" }),
    "--out is required"
  );
  assert.equal(
    validateTemplatePreviewPayload({ input: "a.tmpl", card: "card.json", out: "preview.md" }),
    null
  );
});
