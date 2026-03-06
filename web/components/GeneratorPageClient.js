"use client";

import { useMemo, useState } from "react";

const DEFAULT_FORM = {
  source: "custom",
  model: "demo-model",
  evalFile: "examples/eval_sample.csv",
  metadataFile: "tests/fixtures/custom_metadata.json",
  template: "standard",
  compliance: "eu-ai-act,nist"
};

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

export default function GeneratorPageClient({ locale, dict }) {
  const [form, setForm] = useState(DEFAULT_FORM);
  const [result, setResult] = useState(null);
  const [error, setError] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  const nist = useMemo(
    () => result?.card?.compliance?.find((item) => item.framework === "nist"),
    [result]
  );

  const carbon = result?.card?.carbon;

  const onChange = (event) => {
    const { name, value } = event.target;
    setForm((prev) => ({ ...prev, [name]: value }));
  };

  const onSubmit = async (event) => {
    event.preventDefault();
    setIsLoading(true);
    setError("");
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
            </select>
          </label>

          <label>
            <span>{dict.model}</span>
            <input name="model" value={form.model} onChange={onChange} required />
          </label>

          <label>
            <span>{dict.metadataFile}</span>
            <input name="metadataFile" value={form.metadataFile} onChange={onChange} required />
          </label>

          <label>
            <span>{dict.evalFile}</span>
            <input name="evalFile" value={form.evalFile} onChange={onChange} required />
          </label>

          <label>
            <span>{dict.template}</span>
            <select name="template" value={form.template} onChange={onChange}>
              <option value="standard">standard</option>
              <option value="eu-ai-act">eu-ai-act</option>
              <option value="minimal">minimal</option>
            </select>
          </label>

          <label>
            <span>{dict.compliance}</span>
            <input name="compliance" value={form.compliance} onChange={onChange} />
          </label>

          <button type="submit" disabled={isLoading}>
            {isLoading ? dict.generating : dict.generate}
          </button>
        </form>

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
                <h3>{dict.nistPreview}</h3>
                <p>
                  {dict.status}: <strong>{nist?.status || "n/a"}</strong>
                </p>
                <p>
                  {dict.score}: <strong>{nist?.score ?? "n/a"}</strong>
                </p>
              </div>
            </section>

            <section className="details">
              <h3>{dict.requiredGaps}</h3>
              <ArrayList items={nist?.required_gaps} fallback={dict.noData} />
              <h3>{dict.findings}</h3>
              <ArrayList items={nist?.findings} fallback={dict.noData} />
              <h3>{dict.recommendations}</h3>
              <ArrayList items={nist?.recommended_actions} fallback={dict.noData} />
            </section>

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
      </article>
    </section>
  );
}
