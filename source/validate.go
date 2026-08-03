// Copyright (c) 2026 Katsushi Kagaya. Licensed under the MIT License.
package main

import "fmt"

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
