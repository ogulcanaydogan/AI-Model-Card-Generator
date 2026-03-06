import Link from "next/link";
import { SUPPORTED_LOCALES, normalizeLocale } from "@/lib/i18n";

export default function LocaleLayout({ children, params }) {
  const locale = normalizeLocale(params.locale);

  return (
    <div className="app-shell">
      <header className="topbar">
        <div className="brand">MCG Web</div>
        <nav className="locale-nav" aria-label="Language switcher">
          {SUPPORTED_LOCALES.map((code) => (
            <Link
              key={code}
              className={code === locale ? "locale-link active" : "locale-link"}
              href={`/${code}`}
            >
              {code.toUpperCase()}
            </Link>
          ))}
        </nav>
      </header>
      <main className="content">{children}</main>
    </div>
  );
}
