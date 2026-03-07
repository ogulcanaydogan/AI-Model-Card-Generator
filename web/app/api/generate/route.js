import { promises as fs } from "node:fs";
import os from "node:os";
import path from "node:path";

import { NextResponse } from "next/server";

import {
  buildCLIEnv,
  resolveCLICommand,
  resolveRepoRoot,
  runCommand
} from "@/lib/apiRuntime";
import { resolveSafeRepoPath } from "@/lib/pathGuard";
import { normalizeSource, validateGeneratePayload } from "@/lib/sourceValidation";
import { normalizeTemplateSource, validateTemplateSelection } from "@/lib/templateValidation";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

export async function POST(request) {
  let tempDir;
  try {
    const payload = await request.json();
    const source = normalizeSource(payload?.source);
    const model = String(payload?.model || "").trim();
    const evalFile = String(payload?.evalFile || "").trim() || "examples/eval_sample.csv";
    const metadataFile = String(payload?.metadataFile || "").trim();
    const template = String(payload?.template || "").trim() || "standard";
    const templateSource = normalizeTemplateSource(payload?.templateSource);
    const templateFile = String(payload?.templateFile || "").trim();
    const compliance = String(payload?.compliance || "").trim() || "eu-ai-act,nist,iso42001";
    const lang = String(payload?.locale || "").trim() || "en";

    const validationError = validateGeneratePayload({
      source,
      model,
      metadataFile
    });
    if (validationError) {
      return NextResponse.json({ error: validationError }, { status: 400 });
    }
    const templateValidationError = validateTemplateSelection({
      templateSource,
      template,
      templateFile
    });
    if (templateValidationError) {
      return NextResponse.json({ error: templateValidationError }, { status: 400 });
    }

    const repoRoot = await resolveRepoRoot();
    let safeTemplateFile = "";
    if (templateSource === "template-file") {
      try {
        const resolved = resolveSafeRepoPath(repoRoot, templateFile);
        safeTemplateFile = resolved.relative;
      } catch (error) {
        return NextResponse.json(
          {
            error: `invalid --templateFile: ${error instanceof Error ? error.message : "invalid path"}`
          },
          { status: 400 }
        );
      }
    }

    tempDir = await fs.mkdtemp(path.join(os.tmpdir(), "mcg-web-"));

    const generateArgs = [
      "run",
      "./cmd/mcg-cli",
      "generate",
      "--model",
      model,
      "--source",
      source,
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
    if (safeTemplateFile) {
      generateArgs.push("--template-file", safeTemplateFile);
    }
    if (source === "custom") {
      generateArgs.splice(7, 0, "--uri", metadataFile);
    }
    if (source === "hf" && process.env.MCG_WEB_HF_BASE_URL) {
      generateArgs.push("--hf-base-url", process.env.MCG_WEB_HF_BASE_URL);
    }

    const env = buildCLIEnv();
    const commandTimeoutMs = Number(process.env.MCG_WEB_COMMAND_TIMEOUT_MS || "180000");
    const generateCmd = resolveCLICommand(generateArgs.slice(2), env);
    const generated = await runCommand(generateCmd.bin, generateCmd.args, {
      cwd: repoRoot,
      env,
      timeoutMs: commandTimeoutMs
    });

    const validateCmd = resolveCLICommand([
      "validate",
      "--schema",
      "schemas/model-card.v1.json",
      "--input",
      path.join(tempDir, "model_card.json")
    ], env);
    await runCommand(validateCmd.bin, validateCmd.args, {
      cwd: repoRoot,
      env,
      timeoutMs: commandTimeoutMs
    });

    const checkCmd = resolveCLICommand([
      "check",
      "--framework",
      "nist",
      "--input",
      path.join(tempDir, "model_card.json"),
      "--strict",
      "false"
    ], env);
    const nistCheck = await runCommand(checkCmd.bin, checkCmd.args, {
      cwd: repoRoot,
      env,
      timeoutMs: commandTimeoutMs
    });

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
