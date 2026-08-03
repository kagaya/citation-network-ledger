// Copyright (c) 2026 Katsushi Kagaya. Licensed under the MIT License.
package main

import (
	"fmt"
	"os"
	"strings"
)

func parseGlobals(args []string) (globalOptions, []string, error) {
	g := globalOptions{}
	var rest []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--json":
			g.JSON = true
		case a == "--data-dir":
			if i+1 >= len(args) {
				return g, nil, fmt.Errorf("--data-dir requires a path")
			}
			i++
			g.DataDir = args[i]
		case strings.HasPrefix(a, "--data-dir="):
			g.DataDir = strings.TrimPrefix(a, "--data-dir=")
		default:
			rest = append(rest, a)
		}
	}
	return g, rest, nil
}

func usage() {
	fmt.Printf(`%s %s

Author: %s
Contact: %s
%s
License: %s

Build a citation network one PDF at a time without a browser.

Usage:
  citation-ledger-go [--data-dir DIR] [--json] COMMAND [OPTIONS]

Commands:
  init            create an empty project
  project         show or change the project name
  status          show ledger counts
  papers          list and search papers
  show            show paper details and citation context
  edges           show confirmed citation edges (supports --dot)
  queue           show the next collection candidates
  unresolved      show unresolved references
  extract         extract a reference section with pdftotext
  add             register a PDF and its references
  resolve         connect a reference to an existing paper
  candidate       promote an uncollected reference to a candidate node
  export          export JSON and CSV files as a ZIP
  validate        check ledger reference integrity
  doctor          inspect the execution environment
  version         show the version

Examples:
  citation-ledger-go --data-dir ~/citation_projects/crayfish init --seed-crayfish
  citation-ledger-go --data-dir ~/citation_projects/crayfish status
  citation-ledger-go --data-dir ~/citation_projects/crayfish add paper.pdf \
    --title "Paper title" --authors "Author, A." --year 2026 --references refs.txt
`, appName, appVersion, appAuthor, appContact, appCopyright, appLicense)
}

func run(args []string) error {
	g, args, err := parseGlobals(args)
	if err != nil {
		return err
	}
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		usage()
		return nil
	}
	if args[0] == "version" || args[0] == "--version" {
		fmt.Printf("%s %s\nAuthor: %s\nContact: %s\n%s\nLicense: %s\n", appName, appVersion, appAuthor, appContact, appCopyright, appLicense)
		return nil
	}
	store, err := newStore(g.DataDir)
	if err != nil {
		return err
	}
	commandArgs := args[1:]
	switch args[0] {
	case "init":
		return cmdInit(store, g, commandArgs)
	case "project":
		return cmdProject(store, g, commandArgs)
	case "status":
		return cmdStatus(store, g, commandArgs)
	case "papers":
		return cmdPapers(store, g, commandArgs)
	case "show":
		return cmdShow(store, g, commandArgs)
	case "edges":
		return cmdEdges(store, g, commandArgs)
	case "queue":
		return cmdQueue(store, g, commandArgs)
	case "unresolved":
		return cmdUnresolved(store, g, commandArgs)
	case "extract":
		return cmdExtract(g, commandArgs)
	case "add":
		return cmdAdd(store, g, commandArgs)
	case "resolve":
		return cmdResolve(store, g, commandArgs)
	case "candidate":
		return cmdCandidate(store, g, commandArgs)
	case "export":
		return cmdExport(store, g, commandArgs)
	case "validate":
		return cmdValidate(store, g, commandArgs)
	case "doctor":
		return cmdDoctor(store, g, commandArgs)
	default:
		return fmt.Errorf("unknown command: %s (use help to list commands)", args[0])
	}
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}
