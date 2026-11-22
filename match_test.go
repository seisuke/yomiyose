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

func TestRuleSetDistances(t *testing.T) {
	w := DefaultWeights()
	w.GlideAfterOU = .5
	m := mustNew(t, w)
	for _, tc := range []struct {
		ref, obs               string
		all, japanese, foreign float64
	}{
		{"イシイカオリ", "イシーカオリ", 0, 0, 0},
		{"ヤマザキケント", "ヤマサキケント", .25, .25, .25},
		{"サイトウミホ", "サトウミホ", .25, .25, 2},
		{"カトウミホ", "サトウミホ", .5, .5, 2},
		{"オオノユウスケ", "オノユースケ", .25, .25, 1.25},
		{"デビットマッキー", "デービットマッキー", .25, 2, .25},
		{"デビットマッキー", "デビトマッキー", .25, 2, .25},
		{"アントニオヴィバルディ", "アントニオビバルディ", .25, 2, .25},
		{"ジェームズ・パジェット", "ジェームス・パシェット", .5, 2.25, .5},
		{"ジェームス・パシェット", "ジェームズ・パジェット", .5, 2.25, .5},
		{"ジェームズ・パジェット", "シェームズ・パジェット", 2, 2, 2},
		{"キム・シウ", "キム・シュウ", .5, .5, .5},
		{"kimusyuu", "kimusiu", .5, .5, .5},
		{"タナカしゅうじ", "タナカシウジ", .5, .5, .5},
		{"タナカシウジ", "タナカシュウジ", .5, .5, .5},
		{"ジョシュ・ホーキンソン", "ジョジホーキンソン", .75, 2, .75},
		{"ジョジホーキンソン", "ジョシュ・ホーキンソン", .75, 2, .75},
		{"ジェームズ・バリ", "ジェームス・パル", 1, 4.25, 1},
		{"ジェームス・パル", "ジェームズ・バリ", 1, 4.25, 1},
		{"キム・ミウ", "キム・ムウ", 2, 2, 2},
		{"マーティンホール", "マーチンホール", .25, 2, .25},
		{"ウィリアムモリス", "ウイリアムモリス", .25, 4, .25},
		{"リーエディー", "リーエディ", .125, 1.25, .125},
		{"レジェップタイイップエルドアン", "レジェップタイイップエルドワン", .5, 2, .5},
		{"オオノデビット", "オノデービット", .5, 2.25, 1.5},
		{"キクチユウスケ", "キチユウスケ", 2, 2, 2},
	} {
		for scope, want := range []float64{tc.all, tc.japanese, tc.foreign} {
			if got := m.DistanceWithRules(tc.ref, tc.obs, RuleSet(scope)); got != want {
				t.Errorf("rules %d: %s -> %s = %g, want %g", scope, tc.ref, tc.obs, got, want)
			}
		}
		if got := m.Distance(tc.ref, tc.obs); got != tc.all {
			t.Errorf("Distance(%s, %s) = %g, want %g", tc.ref, tc.obs, got, tc.all)
		}
	}
	w.MedialVoicing = .75
	m = mustNew(t, w, Name{Text: "ジェームズ・パジェット"})
	const transcript = "受付はジェームス・パシェットさんです。"
	if got := m.Mask(transcript, 1.5, "■"); got != "受付は■さんです。" {
		t.Errorf("tuned MedialVoicing at threshold: %q", got)
	}
	if got := m.Mask(transcript, 1.49, "■"); got != transcript {
		t.Errorf("tuned MedialVoicing below threshold: %q", got)
	}
	w.LongInsertion = .5
	m = mustNew(t, w)
	if j, f := m.DistanceWithRules("タナカミホ", "ターナカミホ", JapaneseRules),
		m.DistanceWithRules("タナカミホ", "ターナカミホ", ForeignRules); j != 2 || f != .5 {
		t.Errorf("repeated-vowel penalty: Japanese %g, foreign %g", j, f)
	}
}

func TestHighVowelWeights(t *testing.T) {
	w := DefaultWeights()
	w.ShiShu = .625
	w.RiRu = .875
	w.MedialVoicing = .5
	m := mustNew(t, w, Name{Text: "シウ・キム"})
	const text = "受付でシュウ・キムさんをお呼びします。"
	if got := m.Mask(text, .625, "■"); got != "受付で■さんをお呼びします。" {
		t.Errorf("initial シ/シュ: got %q", got)
	}
	if got := m.Mask(text, .624, "■"); got != text {
		t.Errorf("below the substitution cost: got %q", got)
	}
	if got := m.Distance("ジョシュ・ホーキンソン", "ジョジホーキンソン"); got != 1.125 {
		t.Errorf("composed vowel and voicing cost: got %g, want 1.125", got)
	}
	if got := m.Distance("ジェームズ・バリ", "ジェームズ・バル"); got != .875 {
		t.Errorf("tuned リ/ル: got %g, want .875", got)
	}
	w.ShiShu = math.Inf(1)
	m = mustNew(t, w)
	if got := m.Distance("ジョシュ・ホーキンソン", "ジョジホーキンソン"); got != w.Other {
		t.Errorf("disabled vowel rule: got %g, want %g", got, w.Other)
	}
	w.RiRu = -1
	if _, err := NewReadings(nil, w); err == nil {
		t.Error("accepted a negative high-vowel cost")
	}
}

func TestJapaneseShujiMask(t *testing.T) {
	m, err := NewScoped([]ScopedName{{Name: Name{Text: "タナカしゅうじ"}, Rules: JapaneseRules}}, DefaultWeights())
	if err != nil {
		t.Fatal(err)
	}
	const text = "担当はタナカシウジさん、次はタナカしうじさんです。"
	if got := m.Mask(text, .5, "■"); got != "担当は■さん、次はタナカしうじさんです。" {
		t.Errorf("at the katakana cost: got %q", got)
	}
	if got := m.Mask(text, .875, "■"); got != "担当は■さん、次は■さんです。" {
		t.Errorf("including three hiragana moras: got %q", got)
	}
}

func TestScopedFindAndMask(t *testing.T) {
	// Register each name twice to check that candidates at the same byte
	// offset use the intended rule set, including for aliases.
	jp := Name{Text: "カトウミホ"}
	foreign := Name{Text: "ウィリアムモリス", Readings: []string{"ウィリアムモーリス"}}
	m, err := NewScoped([]ScopedName{
		{Name: jp, Rules: JapaneseRules}, {Name: jp, Rules: ForeignRules},
		{Name: foreign, Rules: JapaneseRules}, {Name: foreign, Rules: ForeignRules},
		{Name: Name{Text: "オオノデビット"}},
	}, DefaultWeights())
	if err != nil {
		t.Fatal(err)
	}
	input := "受付でサトウミホさん、ウイリアム・モーリスさん、オノデービットさんをお呼びします。"
	want := []struct {
		index int
		text  string
		cost  float64
	}{{0, "サトウミホ", .5}, {3, "ウイリアム・モーリス", .25}, {4, "オノデービット", .5}}
	hits := m.Find(input, .5)
	if len(hits) != len(want) {
		t.Fatalf("Find returned %+v, want %d hits", hits, len(want))
	}
	for i, hit := range hits {
		if hit.NameIndex != want[i].index || input[hit.Start:hit.End] != want[i].text || hit.Cost != want[i].cost {
			t.Errorf("unexpected hit %+v for %q", hit, input[hit.Start:hit.End])
		}
	}
	if got := m.Mask(input, .5, "■"); got != "受付で■さん、■さん、■さんをお呼びします。" {
		t.Errorf("Mask = %q", got)
	}
	const invalid RuleSet = 255
	if _, err := NewScoped([]ScopedName{{Name: jp, Rules: invalid}}, DefaultWeights()); err == nil {
		t.Error("NewScoped accepted unknown rule set")
	}
	if got := m.DistanceWithRules("イシイカオリ", "イシーカオリ", invalid); !math.IsInf(got, 1) {
		t.Errorf("unknown rule set gave distance %g", got)
	}
}

func TestPhoneticRules(t *testing.T) {
	m := mustNew(t, DefaultWeights())
	for _, tc := range []struct {
		a, b string
		want float64
	}{
		{"ヤマモトユウスケ", "ヤマモトユースケ", 0},
		{"イシーカオリ", "イシイカオリ", 0}, {"yamamotoyuusuke", "yamamotoyu-suke", 0}, {"isiikaori", "ishi:kaori", 0},
		{"kimuraseisuke", "kimurase:suke", 0}, {"satoumiho", "サトーミホ", 0},
		{"サイトウミホ", "サトウミホ", .25}, {"サトウミホ", "サイトウミホ", .25}, {"katotakumi", "satotakumi", .5},
		{"サイトウミホ", "カトウミホ", .75}, {"いしだりょうた", "いしだりょう", .75},
		{"アントニオヴィバルディ", "アントニオビバルディ", .25}, {"アントニオヴィヴァルディ", "アントニオビバルディ", .5},
		{"デイビットマッキー", "デービットマッキー", 0}, {"デイビットマッキー", "デビットマッキー", .25},
		{"debi^tomakki:", "デビトマッキー", .25}, {"デビトマッキー", "デビットマッキー", .25}, {"ミシェルモラン", "ミッシェルモラン", .25},
		{"マーティンホール", "マーチンホール", .25}, {"ジョセフスミス", "ジョゼフスミス", .25}, {"リーエディー", "リーエディ", .125}, {"ジョンガルーチ", "ジョンガルチ", .25},
		{"デービッドマッキー", "デビトマッキー", .75}, {"タナカミホ", "タナッカミホ", 2}, {"タナカミホ", "タッナカミホ", 2},
		{"ｱﾝﾄﾆｵ･ｳﾞｨｳﾞｧﾙﾃﾞｨ", "アントニオヴィヴァルディ", 0}, {"イシイ・カオリ", "いしーかおり", 0},
	} {
		if got := m.Distance(tc.a, tc.b); got != tc.want {
			t.Errorf("%s -> %s = %g; want %g", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestLongVowelContext(t *testing.T) {
	m := mustNew(t, DefaultWeights())
	for _, tc := range []struct {
		short, long string
		want        float64
	}{
		{"tanakamiho", "ta:nakamiho", 2}, {"タナカミホ", "タアナカミホ", 2},
		{"タナカミホ", "タナーカミホ", 2}, {"タナカミホ", "タナカーミホ", 2},
		{"ヤマダリク", "ヤーマダリク", 2}, {"イシイトモコ", "イシイトーモコ", 2},
		{"デビットマッキー", "デービットマッキー", .25}, {"デビットマッキー", "デイビットマッキー", .25},
		{"オノマイ", "オオノマイ", .25}, {"ウィリアムシェクスピア", "ウィリアムシェークスピア", .25},
		{"ターカタミホ", "タアカタミホ", 0}, // Both readings already have the long vowel.
	} {
		for _, p := range [][2]string{{tc.short, tc.long}, {tc.long, tc.short}} {
			if got := m.Distance(p[0], p[1]); got != tc.want {
				t.Errorf("%s -> %s: got %g, want %g", p[0], p[1], got, tc.want)
			}
		}
	}
}

func TestGlideAfterOU(t *testing.T) {
	w := DefaultWeights()
	w.GlideAfterOU = .5
	m := mustNew(t, w)
	for _, tc := range []struct {
		a, b string
		want float64
	}{
		{"レセップタイップエルドアン", "レセップタイップエルドワン", .5},
		{"トゥプアタマセーゼレアロフィ", "トゥプワタマセーゼレアロフィ", .5},
		{"ヨシダアヤ", "ヨシダワヤ", 2},                 // The preceding vowel is a.
		{"レセップタイップエルドアン", "レセップタイップエルドワル", 4}, // The following sounds differ.
	} {
		if got := m.Distance(tc.a, tc.b); got != tc.want {
			t.Errorf("%s -> %s: got %g, want %g", tc.a, tc.b, got, tc.want)
		}
	}
	if got := m.Distance("トゥプワタマセーゼレアロフィ", "トゥプアタマセーゼレアロフィ"); got != .5 {
		t.Errorf("inverse glide: %g", got)
	}
	t.Run("transcripts", func(t *testing.T) { checkTranscriptCorpus(t, w, nil) })
	w.GlideAfterOU = -1
	if _, err := NewReadings(nil, w); err == nil {
		t.Error("accepted a negative glide cost")
	}
}

func TestLongVowelContextInTranscript(t *testing.T) {
	m, err := NewReadings([]string{"タナカミホ", "デビットマッキー"}, DefaultWeights())
	if err != nil {
		t.Fatal(err)
	}
	text := "ターナカミホさんとデービットマッキーさん。タナカミホさん。"
	if got := m.Mask(text, 1.5, "X"); got != "ターナカミホさんとXさん。Xさん。" {
		t.Error(got)
	}
	// Bi is outside the registered reading, so it cannot license De / De:.
	short := mustNew(t, DefaultWeights(), Name{Text: "ヤマダカナデ"})
	if got := short.Mask("ヤマダカナデービ", .25, "X"); got != "Xービ" {
		t.Error(got)
	}
}

func TestFullSizeKana(t *testing.T) {
	for _, tc := range []struct {
		registered, observed string
		edits                int
		script               float64
	}{
		{"ウィリアムウィリス", "ウイリアム・ウイリス", 2, 0},
		{"ウイリアムウイリス", "ウィリアム・ウィリス", 2, 0},
		{"ウィリアムモリス", "ういリアム・モリス", 1, .25},
		{"ウイリアムモリス", "うぃリアム・モリス", 1, .125},
		{"ハーバートジョージウェルズ", "ハーバート・ジョージ・ウエルズ", 1, 0},
		{"ウィリアムシェクスピア", "ウィリアム・シエクスピア", 1, 0},
		{"wiriamumorisu", "uiriamumorisu", 1, 0},
	} {
		w := DefaultWeights()
		w.KanaSize = .375
		m := mustNew(t, w, Name{Text: tc.registered})
		dist := float64(tc.edits) * w.KanaSize
		if got := m.Distance(tc.registered, tc.observed); got != dist {
			t.Errorf("%s -> %s: distance %g, want %g", tc.registered, tc.observed, got, dist)
		}
		text := "記録は「" + tc.observed + "」です。"
		limit := dist + tc.script
		h := m.Find(text, limit)
		if len(h) != 1 || h[0].Cost != limit || text[h[0].Start:h[0].End] != tc.observed {
			t.Errorf("%s at %g: %+v", tc.observed, limit, h)
		}
		if h := m.Find(text, limit-.01); len(h) != 0 {
			t.Errorf("%s below threshold: %+v", tc.observed, h)
		}
	}
	w := DefaultWeights()
	m := mustNew(t, w, Name{Text: "ウィリアムモリス"})
	for _, text := range []string{"ウ。イリアムモリス", "ウリアムモリス", "ウアリアムモリス"} {
		if h := m.Find(text, .5); len(h) != 0 {
			t.Errorf("incomplete kana pair: %+v", h)
		}
	}
	w.KanaSize = math.Inf(1)
	m = mustNew(t, w, Name{Text: "ウィリアムモリス"})
	if h := m.Find("ウイリアムモリス", .5); len(h) != 0 {
		t.Errorf("disabled kana-size rule: %+v", h)
	}
}

func TestMaskLongSuffix(t *testing.T) {
	w := DefaultWeights()
	m := mustNew(t, w, Name{Text: "セオドアトーマス"})
	text := "指揮はスイオドア・トマス。"
	h := m.Find(text, 1.5)
	if len(h) != 1 || h[0].Cost != 1.5 || text[h[0].Start:h[0].End] != "オドア・トマス" {
		t.Fatalf("long suffix: %+v", h)
	}
	if len(m.Find(text, 1.49)) != 0 || m.Distance("セオドアトーマス", "スイオドア・トマス") <= 1.5 {
		t.Error("head loss must apply only to an eligible search span")
	}
	for _, registered := range []string{"オカダユウスケ", "オカダユースケ"} {
		m := mustNew(t, w, Name{Text: registered})
		if h := m.Find("カダユースケ", 1.5); len(h) != 0 {
			t.Errorf("%s: counted vowel length as an independent mora: %+v", registered, h)
		}
	}
	w.HeadLoss = math.Inf(1)
	m = mustNew(t, w, Name{Text: "セオドアトーマス"})
	if len(m.Find(text, 1.5)) != 0 {
		t.Error("disabled head loss")
	}
	for _, field := range []*float64{&w.HeadLoss, &w.KanaSize} {
		*field = -1
		if _, err := New(nil, w); err == nil {
			t.Error("accepted negative rule cost")
		}
		*field = 0
	}
}

func TestRepeatedVowelPenaltyWeight(t *testing.T) {
	w := DefaultWeights()
	w.LongDeletion = .25
	for _, tc := range []struct{ penalty, want float64 }{{0, .25}, {.5, .75}} {
		w.RepeatedVowelPenalty = tc.penalty
		m := mustNew(t, w)
		if got := m.Distance("ta:nakamiho", "tanakamiho"); got != tc.want {
			t.Errorf("penalty %g: distance = %g, want %g", tc.penalty, got, tc.want)
		}
	}
	w.RepeatedVowelPenalty = -1
	if _, err := NewReadings(nil, w); err == nil {
		t.Error("accepted negative long-vowel context penalty")
	}
}

func TestMaskTranscript(t *testing.T) {
	readings := []string{"アントニオヴィバルディ", "イシーカオリ"}
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
	m := mustNew(t, w, Name{Text: "サイトウミホ"})
	text := "サトウミホ。さいとうミホ。サイトウミホ"
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
	m = mustNew(t, w, Name{Text: "サイトウミホ"})
	if got := m.Find(text, .5); len(got) != 2 || got[0].Cost != 0 {
		t.Errorf("custom weights: %+v", got)
	}
}

func TestBoundariesAndOverlap(t *testing.T) {
	m := mustNew(t, DefaultWeights(), Name{Text: "ウエノマキ"}, Name{Text: "マキサトシ"})
	if got := m.Mask("🟢ウエノマキサトシ、ウエノ。マキ", 0, "X"); got != "🟢X、ウエノ。マキ" {
		t.Error(got)
	}
	m = mustNew(t, DefaultWeights(), Name{Text: "デイビットマッキー", Readings: []string{"デービットマッキー"}})
	text := "『ﾃﾞｰﾋﾞｯﾄ･ﾏｯｷｰ』とデ・イビットマッキー"
	if got := m.Mask(text, 0, "*"); got != "『*』と*" {
		t.Error(got)
	}
	if got := m.Find("デイ。ビットマッキー", 1.5); len(got) != 0 {
		t.Errorf("crossed punctuation: %+v", got)
	}
	// A following vowel in prose is outside the name even if it could be folded.
	m = mustNew(t, DefaultWeights(), Name{Text: "イシイミホ"})
	if got := m.Mask("きいしいミホです", .5, "X"); got != "きXです" {
		t.Error(got)
	}
}

func TestInvalidAndDisabled(t *testing.T) {
	w := DefaultWeights()
	w.BV = math.Inf(1)
	m := mustNew(t, w, Name{Text: "アントニオビバルディ"})
	if got := m.Find("アントニオヴィバルディ", 1.5); len(got) != 0 {
		t.Errorf("disabled b/v: %+v", got)
	}
	if got := m.Find("アントニオビバルディ", math.NaN()); len(got) != 0 {
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
	m := mustNew(t, w, Name{Text: "石井 香織", Readings: []string{"イシーカオリ"}}, Name{Text: "サトウミホ"})
	for _, tc := range []struct {
		a, b string
		want float64
	}{
		{"キムラセイスケ", "キムラセースケ", .125}, {"サトウミホ", "サトーミホ", .5},
		{"デビットマッキー", "デビトマッキー", .5}, {"サトウミホ", "カトウミホ", 1}, {"いしだりょうた", "いしだりょう", 1.25},
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
	m = mustNew(t, DefaultWeights(), Name{Text: "キムラアヤ"})
	hits := m.Find("昨日、タムラアヤさん。", 2)
	if len(hits) != 1 || hits[0].Start != len("昨日、") || hits[0].End != len("昨日、タムラアヤ") || hits[0].Cost != 2 {
		t.Errorf("%+v", hits)
	}
}

func TestSearchDoesNotBorrowLongVowelContext(t *testing.T) {
	m := mustNew(t, DefaultWeights(), Name{Text: "イワタサキ"})
	if h := m.Find("キーワタサキ", 0); len(h) != 0 {
		t.Errorf("borrowed text outside match: %+v", h)
	}
}

func TestDirectionalLengthCosts(t *testing.T) {
	m := mustNew(t, DefaultWeights())
	for _, tc := range []struct {
		short, long  string
		insert, drop float64
	}{
		{"ヨシダアヤ", "ヨーシダアヤ", 2, 1.25}, {"サカケイスケ", "サーカケイスケ", 2, 1.25},
		{"フジタユスケ", "フジタユースケ", 2, 1.25}, {"ミズノレナ", "ミズノレーナ", 2, 1.25},
		{"ミズノレナ", "ミズノレイナ", 2, 1.25}, {"サトウレナ", "サトウレーナ", 2, 1.25},
		{"サトウデビット", "サトウデービット", .25, .25}, {"devi^tomakki:", "de:vi^tomakki:", .25, .25},
		{"ニナフェドロフ", "ニーナフェドロフ", .25, .25}, {"ニコマリ", "ニコマリー", .125, .125},
		{"セオドアトマス", "セオドアトーマス", .25, .25}, {"イトマキコ", "イトーマキコ", 2, 1.25},
		{"ヤマモトヨハン", "ヤマモトヨーハン", .25, .25},
		{"マリニコ", "マリーニコ", .25, .25}, {"タナカミホ", "タナーーカミホ", 4, 3.25},
	} {
		if got := m.Distance(tc.short, tc.long); got != tc.insert {
			t.Errorf("%s -> %s: %g, want %g", tc.short, tc.long, got, tc.insert)
		}
		if got := m.Distance(tc.long, tc.short); got != tc.drop {
			t.Errorf("%s -> %s: %g, want %g", tc.long, tc.short, got, tc.drop)
		}
	}
}

func TestLengthSettings(t *testing.T) {
	w := DefaultWeights()
	w.LongInsertion, w.LongDeletion = .625, .875
	w.LongLoss, w.FinalLongLoss = .375, .5
	m := mustNew(t, w)
	for _, tc := range []struct {
		a, b string
		want float64
	}{
		{"サカケイスケ", "サーカケイスケ", .625}, {"サーカケイスケ", "サカケイスケ", .875},
		{"ジェロームデビス", "ジェロームデービス", .375}, {"リーエディ", "リーエディー", .5},
	} {
		if got := m.Distance(tc.a, tc.b); got != tc.want {
			t.Errorf("%s -> %s: %g, want %g", tc.a, tc.b, got, tc.want)
		}
	}
	// A raised search limit lets the two-mora surname fragment test the cap
	// independently of the cost of dropping the given name.
	for _, cap := range []float64{1, 2.75, 3, math.Inf(1)} {
		w := DefaultWeights()
		w.TwoMoraMaxCost = cap
		m := mustNew(t, w, Name{Text: "オノアキ"})
		for _, sample := range []struct {
			text string
			cost float64
		}{{"オノ", 2.75}, {"おの", 3}} {
			for _, limit := range []float64{1.5, 2.75, 3} {
				got := len(m.Find(sample.text, limit)) != 0
				if want := sample.cost <= min(cap, limit); got != want {
					t.Errorf("%s cap=%g limit=%g hit=%v, want %v", sample.text, cap, limit, got, want)
				}
			}
		}
	}
	w = DefaultWeights()
	w.TwoMoraMaxCost = 0
	m = mustNew(t, w, Name{Text: "カトウアヤ"})
	if got := m.Mask("さとうアヤ", 1, "X"); got != "X" {
		t.Fatalf("two-mora limit prematurely stopped a longer alignment: %q", got)
	}
	for _, field := range []*float64{&w.TwoMoraMaxCost, &w.LongInsertion, &w.LongDeletion} {
		saved := *field
		*field = -1
		if _, err := New(nil, w); err == nil {
			t.Error("accepted a negative length setting")
		}
		*field = saved
	}
}

func TestRelaxedTwoMoraCorpus(t *testing.T) {
	w := DefaultWeights()
	w.TwoMoraMaxCost = 1.25
	t.Run("two-mora", func(t *testing.T) { checkTranscriptCorpus(t, w, nil) })

	w.LongInsertion = 1.5
	want := map[string]string{
		"length-yoshida":   "■と■。",
		"length-yusuke":    "■、■。",
		"length-saka":      "■、■。",
		"length-rena":      "■さん、■さん。",
		"joined-expansion": "■さん。",
	}
	t.Run("including-lengthening", func(t *testing.T) { checkTranscriptCorpus(t, w, want) })
}
