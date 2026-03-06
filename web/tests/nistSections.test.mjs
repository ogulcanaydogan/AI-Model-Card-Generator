import test from "node:test";
import assert from "node:assert/strict";

import { buildNISTFunctionSections, summarizeNISTOverall } from "../lib/nistSections.js";

test("buildNISTFunctionSections groups entries by NIST function", () => {
  const report = {
    framework: "nist",
    status: "warn",
    score: 72,
    required_gaps: ["GOVERN: [GOV-1][required] missing owner", "MAP: [MAP-3][required] missing context"],
    findings: [
      "GOVERN: [GOV-3][advisory] missing timestamp",
      "MEASURE: [MEA-4][advisory] parity threshold exceeded"
    ],
    recommended_actions: [
      "GOVERN: set owner",
      "MAP: add data context",
      "MEASURE: improve subgroup parity",
      "MANAGE: add carbon evidence"
    ]
  };

  const sections = buildNISTFunctionSections(report);
  assert.equal(sections.length, 4);

  const govern = sections.find((item) => item.functionName === "GOVERN");
  assert.equal(govern.status, "fail");
  assert.equal(govern.requiredGaps.length, 1);
  assert.equal(govern.findings.length, 1);
  assert.equal(govern.scoreContribution, -20);

  const manage = sections.find((item) => item.functionName === "MANAGE");
  assert.equal(manage.status, "pass");
  assert.equal(manage.requiredGaps.length, 0);
  assert.equal(manage.findings.length, 0);
  assert.equal(manage.recommendedActions.length, 1);
});

test("summarizeNISTOverall returns status and counts", () => {
  const report = {
    framework: "nist",
    status: "fail",
    score: 45,
    required_gaps: ["GOVERN: missing owner", "MANAGE: missing mitigations"],
    findings: ["MEASURE: parity threshold exceeded"]
  };

  const summary = summarizeNISTOverall(report);
  assert.equal(summary.status, "fail");
  assert.equal(summary.score, 45);
  assert.equal(summary.requiredCount, 2);
  assert.equal(summary.advisoryCount, 1);
});

test("NIST helpers return safe defaults for non-nist reports", () => {
  assert.deepEqual(buildNISTFunctionSections({ framework: "eu-ai-act" }), []);
  assert.deepEqual(summarizeNISTOverall({ framework: "eu-ai-act" }), {
    status: "n/a",
    score: null,
    requiredCount: 0,
    advisoryCount: 0
  });
});
