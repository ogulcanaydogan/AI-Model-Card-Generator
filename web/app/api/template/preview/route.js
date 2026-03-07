import { promises as fs } from "node:fs";

import { NextResponse } from "next/server";

import {
  buildCLIEnv,
  pathExists,
  resolveCLICommand,
  resolveRepoRoot,
  runCommand
} from "@/lib/apiRuntime";
import { resolveSafeRepoPath } from "@/lib/pathGuard";
import { validateTemplatePreviewPayload } from "@/lib/templateValidation";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

export async function POST(request) {
  try {
    const payload = await request.json();
    const validationError = validateTemplatePreviewPayload(payload);
    if (validationError) {
      return NextResponse.json({ error: validationError }, { status: 400 });
    }

    const input = String(payload.input).trim();
    const card = String(payload.card).trim();
    const out = String(payload.out).trim();

    const repoRoot = await resolveRepoRoot();
    let safeInput;
    let safeCard;
    let safeOut;
    try {
      safeInput = resolveSafeRepoPath(repoRoot, input);
      safeCard = resolveSafeRepoPath(repoRoot, card);
      safeOut = resolveSafeRepoPath(repoRoot, out);
    } catch (error) {
      return NextResponse.json(
        {
          error: `invalid path: ${error instanceof Error ? error.message : "invalid path"}`
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
    if (!(await pathExists(safeCard.absolute))) {
      return NextResponse.json(
        {
          error: `card file does not exist: ${safeCard.relative}`
        },
        { status: 400 }
      );
    }

    const env = buildCLIEnv();
    const commandTimeoutMs = Number(process.env.MCG_WEB_COMMAND_TIMEOUT_MS || "180000");
    const cmd = resolveCLICommand(
      [
        "template",
        "preview",
        "--input",
        safeInput.relative,
        "--card",
        safeCard.relative,
        "--out",
        safeOut.relative
      ],
      env
    );

    try {
      const result = await runCommand(cmd.bin, cmd.args, {
        cwd: repoRoot,
        env,
        timeoutMs: commandTimeoutMs
      });
      const markdown = await fs.readFile(safeOut.absolute, "utf-8");
      return NextResponse.json({
        outputPath: safeOut.relative,
        markdown,
        logs: [result.stdout.trim(), result.stderr.trim()].filter(Boolean).join("\n")
      });
    } catch (error) {
      return NextResponse.json(
        {
          error: error instanceof Error ? error.message : "Template preview failed"
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
