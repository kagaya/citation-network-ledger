# Citation Network Ledger Go 3.0.2

**Author:** Katsushi Kagaya  
**Contact:** kkagaya@excyberlab.com  
**Copyright:** Copyright (c) 2026 Katsushi Kagaya  
**Code license:** MIT License

A browser-free command-line ledger for building a citation network one PDF at a time. It is domain-independent.

## Data included in the public release

The bundled crayfish neuroethology sample ledger contains bibliographic records, raw references, citation edges, and matching status. It contains no paper PDFs, Google Drive links, or local PDF paths.

## Features

- Runs as a single executable
- Requires no additional language runtime or package manager
- Binaries for Windows, WSL/Linux, Intel Mac, and Apple Silicon Mac
- Atomically updates the human-readable UTF-8 `ledger.json`
- Copies PDFs into the project's `pdfs/` directory
- Exports CSV, JSON, and Graphviz DOT
- Embeds the existing crayfish neuroethology ledger as a sample project

## Choose a binary

| Environment | Binary |
|---|---|
| Windows 64-bit | `citation-ledger-go-3.0.2-windows-amd64.exe` |
| WSL / Linux 64-bit | `citation-ledger-go-3.0.2-linux-amd64` |
| Intel Mac | `citation-ledger-go-3.0.2-darwin-amd64` |
| Apple Silicon Mac | `citation-ledger-go-3.0.2-darwin-arm64` |

The commands below assume that the selected executable has been renamed to `citation-ledger-go`.

## WSL/Linux

```sh
chmod +x citation-ledger-go
./citation-ledger-go doctor
```

Create an empty project:

```sh
./citation-ledger-go --data-dir ~/citation_projects/limpet \
  init --name "Limpet Homing"
```

Start with the bundled crayfish literature ledger:

```sh
./citation-ledger-go --data-dir ~/citation_projects/crayfish \
  init --seed-crayfish
```

If `--data-dir` is omitted, the program creates `citation-ledger-data/` in the current directory. The data directory can also be set with `CITATION_LEDGER_DATA_DIR`.

## Windows

Run the program from PowerShell or Command Prompt. WSL is not required.

```powershell
.\citation-ledger-go-3.0.2-windows-amd64.exe doctor
.\citation-ledger-go-3.0.2-windows-amd64.exe --data-dir C:\citation_projects\crayfish init --seed-crayfish
.\citation-ledger-go-3.0.2-windows-amd64.exe --data-dir C:\citation_projects\crayfish status
```

The binaries are not commercially code-signed, so Windows may display a warning on first launch. Compare the binary with the published SHA-256 checksum before running it. The Linux binary used in WSL does not produce this Windows warning.

## Basic commands

```sh
# Status
./citation-ledger-go --data-dir PROJECT status

# List and search papers
./citation-ledger-go --data-dir PROJECT papers
./citation-ledger-go --data-dir PROJECT papers --search "readiness"

# Paper details and citation context
./citation-ledger-go --data-dir PROJECT show kagaya_takahata_2010

# Unresolved references and collection priority
./citation-ledger-go --data-dir PROJECT unresolved -n 30
./citation-ledger-go --data-dir PROJECT queue -n 30

# Citation edges
./citation-ledger-go --data-dir PROJECT edges
./citation-ledger-go --data-dir PROJECT edges --dot > network.dot

# JSON output
./citation-ledger-go --json --data-dir PROJECT status
```

## Register a PDF and its references

When a checked and corrected reference list is available, no external program is needed:

```sh
./citation-ledger-go --data-dir PROJECT add paper.pdf \
  --title "Paper title" \
  --authors "Author, A.; Collaborator, B." \
  --year 2026 \
  --venue "Journal name" \
  --doi "10.xxxx/example" \
  --tags "behavior;neuroethology" \
  --references references.txt
```

If `--references` is omitted, the program extracts text from the PDF with `pdftotext -layout`. On WSL/Ubuntu, install it if needed:

```sh
sudo apt install poppler-utils
```

Extract only the reference section to a file for inspection:

```sh
./citation-ledger-go extract paper.pdf -o references.txt
```

Scanned image-only PDFs require OCR. Add a text layer with OCRmyPDF or Tesseract before registration; OCR is not run automatically.

## Confirm citations

Connect an unresolved reference to an existing paper:

```sh
./citation-ledger-go --data-dir PROJECT resolve 123 wine_krasne_1972
```

Promote an uncollected reference to a candidate node:

```sh
./citation-ledger-go --data-dir PROJECT candidate 123 \
  --title "Candidate paper" --authors "Author" --year 1974
```

Registration performs automatic matching using exact DOI matches and author/year/title-token matches. Review anything not automatically confirmed with `unresolved`.

## Export and backup

```sh
./citation-ledger-go --data-dir PROJECT export -o ledger_backup.zip
./citation-ledger-go --data-dir PROJECT export -o ledger_with_pdfs.zip --include-pdfs
```

An export ZIP contains `ledger.json`, `papers.csv`, `raw_references.csv`, `citations.csv`, and `settings.csv`. PDFs are excluded from ordinary exports.

```sh
./citation-ledger-go --data-dir PROJECT validate
```

## Data layout

```text
PROJECT/
├── ledger.json
└── pdfs/
    └── paper_id_original_name.pdf
```

`ledger.json` contains papers, raw references, matching status, and citation edges. Writes complete a temporary file before replacement, reducing the chance of leaving a partial JSON file after interruption. Avoid editing one project from multiple terminals or devices at the same time.

## Dependencies

Ledger operations, search, matching, and export have no additional dependencies.

| Program | Required | Purpose |
|---|---|---|
| `pdftotext` (Poppler) | Optional | Extract text from text-based PDFs |
| `qpdf` | Optional | Repair or normalize damaged PDFs |
| OCRmyPDF / Tesseract | Optional | OCR scanned PDFs |
| Graphviz | Optional | Render DOT files as images |

## Build from source

Users do not need to install Go to run the supplied binaries. Go 1.22 or later is needed only to modify and rebuild the source.

```sh
sh build_all.sh
```

The build uses no external Go modules and depends only on the standard library.

## Author, copyright, and licenses

- Author: Katsushi Kagaya
- Contact: kkagaya@excyberlab.com
- Copyright: Copyright (c) 2026 Katsushi Kagaya
- Code and executables: MIT License (`LICENSE`)
- Original documentation, citation edges, and annotations: CC BY 4.0 (`DATA_LICENSE.txt`)
- Publisher PDFs and third-party literature content: not included and not relicensed by this distribution

The source is included to support reproducibility and future maintenance. It may be used, modified, and redistributed under the MIT License.
