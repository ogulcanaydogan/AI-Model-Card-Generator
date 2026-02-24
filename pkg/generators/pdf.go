package generators

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/yapay/ai-model-card-generator/pkg/core"
)

// PDFGenerator renders HTML and exports it to PDF with headless Chromium.
type PDFGenerator struct{}

func (g *PDFGenerator) Format() string {
	return "pdf"
}

func (g *PDFGenerator) Generate(ctx context.Context, card core.ModelCard, templatePath, outPath string) error {
	htmlContent, err := renderHTML(card, templatePath)
	if err != nil {
		return fmt.Errorf("render HTML for PDF: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("create PDF output dir: %w", err)
	}

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)...)
	defer cancel()

	browserCtx, cancelBrowser := chromedp.NewContext(allocCtx)
	defer cancelBrowser()

	dataURI := "data:text/html;base64," + base64.StdEncoding.EncodeToString([]byte(htmlContent))
	var pdfBuf []byte

	if err := chromedp.Run(browserCtx,
		chromedp.Navigate(dataURI),
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, _, err := page.PrintToPDF().WithPrintBackground(true).Do(ctx)
			if err != nil {
				return fmt.Errorf("print to pdf: %w", err)
			}
			pdfBuf = buf
			return nil
		}),
	); err != nil {
		return fmt.Errorf("run chromium for PDF (ensure Chromium/Chrome is installed): %w", err)
	}

	if err := os.WriteFile(outPath, pdfBuf, 0o644); err != nil {
		return fmt.Errorf("write pdf: %w", err)
	}
	return nil
}
