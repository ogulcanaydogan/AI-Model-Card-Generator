export const NIST_FUNCTIONS = ["GOVERN", "MAP", "MEASURE", "MANAGE"];

const REQUIRED_PENALTY = 12;
const ADVISORY_PENALTY = 4;
const CONTROL_ID_PATTERN = /\[([A-Z]{3}-\d+)\]/;

export const NIST_CONTROL_CATALOG = {
  GOVERN: ["GOV-1", "GOV-2", "GOV-3", "GOV-4"],
  MAP: ["MAP-1", "MAP-2", "MAP-3", "MAP-4", "MAP-5"],
  MEASURE: ["MEA-1", "MEA-2", "MEA-3", "MEA-4", "MEA-5", "MEA-6"],
  MANAGE: ["MAN-1", "MAN-2", "MAN-3", "MAN-4", "MAN-5", "MAN-6"]
};

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

function extractControlID(item) {
  const match = String(item || "").match(CONTROL_ID_PATTERN);
  return match?.[1] || null;
}

function summarizeCoverage(items, catalogIDs) {
  const failedSet = new Set(
    (items || [])
      .map((item) => extractControlID(item))
      .filter((id) => id && catalogIDs.includes(id))
  );
  const totalControls = catalogIDs.length;
  const failingControls = failedSet.size;
  const passingControls = Math.max(0, totalControls - failingControls);
  return { totalControls, passingControls, failingControls };
}

export function buildNISTFunctionSections(report) {
  if (!report || report.framework !== "nist") {
    return [];
  }

  return NIST_FUNCTIONS.map((fn) => {
    const catalogIDs = NIST_CONTROL_CATALOG[fn] || [];
    const requiredGaps = filterByFunction(report.required_gaps, fn);
    const findings = filterByFunction(report.findings, fn);
    const recommendedActions = filterByFunction(report.recommended_actions, fn);
    const status = sectionStatus(requiredGaps, findings);
    const coverage = summarizeCoverage(requiredGaps.concat(findings), catalogIDs);
    const scoreContribution =
      requiredGaps.length * -REQUIRED_PENALTY + findings.length * -ADVISORY_PENALTY;

    return {
      functionName: fn,
      status,
      scoreContribution,
      controlCoverage: `${coverage.passingControls}/${coverage.totalControls}`,
      totalControls: coverage.totalControls,
      requiredCount: requiredGaps.length,
      advisoryCount: findings.length,
      requiredGaps,
      findings,
      recommendedActions,
      shortRemediations: recommendedActions.slice(0, 2)
    };
  });
}

export function summarizeNISTOverall(report) {
  if (!report || report.framework !== "nist") {
    return {
      status: "n/a",
      score: null,
      requiredCount: 0,
      advisoryCount: 0,
      controlCoverage: "0/0"
    };
  }

  const allCatalogIDs = Object.values(NIST_CONTROL_CATALOG).flat();
  const allFindings = (report.required_gaps || []).concat(report.findings || []);
  const coverage = summarizeCoverage(allFindings, allCatalogIDs);

  return {
    status: report.status || "n/a",
    score: report.score,
    requiredCount: (report.required_gaps || []).length,
    advisoryCount: (report.findings || []).length,
    controlCoverage: `${coverage.passingControls}/${coverage.totalControls}`
  };
}
