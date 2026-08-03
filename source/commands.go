// Copyright (c) 2026 Katsushi Kagaya. Licensed under the MIT License.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

type globalOptions struct {
	DataDir string
	JSON    bool
}

type stats struct {
	Collected  int `json:"collected"`
	Candidates int `json:"candidates"`
	Edges      int `json:"edges"`
	RawRefs    int `json:"raw_refs"`
	Unresolved int `json:"unresolved"`
}

func ledgerStats(l *Ledger) stats {
	s := stats{Edges: len(l.Citations), RawRefs: len(l.References)}
	for _, p := range l.Papers {
		if p.Status == "candidate" {
			s.Candidates++
		} else {
			s.Collected++
		}
	}
	for _, r := range l.References {
		if r.Status != "confirmed" {
			s.Unresolved++
		}
	}
	return s
}

func printJSON(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	return fs
}

func requireNoArgs(fs *flag.FlagSet) error {
	if fs.NArg() != 0 {
		return fmt.Errorf("余分な引数があります: %s", strings.Join(fs.Args(), " "))
	}
	return nil
}

func cmdInit(store *Store, g globalOptions, args []string) error {
	fs := newFlagSet("init")
	name := fs.String("name", "Untitled Citation Project", "プロジェクト名")
	reset := fs.Bool("reset", false, "既存台帳を初期化")
	seed := fs.Bool("seed-crayfish", false, "ザリガニ神経行動学の同梱例を使用")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(fs); err != nil {
		return err
	}
	if store.exists() && !*reset {
		return fmt.Errorf("台帳はすでにあります。置き換える場合は --reset を指定してください: %s", store.LedgerPath)
	}
	var l *Ledger
	var err error
	if *seed {
		l, err = seedCrayfishLedger()
		if err != nil {
			return err
		}
		if *name != "Untitled Citation Project" {
			l.ProjectName = *name
		}
	} else {
		l = newLedger(*name)
	}
	if err := store.save(l); err != nil {
		return err
	}
	if g.JSON {
		return printJSON(map[string]any{"project_name": l.ProjectName, "database": store.LedgerPath})
	}
	fmt.Printf("プロジェクトを用意しました: %s\n台帳: %s\n", l.ProjectName, store.LedgerPath)
	return nil
}

func cmdProject(store *Store, g globalOptions, args []string) error {
	fs := newFlagSet("project")
	name := fs.String("name", "", "新しいプロジェクト名")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(fs); err != nil {
		return err
	}
	l, err := store.load()
	if err != nil {
		return err
	}
	if *name != "" {
		l.ProjectName = *name
		if err := store.save(l); err != nil {
			return err
		}
	}
	data := map[string]any{"project_name": l.ProjectName, "database": store.LedgerPath}
	if g.JSON {
		return printJSON(data)
	}
	fmt.Printf("project_name     %s\ndatabase         %s\n", l.ProjectName, store.LedgerPath)
	return nil
}

func cmdStatus(store *Store, g globalOptions, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("status は引数を取りません")
	}
	l, err := store.load()
	if err != nil {
		return err
	}
	s := ledgerStats(l)
	if g.JSON {
		return printJSON(map[string]any{"project": l.ProjectName, "stats": s, "ledger": store.LedgerPath})
	}
	fmt.Printf("%s: %s\n", appName, l.ProjectName)
	fmt.Printf("  収集済み文献       %d\n", s.Collected)
	fmt.Printf("  未収集候補ノード   %d\n", s.Candidates)
	fmt.Printf("  確認済み引用辺     %d\n", s.Edges)
	fmt.Printf("  参考文献原文       %d\n", s.RawRefs)
	fmt.Printf("  照合待ち           %d\n", s.Unresolved)
	fmt.Printf("  台帳               %s\n", store.LedgerPath)
	return nil
}

func sortedPapers(l *Ledger) []Paper {
	papers := append([]Paper(nil), l.Papers...)
	sort.Slice(papers, func(i, j int) bool {
		yi, yj := papers[i].Year, papers[j].Year
		if yi == 0 {
			yi = 9999
		}
		if yj == 0 {
			yj = 9999
		}
		if yi == yj {
			return papers[i].Title < papers[j].Title
		}
		return yi < yj
	})
	return papers
}

func paperLabel(p Paper) string {
	year := "n.d."
	if p.Year > 0 {
		year = strconv.Itoa(p.Year)
	}
	return fmt.Sprintf("%s\t%s\t%s\t%s", p.ID, year, firstAuthor(p.Authors), p.Title)
}

func cmdPapers(store *Store, g globalOptions, args []string) error {
	fs := newFlagSet("papers")
	status := fs.String("status", "all", "all|collected|candidate")
	search := fs.String("search", "", "検索語")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(fs); err != nil {
		return err
	}
	if *status != "all" && *status != "collected" && *status != "candidate" {
		return fmt.Errorf("--status は all, collected, candidate のいずれかです")
	}
	l, err := store.load()
	if err != nil {
		return err
	}
	var papers []Paper
	q := normalize(*search)
	for _, p := range sortedPapers(l) {
		if *status != "all" && p.Status != *status {
			continue
		}
		if q != "" && !strings.Contains(normalize(p.Title+" "+p.Authors+" "+p.Venue+" "+p.Tags), q) {
			continue
		}
		papers = append(papers, p)
	}
	if g.JSON {
		return printJSON(papers)
	}
	for _, p := range papers {
		fmt.Println(paperLabel(p))
	}
	return nil
}

func cmdShow(store *Store, g globalOptions, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("使い方: show PAPER_ID")
	}
	l, err := store.load()
	if err != nil {
		return err
	}
	p := l.paper(args[0])
	if p == nil {
		return fmt.Errorf("文献IDが見つかりません: %s", args[0])
	}
	var cites, citedBy []Paper
	for _, c := range l.Citations {
		if c.SourcePaperID == p.ID {
			if target := l.paper(c.TargetPaperID); target != nil {
				cites = append(cites, *target)
			}
		}
		if c.TargetPaperID == p.ID {
			if source := l.paper(c.SourcePaperID); source != nil {
				citedBy = append(citedBy, *source)
			}
		}
	}
	if g.JSON {
		return printJSON(map[string]any{"paper": p, "cites": cites, "cited_by": citedBy})
	}
	fmt.Println(paperLabel(*p))
	fmt.Printf("掲載誌: %s\nタグ: %s\n", p.Venue, p.Tags)
	if p.DriveURL != "" {
		fmt.Printf("Drive: %s\n", p.DriveURL)
	}
	fmt.Println("引用先:")
	for _, x := range cites {
		fmt.Printf("  -> %s\n", paperLabel(x))
	}
	fmt.Println("この文献を引用:")
	for _, x := range citedBy {
		fmt.Printf("  <- %s\n", paperLabel(x))
	}
	return nil
}

func cmdEdges(store *Store, g globalOptions, args []string) error {
	fs := newFlagSet("edges")
	dot := fs.Bool("dot", false, "Graphviz DOT形式")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(fs); err != nil {
		return err
	}
	l, err := store.load()
	if err != nil {
		return err
	}
	if g.JSON {
		return printJSON(l.Citations)
	}
	if *dot {
		fmt.Println("digraph citations {")
		for _, p := range sortedPapers(l) {
			label := strings.ReplaceAll(fmt.Sprintf("%s %d", firstAuthor(p.Authors), p.Year), "\"", "'")
			fmt.Printf("  %q [label=%q];\n", p.ID, label)
		}
		for _, c := range l.Citations {
			fmt.Printf("  %q -> %q;\n", c.SourcePaperID, c.TargetPaperID)
		}
		fmt.Println("}")
		return nil
	}
	for _, c := range l.Citations {
		fmt.Printf("%s\t->\t%s\n", c.SourcePaperID, c.TargetPaperID)
	}
	return nil
}

type queueItem struct {
	CitedByCount int      `json:"cited_by_count"`
	KeyAuthor    string   `json:"key_author"`
	Year         int      `json:"year,omitempty"`
	Example      string   `json:"example"`
	SourceIDs    []string `json:"source_ids"`
}

func cmdQueue(store *Store, g globalOptions, args []string) error {
	fs := newFlagSet("queue")
	limit := fs.Int("n", 30, "表示件数")
	fs.IntVar(limit, "limit", 30, "表示件数")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(fs); err != nil {
		return err
	}
	l, err := store.load()
	if err != nil {
		return err
	}
	type accumulator struct {
		author  string
		year    int
		example string
		sources map[string]bool
	}
	groups := map[string]*accumulator{}
	for _, r := range l.References {
		if r.Status == "confirmed" || strings.TrimSpace(r.FirstAuthor) == "" {
			continue
		}
		key := normalize(r.FirstAuthor) + "\x00" + strconv.Itoa(r.Year)
		if groups[key] == nil {
			groups[key] = &accumulator{author: strings.ToLower(r.FirstAuthor), year: r.Year, example: r.RawText, sources: map[string]bool{}}
		}
		groups[key].sources[r.SourcePaperID] = true
	}
	var items []queueItem
	for _, a := range groups {
		var sourceIDs []string
		for id := range a.sources {
			sourceIDs = append(sourceIDs, id)
		}
		sort.Strings(sourceIDs)
		items = append(items, queueItem{len(sourceIDs), a.author, a.year, a.example, sourceIDs})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CitedByCount != items[j].CitedByCount {
			return items[i].CitedByCount > items[j].CitedByCount
		}
		if items[i].Year != items[j].Year {
			return items[i].Year < items[j].Year
		}
		return items[i].KeyAuthor < items[j].KeyAuthor
	})
	if *limit >= 0 && len(items) > *limit {
		items = items[:*limit]
	}
	if g.JSON {
		return printJSON(items)
	}
	fmt.Println("被引用元数\t著者・年\t参考文献例")
	for _, x := range items {
		fmt.Printf("%d\t%s %s\t%s\n", x.CitedByCount, x.KeyAuthor, yearText(x.Year), short(x.Example, 92))
	}
	return nil
}

func cmdUnresolved(store *Store, g globalOptions, args []string) error {
	fs := newFlagSet("unresolved")
	limit := fs.Int("n", 50, "表示件数")
	fs.IntVar(limit, "limit", 50, "表示件数")
	source := fs.String("source", "", "引用元文献ID")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(fs); err != nil {
		return err
	}
	l, err := store.load()
	if err != nil {
		return err
	}
	var refs []Reference
	for _, r := range l.References {
		if r.Status == "confirmed" || (*source != "" && r.SourcePaperID != *source) {
			continue
		}
		refs = append(refs, r)
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].ID < refs[j].ID })
	if *limit >= 0 && len(refs) > *limit {
		refs = refs[:*limit]
	}
	if g.JSON {
		return printJSON(refs)
	}
	fmt.Println("ID\t状態\t引用元\t参考文献")
	for _, r := range refs {
		suggestion := ""
		if r.MatchedPaperID != "" {
			suggestion = fmt.Sprintf(" [候補: %s %.0f%%]", r.MatchedPaperID, r.Confidence*100)
		}
		fmt.Printf("%d\t%s\t%s\t%s%s\n", r.ID, r.Status, r.SourcePaperID, short(r.RawText, 92), suggestion)
	}
	return nil
}

func cmdExtract(g globalOptions, args []string) error {
	args = leadingPositionalLast(args)
	fs := newFlagSet("extract")
	output := fs.String("o", "", "出力テキストファイル")
	fs.StringVar(output, "output", "", "出力テキストファイル")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("使い方: extract PDF [-o references.txt]")
	}
	pdf, err := filepath.Abs(fs.Arg(0))
	if err != nil {
		return err
	}
	if _, err := os.Stat(pdf); err != nil {
		return fmt.Errorf("PDFが見つかりません: %s", pdf)
	}
	text, err := extractPDFText(pdf)
	if err != nil {
		return err
	}
	section := referenceSection(text)
	count := len(splitReferences(section))
	if *output != "" {
		if err := os.WriteFile(*output, []byte(section), 0o644); err != nil {
			return err
		}
		fmt.Printf("%s に保存しました（候補 %d 件）\n", *output, count)
		return nil
	}
	if g.JSON {
		return printJSON(map[string]any{"pdf": pdf, "candidate_count": count, "references_text": section})
	}
	fmt.Print(section)
	fmt.Fprintf(os.Stderr, "\n[検出候補: %d件]\n", count)
	return nil
}

func cmdAdd(store *Store, g globalOptions, args []string) error {
	args = leadingPositionalLast(args)
	fs := newFlagSet("add")
	title := fs.String("title", "", "論文名（必須）")
	authors := fs.String("authors", "", "著者（必須）")
	year := fs.Int("year", 0, "出版年（必須）")
	venue := fs.String("venue", "", "掲載誌")
	doi := fs.String("doi", "", "DOI")
	tags := fs.String("tags", "", "タグ")
	driveURL := fs.String("drive-url", "", "Google Drive URL")
	notes := fs.String("notes", "", "注記")
	refsFile := fs.String("references", "", "確認済み参考文献テキスト")
	yes := fs.Bool("y", false, "確認せず登録")
	fs.BoolVar(yes, "yes", false, "確認せず登録")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 || strings.TrimSpace(*title) == "" || strings.TrimSpace(*authors) == "" || *year <= 0 {
		return fmt.Errorf("使い方: add PDF --title TITLE --authors AUTHORS --year YEAR [--references FILE]")
	}
	pdf, err := filepath.Abs(fs.Arg(0))
	if err != nil {
		return err
	}
	if _, err := os.Stat(pdf); err != nil {
		return fmt.Errorf("PDFが見つかりません: %s", pdf)
	}
	var refText string
	if *refsFile != "" {
		b, err := os.ReadFile(*refsFile)
		if err != nil {
			return err
		}
		refText = string(b)
	} else {
		text, err := extractPDFText(pdf)
		if err != nil {
			return err
		}
		refText = referenceSection(text)
	}
	refs := splitReferences(refText)
	fmt.Printf("PDF: %s\n参考文献候補: %d件\n", pdf, len(refs))
	for i, r := range refs {
		if i == 5 {
			fmt.Printf("  …ほか %d件\n", len(refs)-5)
			break
		}
		fmt.Printf("  %s\n", short(r, 92))
	}
	if !*yes {
		fmt.Print("この内容で台帳へ登録しますか [y/N]: ")
		answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		answer = strings.ToLower(strings.TrimSpace(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("登録しませんでした")
			return nil
		}
	}
	l, err := store.load()
	if err != nil {
		return err
	}
	probe := Reference{RawText: *title, DOI: *doi, Year: *year, FirstAuthor: firstAuthor(*authors)}
	var target *Paper
	best := 0.0
	for i := range l.Papers {
		if l.Papers[i].Status != "candidate" {
			continue
		}
		score := paperScore(&probe, &l.Papers[i])
		if score >= 0.72 && score > best {
			best, target = score, &l.Papers[i]
		}
	}
	id := slugID(*authors, *year, *title)
	if target != nil {
		id = target.ID
	} else if l.paper(id) != nil {
		return fmt.Errorf("同じ文献IDがすでにあります: %s", id)
	}
	safeName := safeFilename(filepath.Base(pdf))
	dstName := id + "_" + safeName
	dst := filepath.Join(store.PDFDir, dstName)
	if err := copyFileAtomic(pdf, dst); err != nil {
		return fmt.Errorf("PDFをコピーできません: %w", err)
	}
	p := Paper{ID: id, Title: strings.TrimSpace(*title), Authors: strings.TrimSpace(*authors), Year: *year,
		Venue: *venue, DOI: *doi, Tags: *tags, Status: "collected", PDFPath: filepath.ToSlash(filepath.Join("pdfs", dstName)),
		DriveURL: *driveURL, Notes: *notes, CreatedAt: nowISO()}
	if target != nil {
		*target = p
	} else {
		l.Papers = append(l.Papers, p)
	}
	for ordinal, raw := range refs {
		d, y, a := parseReference(raw)
		l.References = append(l.References, Reference{ID: l.NextRefID, SourcePaperID: id, Ordinal: ordinal + 1,
			RawText: raw, DOI: d, Year: y, FirstAuthor: a, Status: "unresolved"})
		l.NextRefID++
	}
	reconcile(l)
	if err := store.save(l); err != nil {
		return err
	}
	result := map[string]any{"paper_id": id, "references": len(refs), "ledger": store.LedgerPath}
	if g.JSON {
		return printJSON(result)
	}
	fmt.Printf("登録完了: %s（参考文献 %d件）\n", id, len(refs))
	return nil
}

func cmdResolve(store *Store, g globalOptions, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("使い方: resolve REFERENCE_ID TARGET_PAPER_ID")
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("参考文献IDが不正です: %s", args[0])
	}
	l, err := store.load()
	if err != nil {
		return err
	}
	r := l.reference(id)
	if r == nil {
		return fmt.Errorf("参考文献IDが見つかりません: %d", id)
	}
	if l.paper(args[1]) == nil {
		return fmt.Errorf("照合先文献IDが見つかりません: %s", args[1])
	}
	l.removeCitationForReference(r.ID)
	r.MatchedPaperID, r.Confidence, r.Status = args[1], 1, "confirmed"
	l.addCitation(r.SourcePaperID, args[1], r.ID)
	if err := store.save(l); err != nil {
		return err
	}
	if g.JSON {
		return printJSON(map[string]any{"reference_id": id, "target_paper_id": args[1]})
	}
	fmt.Printf("参考文献 %d を %s に結びました\n", id, args[1])
	return nil
}

func cmdCandidate(store *Store, g globalOptions, args []string) error {
	args = leadingPositionalLast(args)
	fs := newFlagSet("candidate")
	title := fs.String("title", "", "候補文献名")
	authors := fs.String("authors", "", "候補著者")
	year := fs.Int("year", 0, "候補出版年")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("使い方: candidate REFERENCE_ID [--title TITLE --authors AUTHORS --year YEAR]")
	}
	id, err := strconv.ParseInt(fs.Arg(0), 10, 64)
	if err != nil {
		return fmt.Errorf("参考文献IDが不正です")
	}
	l, err := store.load()
	if err != nil {
		return err
	}
	r := l.reference(id)
	if r == nil {
		return fmt.Errorf("参考文献IDが見つかりません: %d", id)
	}
	if *title == "" {
		*title = r.RawText
	}
	if *authors == "" {
		*authors = r.FirstAuthor
	}
	if *year == 0 {
		*year = r.Year
	}
	pid := slugID(*authors, *year, *title)
	if l.paper(pid) == nil {
		l.Papers = append(l.Papers, Paper{ID: pid, Title: *title, Authors: *authors, Year: *year, DOI: r.DOI, Status: "candidate", CreatedAt: nowISO()})
	}
	l.removeCitationForReference(r.ID)
	r.MatchedPaperID, r.Confidence, r.Status = pid, 1, "confirmed"
	l.addCitation(r.SourcePaperID, pid, r.ID)
	if err := store.save(l); err != nil {
		return err
	}
	if g.JSON {
		return printJSON(map[string]any{"reference_id": id, "candidate_paper_id": pid})
	}
	fmt.Printf("参考文献 %d を候補ノード %s にしました\n", id, pid)
	return nil
}

func cmdExport(store *Store, g globalOptions, args []string) error {
	fs := newFlagSet("export")
	output := fs.String("o", "citation_ledger_export.zip", "出力ZIP")
	fs.StringVar(output, "output", "citation_ledger_export.zip", "出力ZIP")
	includePDFs := fs.Bool("include-pdfs", false, "PDFもZIPに含める")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(fs); err != nil {
		return err
	}
	l, err := store.load()
	if err != nil {
		return err
	}
	if err := exportBundle(store, l, *output, *includePDFs); err != nil {
		return err
	}
	abs, _ := filepath.Abs(*output)
	if g.JSON {
		return printJSON(map[string]any{"output": abs, "include_pdfs": *includePDFs})
	}
	fmt.Println(abs)
	return nil
}

func cmdValidate(store *Store, g globalOptions, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("validate は引数を取りません")
	}
	l, err := store.load()
	if err != nil {
		return err
	}
	errs := validateLedger(l)
	if g.JSON {
		return printJSON(map[string]any{"valid": len(errs) == 0, "errors": errs})
	}
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Printf("ERROR: %s\n", e)
		}
		return fmt.Errorf("整合性エラーが %d 件あります", len(errs))
	}
	fmt.Println("OK: 台帳の参照整合性に問題はありません")
	return nil
}

func cmdDoctor(store *Store, g globalOptions, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("doctor は引数を取りません")
	}
	tool := func(name, missing string) string {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
		return missing
	}
	checks := map[string]any{
		"application": appName, "version": appVersion, "go_runtime": runtime.Version(),
		"os": runtime.GOOS, "architecture": runtime.GOARCH, "ledger": store.LedgerPath,
		"ledger_exists": store.exists(), "pdftotext": tool("pdftotext", "not found (PDF extraction unavailable)"),
		"qpdf":      tool("qpdf", "not found (optional PDF repair)"),
		"tesseract": tool("tesseract", "not found (optional OCR)"),
	}
	if g.JSON {
		return printJSON(checks)
	}
	keys := []string{"application", "version", "go_runtime", "os", "architecture", "ledger", "ledger_exists", "pdftotext", "qpdf", "tesseract"}
	for _, key := range keys {
		fmt.Printf("%-16s %v\n", key, checks[key])
	}
	return nil
}

func yearText(year int) string {
	if year == 0 {
		return ""
	}
	return strconv.Itoa(year)
}

func short(s string, width int) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	return string(r[:width-1]) + "…"
}

func safeFilename(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || strings.ContainsRune("._-", r) {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "paper.pdf"
	}
	return b.String()
}

// The standard flag package stops at the first positional argument. Our CUI
// accepts the natural "add paper.pdf --title ..." form by moving one leading
// positional argument behind the options before parsing.
func leadingPositionalLast(args []string) []string {
	if len(args) > 1 && !strings.HasPrefix(args[0], "-") {
		out := append([]string(nil), args[1:]...)
		return append(out, args[0])
	}
	return args
}
