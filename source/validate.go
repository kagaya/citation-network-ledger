// Copyright (c) 2026 Katsushi Kagaya. Licensed under the MIT License.
package main

import "fmt"

func validateLedger(l *Ledger) []string {
	var errs []string
	papers := map[string]bool{}
	refs := map[int64]bool{}
	for _, p := range l.Papers {
		if p.ID == "" {
			errs = append(errs, "a paper has an empty ID")
		} else if papers[p.ID] {
			errs = append(errs, "duplicate paper ID: "+p.ID)
		}
		papers[p.ID] = true
	}
	for _, r := range l.References {
		if refs[r.ID] {
			errs = append(errs, fmt.Sprintf("duplicate reference ID: %d", r.ID))
		}
		refs[r.ID] = true
		if !papers[r.SourcePaperID] {
			errs = append(errs, fmt.Sprintf("reference %d has no source paper: %s", r.ID, r.SourcePaperID))
		}
		if r.MatchedPaperID != "" && !papers[r.MatchedPaperID] {
			errs = append(errs, fmt.Sprintf("reference %d has no matched paper: %s", r.ID, r.MatchedPaperID))
		}
	}
	for _, c := range l.Citations {
		if !papers[c.SourcePaperID] || !papers[c.TargetPaperID] {
			errs = append(errs, fmt.Sprintf("citation edge has a missing endpoint: %s -> %s", c.SourcePaperID, c.TargetPaperID))
		}
	}
	return errs
}
