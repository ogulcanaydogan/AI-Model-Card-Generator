import { NextResponse } from "next/server";

import {
  buildCLIEnv,
  pathExists,
  resolveCLICommand,
  resolveRepoRoot,
  runCommand
} from "@/lib/apiRuntime";
import { resolveSafeRepoPath } from "@/lib/pathGuard";
import { validateTemplateValidatePayload } from "@/lib/templateValidation";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

export async function POST(request) {
  try {
    const payload = await request.json();
    const validationError = validateTemplateValidatePayload(payload);
    if (validationError) {
      return NextResponse.json({ error: validationError }, { status: 400 });
    }

    const input = String(payload.input).trim();
    const repoRoot = await resolveRepoRoot();
    let safeInput;
    try {
      safeInput = resolveSafeRepoPath(repoRoot, input);
    } catch (error) {
      return NextResponse.json(
        {
          error: `invalid --input path: ${error instanceof Error ? error.message : "invalid path"}`
        },
        { status: 400 }
      );
    }
    if (!(await pathExists(safeInput.absolute))) {
      return NextResponse.json(
        {
          error: `template file does not exist: ${safeInput.relative}`
        },
        { status: 400 }
      );
    }

    const env = buildCLIEnv();
    const commandTimeoutMs = Number(process.env.MCG_WEB_COMMAND_TIMEOUT_MS || "180000");
    const cmd = resolveCLICommand(["template", "validate", "--input", safeInput.relative], env);

    try {
      const result = await runCommand(cmd.bin, cmd.args, {
        cwd: repoRoot,
        env,
        timeoutMs: commandTimeoutMs
      });
      return NextResponse.json({
        valid: true,
        inputPath: safeInput.relative,
        logs: [result.stdout.trim(), result.stderr.trim()].filter(Boolean).join("\n")
      });
    } catch (error) {
      return NextResponse.json(
        {
          error: error instanceof Error ? error.message : "Template validation failed"
        },
        { status: 400 }
      );
    }
  } catch (error) {
    return NextResponse.json(
      {
        error: error instanceof Error ? error.message : "Unknown error"
      },
      { status: 500 }
    );
  }
}
