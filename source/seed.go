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
		return nil, fmt.Errorf("同梱サンプルを読めません: %w", err)
	}
	if l.FormatVersion != formatVersion {
		return nil, fmt.Errorf("同梱サンプルの形式が不正です")
	}
	return &l, nil
}
