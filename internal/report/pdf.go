package report

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/meysam81/scry/core/model"
	"github.com/meysam81/scry/internal/logger"
)

// PDFReporter renders a CrawlResult as a PDF document by first generating
// an HTML report and then printing it to PDF via a headless browser.
type PDFReporter struct{}

// Name returns "pdf".
func (r *PDFReporter) Name() string { return "pdf" }

// Write generates a PDF report and writes it to w.
// Note: each invocation launches a new headless browser instance to render
// the HTML-to-PDF conversion. This is intentional as the browser is short-lived
// and isolated per report generation.
func (r *PDFReporter) Write(ctx context.Context, result *model.CrawlResult, w io.Writer) error {
	if result == nil {
		return nil
	}

	l := logger.FromContext(ctx)

	// 1. Generate HTML report to a temporary file.
	tmpDir, err := os.MkdirTemp("", "scry-pdf-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			l.Warn().Err(err).Msg("failed to remove temp dir")
		}
	}()

	htmlPath := filepath.Join(tmpDir, "report.html")
	htmlFile, err := os.Create(htmlPath)
	if err != nil {
		return fmt.Errorf("create temp html: %w", err)
	}

	htmlReporter := &HTMLReporter{}
	if err := htmlReporter.Write(ctx, result, htmlFile); err != nil {
		if closeErr := htmlFile.Close(); closeErr != nil {
			return fmt.Errorf("generate html: %w (also failed to close file: %w)", err, closeErr)
		}
		return fmt.Errorf("generate html: %w", err)
	}
	if err := htmlFile.Close(); err != nil {
		return fmt.Errorf("close temp html: %w", err)
	}

	// 2. Launch headless browser and convert HTML to PDF.
	controlURL, err := launcher.New().Headless(true).Launch()
	if err != nil {
		return fmt.Errorf("launch browser for PDF: %w", err)
	}

	browser := rod.New().ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		return fmt.Errorf("connect browser: %w", err)
	}
	defer func() {
		if err := browser.Close(); err != nil {
			l.Warn().Err(err).Msg("failed to close browser")
		}
	}()

	page, err := browser.Page(proto.TargetCreateTarget{URL: "file://" + htmlPath})
	if err != nil {
		return fmt.Errorf("open page: %w", err)
	}

	// Wait for page to fully render.
	if err := page.WaitStable(0); err != nil {
		return fmt.Errorf("wait for page stable: %w", err)
	}

	// 3. Generate PDF with reasonable margins.
	margin := 0.5
	printBg := true
	pdfReader, err := page.PDF(&proto.PagePrintToPDF{
		PrintBackground: printBg,
		MarginTop:       &margin,
		MarginBottom:    &margin,
		MarginLeft:      &margin,
		MarginRight:     &margin,
	})
	if err != nil {
		return fmt.Errorf("generate pdf: %w", err)
	}

	if _, err := io.Copy(w, pdfReader); err != nil {
		return fmt.Errorf("write pdf: %w", err)
	}

	return nil
}
