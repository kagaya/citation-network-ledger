// Copyright (c) 2026 Katsushi Kagaya. Licensed under the MIT License.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

type Store struct {
	Dir        string
	LedgerPath string
	PDFDir     string
}

func newStore(dir string) (*Store, error) {
	if dir == "" {
		dir = os.Getenv("CITATION_LEDGER_DATA_DIR")
	}
	if dir == "" {
		dir = "citation-ledger-data"
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	return &Store{
		Dir:        abs,
		LedgerPath: filepath.Join(abs, "ledger.json"),
		PDFDir:     filepath.Join(abs, "pdfs"),
	}, nil
}

func (s *Store) exists() bool {
	_, err := os.Stat(s.LedgerPath)
	return err == nil
}

func (s *Store) ensureDirs() error {
	if err := os.MkdirAll(s.PDFDir, 0o755); err != nil {
		return fmt.Errorf("cannot create data directory: %w", err)
	}
	return nil
}

func (s *Store) load() (*Ledger, error) {
	b, err := os.ReadFile(s.LedgerPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("ledger not found; run init first: %s", s.LedgerPath)
		}
		return nil, err
	}
	var l Ledger
	if err := json.Unmarshal(b, &l); err != nil {
		return nil, fmt.Errorf("cannot read ledger.json: %w", err)
	}
	if l.FormatVersion != formatVersion {
		return nil, fmt.Errorf("unsupported ledger format: %d", l.FormatVersion)
	}
	if l.NextRefID < 1 {
		l.NextRefID = 1
		for _, r := range l.References {
			if r.ID >= l.NextRefID {
				l.NextRefID = r.ID + 1
			}
		}
	}
	return &l, nil
}

func (s *Store) save(l *Ledger) error {
	if err := s.ensureDirs(); err != nil {
		return err
	}
	l.UpdatedAt = nowISO()
	b, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	tmp, err := os.CreateTemp(s.Dir, ".ledger-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(b); err != nil {
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
	if runtime.GOOS == "windows" {
		_ = os.Remove(s.LedgerPath)
	}
	if err := os.Rename(tmpName, s.LedgerPath); err != nil {
		return fmt.Errorf("cannot update ledger: %w", err)
	}
	return nil
}

func copyFileAtomic(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".copy-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := io.Copy(tmp, in); err != nil {
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
	if runtime.GOOS == "windows" {
		_ = os.Remove(dst)
	}
	return os.Rename(tmpName, dst)
}
