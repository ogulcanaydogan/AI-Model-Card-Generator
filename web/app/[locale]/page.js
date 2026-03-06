import GeneratorPageClient from "@/components/GeneratorPageClient";
import { getDictionary, normalizeLocale } from "@/lib/i18n";

export default function LocalePage({ params }) {
  const locale = normalizeLocale(params.locale);
  const dictionary = getDictionary(locale);
  return <GeneratorPageClient locale={locale} dict={dictionary} />;
}
