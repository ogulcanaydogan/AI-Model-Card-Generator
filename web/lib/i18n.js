export const SUPPORTED_LOCALES = ["en", "tr"];
export const DEFAULT_LOCALE = "en";

const dictionaries = {
  en: {
    appName: "AI Model Card Generator",
    subtitle: "Generate model cards and preview Carbon + NIST sections.",
    localeLabel: "Language",
    source: "Source",
    model: "Model ID",
    evalFile: "Eval CSV path",
    metadataFile: "Custom metadata JSON path",
    template: "Template",
    compliance: "Compliance frameworks",
    generate: "Generate Preview",
    generating: "Generating...",
    preview: "Preview",
    markdownPreview: "Markdown Preview",
    carbonPreview: "Carbon",
    compliancePreview: "Compliance",
    overallComplianceState: "Overall Compliance State",
    complianceTabs: "Compliance tabs",
    euAiActTab: "EU AI Act",
    nistPreview: "NIST",
    iso42001Tab: "ISO42001",
    nistFunctionBreakdown: "NIST Function Breakdown",
    scoreContribution: "Score Contribution",
    requiredCount: "Required Gaps",
    advisoryCount: "Advisory Findings",
    nistGOVERN: "GOVERN",
    nistMAP: "MAP",
    nistMEASURE: "MEASURE",
    nistMANAGE: "MANAGE",
    noData: "No data yet.",
    status: "Status",
    score: "Score",
    findings: "Findings",
    requiredGaps: "Required Gaps",
    recommendations: "Recommendations",
    outputs: "Generated Files",
    logs: "CLI Output",
    note: "Source parity is enabled for custom/hf/wandb/mlflow preview flow.",
    modelHintCustom: "Custom model id can be any label (example: demo-model).",
    modelHintHF: "Hugging Face expects a model id (example: bert-base-uncased).",
    modelHintWandB: "W&B requires <entity>/<project>/<run_id>.",
    modelHintMLflow: "MLflow requires run:<run_id>."
  },
  tr: {
    appName: "AI Model Card Generator",
    subtitle: "Model kart üret ve Carbon + NIST bölümlerini önizle.",
    localeLabel: "Dil",
    source: "Kaynak",
    model: "Model ID",
    evalFile: "Eval CSV yolu",
    metadataFile: "Custom metadata JSON yolu",
    template: "Şablon",
    compliance: "Uyumluluk çerçeveleri",
    generate: "Önizleme Üret",
    generating: "Üretiliyor...",
    preview: "Önizleme",
    markdownPreview: "Markdown Önizleme",
    carbonPreview: "Carbon",
    compliancePreview: "Uyumluluk",
    overallComplianceState: "Genel Uyumluluk Durumu",
    complianceTabs: "Uyumluluk sekmeleri",
    euAiActTab: "EU AI Act",
    nistPreview: "NIST",
    iso42001Tab: "ISO42001",
    nistFunctionBreakdown: "NIST Fonksiyon Kırılımı",
    scoreContribution: "Skor Katkısı",
    requiredCount: "Zorunlu Eksik",
    advisoryCount: "Danışmanlık Bulgusu",
    nistGOVERN: "GOVERN",
    nistMAP: "MAP",
    nistMEASURE: "MEASURE",
    nistMANAGE: "MANAGE",
    noData: "Henüz veri yok.",
    status: "Durum",
    score: "Skor",
    findings: "Bulgular",
    requiredGaps: "Zorunlu Eksikler",
    recommendations: "Öneriler",
    outputs: "Üretilen Dosyalar",
    logs: "CLI Çıktısı",
    note: "Kaynak paritesi custom/hf/wandb/mlflow önizleme akışı için açık.",
    modelHintCustom: "Custom model id herhangi bir etiket olabilir (örnek: demo-model).",
    modelHintHF: "Hugging Face model id bekler (örnek: bert-base-uncased).",
    modelHintWandB: "W&B için format <entity>/<project>/<run_id> olmalı.",
    modelHintMLflow: "MLflow için format run:<run_id> olmalı."
  }
};

export function normalizeLocale(locale) {
  if (SUPPORTED_LOCALES.includes(locale)) {
    return locale;
  }
  return DEFAULT_LOCALE;
}

export function getDictionary(locale) {
  return dictionaries[normalizeLocale(locale)];
}
