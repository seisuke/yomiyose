package yomiyose

import (
	"encoding/csv"
	"os"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

type nameSpan struct {
	start, end int
	optional   bool
}

// Brackets mark required mentions; a leading ? permits masking an extra span.
func annotatedText(t *testing.T, marked string) (string, []nameSpan) {
	t.Helper()
	var plain strings.Builder
	var spans []nameSpan
	for {
		prefix, rest, open := strings.Cut(marked, "【")
		if strings.Contains(prefix, "】") {
			t.Fatal("unmatched closing bracket", marked)
		}
		plain.WriteString(prefix)
		if !open {
			return plain.String(), spans
		}
		name, tail, closed := strings.Cut(rest, "】")
		name, optional := strings.CutPrefix(name, "?")
		if !closed || name == "" || strings.Contains(name, "【") {
			t.Fatal("invalid target", marked)
		}
		start := plain.Len()
		plain.WriteString(name)
		spans = append(spans, nameSpan{start, plain.Len(), optional})
		marked = tail
	}
}

func TestTranscriptCorpus(t *testing.T) {
	checkTranscriptCorpus(t, DefaultWeights(), nil)
}

func readCorpus(t *testing.T, path string, fields int) [][]string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	reader := csv.NewReader(f)
	reader.Comma = '\t'
	reader.FieldsPerRecord = fields
	rows, err := reader.ReadAll()
	if err != nil || len(rows) < 2 {
		t.Fatalf("read %s: %v", path, err)
	}
	return rows
}

func checkTranscriptCorpus(t *testing.T, w Weights, overrides map[string]string) {
	t.Helper()
	rows := readCorpus(t, "testdata/transcripts.tsv", 4)
	var full, partial, missed, allowed, extra int
	for _, row := range rows[1:] {
		t.Run(row[0], func(t *testing.T) {
			text, targets := annotatedText(t, row[2])
			m, err := NewReadings(strings.Split(row[1], "|"), w)
			if err != nil {
				t.Fatal(err)
			}
			hidden := make([]bool, len(text))
			for _, h := range m.Find(text, 1.5) {
				if h.Start < 0 || h.End > len(text) || h.Start >= h.End ||
					!utf8.RuneStart(text[h.Start]) || (h.End < len(text) && !utf8.RuneStart(text[h.End])) {
					t.Fatalf("invalid UTF-8 span %+v", h)
				}
				for i := h.Start; i < h.End; i++ {
					hidden[i] = true
				}
				required, permitted := false, false
				for _, s := range targets {
					if s.optional {
						permitted = permitted || (s.start <= h.Start && h.End <= s.end)
					} else if s.start < h.End && h.Start < s.end {
						required = true
						break
					}
				}
				if required {
					continue
				}
				if permitted {
					allowed++
					t.Logf("allowed: %q", text[h.Start:h.End])
				} else {
					extra++
					t.Logf("extra: %q", text[h.Start:h.End])
				}
			}
			for _, s := range targets {
				if s.optional {
					continue
				}
				covered, letters := 0, 0
				for offset, r := range text[s.start:s.end] {
					if unicode.IsLetter(r) || unicode.IsDigit(r) {
						letters++
						if hidden[s.start+offset] {
							covered++
						}
					}
				}
				if letters == 0 {
					t.Fatal("annotation has no name characters")
				}
				switch covered {
				case 0:
					missed++
					t.Errorf("fully exposed: %q in %q", text[s.start:s.end], text)
				case letters:
					full++
				default:
					partial++
					t.Logf("partly masked: %q", text[s.start:s.end])
				}
			}
			want := row[3]
			if override, ok := overrides[row[0]]; ok {
				want = override
			}
			if got := m.Mask(text, 1.5, "■"); got != want {
				t.Errorf("mask = %q, want %q", got, want)
			}
		})
	}
	t.Logf("texts=%d mentions=%d full=%d partial=%d exposed=%d allowed_hits=%d extra_hits=%d",
		len(rows)-1, full+partial+missed, full, partial, missed, allowed, extra)
}
