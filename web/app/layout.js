import "./globals.css";

export const metadata = {
  title: "AI Model Card Generator",
  description: "Web UI for model card generation, validation and compliance preview."
};

export default function RootLayout({ children }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
