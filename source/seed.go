// Copyright (c) 2026 Katsushi Kagaya. Licensed under the MIT License.
package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed seed_crayfish.json
var seedCrayfishJSON []byte

func seedCrayfishLedger() (*Ledger, error) {
	var l Ledger
	if err := json.Unmarshal(seedCrayfishJSON, &l); err != nil {
		return nil, fmt.Errorf("cannot read bundled sample: %w", err)
	}
	if l.FormatVersion != formatVersion {
		return nil, fmt.Errorf("invalid bundled sample format")
	}
	return &l, nil
}
