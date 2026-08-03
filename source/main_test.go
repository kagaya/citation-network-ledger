// Copyright (c) 2026 Katsushi Kagaya. Licensed under the MIT License.
package main

import (
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
