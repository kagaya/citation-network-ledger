// Copyright (c) 2026 Katsushi Kagaya. Licensed under the MIT License.
package main

import "time"

const (
	appName       = "Citation Network Ledger Go"
	appVersion    = "3.0.2"
	appAuthor     = "Katsushi Kagaya"
	appContact    = "kkagaya@excyberlab.com"
	appCopyright  = "Copyright (c) 2026 Katsushi Kagaya"
	appLicense    = "MIT License"
	formatVersion = 1
)

type Ledger struct {
	FormatVersion int         `json:"format_version"`
	ProjectName   string      `json:"project_name"`
	NextRefID     int64       `json:"next_reference_id"`
	Papers        []Paper     `json:"papers"`
	References    []Reference `json:"references"`
	Citations     []Citation  `json:"citations"`
	UpdatedAt     string      `json:"updated_at"`
}

type Paper struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Authors   string `json:"authors,omitempty"`
	Year      int    `json:"year,omitempty"`
	Venue     string `json:"venue,omitempty"`
	DOI       string `json:"doi,omitempty"`
	Tags      string `json:"tags,omitempty"`
	Status    string `json:"status"`
	PDFPath   string `json:"pdf_path,omitempty"`
	DriveURL  string `json:"drive_url,omitempty"`
	Notes     string `json:"notes,omitempty"`
	CreatedAt string `json:"created_at"`
}

type Reference struct {
	ID             int64   `json:"id"`
	SourcePaperID  string  `json:"source_paper_id"`
	Ordinal        int     `json:"ordinal"`
	RawText        string  `json:"raw_text"`
	DOI            string  `json:"doi,omitempty"`
	Year           int     `json:"year,omitempty"`
	FirstAuthor    string  `json:"first_author,omitempty"`
	MatchedPaperID string  `json:"matched_paper_id,omitempty"`
	Confidence     float64 `json:"confidence"`
	Status         string  `json:"status"`
}

type Citation struct {
	SourcePaperID  string `json:"source_paper_id"`
	TargetPaperID  string `json:"target_paper_id"`
	RawReferenceID int64  `json:"raw_reference_id,omitempty"`
	Status         string `json:"status"`
	CreatedAt      string `json:"created_at"`
}

func newLedger(name string) *Ledger {
	if name == "" {
		name = "Untitled Citation Project"
	}
	return &Ledger{
		FormatVersion: formatVersion,
		ProjectName:   name,
		NextRefID:     1,
		Papers:        []Paper{},
		References:    []Reference{},
		Citations:     []Citation{},
		UpdatedAt:     nowISO(),
	}
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func (l *Ledger) paper(id string) *Paper {
	for i := range l.Papers {
		if l.Papers[i].ID == id {
			return &l.Papers[i]
		}
	}
	return nil
}

func (l *Ledger) reference(id int64) *Reference {
	for i := range l.References {
		if l.References[i].ID == id {
			return &l.References[i]
		}
	}
	return nil
}

func (l *Ledger) hasCitation(source, target string) bool {
	for _, c := range l.Citations {
		if c.SourcePaperID == source && c.TargetPaperID == target {
			return true
		}
	}
	return false
}

func (l *Ledger) addCitation(source, target string, refID int64) {
	if l.hasCitation(source, target) {
		return
	}
	l.Citations = append(l.Citations, Citation{
		SourcePaperID:  source,
		TargetPaperID:  target,
		RawReferenceID: refID,
		Status:         "confirmed",
		CreatedAt:      nowISO(),
	})
}

func (l *Ledger) removeCitationForReference(refID int64) {
	if refID == 0 {
		return
	}
	kept := l.Citations[:0]
	for _, c := range l.Citations {
		if c.RawReferenceID != refID {
			kept = append(kept, c)
		}
	}
	l.Citations = kept
}
