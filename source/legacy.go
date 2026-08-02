// Copyright (c) 2026 Katsushi Kagaya. Licensed under the MIT License.
package main

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func csvRowsFromZip(zr *zip.ReadCloser, name string) ([]map[string]string, error) {
	var target *zip.File
	for _, f := range zr.File {
		if f.Name == name {
			target = f
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("%s がZIP内にありません", name)
	}
	rc, err := target.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	r := csv.NewReader(rc)
	r.FieldsPerRecord = -1
	header, err := r.Read()
	if err == io.EOF {
		return []map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	var rows []map[string]string
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		row := map[string]string{}
		for i, key := range header {
			if i < len(record) {
				row[key] = record[i]
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

func atoi64(s string) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return n
}

func atof(s string) float64 {
	n, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return n
}

func importLegacyZip(path string) (*Ledger, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("移行ZIPを開けません: %w", err)
	}
	defer zr.Close()
	settings, err := csvRowsFromZip(zr, "settings.csv")
	if err != nil {
		return nil, err
	}
	papers, err := csvRowsFromZip(zr, "papers.csv")
	if err != nil {
		return nil, err
	}
	refs, err := csvRowsFromZip(zr, "raw_references.csv")
	if err != nil {
		return nil, err
	}
	citations, err := csvRowsFromZip(zr, "citations.csv")
	if err != nil {
		return nil, err
	}
	name := "Imported Citation Project"
	for _, row := range settings {
		if row["key"] == "project_name" && row["value"] != "" {
			name = row["value"]
		}
	}
	l := newLedger(name)
	for _, row := range papers {
		status := row["status"]
		if status == "" {
			status = "collected"
		}
		created := row["created_at"]
		if created == "" {
			created = nowISO()
		}
		l.Papers = append(l.Papers, Paper{
			ID: row["id"], Title: row["title"], Authors: row["authors"], Year: atoi(row["year"]),
			Venue: row["venue"], DOI: row["doi"], Tags: row["tags"], Status: status,
			PDFPath: row["pdf_path"], DriveURL: row["drive_url"], Notes: row["notes"], CreatedAt: created,
		})
	}
	for _, row := range refs {
		id := atoi64(row["id"])
		if id >= l.NextRefID {
			l.NextRefID = id + 1
		}
		status := row["status"]
		if status == "" {
			status = "unresolved"
		}
		l.References = append(l.References, Reference{
			ID: id, SourcePaperID: row["source_paper_id"], Ordinal: atoi(row["ordinal"]),
			RawText: row["raw_text"], DOI: row["doi"], Year: atoi(row["year"]),
			FirstAuthor: row["first_author"], MatchedPaperID: row["matched_paper_id"],
			Confidence: atof(row["confidence"]), Status: status,
		})
	}
	for _, row := range citations {
		created := row["created_at"]
		if created == "" {
			created = nowISO()
		}
		status := row["status"]
		if status == "" {
			status = "confirmed"
		}
		l.Citations = append(l.Citations, Citation{
			SourcePaperID: row["source_paper_id"], TargetPaperID: row["target_paper_id"],
			RawReferenceID: atoi64(row["raw_reference_id"]), Status: status, CreatedAt: created,
		})
	}
	if errs := validateLedger(l); len(errs) > 0 {
		return nil, fmt.Errorf("移行データの整合性エラー: %s", strings.Join(errs, "; "))
	}
	return l, nil
}

func validateLedger(l *Ledger) []string {
	var errs []string
	papers := map[string]bool{}
	refs := map[int64]bool{}
	for _, p := range l.Papers {
		if p.ID == "" {
			errs = append(errs, "IDが空の文献があります")
		} else if papers[p.ID] {
			errs = append(errs, "文献IDが重複しています: "+p.ID)
		}
		papers[p.ID] = true
	}
	for _, r := range l.References {
		if refs[r.ID] {
			errs = append(errs, fmt.Sprintf("参考文献IDが重複しています: %d", r.ID))
		}
		refs[r.ID] = true
		if !papers[r.SourcePaperID] {
			errs = append(errs, fmt.Sprintf("参考文献%dの引用元がありません: %s", r.ID, r.SourcePaperID))
		}
		if r.MatchedPaperID != "" && !papers[r.MatchedPaperID] {
			errs = append(errs, fmt.Sprintf("参考文献%dの照合先がありません: %s", r.ID, r.MatchedPaperID))
		}
	}
	for _, c := range l.Citations {
		if !papers[c.SourcePaperID] || !papers[c.TargetPaperID] {
			errs = append(errs, fmt.Sprintf("引用辺の端点がありません: %s -> %s", c.SourcePaperID, c.TargetPaperID))
		}
	}
	return errs
}
