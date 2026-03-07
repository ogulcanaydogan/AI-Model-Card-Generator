import { NextResponse } from "next/server";

import {
  buildCLIEnv,
  resolveCLICommand,
  resolveRepoRoot,
  runCommand
} from "@/lib/apiRuntime";
import { resolveSafeRepoPath } from "@/lib/pathGuard";
import { validateTemplateInitPayload } from "@/lib/templateValidation";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

export async function POST(request) {
  try {
    const payload = await request.json();
    const validationError = validateTemplateInitPayload(payload);
    if (validationError) {
      return NextResponse.json({ error: validationError }, { status: 400 });
    }

    const name = String(payload.name).trim();
    const base = String(payload.base || "standard")
      .trim()
      .toLowerCase();
    const out = String(payload.out).trim();

    const repoRoot = await resolveRepoRoot();
    let safeOut;
    try {
      safeOut = resolveSafeRepoPath(repoRoot, out);
    } catch (error) {
      return NextResponse.json(
        {
          error: `invalid --out path: ${error instanceof Error ? error.message : "invalid path"}`
        },
        { status: 400 }
      );
    }

    const env = buildCLIEnv();
    const commandTimeoutMs = Number(process.env.MCG_WEB_COMMAND_TIMEOUT_MS || "180000");
    const cmd = resolveCLICommand(
      ["template", "init", "--name", name, "--out", safeOut.relative, "--base", base],
      env
    );
    const result = await runCommand(cmd.bin, cmd.args, {
      cwd: repoRoot,
      env,
      timeoutMs: commandTimeoutMs
    });

    return NextResponse.json({
      outputPath: safeOut.relative,
      logs: [result.stdout.trim(), result.stderr.trim()].filter(Boolean).join("\n")
    });
  } catch (error) {
    return NextResponse.json(
      {
        error: error instanceof Error ? error.message : "Unknown error"
      },
      { status: 500 }
    );
  }
}
