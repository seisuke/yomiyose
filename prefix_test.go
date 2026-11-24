package yomiyose

import (
	"math"
	"testing"
)

func TestNamePrefixes(t *testing.T) {
	const name = "サトウマサヒロ"
	const text = "サト、サトウ、サトーマザ、サトウマサヒロ、マサヒロ。"
	const ordinaryText = "おさとうを加えてください。サトウユウコさんも来ます。"
	w := DefaultWeights()
	m := mustNew(t, w, Name{Text: name})
	if got := m.MaskDefault(text, "■"); got != "サト、サトウ、サトーマザ、■、マサヒロ。" {
		t.Fatalf("default mask: %q", got)
	}
	if got := m.MaskDefault(ordinaryText, "■"); got != ordinaryText {
		t.Fatalf("default mask on ordinary text: %q", got)
	}
	w.MatchPrefixes = true
	m = mustNew(t, w, Name{Text: name})
	if got := m.MaskDefault(text, "■"); got != "サト、■、■、■、マサヒロ。" {
		t.Fatalf("prefix mask: %q", got)
	}
	hits := m.FindDefault(text)
	wantSpans := []string{"サトウ", "サトーマザ", "サトウマサヒロ"}
	wantCosts := []float64{0, .25, 0}
	if len(hits) != len(wantSpans) {
		t.Fatalf("hits: %+v", hits)
	}
	for i, hit := range hits {
		if hit.Name != name || hit.NameIndex != 0 || text[hit.Start:hit.End] != wantSpans[i] || hit.Cost != wantCosts[i] {
			t.Errorf("hit %d: %+v", i, hit)
		}
	}
	if got := m.MaskDefault(ordinaryText, "■"); got != "お■加えてください。■ウコさんも来ます。" {
		t.Errorf("matches without a name boundary: %q", got)
	}
	if hits := m.FindDefault("おさとう。"); len(hits) != 1 || hits[0].Cost != 0 {
		t.Errorf("normalized prefix cost: %+v", hits)
	}
}

func TestPrefixLimits(t *testing.T) {
	const name = "サイトウユウスケ"
	w := DefaultWeights()
	w.MatchPrefixes = true
	m := mustNew(t, w, Name{Text: name})
	const text = "🎤カトウさん。"
	hits := m.Find(text, .5)
	if len(hits) != 1 || hits[0].Cost != .5 || text[hits[0].Start:hits[0].End] != "カトウ" {
		t.Fatalf("inclusive limit: %+v", hits)
	}
	if got := m.Mask(text, .49, "■"); got != text {
		t.Errorf("caller limit: %q", got)
	}
	if got := m.Distance(name, "カトウ"); got <= .75 {
		t.Errorf("whole-reading distance: %g", got)
	}
	w.PrefixMaxCost = .49
	if got := mustNew(t, w, Name{Text: name}).MaskDefault(text, "■"); got != text {
		t.Errorf("configured cost: %q", got)
	}
	w.MinPrefixChars = 4
	m = mustNew(t, w, Name{Text: name})
	if got := m.MaskDefault("サトウ。サイトウ。", "■"); got != "サトウ。■。" {
		t.Errorf("configured minimum: %q", got)
	}
	w.MinPrefixChars, w.PrefixMaxCost = 3, 1
	w.MoraLimits.UpTo6 = .49
	if got := mustNew(t, w, Name{Text: name}).MaskDefault(text, "■"); got != text {
		t.Errorf("three-mora cap: %q", got)
	}
}

func TestPrefixAndFullNameOverlap(t *testing.T) {
	w := DefaultWeights()
	w.MatchPrefixes = true
	m, err := NewScoped([]ScopedName{
		{Name: Name{Text: "サイトウユウスケ"}, Rules: ForeignRules},
		{Name: Name{Text: "イシイカオリ", Readings: []string{"サイトウユウスケ"}}, Rules: JapaneseRules},
	}, w)
	if err != nil {
		t.Fatal(err)
	}
	const text = "カトウ。サイトウユウスコ。"
	hits := m.FindDefault(text)
	// Only the Japanese-scoped alias can combine the two cheap edits in カトウ.
	// Both complete readings match the second mention by dropping the final mora.
	if len(hits) != 3 || hits[0].NameIndex != 1 || hits[0].Name != "イシイカオリ" ||
		text[hits[0].Start:hits[0].End] != "カトウ" || hits[0].Cost != .5 {
		t.Fatalf("scoped and aliased hits: %+v", hits)
	}
	for _, hit := range hits[1:] {
		if text[hit.Start:hit.End] != "サイトウユウス" || hit.Cost != w.TailLoss {
			t.Errorf("full-name match suppressed by a cheaper prefix: %+v", hit)
		}
	}
	m = mustNew(t, w, Name{Text: "サトウマサヒロ"})
	if got := m.MaskDefault("サトウマコヒロさん", "■"); got != "■さん" {
		t.Errorf("full-name match above the prefix cost cap: %q", got)
	}
}

func TestPrefixKanaCharacters(t *testing.T) {
	w := DefaultWeights()
	w.MatchPrefixes = true
	m := mustNew(t, w, Name{Text: "シュンタカヒロ"})
	if got := m.MaskDefault("シュ、シュ・ン、シュ。ン。", "■"); got != "シュ、■、シュ。ン。" {
		t.Errorf("small kana character count: %q", got)
	}
	m = mustNew(t, w, Name{Text: "ザキヤマリョウタ"})
	if got := m.MaskDefault("ザ・キ、サ\u3099キ、ザ キ、ザキヤ。", "■"); got != "ザ・キ、サ\u3099キ、ザ キ、■。" {
		t.Errorf("normalized character count: %q", got)
	}
}

func TestPrefixCharacterBands(t *testing.T) {
	w := DefaultWeights()
	w.MatchPrefixes = true
	for _, tc := range []struct {
		name, text string
		cost       float64
	}{
		{"サイトウユウスケ", "カトウ", .5},
		{"タカハシリョウタ", "タケハシ", 1},
		{"オオバヤシタカヒロ", "オイハヤシ", 1.25},
	} {
		m := mustNew(t, w, Name{Text: tc.name})
		hits := m.FindDefault(tc.text)
		if len(hits) != 1 || hits[0].Start != 0 || hits[0].End != len(tc.text) || hits[0].Cost != tc.cost {
			t.Errorf("%s / %s: %+v", tc.name, tc.text, hits)
		}
	}
	for _, tc := range []struct{ name, text, want string }{
		{"タカハシリョウタ", "タケハ", "タケハ"},     // one ordinary edit exceeds the three-character cap.
		{"タカハシリョウタ", "タガミハシ", "タガミハシ"}, // five observed vs four registered: cost 1.25 exceeds 1.
		{"ヤマノウチダアキヒロ", "マノウチダ", "■ダ"},  // a sixth registered character cannot extend the prefix.
		{"コバヤカワタカヒロ", "コバヤカワタ。コバヤカワタカヒロ。", "■タ。■。"},
	} {
		m := mustNew(t, w, Name{Text: tc.name})
		if got := m.MaskDefault(tc.text, "■"); got != tc.want {
			t.Errorf("%s / %s: got %s, want %s", tc.name, tc.text, got, tc.want)
		}
	}
}

func TestPrefixOptionsValidation(t *testing.T) {
	w := DefaultWeights()
	w.MatchPrefixes = true
	for _, invalid := range []int{0, 2, 6} {
		w.MinPrefixChars = invalid
		if _, err := NewReadings(nil, w); err == nil {
			t.Errorf("accepted prefix minimum %d", invalid)
		}
	}
	w.MinPrefixChars = 3
	for _, field := range []*float64{&w.ThreeCharPrefixMaxCost, &w.FourCharPrefixMaxCost, &w.PrefixMaxCost} {
		for _, invalid := range []float64{-1, math.NaN()} {
			*field = invalid
			if _, err := NewReadings(nil, w); err == nil {
				t.Errorf("accepted invalid prefix cost %v", invalid)
			}
		}
		*field = 1
	}
}
