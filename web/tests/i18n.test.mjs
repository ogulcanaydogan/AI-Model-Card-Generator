import test from "node:test";
import assert from "node:assert/strict";

import { DEFAULT_LOCALE, getDictionary, normalizeLocale } from "../lib/i18n.js";

test("normalizeLocale falls back to default for unknown locales", () => {
  assert.equal(normalizeLocale("de"), DEFAULT_LOCALE);
  assert.equal(normalizeLocale(""), DEFAULT_LOCALE);
});

test("normalizeLocale keeps supported locales", () => {
  assert.equal(normalizeLocale("en"), "en");
  assert.equal(normalizeLocale("tr"), "tr");
});

test("getDictionary uses fallback dictionary for unknown locales", () => {
  const dictionary = getDictionary("unknown");
  assert.equal(dictionary.generate, "Generate Preview");
});
