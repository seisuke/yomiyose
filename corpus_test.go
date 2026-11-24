package yomiyose

import (
	"bytes"
	"encoding/csv"
	"math"
	"os"
	"regexp"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

type nameSpan struct {
	start, end int
	optional   bool
}

var katakanaName = regexp.MustCompile("^[ァ-ヺー・\u3099\u309a]+$")

func requireKatakanaNames(t *testing.T, names ...string) {
	t.Helper()
	for _, name := range names {
		if !katakanaName.MatchString(name) {
			t.Fatalf("expected a katakana name, got %q", name)
		}
	}
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
		requireKatakanaNames(t, name)
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
	length, noGlide := base, base
	length.LongInsertion = 1.5
	noGlide.GlideAfterOU = math.Inf(1)
	for _, tc := range []struct {
		label    string
		column   int
		weights  Weights
		rules    RuleSet
		capAt1_5 bool
	}{
		{"default_mask", 7, base, AllRules, false}, {"long_mask", 5, length, AllRules, false},
		{"no_glide_mask", 6, noGlide, AllRules, false}, {"foreign_rules", 7, base, ForeignRules, false},
		{"max_1_5_mask", 4, base, AllRules, true},
	} {
		t.Run(tc.label, func(t *testing.T) {
			const prefix, suffix = "記録は「", "」です。"
			full, partial, exposed := 0, 0, 0
			for _, row := range rows[1:] {
				requireKatakanaNames(t, row[2], row[3])
				m, err := NewScoped([]ScopedName{{Name: Name{Text: row[2]}, Rules: tc.rules}}, tc.weights)
				if err != nil {
					t.Fatal(err)
				}
				input := prefix + row[3] + suffix
				var got string
				if tc.capAt1_5 {
					got = m.Mask(input, 1.5, "■")
				} else {
					got = m.MaskDefault(input, "■")
				}
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
	checkKanaNameMasks(t, "testdata/kotoba_kana_names.tsv")
}

func TestKoreanASRNameMasks(t *testing.T) {
	checkKanaNameMasks(t, "testdata/korean_asr_names.tsv")
}

func TestJapaneseFullNameMasks(t *testing.T) {
	const prefix, suffix = "記録は「", "」です。"
	type totals struct{ full, partial, exposed int }
	byVariation := make(map[string]totals)
	var names []Name
	matchers := make(map[string]*Matcher)
	seen := make(map[[2]string]bool)
	rows := readCorpus(t, "testdata/japanese_full_names.tsv", 4)
	for _, row := range rows[1:] {
		if name := row[1]; matchers[name] == nil {
			entry := Name{Text: name}
			matchers[name] = mustNew(t, DefaultWeights(), entry)
			names = append(names, entry)
		}
	}
	roster := mustNew(t, DefaultWeights(), names...)
	w := DefaultWeights()
	w.MatchSubstrings = true
	withParts := mustNew(t, w, names...)
	for _, row := range rows[1:] {
		variation, name, observed, expected := row[0], row[1], row[2], row[3]
		requireKatakanaNames(t, name, observed)
		pair := [2]string{name, observed}
		if seen[pair] || name == observed {
			t.Fatalf("duplicate or unchanged spelling: %q → %q", name, observed)
		}
		seen[pair] = true
		input := prefix + observed + suffix
		got := matchers[name].MaskDefault(input, "■")
		if got != prefix+expected+suffix {
			t.Errorf("%s / %s → %s: got %q, want %q", variation, name, observed, got, prefix+expected+suffix)
		}
		if withRoster := roster.MaskDefault(input, "■"); withRoster != got {
			t.Errorf("%s → %s: roster masked %q, single registration masked %q", name, observed, withRoster, got)
		}
		if masked := mustNew(t, w, Name{Text: name}).MaskDefault(input, "■"); masked != got {
			t.Errorf("single with substrings: %s: %s", name, masked)
		}
		wantParts := expected
		switch observed {
		case "オオノマドカ", "ヤマモリケンタ", "サクライユウコ":
			wantParts = "■"
		}
		if masked := withParts.MaskDefault(input, "■"); masked != prefix+wantParts+suffix {
			t.Errorf("MatchSubstrings: %s → %s: %q, want %q", name, observed, masked, prefix+wantParts+suffix)
		}
		n := byVariation[variation]
		switch got {
		case prefix + "■" + suffix:
			n.full++
		case input:
			n.exposed++
			t.Errorf("%s → %s: name fully exposed", name, observed)
		default:
			n.partial++
		}
		byVariation[variation] = n
	}
	var all totals
	for variation, n := range byVariation {
		t.Logf("%s full=%d partial=%d exposed=%d", variation, n.full, n.partial, n.exposed)
		all.full += n.full
		all.partial += n.partial
		all.exposed += n.exposed
	}
	t.Logf("names=%d spellings=%d full=%d partial=%d exposed=%d", len(names), len(seen), all.full, all.partial, all.exposed)
}

func checkKanaNameMasks(t *testing.T, path string) {
	t.Helper()
	const prefix, suffix = "記録は「", "」です。"
	rows := readCorpus(t, path, 5)
	for _, row := range rows[1:] {
		t.Run(row[0]+"/"+row[1], func(t *testing.T) {
			requireKatakanaNames(t, row[2], row[3])
			m := mustNew(t, DefaultWeights(), Name{Text: row[2]})
			want := row[4]
			if want == "=" {
				want = row[3]
			}
			if got := m.MaskDefault(prefix+row[3]+suffix, "■"); got != prefix+want+suffix {
				t.Errorf("registered %q, observed %q: got %q, want %q", row[2], row[3], got, prefix+want+suffix)
			}
		})
	}
}

func TestJapaneseASRReadingExamples(t *testing.T) {
	rows := readCorpus(t, "testdata/japanese_asr_names.tsv", 5)
	for _, row := range rows[1:] {
		t.Run(row[1], func(t *testing.T) {
			registered, assumed, expected := row[2], row[3], row[4]
			requireKatakanaNames(t, registered, assumed)
			for _, scope := range []RuleSet{AllRules, JapaneseRules} {
				m, err := NewScoped([]ScopedName{{Name: Name{Text: registered}, Rules: scope}}, DefaultWeights())
				if err != nil {
					t.Fatal(err)
				}
				got := m.MaskDefault("記録は「"+assumed+"」です。", "■")
				if want := "記録は「" + expected + "」です。"; got != want {
					t.Errorf("rules %d: %s -> %s: got %q, want %q", scope, registered, assumed, got, want)
				}
			}
		})
	}
}

func checkTranscriptCorpus(t *testing.T, w Weights, overrides map[string]string) {
	t.Helper()
	rows := readCorpus(t, "testdata/transcripts.tsv", 4)
	var texts, full, partial, missed, allowed, extra int
	for _, row := range rows[1:] {
		t.Run(row[0], func(t *testing.T) {
			texts++
			text, targets := annotatedText(t, row[2])
			names := strings.Split(row[1], "|")
			requireKatakanaNames(t, names...)
			m, err := NewReadings(names, w)
			if err != nil {
				t.Fatal(err)
			}
			hidden := make([]bool, len(text))
			for _, h := range m.FindDefault(text) {
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
			if got := m.MaskDefault(text, "■"); got != want {
				t.Errorf("mask = %q, want %q", got, want)
			}
		})
	}
	t.Logf("texts=%d mentions=%d full=%d partial=%d exposed=%d allowed_hits=%d extra_hits=%d",
		texts, full+partial+missed, full, partial, missed, allowed, extra)
}
