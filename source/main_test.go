// Copyright (c) 2026 Katsushi Kagaya. Licensed under the MIT License.
package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSplitNumberedReferences(t *testing.T) {
	input := "References\n1. Wine, J. J. (1972). A sufficiently long reference title for testing.\ncontinued journal information\n2. Bowerman, R. F. (1974). Another sufficiently long reference title."
	refs := splitReferences(referenceSection(input))
	if len(refs) != 2 {
		t.Fatalf("got %d references: %#v", len(refs), refs)
	}
	if !strings.Contains(refs[0], "continued journal") {
		t.Fatalf("continuation line was lost: %q", refs[0])
	}
}

func TestSplitUnnumberedReferences(t *testing.T) {
	input := "REFERENCES\nWine, J. J. (1972). The organization of escape behaviour in the crayfish. Journal text.\nKrasne, F. B. (1969). A second classic reference with enough characters."
	refs := splitReferences(referenceSection(input))
	if len(refs) != 2 {
		t.Fatalf("got %d references: %#v", len(refs), refs)
	}
}

func TestJapaneseSpacedReferenceHeader(t *testing.T) {
	input := "本文です。\n参 考 文 献\nWine, J. J. (1972). A sufficiently long reference for the test."
	section := referenceSection(input)
	if strings.Contains(section, "本文です") || !strings.Contains(section, "Wine") {
		t.Fatalf("unexpected section: %q", section)
	}
}

func TestParseReference(t *testing.T) {
	doi, year, author := parseReference("[12] O'Connor, A. (2011). Title. doi:10.1000/ABC.123")
	if doi != "10.1000/ABC.123" || year != 2011 || author != "O'Connor" {
		t.Fatalf("unexpected metadata: doi=%q year=%d author=%q", doi, year, author)
	}
}

func TestStoreRoundTrip(t *testing.T) {
	store, err := newStore(filepath.Join(t.TempDir(), "project"))
	if err != nil {
		t.Fatal(err)
	}
	l := newLedger("Test Project")
	l.Papers = append(l.Papers, Paper{ID: "paper_1", Title: "日本語とEnglish", Status: "collected", CreatedAt: nowISO()})
	if err := store.save(l); err != nil {
		t.Fatal(err)
	}
	got, err := store.load()
	if err != nil {
		t.Fatal(err)
	}
	if got.ProjectName != l.ProjectName || got.Papers[0].Title != l.Papers[0].Title {
		t.Fatalf("round trip mismatch: %#v", got)
	}
}

func TestReconcileExactDOI(t *testing.T) {
	l := newLedger("Test")
	l.Papers = []Paper{
		{ID: "source", Title: "Source", Status: "collected"},
		{ID: "target", Title: "Target", DOI: "10.1000/example", Status: "collected"},
	}
	l.References = []Reference{{ID: 1, SourcePaperID: "source", RawText: "Some citation", DOI: "10.1000/example", Status: "unresolved"}}
	reconcile(l)
	if l.References[0].Status != "confirmed" || l.References[0].MatchedPaperID != "target" || len(l.Citations) != 1 {
		t.Fatalf("DOI reconciliation failed: %#v %#v", l.References[0], l.Citations)
	}
}

func TestImportLegacyZip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	files := map[string]string{
		"settings.csv":       "key,value\nproject_name,Legacy Test\n",
		"papers.csv":         "id,title,authors,year,venue,doi,tags,status,pdf_path,drive_url,notes,created_at\na,Paper A,Author A,2000,,,,collected,,,,2026-01-01T00:00:00Z\nb,Paper B,Author B,2001,,,,collected,,,,2026-01-01T00:00:00Z\n",
		"raw_references.csv": "id,source_paper_id,ordinal,raw_text,doi,year,first_author,matched_paper_id,confidence,status\n1,a,1,Paper B,,2001,Author,b,1,confirmed\n",
		"citations.csv":      "source_paper_id,target_paper_id,raw_reference_id,status,created_at\na,b,1,confirmed,2026-01-01T00:00:00Z\n",
	}
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	l, err := importLegacyZip(path)
	if err != nil {
		t.Fatal(err)
	}
	if l.ProjectName != "Legacy Test" || len(l.Papers) != 2 || len(l.References) != 1 || len(l.Citations) != 1 {
		t.Fatalf("unexpected import: %#v", l)
	}
}

func TestBundledCrayfishSeed(t *testing.T) {
	l, err := seedCrayfishLedger()
	if err != nil {
		t.Fatal(err)
	}
	s := ledgerStats(l)
	if s.Collected != 12 || s.Edges != 21 || s.RawRefs != 450 || s.Unresolved != 434 {
		t.Fatalf("unexpected seed counts: %#v", s)
	}
	if errs := validateLedger(l); len(errs) != 0 {
		t.Fatalf("invalid seed: %#v", errs)
	}
}
