// Copyright (c) 2026 Katsushi Kagaya. Licensed under the MIT License.
package main

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

func csvBytes(header []string, rows [][]string) ([]byte, error) {
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	if err := w.Write(header); err != nil {
		return nil, err
	}
	if err := w.WriteAll(rows); err != nil {
		return nil, err
	}
	w.Flush()
	return b.Bytes(), w.Error()
}

func addZipBytes(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func exportBundle(store *Store, l *Ledger, output string, includePDFs bool) error {
	abs, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(abs), ".citation-ledger-*.zip")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	zw := zip.NewWriter(tmp)

	ledgerJSON, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	if err := addZipBytes(zw, "ledger.json", append(ledgerJSON, '\n')); err != nil {
		return err
	}

	settings, _ := csvBytes([]string{"key", "value"}, [][]string{{"project_name", l.ProjectName}, {"format_version", strconv.Itoa(l.FormatVersion)}})
	if err := addZipBytes(zw, "settings.csv", settings); err != nil {
		return err
	}

	var paperRows [][]string
	for _, p := range l.Papers {
		paperRows = append(paperRows, []string{p.ID, p.Title, p.Authors, strconv.Itoa(p.Year), p.Venue, p.DOI, p.Tags, p.Status, p.PDFPath, p.DriveURL, p.Notes, p.CreatedAt})
	}
	papers, _ := csvBytes([]string{"id", "title", "authors", "year", "venue", "doi", "tags", "status", "pdf_path", "drive_url", "notes", "created_at"}, paperRows)
	if err := addZipBytes(zw, "papers.csv", papers); err != nil {
		return err
	}

	var refRows [][]string
	for _, r := range l.References {
		refRows = append(refRows, []string{strconv.FormatInt(r.ID, 10), r.SourcePaperID, strconv.Itoa(r.Ordinal), r.RawText, r.DOI, strconv.Itoa(r.Year), r.FirstAuthor, r.MatchedPaperID, strconv.FormatFloat(r.Confidence, 'f', 6, 64), r.Status})
	}
	refs, _ := csvBytes([]string{"id", "source_paper_id", "ordinal", "raw_text", "doi", "year", "first_author", "matched_paper_id", "confidence", "status"}, refRows)
	if err := addZipBytes(zw, "raw_references.csv", refs); err != nil {
		return err
	}

	var edgeRows [][]string
	for _, c := range l.Citations {
		edgeRows = append(edgeRows, []string{c.SourcePaperID, c.TargetPaperID, strconv.FormatInt(c.RawReferenceID, 10), c.Status, c.CreatedAt})
	}
	edges, _ := csvBytes([]string{"source_paper_id", "target_paper_id", "raw_reference_id", "status", "created_at"}, edgeRows)
	if err := addZipBytes(zw, "citations.csv", edges); err != nil {
		return err
	}

	if includePDFs {
		for _, p := range l.Papers {
			if p.PDFPath == "" {
				continue
			}
			path := p.PDFPath
			if !filepath.IsAbs(path) {
				path = filepath.Join(store.Dir, path)
			}
			in, err := os.Open(path)
			if err != nil {
				continue
			}
			w, err := zw.Create(filepath.ToSlash(filepath.Join("pdfs", filepath.Base(path))))
			if err == nil {
				_, err = io.Copy(w, in)
			}
			in.Close()
			if err != nil {
				return err
			}
		}
	}

	if err := zw.Close(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, abs); err != nil {
		return fmt.Errorf("cannot save ZIP: %w", err)
	}
	return nil
}
