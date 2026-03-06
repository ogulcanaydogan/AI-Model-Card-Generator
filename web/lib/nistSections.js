export const NIST_FUNCTIONS = ["GOVERN", "MAP", "MEASURE", "MANAGE"];

const REQUIRED_PENALTY = 15;
const ADVISORY_PENALTY = 5;

function filterByFunction(items, fn) {
  const prefix = `${fn}:`;
  return (items || []).filter((item) => String(item || "").startsWith(prefix));
}

function sectionStatus(requiredGaps, findings) {
  if (requiredGaps.length > 0) {
    return "fail";
  }
  if (findings.length > 0) {
    return "warn";
  }
  return "pass";
}

export function buildNISTFunctionSections(report) {
  if (!report || report.framework !== "nist") {
    return [];
  }

  return NIST_FUNCTIONS.map((fn) => {
    const requiredGaps = filterByFunction(report.required_gaps, fn);
    const findings = filterByFunction(report.findings, fn);
    const recommendedActions = filterByFunction(report.recommended_actions, fn);
    const status = sectionStatus(requiredGaps, findings);
    const scoreContribution =
      requiredGaps.length * -REQUIRED_PENALTY + findings.length * -ADVISORY_PENALTY;

    return {
      functionName: fn,
      status,
      scoreContribution,
      requiredGaps,
      findings,
      recommendedActions
    };
  });
}

export function summarizeNISTOverall(report) {
  if (!report || report.framework !== "nist") {
    return {
      status: "n/a",
      score: null,
      requiredCount: 0,
      advisoryCount: 0
    };
  }

  return {
    status: report.status || "n/a",
    score: report.score,
    requiredCount: (report.required_gaps || []).length,
    advisoryCount: (report.findings || []).length
  };
}
