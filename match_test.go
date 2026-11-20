package yomiyose

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

func mustNew(t *testing.T, w Weights, names ...Name) *Matcher {
	t.Helper()
	m, err := New(names, w)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestPhoneticRules(t *testing.T) {
	m := mustNew(t, DefaultWeights())
	for _, tc := range []struct {
		a, b string
		want float64
	}{
		{"ヤマモトユウスケ", "ヤマモトユースケ", 0},
		{"イシーカオリ", "イシイカオリ", 0}, {"yuusuke", "yu-suke", 0}, {"isii", "ishi:", 0},
		{"seisuke", "se:suke", 0}, {"satou", "サトー", 0},
		{"サイトウ", "サトウ", .25}, {"サトウ", "サイトウ", .25}, {"kato", "sato", .5},
		{"サイトウ", "カトウ", .75}, {"りょうた", "りょう", .75},
		{"ヴィバルディ", "ビバルディ", .25}, {"ヴィヴァルディ", "ビバルディ", .5},
		{"デイビット", "デービット", 0}, {"デイビット", "デビット", .25},
		{"debi^to", "デビト", .25}, {"デビト", "デビット", .25}, {"ミシェル", "ミッシェル", .25},
		{"マーティン", "マーチン", .25}, {"ジョセフ", "ジョゼフ", .25}, {"エディー", "エディ", .125}, {"ガルーチ", "ガルチ", .25},
		{"デービッド", "デビト", .75}, {"タナカ", "タナッカ", 2}, {"タナカ", "タッナカ", 2},
		{"ｳﾞｨｳﾞｧﾙﾃﾞｨ", "ヴィヴァルディ", 0}, {"イシイ・カオリ", "いしーかおり", 0},
	} {
		if got := m.Distance(tc.a, tc.b); got != tc.want {
			t.Errorf("%s -> %s = %g; want %g", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestMaskTranscript(t *testing.T) {
	readings := []string{"ヴィバルディ", "イシーカオリ"}
	m, err := NewReadings(readings, DefaultWeights())
	if err != nil {
		t.Fatal(err)
	}
	prefix := strings.Repeat("この会議の記録です。", 500)
	input := prefix + "🙂イシイ・カオリさんが来て、イシイカオさんが帰った。いしいかおりさん。"
	want := []string{"イシイ・カオリ", "イシイカオ", "いしいかおり"}
	cost := []float64{0, .75, .75}
	hits := m.Find(input, 1.5)
	if len(hits) != len(want) {
		t.Fatalf("hits = %+v", hits)
	}
	for i, h := range hits {
		if input[h.Start:h.End] != want[i] || h.Cost != cost[i] || h.NameIndex != 1 || h.Name != readings[1] {
			t.Errorf("got %+v (%s)", h, input[h.Start:h.End])
		}
	}
	if got := m.Mask(input, 1.5, "＊＊"); got != prefix+"🙂＊＊さんが来て、＊＊さんが帰った。＊＊さん。" {
		t.Error("incorrect mask")
	}
}

func TestThresholdAndScript(t *testing.T) {
	w := DefaultWeights()
	m := mustNew(t, w, Name{Text: "サイトウ"})
	text := "サトウ。さいとう。サイトウ"
	for _, tc := range []struct {
		threshold float64
		want      int
	}{{0, 1}, {.24, 1}, {.25, 2}, {.5, 3}} {
		if got := m.Find(text, tc.threshold); len(got) != tc.want {
			t.Errorf("threshold %g: %+v", tc.threshold, got)
		}
	}
	w.Hiragana = 0
	w.VowelLoss = 1.25
	m = mustNew(t, w, Name{Text: "サイトウ"})
	if got := m.Find(text, .5); len(got) != 2 || got[0].Cost != 0 {
		t.Errorf("custom weights: %+v", got)
	}
}

func TestBoundariesAndOverlap(t *testing.T) {
	m := mustNew(t, DefaultWeights(), Name{Text: "ナナ"}, Name{Text: "ナナコ"})
	if got := m.Mask("🟢ナナナコ、ナ。ナ", 0, "X"); got != "🟢X、ナ。ナ" {
		t.Error(got)
	}
	m = mustNew(t, DefaultWeights(), Name{Text: "デイビット", Readings: []string{"デービット"}})
	text := "『ﾃﾞｰﾋﾞｯﾄ』とデ・イビット"
	if got := m.Mask(text, 0, "*"); got != "『*』と*" {
		t.Error(got)
	}
	if got := m.Find("デイ。ビット", 1.5); len(got) != 0 {
		t.Errorf("crossed punctuation: %+v", got)
	}
	// A following vowel in prose is outside the name even if it could be folded.
	m = mustNew(t, DefaultWeights(), Name{Text: "イシイ"})
	if got := m.Mask("きいしいです", .5, "X"); got != "きXです" {
		t.Error(got)
	}
}

func TestInvalidAndDisabled(t *testing.T) {
	w := DefaultWeights()
	w.BV = math.Inf(1)
	m := mustNew(t, w, Name{Text: "ビバルディ"})
	if got := m.Find("ヴィバルディ", 1.5); len(got) != 0 {
		t.Errorf("disabled b/v: %+v", got)
	}
	if got := m.Find("ビバルディ", math.NaN()); len(got) != 0 {
		t.Error(got)
	}
	for _, name := range []string{"", "・", "ア。イ"} {
		if _, err := New([]Name{{Text: name}}, w); err == nil {
			t.Error(name)
		}
	}
	w.Other = -1
	if _, err := New(nil, w); err == nil {
		t.Error("negative cost")
	}
}

func ExampleMatcher_Find() {
	m, err := NewReadings([]string{"イシーカオリ"}, DefaultWeights())
	if err != nil {
		panic(err)
	}
	text := "受付でイシイカオさんが待っています。"
	for _, h := range m.Find(text, 1.5) {
		fmt.Println(h.Name, text[h.Start:h.End], h.Cost)
	}
	fmt.Println(m.Mask(text, 1.5, "［氏名］"))
	// Output:
	// イシーカオリ イシイカオ 0.75
	// 受付で［氏名］さんが待っています。
}

func TestTunableRulesAndSearchFallback(t *testing.T) {
	w := DefaultWeights()
	w.EiSpelling, w.OuSpelling = .125, .5
	w.Geminate, w.InitialKS, w.TailLoss = .5, 1, 1.25
	m := mustNew(t, w, Name{Text: "石井 香織", Readings: []string{"イシーカオリ"}}, Name{Text: "サトウ"})
	for _, tc := range []struct {
		a, b string
		want float64
	}{
		{"セイスケ", "セースケ", .125}, {"サトウ", "サトー", .5},
		{"デビット", "デビト", .5}, {"サトウ", "カトウ", 1}, {"りょうた", "りょう", 1.25},
	} {
		if got := m.Distance(tc.a, tc.b); got != tc.want {
			t.Errorf("%s/%s = %g, want %g", tc.a, tc.b, got, tc.want)
		}
	}
	if got := m.Mask("🙂石井　香織さん", 0, "X"); got != "🙂Xさん" {
		t.Error(got)
	}
	// Once ordinary edits are affordable, a name can start with an
	// arbitrary substituted mora. The indexed path must not omit this hit.
	m = mustNew(t, DefaultWeights(), Name{Text: "キムラ"})
	hits := m.Find("昨日、タムラさん。", 2)
	if len(hits) != 1 || hits[0].Start != len("昨日、") || hits[0].End != len("昨日、タムラ") || hits[0].Cost != 2 {
		t.Errorf("%+v", hits)
	}
}

func TestSearchDoesNotBorrowLongVowelContext(t *testing.T) {
	m := mustNew(t, DefaultWeights(), Name{Text: "イワタ"})
	if h := m.Find("キーワタ", 0); len(h) != 0 {
		t.Errorf("borrowed text outside match: %+v", h)
	}
}
