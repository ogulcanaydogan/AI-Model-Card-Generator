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
    nistPreview: "NIST",
    noData: "No data yet.",
    status: "Status",
    score: "Score",
    findings: "Findings",
    requiredGaps: "Required Gaps",
    recommendations: "Recommendations",
    outputs: "Generated Files",
    logs: "CLI Output",
    note: "Current web sprint supports custom source flow first."
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
    nistPreview: "NIST",
    noData: "Henüz veri yok.",
    status: "Durum",
    score: "Skor",
    findings: "Bulgular",
    requiredGaps: "Zorunlu Eksikler",
    recommendations: "Öneriler",
    outputs: "Üretilen Dosyalar",
    logs: "CLI Çıktısı",
    note: "Bu web sprintinde önce custom source akışı destekleniyor."
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
