package yomiyose

import (
	"bytes"
	"encoding/csv"
	"math"
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
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	reader := csv.NewReader(bytes.NewReader(data))
	reader.Comma = '\t'
	reader.FieldsPerRecord = fields
	rows, err := reader.ReadAll()
	if err != nil || len(rows) < 2 {
		t.Fatalf("read %s: %v", path, err)
	}
	return rows
}

// Each hypothesis is a verbatim name span from a published ASR run. Keep the
// carrier fixed to compare edit rules without changing the surrounding text.
func TestFLEURSNameMasks(t *testing.T) {
	rows := readCorpus(t, "testdata/fleurs_names.tsv", 8)
	base := DefaultWeights()
	length, noGlide, byLength := base, base, base
	length.LongInsertion = 1.5
	noGlide.GlideAfterOU = math.Inf(1)
	byLength.LimitByMoraCount = true
	for _, tc := range []struct {
		label   string
		column  int
		weights Weights
		rules   RuleSet
		limit   float64
	}{
		{"default_mask", 4, base, AllRules, 1.5}, {"long_mask", 5, length, AllRules, 1.5},
		{"no_glide_mask", 6, noGlide, AllRules, 1.5}, {"foreign_rules", 4, base, ForeignRules, 1.5},
		{"default_by_length", 7, byLength, AllRules, 2}, {"foreign_by_length", 7, byLength, ForeignRules, 2},
	} {
		t.Run(tc.label, func(t *testing.T) {
			const prefix, suffix = "記録は「", "」です。"
			full, partial, exposed := 0, 0, 0
			for _, row := range rows[1:] {
				m, err := NewScoped([]ScopedName{{Name: Name{Text: row[2]}, Rules: tc.rules}}, tc.weights)
				if err != nil {
					t.Fatal(err)
				}
				input := prefix + row[3] + suffix
				got := m.Mask(input, tc.limit, "■")
				want := row[tc.column]
				if want == "=" {
					want = row[3]
				}
				if got != prefix+want+suffix {
					t.Errorf("%s %s: got %q, want %q", row[0], row[1], got, prefix+want+suffix)
				}
				switch got {
				case input:
					exposed++
				case prefix + "■" + suffix:
					full++
				default:
					partial++
				}
			}
			t.Logf("mentions=%d full=%d partial=%d exposed=%d", len(rows)-1, full, partial, exposed)
		})
	}
}

func TestKotobaKanaNameMasks(t *testing.T) {
	const prefix, suffix = "記録は「", "」です。"
	rows := readCorpus(t, "testdata/kotoba_kana_names.tsv", 5)
	for _, row := range rows[1:] {
		t.Run(row[0]+"/"+row[1], func(t *testing.T) {
			m := mustNew(t, DefaultWeights(), Name{Text: row[2]})
			want := row[4]
			if want == "=" {
				want = row[3]
			}
			if got := m.Mask(prefix+row[3]+suffix, 1.5, "■"); got != prefix+want+suffix {
				t.Errorf("registered %q, observed %q: got %q, want %q", row[2], row[3], got, prefix+want+suffix)
			}
		})
	}
}

func TestJapaneseASRReadingExamples(t *testing.T) {
	rows := readCorpus(t, "testdata/japanese_asr_names.tsv", 7)
	for _, row := range rows[1:] {
		if row[5] == "-" { // Only the written reference and prediction were recorded.
			continue
		}
		t.Run(row[1], func(t *testing.T) {
			for _, scope := range []RuleSet{AllRules, JapaneseRules} {
				m, err := NewScoped([]ScopedName{{Name: Name{Text: row[4]}, Rules: scope}}, DefaultWeights())
				if err != nil {
					t.Fatal(err)
				}
				got := m.Mask("記録は「"+row[5]+"」です。", 1.5, "■")
				if want := "記録は「" + row[6] + "」です。"; got != want {
					t.Errorf("rules %d: %s -> %s: got %q, want %q", scope, row[4], row[5], got, want)
				}
			}
		})
	}
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
