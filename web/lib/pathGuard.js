import path from "node:path";

function normalizeSeparators(value) {
  return value.split(path.sep).join("/");
}

export function resolveSafeRepoPath(repoRoot, userPath) {
  const input = String(userPath || "").trim();
  if (!input) {
    throw new Error("path is required");
  }
  if (path.isAbsolute(input)) {
    throw new Error("path must be relative to repository root");
  }

  const cleaned = path.normalize(input);
  if (cleaned === "." || cleaned === ".." || cleaned.startsWith(`..${path.sep}`)) {
    throw new Error("path must stay inside repository root");
  }

  const root = path.resolve(repoRoot);
  const resolved = path.resolve(root, cleaned);
  const relativeFromRoot = path.relative(root, resolved);
  if (
    relativeFromRoot === ".." ||
    relativeFromRoot.startsWith(`..${path.sep}`) ||
    path.isAbsolute(relativeFromRoot)
  ) {
    throw new Error("path must stay inside repository root");
  }

  return {
    relative: normalizeSeparators(relativeFromRoot),
    absolute: resolved
  };
}
