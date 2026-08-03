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
				return g, nil, fmt.Errorf("--data-dir にはパスが必要です")
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

PDFを一つずつ追加しながら育てる、ブラウザー不要の引用ネットワーク台帳。

使い方:
  citation-ledger-go [--data-dir DIR] [--json] COMMAND [OPTIONS]

コマンド:
  init            空のプロジェクトを作成
  project         プロジェクト名を表示・変更
  status          台帳の件数を表示
  papers          文献一覧・検索
  show            文献詳細と前後の引用
  edges           確認済み引用辺（--dot対応）
  queue           次に収集する候補
  unresolved      照合待ち参考文献
  extract         pdftotextで参考文献節を抽出
  add             PDFと参考文献を登録
  resolve         参考文献を既存文献へ結ぶ
  candidate       未収集参考文献を候補ノード化
  export          JSON・CSVをZIP出力
  validate        台帳の参照整合性を検査
  doctor          実行環境を点検
  version         バージョン表示

例:
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
		return fmt.Errorf("不明なコマンドです: %s（helpで一覧を表示）", args[0])
	}
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}
