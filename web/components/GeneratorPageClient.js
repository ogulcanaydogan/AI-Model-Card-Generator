"use client";

import { useMemo, useState } from "react";
import { normalizeSource, validateGeneratePayload } from "@/lib/sourceValidation";
import { buildNISTFunctionSections, summarizeNISTOverall } from "@/lib/nistSections";
import { normalizeTemplateSource, validateTemplateSelection } from "@/lib/templateValidation";

const DEFAULT_FORM = {
  source: "custom",
  model: "demo-model",
  evalFile: "examples/eval_sample.csv",
  metadataFile: "tests/fixtures/custom_metadata.json",
  templateSource: "built-in",
  template: "standard",
  templateFile: "tests/fixtures/batch/custom-template.tmpl",
  templateInitName: "Web Custom Template",
  templateInitBase: "standard",
  templateInitOut: "artifacts/web/templates/web-custom.tmpl",
  templatePreviewCard: "tests/fixtures/strict_fail_model_card.json",
  templatePreviewOut: "artifacts/web/templates/web-preview.md",
  compliance: "eu-ai-act,nist,iso42001"
};

const COMPLIANCE_TABS = ["eu-ai-act", "nist", "iso42001"];

function ArrayList({ items, fallback }) {
  if (!items || items.length === 0) {
    return <p className="muted">{fallback}</p>;
  }
  return (
    <ul className="list">
      {items.map((item, idx) => (
        <li key={`${item}-${idx}`}>{item}</li>
      ))}
    </ul>
  );
}

function formatScoreContribution(value) {
  if (value === 0) {
    return "0";
  }
  return value > 0 ? `+${value}` : `${value}`;
}

export default function GeneratorPageClient({ locale, dict }) {
  const [form, setForm] = useState(DEFAULT_FORM);
  const [result, setResult] = useState(null);
  const [templateAction, setTemplateAction] = useState(null);
  const [error, setError] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [isTemplateLoading, setIsTemplateLoading] = useState(false);
  const [activeTab, setActiveTab] = useState("nist");

  const complianceMap = useMemo(() => {
    const entries = result?.card?.compliance || [];
    return Object.fromEntries(entries.map((entry) => [entry.framework, entry]));
  }, [result]);

  const activeReport = complianceMap[activeTab];
  const nistReport = complianceMap.nist;
  const nistSections = useMemo(() => buildNISTFunctionSections(nistReport), [nistReport]);
  const nistOverall = useMemo(() => summarizeNISTOverall(nistReport), [nistReport]);

  const carbon = result?.card?.carbon;
  const normalizedSource = normalizeSource(form.source);
  const normalizedTemplateSource = normalizeTemplateSource(form.templateSource);

  const onChange = (event) => {
    const { name, value } = event.target;
    setForm((prev) => ({ ...prev, [name]: value }));
  };

  const runTemplateAction = async (action) => {
    const endpointByAction = {
      init: "/api/template/init",
      validate: "/api/template/validate",
      preview: "/api/template/preview"
    };
    const payloadByAction = {
      init: {
        name: form.templateInitName,
        base: form.templateInitBase,
        out: form.templateInitOut
      },
      validate: {
        input: form.templateFile
      },
      preview: {
        input: form.templateFile,
        card: form.templatePreviewCard,
        out: form.templatePreviewOut
      }
    };

    const endpoint = endpointByAction[action];
    const payload = payloadByAction[action];
    if (!endpoint || !payload) {
      return;
    }

    setError("");
    setIsTemplateLoading(true);
    try {
      const response = await fetch(endpoint, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload)
      });
      const body = await response.json();
      if (!response.ok) {
        throw new Error(body?.error || "Template action failed");
      }
      setTemplateAction({ action, ...body });
      if (action === "init" && body?.outputPath) {
        setForm((prev) => ({
          ...prev,
          templateFile: body.outputPath,
          templateSource: "template-file"
        }));
      }
    } catch (err) {
      setTemplateAction(null);
      setError(err.message);
    } finally {
      setIsTemplateLoading(false);
    }
  };

  const onSubmit = async (event) => {
    event.preventDefault();
    setIsLoading(true);
    setError("");
    const validationError = validateGeneratePayload(form);
    if (validationError) {
      setIsLoading(false);
      setError(validationError);
      return;
    }
    const templateValidationError = validateTemplateSelection(form);
    if (templateValidationError) {
      setIsLoading(false);
      setError(templateValidationError);
      return;
    }
    try {
      const response = await fetch("/api/generate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          locale,
          ...form
        })
      });
      const payload = await response.json();
      if (!response.ok) {
        throw new Error(payload?.error || "Failed to generate");
      }
      setResult(payload);
      const available = COMPLIANCE_TABS.find((tab) => payload?.card?.compliance?.some((item) => item.framework === tab));
      if (available) {
        setActiveTab(available);
      }
    } catch (err) {
      setResult(null);
      setError(err.message);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <section className="page-grid">
      <article className="panel form-panel">
        <h1>{dict.appName}</h1>
        <p className="subtitle">{dict.subtitle}</p>
        <p className="badge">{dict.note}</p>

        <form onSubmit={onSubmit} className="form-grid">
          <label>
            <span>{dict.source}</span>
            <select name="source" value={form.source} onChange={onChange}>
              <option value="custom">custom</option>
              <option value="hf">hf</option>
              <option value="wandb">wandb</option>
              <option value="mlflow">mlflow</option>
            </select>
          </label>

          <label>
            <span>{dict.model}</span>
            <input name="model" value={form.model} onChange={onChange} required />
          </label>

          {normalizedSource === "custom" ? (
            <label>
              <span>{dict.metadataFile}</span>
              <input name="metadataFile" value={form.metadataFile} onChange={onChange} required />
            </label>
          ) : null}

          <p className="hint">
            {normalizedSource === "custom" ? dict.modelHintCustom : null}
            {normalizedSource === "hf" ? dict.modelHintHF : null}
            {normalizedSource === "wandb" ? dict.modelHintWandB : null}
            {normalizedSource === "mlflow" ? dict.modelHintMLflow : null}
          </p>

          <label>
            <span>{dict.evalFile}</span>
            <input name="evalFile" value={form.evalFile} onChange={onChange} required />
          </label>

          <label>
            <span>{dict.templateSource}</span>
            <select name="templateSource" value={form.templateSource} onChange={onChange}>
              <option value="built-in">{dict.templateSourceBuiltIn}</option>
              <option value="template-file">{dict.templateSourceFile}</option>
            </select>
          </label>

          <label>
            <span>{dict.template}</span>
            <select
              name="template"
              value={form.template}
              onChange={onChange}
              disabled={normalizedTemplateSource !== "built-in"}
            >
              <option value="standard">standard</option>
              <option value="eu-ai-act">eu-ai-act</option>
              <option value="minimal">minimal</option>
            </select>
          </label>

          {normalizedTemplateSource === "template-file" ? (
            <label>
              <span>{dict.templateFile}</span>
              <input name="templateFile" value={form.templateFile} onChange={onChange} required />
            </label>
          ) : null}

          <label>
            <span>{dict.compliance}</span>
            <input name="compliance" value={form.compliance} onChange={onChange} />
          </label>

          <button type="submit" disabled={isLoading}>
            {isLoading ? dict.generating : dict.generate}
          </button>
        </form>

        <section className="details">
          <h3>{dict.templateActions}</h3>
          <label>
            <span>{dict.templateInitName}</span>
            <input
              name="templateInitName"
              value={form.templateInitName}
              onChange={onChange}
              disabled={isTemplateLoading}
            />
          </label>
          <label>
            <span>{dict.templateInitBase}</span>
            <select
              name="templateInitBase"
              value={form.templateInitBase}
              onChange={onChange}
              disabled={isTemplateLoading}
            >
              <option value="standard">standard</option>
              <option value="eu-ai-act">eu-ai-act</option>
              <option value="minimal">minimal</option>
            </select>
          </label>
          <label>
            <span>{dict.templateInitOut}</span>
            <input
              name="templateInitOut"
              value={form.templateInitOut}
              onChange={onChange}
              disabled={isTemplateLoading}
            />
          </label>
          <label>
            <span>{dict.templateFile}</span>
            <input
              name="templateFile"
              value={form.templateFile}
              onChange={onChange}
              disabled={isTemplateLoading}
            />
          </label>
          <label>
            <span>{dict.templatePreviewCard}</span>
            <input
              name="templatePreviewCard"
              value={form.templatePreviewCard}
              onChange={onChange}
              disabled={isTemplateLoading}
            />
          </label>
          <label>
            <span>{dict.templatePreviewOut}</span>
            <input
              name="templatePreviewOut"
              value={form.templatePreviewOut}
              onChange={onChange}
              disabled={isTemplateLoading}
            />
          </label>

          <div className="inline-actions">
            <button type="button" onClick={() => runTemplateAction("init")} disabled={isTemplateLoading}>
              {dict.initTemplate}
            </button>
            <button
              type="button"
              onClick={() => runTemplateAction("validate")}
              disabled={isTemplateLoading}
            >
              {dict.validateTemplate}
            </button>
            <button
              type="button"
              onClick={() => runTemplateAction("preview")}
              disabled={isTemplateLoading}
            >
              {dict.previewTemplate}
            </button>
          </div>
        </section>

        {error ? <p className="error">{error}</p> : null}
      </article>

      <article className="panel preview-panel">
        <h2>{dict.preview}</h2>
        {!result ? <p className="muted">{dict.noData}</p> : null}

        {result ? (
          <>
            <section className="card-grid">
              <div className="mini-card">
                <h3>{dict.carbonPreview}</h3>
                <p>
                  {carbon?.estimated_kg_co2e ?? 0} kgCO2e ({carbon?.method || "unavailable"})
                </p>
              </div>
              <div className="mini-card">
                <h3>{dict.overallComplianceState}</h3>
                <p>
                  {dict.status}: <strong>{activeReport?.status || "n/a"}</strong>
                </p>
                <p>
                  {dict.score}: <strong>{activeReport?.score ?? "n/a"}</strong>
                </p>
                <p>
                  {dict.requiredCount}: <strong>{(activeReport?.required_gaps || []).length}</strong>
                </p>
                <p>
                  {dict.advisoryCount}: <strong>{(activeReport?.findings || []).length}</strong>
                </p>
                {activeTab === "nist" ? (
                  <p>
                    {dict.controlCoverage}: <strong>{nistOverall.controlCoverage}</strong>
                  </p>
                ) : null}
              </div>
            </section>

            <section className="tabs" aria-label={dict.complianceTabs}>
              {COMPLIANCE_TABS.map((tab) => {
                const label =
                  tab === "eu-ai-act"
                    ? dict.euAiActTab
                    : tab === "nist"
                      ? dict.nistPreview
                      : dict.iso42001Tab;
                const isActive = activeTab === tab;
                return (
                  <button
                    key={tab}
                    type="button"
                    className={isActive ? "tab active" : "tab"}
                    onClick={() => setActiveTab(tab)}
                  >
                    {label}
                  </button>
                );
              })}
            </section>

            <section className="details">
              <h3>{dict.requiredGaps}</h3>
              <ArrayList items={activeReport?.required_gaps} fallback={dict.noData} />
              <h3>{dict.findings}</h3>
              <ArrayList items={activeReport?.findings} fallback={dict.noData} />
              <h3>{dict.recommendations}</h3>
              <ArrayList items={activeReport?.recommended_actions} fallback={dict.noData} />
            </section>

            {activeTab === "nist" ? (
              <section className="details">
                <h3>{dict.nistFunctionBreakdown}</h3>
                <p className="muted">
                  {dict.status}: <strong>{nistOverall.status}</strong> | {dict.score}:{" "}
                  <strong>{nistOverall.score ?? "n/a"}</strong> | {dict.requiredCount}:{" "}
                  <strong>{nistOverall.requiredCount}</strong> | {dict.advisoryCount}:{" "}
                  <strong>{nistOverall.advisoryCount}</strong> | {dict.controlCoverage}:{" "}
                  <strong>{nistOverall.controlCoverage}</strong>
                </p>
                <div className="nist-grid">
                  {nistSections.map((section) => (
                    <article key={section.functionName} className="nist-card">
                      <div className="nist-card-header">
                        <h4>{dict[`nist${section.functionName}`] || section.functionName}</h4>
                        <span className={`status-badge ${section.status}`}>
                          {section.status}
                        </span>
                      </div>
                      <p>
                        {dict.scoreContribution}:{" "}
                        <strong>{formatScoreContribution(section.scoreContribution)}</strong>
                      </p>
                      <p>
                        {dict.controlCoverage}: <strong>{section.controlCoverage}</strong> (
                        {section.totalControls} {dict.totalControls})
                      </p>
                      <p>
                        {dict.requiredCount}: <strong>{section.requiredCount}</strong> |{" "}
                        {dict.advisoryCount}: <strong>{section.advisoryCount}</strong>
                      </p>
                      <h5>{dict.shortRemediation}</h5>
                      <ArrayList items={section.shortRemediations} fallback={dict.noData} />
                      <h5>{dict.requiredGaps}</h5>
                      <ArrayList items={section.requiredGaps} fallback={dict.noData} />
                      <h5>{dict.findings}</h5>
                      <ArrayList items={section.findings} fallback={dict.noData} />
                      <h5>{dict.recommendations}</h5>
                      <ArrayList items={section.recommendedActions} fallback={dict.noData} />
                    </article>
                  ))}
                </div>
              </section>
            ) : null}

            <section className="details">
              <h3>{dict.outputs}</h3>
              <pre>{JSON.stringify(result.files, null, 2)}</pre>
            </section>

            <section className="details">
              <h3>{dict.markdownPreview}</h3>
              <pre className="markdown-preview">{result.markdown || dict.noData}</pre>
            </section>

            <section className="details">
              <h3>{dict.logs}</h3>
              <pre>{result.logs || dict.noData}</pre>
            </section>
          </>
        ) : null}

        <section className="details">
          <h3>{dict.templateActionResult}</h3>
          {!templateAction ? <p className="muted">{dict.templateActionNone}</p> : null}
          {templateAction?.action ? (
            <p>
              <strong>{templateAction.action}</strong>
            </p>
          ) : null}
          {templateAction?.outputPath ? <p>{templateAction.outputPath}</p> : null}
          {templateAction?.logs ? <pre>{templateAction.logs}</pre> : null}
          {templateAction?.markdown ? (
            <pre className="markdown-preview">{templateAction.markdown}</pre>
          ) : null}
        </section>
      </article>
    </section>
  );
}
