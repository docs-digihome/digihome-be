package pkg

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func IsPDF(fileName string, reader io.ReadSeeker) (bool, error) {
	if !strings.EqualFold(filepath.Ext(fileName), ".pdf") {
		return false, nil
	}
	buf := make([]byte, 1024)
	n, err := reader.Read(buf)
	if err != nil && err != io.EOF {
		return false, err
	}
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return false, err
	}
	return bytes.Contains(buf[:n], []byte("%PDF-")), nil
}

const (
	ocrLang             = "eng+ind"
	confidenceThreshold = 0.65
	ocrTimeout          = 120 * time.Second
)

type PdfClass struct {
	Kind       string
	Pages      int
	Confidence float64
	NeedOCR    int // -1 if not present (e.g. Digital), otherwise pages needing OCR from "Mixed" output
}

var pdfInfoRegex = regexp.MustCompile(
	`(?i)^([A-Za-z]+)\s+\((\d+)\s+pages?,\s+confidence:\s+([0-9.]+)(?:,\s*(\d+)\s+pages?\s+need\s+OCR)?\)$`,
)

func ClassifyPDF(ctx context.Context, path string) (PdfClass, error) {
	cmd := exec.CommandContext(ctx,
		"pdf-inspector",
		"detect",
		path,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return PdfClass{}, fmt.Errorf("pdf-inspector detect failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	result := strings.TrimSpace(string(output))
	matches := pdfInfoRegex.FindStringSubmatch(result)
	if len(matches) == 0 {
		return PdfClass{}, fmt.Errorf("unparsable detect output: %q", result)
	}
	pages, err := strconv.Atoi(matches[2])
	if err != nil {
		return PdfClass{}, fmt.Errorf("invalid page count %q: %w", matches[2], err)
	}
	confidence, err := strconv.ParseFloat(matches[3], 64)
	if err != nil {
		return PdfClass{}, fmt.Errorf("invalid confidence %q: %w", matches[3], err)
	}
	needOCR := -1
	if len(matches) >= 5 && matches[4] != "" {
		if n, err := strconv.Atoi(matches[4]); err == nil {
			needOCR = n
		}
	}
	return PdfClass{
		Kind:       matches[1],
		Pages:      pages,
		Confidence: confidence,
		NeedOCR:    needOCR,
	}, nil
}

func isDocumentBased(c PdfClass) bool {
	kindOK := strings.EqualFold(c.Kind, "Digital") ||
		strings.EqualFold(c.Kind, "TextBased") ||
		strings.EqualFold(c.Kind, "Text")
	return kindOK && c.Confidence >= confidenceThreshold
}

func ocrPDF(ctx context.Context, src string) (string, error) {
	tmpFile, err := os.CreateTemp("", "ocr-*.pdf")
	if err != nil {
		return "", fmt.Errorf("create temp ocr file: %w", err)
	}
	dst := tmpFile.Name()
	tmpFile.Close()

	ocrCtx, cancel := context.WithTimeout(ctx, ocrTimeout)
	defer cancel()

	cmd := exec.CommandContext(ocrCtx,
		"ocrmypdf",
		"-l", ocrLang,
		"--skip-text",
		"--output-type", "pdf",
		"--optimize", "1",
		"--jobs", "1",
		"--quiet",
		src,
		dst,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		os.Remove(dst)
		if ocrCtx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("ocrmypdf timed out after %s: %w: %s", ocrTimeout, err, strings.TrimSpace(string(output)))
		}
		return "", fmt.Errorf("ocrmypdf failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return dst, nil
}

func inspectDirect(ctx context.Context, path string) ([]byte, error) {
	cmd := exec.CommandContext(ctx,
		"pdf-inspector",
		path,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("pdf-inspector failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func inspectDirectByPage(ctx context.Context, path string, page int) ([]byte, error) {
	cmd := exec.CommandContext(ctx,
		"pdf-inspector",
		path,
		"--pages",
		strconv.Itoa(page),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("pdf-inspector failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func InspectPDF(path string) ([]byte, error) {
	return InspectPDFContext(context.Background(), path)
}

func InspectPDFContext(ctx context.Context, path string) ([]byte, error) {
	class, err := ClassifyPDF(ctx, path)
	if err != nil {
		// fallback to OCR on classification failure (safer than failing ingest)
		ocrPath, ocrErr := ocrPDF(ctx, path)
		if ocrErr != nil {
			// OCR can fail on corrupt embedded images (e.g. "Not a JPEG file: starts with 0x48 0x89");
			// degrade gracefully to direct inspect so text-based pages are still ingested.
			if out, directErr := inspectDirect(ctx, path); directErr == nil {
				return out, nil
			}
			return nil, fmt.Errorf("classify failed (%v) and ocr fallback failed: %w", err, ocrErr)
		}
		defer os.Remove(ocrPath)
		if out, directErr := inspectDirect(ctx, ocrPath); directErr == nil {
			return out, nil
		} else if out2, fallbackErr := inspectDirect(ctx, path); fallbackErr == nil {
			return out2, nil
		} else {
			return nil, directErr
		}
	}

	if isDocumentBased(class) {
		return inspectDirect(ctx, path)
	}

	ocrPath, err := ocrPDF(ctx, path)
	if err != nil {
		// Mixed / scanned PDFs where OCR fails should still ingest via direct inspect
		if out, directErr := inspectDirect(ctx, path); directErr == nil {
			return out, nil
		}
		return nil, err
	}
	defer os.Remove(ocrPath)
	if out, directErr := inspectDirect(ctx, ocrPath); directErr == nil {
		return out, nil
	} else if out2, fallbackErr := inspectDirect(ctx, path); fallbackErr == nil {
		return out2, nil
	} else {
		return nil, directErr
	}
}

func InspectPDFByPage(path string, page int) ([]byte, error) {
	return InspectPDFByPageContext(context.Background(), path, page)
}

func InspectPDFByPageContext(ctx context.Context, path string, page int) ([]byte, error) {
	class, err := ClassifyPDF(ctx, path)
	if err != nil {
		ocrPath, ocrErr := ocrPDF(ctx, path)
		if ocrErr != nil {
			if out, directErr := inspectDirectByPage(ctx, path, page); directErr == nil {
				return out, nil
			}
			return nil, fmt.Errorf("classify failed (%v) and ocr fallback failed: %w", err, ocrErr)
		}
		defer os.Remove(ocrPath)
		if out, directErr := inspectDirectByPage(ctx, ocrPath, page); directErr == nil {
			return out, nil
		} else if out2, fallbackErr := inspectDirectByPage(ctx, path, page); fallbackErr == nil {
			return out2, nil
		} else {
			return nil, directErr
		}
	}

	if isDocumentBased(class) {
		return inspectDirectByPage(ctx, path, page)
	}

	ocrPath, err := ocrPDF(ctx, path)
	if err != nil {
		if out, directErr := inspectDirectByPage(ctx, path, page); directErr == nil {
			return out, nil
		}
		return nil, err
	}
	defer os.Remove(ocrPath)
	if out, directErr := inspectDirectByPage(ctx, ocrPath, page); directErr == nil {
		return out, nil
	} else if out2, fallbackErr := inspectDirectByPage(ctx, path, page); fallbackErr == nil {
		return out2, nil
	} else {
		return nil, directErr
	}
}

func GetTotalPage(path string) (int, error) {
	return GetTotalPageContext(context.Background(), path)
}

func GetTotalPageContext(ctx context.Context, path string) (int, error) {
	class, err := ClassifyPDF(ctx, path)
	if err != nil {
		return -1, err
	}
	return class.Pages, nil
}


