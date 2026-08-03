// Copyright (c) 2026 Katsushi Kagaya. Licensed under the MIT License.
package main

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

var (
	doiPattern      = regexp.MustCompile(`(?i)10\.[0-9]{4,9}/[-._;()/:a-z0-9]+`)
	yearPattern     = regexp.MustCompile(`(^|[^0-9])(18[0-9]{2}|19[0-9]{2}|20[0-9]{2})([^0-9]|$)`)
	numberedPattern = regexp.MustCompile(`^(\[[0-9]+\]|[0-9]+[.):])[[:space:]]+`)
	authorYear      = regexp.MustCompile(`(?i)\((18|19|20)[0-9]{2}[a-z]?\)`)
	referenceHeader = regexp.MustCompile(`(?im)^[[:space:]]*(references|literature cited)[[:space:]]*$`)
	japaneseHeader  = regexp.MustCompile(`参[[:space:]]*考[[:space:]]*文[[:space:]]*献`)
)

var latinFold = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ä", "a", "ã", "a", "å", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "ô", "o", "ö", "o", "õ", "o", "ø", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"ñ", "n", "ç", "c", "ý", "y", "ÿ", "y", "ß", "ss", "æ", "ae", "œ", "oe",
)

func normalize(s string) string {
	s = latinFold.Replace(strings.ToLower(s))
	var b strings.Builder
	space := true
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			b.WriteRune(r)
			space = false
		} else if !space {
			b.WriteByte(' ')
			space = true
		}
	}
	return strings.TrimSpace(b.String())
}

func firstAuthor(authors string) string {
	authors = strings.TrimSpace(authors)
	if i := strings.IndexAny(authors, ",;"); i >= 0 {
		return strings.TrimSpace(authors[:i])
	}
	fields := strings.Fields(authors)
	if len(fields) > 0 {
		return fields[0]
	}
	return ""
}

func slugID(authors string, year int, title string) string {
	prefix := normalize(firstAuthor(authors))
	prefix = strings.ReplaceAll(prefix, " ", "_")
	var ascii strings.Builder
	for _, r := range prefix {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			ascii.WriteRune(r)
		}
	}
	if ascii.Len() == 0 {
		ascii.WriteString("paper")
	}
	y := "nd"
	if year > 0 {
		y = strconv.Itoa(year)
	}
	sum := sha1.Sum([]byte(normalize(title)))
	return fmt.Sprintf("%s_%s_%s", ascii.String(), y, hex.EncodeToString(sum[:4]))
}

func parseReference(raw string) (doi string, year int, author string) {
	if m := doiPattern.FindString(raw); m != "" {
		doi = strings.TrimRight(m, ".,;)")
	}
	if m := yearPattern.FindStringSubmatch(raw); len(m) >= 3 {
		year, _ = strconv.Atoi(m[2])
	}
	clean := strings.TrimSpace(numberedPattern.ReplaceAllString(raw, ""))
	runes := []rune(clean)
	for i, r := range runes {
		if r == ',' || unicode.IsSpace(r) {
			author = strings.TrimSpace(string(runes[:i]))
			break
		}
	}
	if author == "" && len(runes) > 0 {
		author = string(runes)
	}
	return
}

func looksLikeAuthorStart(line string) bool {
	if !authorYear.MatchString(line) {
		return false
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}
	r, _ := utf8FirstRune(line)
	return unicode.IsUpper(r) || unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana)
}

func utf8FirstRune(s string) (rune, int) {
	for _, r := range s {
		return r, len(string(r))
	}
	return 0, 0
}

func splitReferences(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	scanner := bufio.NewScanner(strings.NewReader(text))
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	var lines []string
	for scanner.Scan() {
		line := strings.Join(strings.Fields(scanner.Text()), " ")
		if line != "" {
			lines = append(lines, line)
		}
	}
	numbered := 0
	for _, line := range lines {
		if numberedPattern.MatchString(line) {
			numbered++
		}
	}
	var entries []string
	var current []string
	flush := func() {
		if len(current) == 0 {
			return
		}
		joined := strings.Join(current, " ")
		if len([]rune(joined)) >= 25 {
			entries = append(entries, joined)
		}
		current = nil
	}
	for _, line := range lines {
		isStart := false
		if numbered > 0 {
			isStart = numberedPattern.MatchString(line)
		} else {
			isStart = looksLikeAuthorStart(line)
		}
		if isStart {
			flush()
			current = []string{line}
		} else if len(current) > 0 {
			current = append(current, line)
		}
	}
	flush()
	return entries
}

func referenceSection(text string) string {
	last := -1
	lastEnd := -1
	for _, loc := range japaneseHeader.FindAllStringIndex(text, -1) {
		if loc[0] > last {
			last, lastEnd = loc[0], loc[1]
		}
	}
	for _, loc := range referenceHeader.FindAllStringIndex(text, -1) {
		if loc[0] > last {
			last, lastEnd = loc[0], loc[1]
		}
	}
	if last >= 0 {
		return text[lastEnd:]
	}
	start := len(text) * 65 / 100
	return text[start:]
}

func extractPDFText(path string) (string, error) {
	pdftotext, err := exec.LookPath("pdftotext")
	if err != nil {
		return "", fmt.Errorf("pdftotext was not found; install Poppler or use add --references FILE")
	}
	cmd := exec.Command(pdftotext, "-layout", path, "-")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pdftotext failed: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	if strings.TrimSpace(stdout.String()) == "" {
		return "", fmt.Errorf("could not extract text from PDF; run OCR first for an image-only PDF")
	}
	return stdout.String(), nil
}

func paperScore(ref *Reference, p *Paper) float64 {
	score := 0.0
	if ref.DOI != "" && p.DOI != "" && strings.EqualFold(ref.DOI, p.DOI) {
		score += 0.9
	}
	if ref.Year > 0 && p.Year > 0 && ref.Year == p.Year {
		score += 0.25
	}
	fa := normalize(ref.FirstAuthor)
	if fa != "" {
		for _, token := range strings.Fields(normalize(p.Authors)) {
			if token == fa {
				score += 0.32
				break
			}
		}
	}
	words := []string{}
	for _, w := range strings.Fields(normalize(p.Title)) {
		if len([]rune(w)) > 3 {
			words = append(words, w)
		}
	}
	if len(words) > 0 {
		raw := normalize(ref.RawText)
		hits := 0
		for _, w := range words {
			if strings.Contains(raw, w) {
				hits++
			}
		}
		score += 0.4 * float64(hits) / float64(len(words))
	}
	if score > 1 {
		return 1
	}
	return score
}

func reconcile(l *Ledger) {
	for i := range l.References {
		ref := &l.References[i]
		if ref.Status == "confirmed" {
			continue
		}
		bestScore := 0.0
		bestID := ""
		for j := range l.Papers {
			p := &l.Papers[j]
			if p.ID == ref.SourcePaperID {
				continue
			}
			s := paperScore(ref, p)
			if s > bestScore || (s == bestScore && p.ID < bestID) {
				bestScore, bestID = s, p.ID
			}
		}
		ref.Confidence = bestScore
		switch {
		case bestScore >= 0.84:
			ref.Status = "confirmed"
			ref.MatchedPaperID = bestID
			l.addCitation(ref.SourcePaperID, bestID, ref.ID)
		case bestScore >= 0.58:
			ref.Status = "suggested"
			ref.MatchedPaperID = bestID
		default:
			ref.Status = "unresolved"
			ref.MatchedPaperID = ""
		}
	}
	sort.Slice(l.Citations, func(i, j int) bool {
		a, b := l.Citations[i], l.Citations[j]
		if a.SourcePaperID == b.SourcePaperID {
			return a.TargetPaperID < b.TargetPaperID
		}
		return a.SourcePaperID < b.SourcePaperID
	})
}
