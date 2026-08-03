// Copyright (c) 2026 Katsushi Kagaya. Licensed under the MIT License.
package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strconv"
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

func TestStoreSaveReplacesLedger(t *testing.T) {
	store, err := newStore(filepath.Join(t.TempDir(), "project"))
	if err != nil {
		t.Fatal(err)
	}
	ledger := newLedger("First Name")
	if err := store.save(ledger); err != nil {
		t.Fatal(err)
	}
	ledger.ProjectName = "Replacement Name"
	if err := store.save(ledger); err != nil {
		t.Fatal(err)
	}
	got, err := store.load()
	if err != nil {
		t.Fatal(err)
	}
	if got.ProjectName != "Replacement Name" {
		t.Fatalf("replacement failed: got %q", got.ProjectName)
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

func TestEndToEndLedgerWorkflow(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project with spaces 日本語")
	targetPDF := filepath.Join(root, "target paper.pdf")
	sourcePDF := filepath.Join(root, "source paper.pdf")
	emptyRefs := filepath.Join(root, "empty references.txt")
	sourceRefs := filepath.Join(root, "source references.txt")

	for _, path := range []string{targetPDF, sourcePDF} {
		if err := os.WriteFile(path, []byte("%PDF-1.4\nportable test fixture\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(emptyRefs, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	refText := "1. Candidate, A. (2001). A sufficiently long candidate reference for end-to-end testing.\n" +
		"2. Target, T. (2002). A sufficiently long collected reference for resolution testing.\n"
	if err := os.WriteFile(sourceRefs, []byte(refText), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := run([]string{"--data-dir", project, "init", "--name", "Portable Test Project"}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"--data-dir", project, "add", targetPDF,
		"--title", "Collected Target", "--authors", "Target, T.", "--year", "2002",
		"--references", emptyRefs, "--yes"}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"--data-dir", project, "add", sourcePDF,
		"--title", "Source Study", "--authors", "Source, S.", "--year", "2026",
		"--references", sourceRefs, "--yes"}); err != nil {
		t.Fatal(err)
	}

	store, err := newStore(project)
	if err != nil {
		t.Fatal(err)
	}
	ledger, err := store.load()
	if err != nil {
		t.Fatal(err)
	}
	if len(ledger.Papers) != 2 || len(ledger.References) != 2 {
		t.Fatalf("unexpected initial counts: papers=%d references=%d", len(ledger.Papers), len(ledger.References))
	}
	targetID := slugID("Target, T.", 2002, "Collected Target")
	for _, paper := range ledger.Papers {
		if paper.PDFPath == "" {
			t.Fatalf("collected paper has no PDF path: %#v", paper)
		}
		if _, err := os.Stat(filepath.Join(project, filepath.FromSlash(paper.PDFPath))); err != nil {
			t.Fatalf("copied PDF is unavailable: %v", err)
		}
	}

	firstRef := strconv.FormatInt(ledger.References[0].ID, 10)
	secondRef := strconv.FormatInt(ledger.References[1].ID, 10)
	if err := run([]string{"--data-dir", project, "candidate", firstRef,
		"--title", "Candidate Work", "--authors", "Candidate, A.", "--year", "2001"}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"--data-dir", project, "resolve", secondRef, targetID}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"--json", "--data-dir", project, "status"}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"--data-dir", project, "validate"}); err != nil {
		t.Fatal(err)
	}

	ledger, err = store.load()
	if err != nil {
		t.Fatal(err)
	}
	stats := ledgerStats(ledger)
	if stats.Collected != 2 || stats.Candidates != 1 || stats.Edges != 2 || stats.RawRefs != 2 || stats.Unresolved != 0 {
		t.Fatalf("unexpected final statistics: %#v", stats)
	}

	withoutPDFs := filepath.Join(root, "ledger backup.zip")
	withPDFs := filepath.Join(root, "ledger backup with PDFs.zip")
	if err := run([]string{"--data-dir", project, "export", "--output", withoutPDFs}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"--data-dir", project, "export", "--output", withPDFs, "--include-pdfs"}); err != nil {
		t.Fatal(err)
	}

	plainEntries := zipEntries(t, withoutPDFs)
	for _, required := range []string{"ledger.json", "papers.csv", "raw_references.csv", "citations.csv", "settings.csv"} {
		if !plainEntries[required] {
			t.Errorf("ordinary export is missing %s", required)
		}
	}
	for name := range plainEntries {
		if strings.HasPrefix(name, "pdfs/") {
			t.Fatalf("ordinary export unexpectedly contains %s", name)
		}
	}
	withPDFEntries := zipEntries(t, withPDFs)
	pdfCount := 0
	for name := range withPDFEntries {
		if strings.HasPrefix(name, "pdfs/") {
			pdfCount++
		}
	}
	if pdfCount != 2 {
		t.Fatalf("PDF export contains %d PDFs, want 2", pdfCount)
	}
}

func TestStoreLoadRejectsMalformedAndUnsupportedLedger(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "malformed JSON", content: "{", want: "cannot read ledger.json"},
		{name: "unsupported format", content: `{"format_version": 99}`, want: "unsupported ledger format: 99"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store, err := newStore(filepath.Join(t.TempDir(), "project"))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(store.Dir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(store.LedgerPath, []byte(tc.content), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err = store.load()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got error %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestValidateLedgerReportsIntegrityErrors(t *testing.T) {
	ledger := newLedger("Invalid")
	ledger.Papers = []Paper{{ID: "duplicate"}, {ID: "duplicate"}}
	ledger.References = []Reference{
		{ID: 1, SourcePaperID: "missing-source", MatchedPaperID: "missing-target"},
		{ID: 1, SourcePaperID: "duplicate"},
	}
	ledger.Citations = []Citation{{SourcePaperID: "duplicate", TargetPaperID: "missing-target"}}
	errs := strings.Join(validateLedger(ledger), "\n")
	for _, want := range []string{
		"duplicate paper ID: duplicate",
		"duplicate reference ID: 1",
		"reference 1 has no source paper: missing-source",
		"reference 1 has no matched paper: missing-target",
		"citation edge has a missing endpoint: duplicate -> missing-target",
	} {
		if !strings.Contains(errs, want) {
			t.Errorf("validation output is missing %q:\n%s", want, errs)
		}
	}
}

func TestInitRefusesToOverwriteExistingLedger(t *testing.T) {
	project := filepath.Join(t.TempDir(), "project")
	if err := run([]string{"--data-dir", project, "init", "--name", "Original"}); err != nil {
		t.Fatal(err)
	}
	err := run([]string{"--data-dir", project, "init", "--name", "Replacement"})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("unexpected error: %v", err)
	}
	store, _ := newStore(project)
	ledger, err := store.load()
	if err != nil {
		t.Fatal(err)
	}
	if ledger.ProjectName != "Original" {
		t.Fatalf("existing ledger was changed: %q", ledger.ProjectName)
	}
}

func TestAddMissingPDFFailsWithoutMutation(t *testing.T) {
	project := filepath.Join(t.TempDir(), "project")
	if err := run([]string{"--data-dir", project, "init"}); err != nil {
		t.Fatal(err)
	}
	err := run([]string{"--data-dir", project, "add", filepath.Join(project, "missing.pdf"),
		"--title", "Missing", "--authors", "Author, A.", "--year", "2026", "--yes"})
	if err == nil || !strings.Contains(err.Error(), "PDF not found") {
		t.Fatalf("unexpected error: %v", err)
	}
	store, _ := newStore(project)
	ledger, err := store.load()
	if err != nil {
		t.Fatal(err)
	}
	if len(ledger.Papers) != 0 || len(ledger.References) != 0 {
		t.Fatalf("failed add changed the ledger: %#v", ledger)
	}
}

func TestExtractPDFTextReportsMissingDependency(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, err := extractPDFText(filepath.Join(t.TempDir(), "paper.pdf"))
	if err == nil || !strings.Contains(err.Error(), "pdftotext was not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCopyFileAtomicReplacesExistingFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source")
	dst := filepath.Join(dir, "destination")
	if err := os.WriteFile(src, []byte("replacement"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("old content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyFileAtomic(src, dst); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "replacement" {
		t.Fatalf("destination contains %q", got)
	}
}

func TestVersionMetadataIsConsistent(t *testing.T) {
	files := map[string]string{
		filepath.Join("..", "VERSION"):        appVersion,
		filepath.Join("..", "README.md"):      "Citation Network Ledger Go " + appVersion,
		filepath.Join("..", "CITATION.cff"):   `version: "` + appVersion + `"`,
		filepath.Join("..", "BUILD_INFO.txt"): "Citation Network Ledger Go " + appVersion,
	}
	for path, want := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(content), want) {
			t.Errorf("%s does not contain %q", path, want)
		}
	}
}

func zipEntries(t *testing.T, path string) map[string]bool {
	t.Helper()
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	entries := make(map[string]bool, len(reader.File))
	for _, file := range reader.File {
		entries[file.Name] = true
	}
	return entries
}
