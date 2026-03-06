import { spawn } from "node:child_process";
import { promises as fs } from "node:fs";
import os from "node:os";
import path from "node:path";

import { NextResponse } from "next/server";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

async function pathExists(candidate) {
  try {
    await fs.access(candidate);
    return true;
  } catch {
    return false;
  }
}

async function resolveRepoRoot() {
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

function runCommand(bin, args, options = {}) {
  return new Promise((resolve, reject) => {
    const child = spawn(bin, args, options);
    let stdout = "";
    let stderr = "";

    child.stdout.on("data", (chunk) => {
      stdout += chunk.toString();
    });
    child.stderr.on("data", (chunk) => {
      stderr += chunk.toString();
    });
    child.on("error", reject);
    child.on("close", (code) => {
      if (code !== 0) {
        reject(new Error(`${bin} ${args.join(" ")} failed (${code}): ${stderr || stdout}`));
        return;
      }
      resolve({ stdout, stderr });
    });
  });
}

export async function POST(request) {
  let tempDir;
  try {
    const payload = await request.json();
    const source = payload?.source || "custom";
    if (source !== "custom") {
      return NextResponse.json(
        { error: "Sprint 4 web skeleton currently supports only --source custom" },
        { status: 400 }
      );
    }

    const repoRoot = await resolveRepoRoot();
    const model = payload?.model || "demo-model";
    const evalFile = payload?.evalFile || "examples/eval_sample.csv";
    const metadataFile = payload?.metadataFile || "tests/fixtures/custom_metadata.json";
    const template = payload?.template || "standard";
    const compliance = payload?.compliance || "eu-ai-act,nist";
    const lang = payload?.locale || "en";

    tempDir = await fs.mkdtemp(path.join(os.tmpdir(), "mcg-web-"));

    const generateArgs = [
      "run",
      "./cmd/mcg-cli",
      "generate",
      "--model",
      model,
      "--source",
      source,
      "--uri",
      metadataFile,
      "--eval-file",
      evalFile,
      "--template",
      template,
      "--formats",
      "md,json",
      "--out-dir",
      tempDir,
      "--lang",
      lang,
      "--compliance",
      compliance
    ];

    const env = {
      ...process.env,
      MCG_PYTHON_BIN: process.env.MCG_PYTHON_BIN || "python3",
      MCG_FAIRNESS_SCRIPT:
        process.env.MCG_FAIRNESS_SCRIPT || "tests/fixtures/fairness_stub.py",
      MCG_CARBON_FIXTURE:
        process.env.MCG_CARBON_FIXTURE || "tests/fixtures/carbon/carbon_fixture.json"
    };

    const generated = await runCommand("go", generateArgs, {
      cwd: repoRoot,
      env
    });

    await runCommand(
      "go",
      [
        "run",
        "./cmd/mcg-cli",
        "validate",
        "--schema",
        "schemas/model-card.v1.json",
        "--input",
        path.join(tempDir, "model_card.json")
      ],
      { cwd: repoRoot, env }
    );

    const nistCheck = await runCommand(
      "go",
      [
        "run",
        "./cmd/mcg-cli",
        "check",
        "--framework",
        "nist",
        "--input",
        path.join(tempDir, "model_card.json"),
        "--strict",
        "false"
      ],
      { cwd: repoRoot, env }
    );

    const [jsonRaw, mdRaw, complianceRaw] = await Promise.all([
      fs.readFile(path.join(tempDir, "model_card.json"), "utf-8"),
      fs.readFile(path.join(tempDir, "model_card.md"), "utf-8"),
      fs.readFile(path.join(tempDir, "compliance_report.json"), "utf-8")
    ]);

    return NextResponse.json({
      card: JSON.parse(jsonRaw),
      markdown: mdRaw,
      complianceReport: JSON.parse(complianceRaw),
      nistCheck: JSON.parse(nistCheck.stdout),
      files: {
        modelCardJson: path.join(tempDir, "model_card.json"),
        modelCardMarkdown: path.join(tempDir, "model_card.md"),
        complianceReportJson: path.join(tempDir, "compliance_report.json")
      },
      logs: [generated.stdout.trim(), generated.stderr.trim()].filter(Boolean).join("\n")
    });
  } catch (error) {
    return NextResponse.json(
      {
        error: error instanceof Error ? error.message : "Unknown error"
      },
      { status: 500 }
    );
  } finally {
    if (tempDir) {
      // Keep artifacts for local inspection in sprint skeleton.
      // eslint-disable-next-line no-console
      console.log(`mcg-web artifacts retained at ${tempDir}`);
    }
  }
}
