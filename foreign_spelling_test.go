package yomiyose

import (
	"math"
	"testing"
)

func TestForeignSpellingCosts(t *testing.T) {
	w := DefaultWeights()
	w.TuToWa, w.PuaContraction = .375, .5
	for _, tc := range []struct {
		name, observed string
		cost           float64
	}{
		{"トゥプア・タマセーゼ", "トプア・タマセーゼ", .375},
		{"トゥプア・タマセーゼ", "トワプア・タマセーゼ", .375},
		{"プア・タマセーゼ", "パ・タマセーゼ", .5},
		{"プア・タマセーゼ", "ッパ・タマセーゼ", .5},
	} {
		m := mustNew(t, w, Name{Text: tc.name})
		if got := m.Distance(tc.name, tc.observed); got != tc.cost {
			t.Errorf("%s → %s: distance %g, want %g", tc.name, tc.observed, got, tc.cost)
		}
		text := "受付は「" + tc.observed + "」です。"
		if got := m.Mask(text, tc.cost, "■"); got != "受付は「■」です。" {
			t.Errorf("mask at threshold for %s: %q", tc.observed, got)
		}
		if got := m.Mask(text, math.Nextafter(tc.cost, 0), "■"); got != text {
			t.Errorf("below threshold for %s: %q", tc.observed, got)
		}
		if got := m.DistanceWithRules(tc.name, tc.observed, JapaneseRules); got <= tc.cost {
			t.Errorf("JapaneseRules for %s: %g", tc.observed, got)
		}
	}
	const name, phrase = "トゥプア・タマセーゼ", "とわっぱ・タマセーゼ"
	m := mustNew(t, w, Name{Text: name})
	want := w.TuToWa + w.PuaContraction + 4*w.Hiragana
	if hits := m.Find(phrase, want); len(hits) != 1 || hits[0].Cost != want || phrase[hits[0].Start:hits[0].End] != phrase {
		t.Errorf("hiragana and original span: %+v", hits)
	}
	w.TuToWa, w.PuaContraction = math.Inf(1), math.Inf(1)
	if got := mustNew(t, w, Name{Text: name}).Mask("トワッパ・タマセーゼ", 1, "■"); got != "トワッパ・タマセーゼ" {
		t.Errorf("disabled spelling rules: %q", got)
	}
	for _, cost := range []*float64{&w.TuToWa, &w.PuaContraction, &w.ForeignLongInsertion, &w.ForeignLongDeletion} {
		*cost = -1
		if _, err := NewReadings(nil, w); err == nil {
			t.Error("accepted negative spelling or length cost")
		}
		*cost = 0
	}
}

func TestTupuaFullNameMask(t *testing.T) {
	const name = "トゥプア・タマセーゼ・レアロフィ"
	const observed = "トワッパー・タマセ・ゼ・デアロフィ"
	w := DefaultWeights()
	w.LimitByMoraCount = true
	m, err := NewReadings([]string{name}, w)
	if err != nil {
		t.Fatal(err)
	}
	if got := m.DistanceWithRules(name, observed, ForeignRules); got != 2 {
		t.Errorf("foreign distance = %g", got)
	}
	if got := m.Distance(name, observed); got != 2 {
		t.Errorf("all-rules distance = %g", got)
	}
	const text = "受付は「" + observed + "」です。"
	if hits := m.Find(text, 2); len(hits) != 1 || hits[0].Name != name || hits[0].Cost != 2 || text[hits[0].Start:hits[0].End] != observed {
		t.Errorf("full name span at 2: %+v", hits)
	}
	if got := m.Mask(text, 2, "■"); got != "受付は「■」です。" {
		t.Errorf("mask at 2: %q", got)
	}
	if got := m.Mask(text, math.Nextafter(2, 0), "■"); got != text {
		t.Errorf("mask below 2: %q", got)
	}
	w.ForeignLongInsertion, w.ForeignLongDeletion = .625, .875
	m = mustNew(t, w)
	for _, scope := range []RuleSet{AllRules, ForeignRules} {
		if got := m.DistanceWithRules(name, observed, scope); got != 2.5 {
			t.Errorf("custom length costs with rules %d: %g", scope, got)
		}
	}
}

func TestLongVowelsInAlternateReadings(t *testing.T) {
	m := mustNew(t, DefaultWeights(), Name{Text: "レナ・ビダル", Readings: []string{"レナ・ヴィダル"}})
	const text = "記録は「レーナ・ビダル」「レーナ・ヴィダル」です。"
	if got := m.Mask(text, .5, "■"); got != "記録は「レーナ・ビダル」「■」です。" {
		t.Errorf("length cost per reading: %q", got)
	}
	if got := m.Mask(text, .75, "■"); got != "記録は「■」「■」です。" {
		t.Errorf("both readings: %q", got)
	}
}

func TestShortenedTupuaTranscript(t *testing.T) {
	// Both shortened observations also match with the new spelling rules disabled.
	const name = "トゥプア・タマセーゼ・レアロフィ"
	w := DefaultWeights()
	w.TuToWa, w.PuaContraction = math.Inf(1), math.Inf(1)
	m := mustNew(t, w, Name{Text: name})
	for _, tc := range []struct {
		observed string
		cost     float64
	}{
		{"トゥプア・タマセゼ・レアロ", 1.25}, // missing ー and final フィ
		{"トゥプア・タマセーゼ・レアフィ", 1},  // missing medial ロ
	} {
		const prefix, suffix = "先ほど、", "さんが受付に来ました。"
		text := prefix + tc.observed + suffix
		hits := m.Find(text, 1.5)
		if len(hits) != 1 || hits[0].Name != name || hits[0].Cost != tc.cost ||
			hits[0].Start != len(prefix) || hits[0].End != len(prefix)+len(tc.observed) {
			t.Errorf("Find(%q) = %+v", text, hits)
		}
		if got := m.Mask(text, 1.5, "［氏名］"); got != prefix+"［氏名］"+suffix {
			t.Errorf("Mask(%q) = %q", text, got)
		}
	}
}
