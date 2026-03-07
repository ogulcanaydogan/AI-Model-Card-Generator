import { spawn } from "node:child_process";
import { promises as fs } from "node:fs";
import path from "node:path";

export async function pathExists(candidate) {
  try {
    await fs.access(candidate);
    return true;
  } catch {
    return false;
  }
}

export async function resolveRepoRoot() {
  const cwd = process.cwd();
  if (await pathExists(path.join(cwd, "go.mod"))) {
    return cwd;
  }
  const parent = path.resolve(cwd, "..");
  if (await pathExists(path.join(parent, "go.mod"))) {
    return parent;
  }
  throw new Error("Could not resolve repository root containing go.mod");
}

export function buildCLIEnv() {
  return {
    ...process.env,
    MCG_PYTHON_BIN: process.env.MCG_PYTHON_BIN || "python3",
    MCG_FAIRNESS_SCRIPT:
      process.env.MCG_FAIRNESS_SCRIPT || "tests/fixtures/fairness_stub.py",
    MCG_CARBON_FIXTURE:
      process.env.MCG_CARBON_FIXTURE || "tests/fixtures/carbon/carbon_fixture.json",
    MCG_WANDB_FIXTURE: process.env.MCG_WANDB_FIXTURE || "",
    MCG_MLFLOW_FIXTURE: process.env.MCG_MLFLOW_FIXTURE || ""
  };
}

export function resolveCLICommand(args, env) {
  const cliBin = String(env.MCG_CLI_BIN || "").trim();
  if (cliBin) {
    return {
      bin: cliBin,
      args
    };
  }
  return {
    bin: "go",
    args: ["run", "./cmd/mcg-cli", ...args]
  };
}

export function runCommand(bin, args, options = {}) {
  return new Promise((resolve, reject) => {
    const child = spawn(bin, args, options);
    let stdout = "";
    let stderr = "";
    const timeoutMs = Number(options.timeoutMs || 180000);
    let completed = false;
    const timeoutID = setTimeout(() => {
      if (completed) {
        return;
      }
      child.kill("SIGTERM");
      setTimeout(() => {
        if (!completed) {
          child.kill("SIGKILL");
        }
      }, 2000);
      completed = true;
      reject(
        new Error(
          `${bin} ${args.join(" ")} timed out after ${timeoutMs}ms: ${stderr || stdout}`
        )
      );
    }, timeoutMs);

    child.stdout.on("data", (chunk) => {
      stdout += chunk.toString();
    });
    child.stderr.on("data", (chunk) => {
      stderr += chunk.toString();
    });
    child.on("error", (err) => {
      if (completed) {
        return;
      }
      completed = true;
      clearTimeout(timeoutID);
      reject(err);
    });
    child.on("close", (code) => {
      if (completed) {
        return;
      }
      completed = true;
      clearTimeout(timeoutID);
      if (code !== 0) {
        reject(new Error(`${bin} ${args.join(" ")} failed (${code}): ${stderr || stdout}`));
        return;
      }
      resolve({ stdout, stderr });
    });
  });
}
