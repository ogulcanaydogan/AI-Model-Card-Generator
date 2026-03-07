import test from "node:test";
import assert from "node:assert/strict";
import path from "node:path";

import { resolveSafeRepoPath } from "../lib/pathGuard.js";

test("resolveSafeRepoPath accepts relative path under repository", () => {
  const repoRoot = path.resolve("/tmp/repo");
  const resolved = resolveSafeRepoPath(repoRoot, "artifacts/web/template.tmpl");
  assert.equal(resolved.relative, "artifacts/web/template.tmpl");
  assert.equal(resolved.absolute, path.resolve(repoRoot, "artifacts/web/template.tmpl"));
});

test("resolveSafeRepoPath rejects absolute paths", () => {
  const repoRoot = path.resolve("/tmp/repo");
  assert.throws(
    () => resolveSafeRepoPath(repoRoot, "/etc/passwd"),
    /path must be relative to repository root/
  );
});

test("resolveSafeRepoPath rejects traversal paths", () => {
  const repoRoot = path.resolve("/tmp/repo");
  assert.throws(
    () => resolveSafeRepoPath(repoRoot, "../outside.tmpl"),
    /path must stay inside repository root/
  );
});
