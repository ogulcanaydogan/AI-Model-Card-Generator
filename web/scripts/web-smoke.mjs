import { spawn } from "node:child_process";
import http from "node:http";
import { setTimeout as sleep } from "node:timers/promises";
import { once } from "node:events";

async function startHFMockServer() {
  const server = http.createServer((req, res) => {
    if (req.method === "GET" && req.url?.startsWith("/api/models/")) {
      const modelID = req.url.split("/").pop() || "unknown-model";
      const payload = {
        id: modelID,
        modelId: modelID,
        author: "hf-mock-owner",
        tags: ["license:apache-2.0", "nlp"],
        cardData: {
          license: "apache-2.0",
          model_summary: "HF mock summary",
          limitations: "HF mock limitations",
          datasets: "HF mock training dataset",
          eval_results: "HF mock eval dataset"
        }
      };
      res.writeHead(200, {
        "Content-Type": "application/json",
        Connection: "close"
      });
      res.end(JSON.stringify(payload));
      return;
    }

    res.writeHead(404, {
      "Content-Type": "application/json",
      Connection: "close"
    });
    res.end(JSON.stringify({ error: "not-found" }));
  });

  server.listen(0, "127.0.0.1");
  await once(server, "listening");
  const address = server.address();
  if (!address || typeof address === "string") {
    throw new Error("Unable to read HF mock server address");
  }
  return { server, baseURL: `http://127.0.0.1:${address.port}` };
}

async function waitForReady(url, attempts = 60) {
  for (let i = 0; i < attempts; i += 1) {
    try {
      const response = await fetch(url);
      if (response.ok) {
        return;
      }
    } catch {
      // ignore and retry
    }
    await sleep(500);
  }
  throw new Error(`Timed out waiting for ${url}`);
}

async function postGenerate(port, payload) {
  const response = await fetch(`http://127.0.0.1:${port}/api/generate`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });
  let body;
  try {
    body = await response.json();
  } catch {
    body = {};
  }
  return { status: response.status, body };
}

async function stopChildProcess(child) {
  if (child.exitCode !== null) {
    return;
  }

  child.kill("SIGTERM");
  const graceful = await Promise.race([
    once(child, "exit").then(() => true),
    sleep(5000).then(() => false)
  ]);
  if (graceful) {
    return;
  }

  child.kill("SIGKILL");
  await Promise.race([once(child, "exit"), sleep(2000)]);
}

async function run() {
  const port = 3110;
  const hfMock = await startHFMockServer();
  const env = {
    ...process.env,
    MCG_WEB_HF_BASE_URL: hfMock.baseURL,
    MCG_FAIRNESS_SCRIPT: process.env.MCG_FAIRNESS_SCRIPT || "tests/fixtures/fairness_stub.py",
    MCG_CARBON_FIXTURE:
      process.env.MCG_CARBON_FIXTURE || "tests/fixtures/carbon/carbon_fixture.json"
  };

  const app = spawn("npm", ["run", "start", "--", "-p", String(port)], {
    env,
    stdio: ["ignore", "pipe", "pipe"]
  });

  let appLogs = "";
  app.stdout.on("data", (chunk) => {
    appLogs += chunk.toString();
  });
  app.stderr.on("data", (chunk) => {
    appLogs += chunk.toString();
  });

  try {
    await waitForReady(`http://127.0.0.1:${port}/en`);

    const custom = await postGenerate(port, {
      locale: "en",
      source: "custom",
      model: "demo-model",
      metadataFile: "tests/fixtures/custom_metadata.json",
      evalFile: "examples/eval_sample.csv",
      template: "standard",
      compliance: "eu-ai-act,nist"
    });
    if (custom.status !== 200 || !custom.body?.card?.carbon) {
      throw new Error(`custom flow failed: status=${custom.status} body=${JSON.stringify(custom.body)}`);
    }

    const hf = await postGenerate(port, {
      locale: "en",
      source: "hf",
      model: "hf-mock-model",
      evalFile: "examples/eval_sample.csv",
      template: "standard",
      compliance: "eu-ai-act,nist"
    });
    if (hf.status !== 200 || !hf.body?.card?.carbon) {
      throw new Error(`hf flow failed: status=${hf.status} body=${JSON.stringify(hf.body)}`);
    }

    const badWandb = await postGenerate(port, {
      locale: "en",
      source: "wandb",
      model: "acme/project",
      evalFile: "examples/eval_sample.csv",
      template: "standard",
      compliance: "eu-ai-act,nist"
    });
    if (
      badWandb.status !== 400 ||
      !String(badWandb.body?.error || "").includes("invalid --model for wandb source")
    ) {
      throw new Error(`wandb validation failed: status=${badWandb.status} body=${JSON.stringify(badWandb.body)}`);
    }

    const badMLflow = await postGenerate(port, {
      locale: "en",
      source: "mlflow",
      model: "abc123",
      evalFile: "examples/eval_sample.csv",
      template: "standard",
      compliance: "eu-ai-act,nist"
    });
    if (
      badMLflow.status !== 400 ||
      !String(badMLflow.body?.error || "").includes("invalid --model for mlflow source")
    ) {
      throw new Error(
        `mlflow validation failed: status=${badMLflow.status} body=${JSON.stringify(badMLflow.body)}`
      );
    }

    console.log(
      JSON.stringify(
        {
          custom: custom.status,
          hf: hf.status,
          badWandb: badWandb.status,
          badMLflow: badMLflow.status
        },
        null,
        2
      )
    );
  } finally {
    await stopChildProcess(app);
    hfMock.server.closeIdleConnections?.();
    hfMock.server.closeAllConnections?.();
    await new Promise((resolve) => hfMock.server.close(resolve));
    if (process.env.DEBUG_WEB_SMOKE === "1") {
      console.log(appLogs);
    }
  }
}

run()
  .then(() => {
    process.exit(0);
  })
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
