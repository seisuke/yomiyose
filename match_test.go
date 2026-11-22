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
		{"サイトウミホ", "サトウミホ", .25, .25, 1},
		{"カトウミホ", "サトウミホ", .5, .5, 1},
		{"オオノユウスケ", "オノユースケ", .25, .25, .5},
		{"デビットマッキー", "デービットマッキー", .25, .75, .25},
		{"デビットマッキー", "デビトマッキー", .25, 1, .25},
		{"アントニオヴィバルディ", "アントニオビバルディ", .25, 1, .25},
		{"ジェームズ・パジェット", "ジェームス・パシェット", .5, 1.25, .5},
		{"ジェームス・パシェット", "ジェームズ・パジェット", .5, 1.25, .5},
		{"ジェームズ・パジェット", "シェームズ・パジェット", 1, 1, 1},
		{"キム・シウ", "キム・シュウ", .5, .5, .5},
		{"kimusyuu", "kimusiu", .5, .5, .5},
		{"タナカしゅうじ", "タナカシウジ", .5, .5, .5},
		{"タナカシウジ", "タナカシュウジ", .5, .5, .5},
		{"ジョシュ・ホーキンソン", "ジョジホーキンソン", .75, 1, .75},
		{"ジョジホーキンソン", "ジョシュ・ホーキンソン", .75, 1, .75},
		{"ジェームズ・バリ", "ジェームス・パル", 1, 2.25, 1},
		{"ジェームス・パル", "ジェームズ・バリ", 1, 2.25, 1},
		{"キム・ミウ", "キム・ムウ", 1, 1, 1},
		{"マーティンホール", "マーチンホール", .25, 1, .25},
		{"ウィリアムモリス", "ウイリアムモリス", .25, 2, .25},
		{"リーエディー", "リーエディ", .125, .75, .125},
		{"レジェップタイイップエルドアン", "レジェップタイイップエルドワン", .5, 1, .5},
		{"エルモア・ジェイムズ", "エルモはジェームズ", .625, 1, .625},
		{"オトワユウスケ", "オトはユウスケ", .125, .125, .125},
		{"オトハユウスケ", "オトワユウスケ", 1, 1, 1},
		{"オオノデビット", "オノデービット", .5, 1, .75},
		{"キクチユウスケ", "キチユウスケ", 1, 1, 1},
		{"キチユウスケ", "キクチユウスケ", 1, 1, 1},
		{"ジャンカルロ・フィジケラ", "ジャン・カルド・フィジケラ", 0.5, 1, 0.5},
		{"リー・シェン・ルーン", "ディー・シェン・ルーン", 0.5, 1, 0.5},
		{"ジャン・カルド", "ジャン・カルロ", 0.5, 1, 0.5},
		{"リー・シェン・ルーン", "ティー・シェン・ルーン", 1, 1, 1},
		{"タナカキクコ", "タナカキッコ", .5, .5, 1},
		{"フクカワミホ", "フッカワミホ", .5, .5, 1},
		{"satoukikuko", "satoukikko", .5, .5, 1},
		{"サトウキッコ", "サトウキクコ", 1, 1, 1},
		{"タナカイクミ", "タナカイッミ", 1, 1, 1},
		{"タナカアキコ", "タナカアッコ", 1, 1, 1},
		{"サトウキクコ", "サトウキッケ", 2, 2, 2},
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
	w.LongInsertion, w.RepeatedVowelPenalty = .5, .25
	m = mustNew(t, w)
	if j, f := m.DistanceWithRules("タナカミホ", "ターナカミホ", JapaneseRules),
		m.DistanceWithRules("タナカミホ", "ターナカミホ", ForeignRules); j != .75 || f != .5 {
		t.Errorf("repeated-vowel penalty: Japanese %g, foreign %g", j, f)
	}
}

func TestHighVowelWeights(t *testing.T) {
	w := DefaultWeights()
	w.Other = 2 // Exercise a combined phonetic cost above one ordinary edit.
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

func TestRDCostAndContext(t *testing.T) {
	const name, text = "リー・シェン・ルーン", "会談にはディー・シェン・ルーン首相が出席。"
	w := DefaultWeights()
	w.RD = .75
	m := mustNew(t, w, Name{Text: name})
	if got := m.Mask(text, .75, "■"); got != "会談には■首相が出席。" {
		t.Errorf("mask with initial RD at threshold: %q", got)
	}
	if got := m.Mask(text, .74, "■"); got != text {
		t.Errorf("mask below threshold: %q", got)
	}
	if got := m.Distance(name, "ディ・シェン・ルーン"); got != w.RD+w.LongDeletion {
		t.Errorf("RD with a missing length mark: got %g", got)
	}
	w.RD = math.Inf(1)
	disabled := mustNew(t, w, Name{Text: name})
	const differentNeighbor = "ディ・セン・ルーン"
	if got, want := m.Distance(name, differentNeighbor), disabled.Distance(name, differentNeighbor); got != want {
		t.Errorf("RD with a mismatched following mora: got %g, want %g", got, want)
	}
	if got := disabled.Mask(text, .75, "■"); got != text {
		t.Errorf("disabled RD: %q", got)
	}
	w.RD = -1
	if _, err := NewReadings(nil, w); err == nil {
		t.Error("accepted a negative RD cost")
	}
}

func TestDefaultMoraEdits(t *testing.T) {
	const name = "ヤマモトユウスケ"
	m := mustNew(t, DefaultWeights(), Name{Text: "ジャンカルロ・フィジケラ"})
	for _, observed := range []string{"ヤマモトユススケ", "ヤマモトフユウスケ", "ヤモトユウスケ"} {
		if got := m.Distance(name, observed); got != 1 {
			t.Errorf("Distance(%q, %q) = %g, want 1", name, observed, got)
		}
	}
	if got := m.Mask("受付は「ジャン・カルド・フィケラ」です。", 1.5, "■"); got != "受付は「■」です。" {
		t.Errorf("RD plus a missing medial mora: %q", got)
	}
}

func TestMoraCountLimits(t *testing.T) {
	for n, want := range map[int]float64{6: 1.5, 7: 1.75, 8: 1.75, 9: 1.75, 10: 2, 20: 2} {
		if got := moraCostLimit(n); got != want {
			t.Errorf("limit at %d moras = %g, want %g", n, got, want)
		}
	}
	w := DefaultWeights()
	w.LimitByMoraCount = true
	for _, tc := range []struct {
		name, observed string
		limit          float64
		match          bool
	}{
		{"ヤマモトユウスケ", "やまもとゆうハケ", 2, true},      // 8 moras, cost 1.75
		{"ヤマモトユウスケ", "やまもとゆうハケ", 1.5, false},   // caller's lower ceiling
		{"アレクサンドル・ペトロ", "アレクサンドルモタロ", 2, true}, // 10 moras, cost 2
		{"アレクサンドル・ペトロ", "アレクンドルモトロ", 2, false}, // 9 observed moras, cost 2
	} {
		m := mustNew(t, w, Name{Text: tc.name})
		text := "受付は「" + tc.observed + "」です。"
		want := text
		if tc.match {
			want = "受付は「■」です。"
		}
		if got := m.Mask(text, tc.limit, "■"); got != want {
			t.Errorf("Mask(%q, %g) = %q, want %q", text, tc.limit, got, want)
		}
	}
}

func TestMedialLossMask(t *testing.T) {
	const name, text = "タナカユウスケ", "受付は「タカユウスケ」です。"
	w := DefaultWeights()
	w.Other = 2 // Isolate the configurable medial deletion from its fallback.
	for _, cost := range []float64{w.MedialLoss, 1.5, 1.75, math.Inf(1)} {
		w.MedialLoss = cost
		m := mustNew(t, w, Name{Text: name})
		want := text
		if cost <= 1.5 {
			want = "受付は「■」です。"
		}
		if got := m.Mask(text, 1.5, "■"); got != want {
			t.Errorf("MedialLoss=%g: got %q, want %q", cost, got, want)
		}
	}
	w.MedialLoss = -1
	if _, err := NewReadings(nil, w); err == nil {
		t.Error("accepted a negative MedialLoss cost")
	}
}

func TestKuGeminationMask(t *testing.T) {
	const name = "サトウキクコ"
	w := DefaultWeights()
	m := mustNew(t, w, Name{Text: name})
	for _, tc := range []struct {
		observed string
		cost     float64
	}{
		{"サトウキッコ", .5},
		{"さとうきっこ", 1.25},
		{"satoukikko", .5},
	} {
		text := "受付は「" + tc.observed + "」です。"
		hits := m.Find(text, tc.cost)
		if len(hits) != 1 || hits[0].Cost != tc.cost || text[hits[0].Start:hits[0].End] != tc.observed {
			t.Errorf("Find(%q): %+v", text, hits)
		}
		if m.Mask(text, 1.5, "■") != "受付は「■」です。" || m.Mask(text, tc.cost-.01, "■") != text {
			t.Errorf("unexpected mask or threshold for %q", text)
		}
	}
	w.KuGemination = .25
	m = mustNew(t, w, Name{Text: name})
	if got := m.Mask("サトウキッコ。サトウキッ。コ", .25, "■"); got != "■。サトウキッ。コ" {
		t.Errorf("tuned cost and span: %q", got)
	}
	w.KuGemination = math.Inf(1)
	m = mustNew(t, w, Name{Text: name})
	if got := m.Mask("サトウキッコ", .5, "■"); got != "サトウキッコ" {
		t.Errorf("disabled rule: %q", got)
	}
	w.KuGemination = -1
	if _, err := NewReadings(nil, w); err == nil {
		t.Error("accepted a negative KuGemination cost")
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
		{"デービッドマッキー", "デビトマッキー", .75}, {"タナカミホ", "タナッカミホ", 1}, {"タナカミホ", "タッナカミホ", 1},
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
		{"tanakamiho", "ta:nakamiho", .75}, {"タナカミホ", "タアナカミホ", .75},
		{"タナカミホ", "タナーカミホ", .75}, {"タナカミホ", "タナカーミホ", .75},
		{"ヤマダリク", "ヤーマダリク", .75}, {"イシイトモコ", "イシイトーモコ", .75},
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
		{"ヨシダアヤ", "ヨシダワヤ", 1},                 // The preceding vowel is a.
		{"レセップタイップエルドアン", "レセップタイップエルドワル", 2}, // The following sounds differ.
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

func TestObservedHaAsWa(t *testing.T) {
	const name = "エルモア・ジェイムズ"
	const text = "担当はエルモはジェームズさんです。"
	w := DefaultWeights()
	m := mustNew(t, w, Name{Text: name})
	for _, tc := range []struct {
		ref, obs string
		want     float64
	}{
		{name, "エルモワジェームズ", .25},
		{name, "エルモはジェームズ", .375},
		{"エルモワジェイムズ", "エルモはジェームズ", .125},
		{name, "エルモハジェームズ", 1},
		{name, "エルモhaジェームズ", 1},
		{"エルモはジェームズ", name, 1},
		{"ヨシダアヤ", "ヨシダはヤ", 1},
		{name, "エルモはシェームズ", 1.25},
	} {
		if got := m.Distance(tc.ref, tc.obs); got != tc.want {
			t.Errorf("Distance(%q, %q) = %g, want %g", tc.ref, tc.obs, got, tc.want)
		}
	}
	if got := m.Mask(text, .5, "■"); got != "担当は■さんです。" {
		t.Errorf("at threshold: %q", got)
	}
	if got := m.Mask(text, .499, "■"); got != text {
		t.Errorf("below threshold: %q", got)
	}
	w.HaAsWa = math.Inf(1)
	m = mustNew(t, w)
	if got := m.Distance(name, "エルモはジェームズ"); got != w.Other {
		t.Errorf("disabled HaAsWa: %g", got)
	}
	w.HaAsWa = -1
	if _, err := NewReadings(nil, w); err == nil {
		t.Error("accepted a negative HaAsWa cost")
	}
}

func TestLongVowelContextInTranscript(t *testing.T) {
	m, err := NewReadings([]string{"タナカミホ", "デビットマッキー"}, DefaultWeights())
	if err != nil {
		t.Fatal(err)
	}
	text := "ターナカミホさんとデービットマッキーさん。タナカミホさん。"
	if got := m.Mask(text, 1.5, "X"); got != "XさんとXさん。Xさん。" {
		t.Error(got)
	}
	if got := m.Mask(text, .749, "X"); got != "ターナカミホさんとXさん。Xさん。" {
		t.Error(got)
	}
	// Bi is outside the registered reading, so it cannot license De / De:.
	short := mustNew(t, DefaultWeights(), Name{Text: "ヤマダカナデ"})
	if got := short.Mask("ヤマダカナデービ", .25, "X"); got != "Xービ" {
		t.Error(got)
	}
}

func TestMaskTwoLengthChanges(t *testing.T) {
	const name, input = "サトウレーナ", "担当はサトレナさん。"
	m := mustNew(t, DefaultWeights(), Name{Text: name})
	if got := m.Mask(input, 1.5, "■"); got != "担当は■さん。" {
		t.Errorf("two missing long vowels: %q", got)
	}
	if got := m.Mask(input, 1.49, "■"); got != input {
		t.Errorf("below the combined cost: %q", got)
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
	w.Other, w.HeadLoss = 2, 1.25 // Isolate the search-only head discount.
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
	if got := m.Find("アントニオヴィバルディ", .5); len(got) != 0 {
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
	w.Geminate, w.InitialKS, w.TailLoss = .5, 1, .875
	m := mustNew(t, w, Name{Text: "石井 香織", Readings: []string{"イシーカオリ"}}, Name{Text: "サトウミホ"})
	for _, tc := range []struct {
		a, b string
		want float64
	}{
		{"キムラセイスケ", "キムラセースケ", .125}, {"サトウミホ", "サトーミホ", .5},
		{"デビットマッキー", "デビトマッキー", .5}, {"サトウミホ", "カトウミホ", 1}, {"いしだりょうた", "いしだりょう", .875},
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
	hits := m.Find("昨日、タムラアヤさん。", 1)
	if len(hits) != 1 || hits[0].Start != len("昨日、") || hits[0].End != len("昨日、タムラアヤ") || hits[0].Cost != 1 {
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
		{"ヨシダアヤ", "ヨーシダアヤ", .75, .75}, {"サカケイスケ", "サーカケイスケ", .75, .75},
		{"フジタユスケ", "フジタユースケ", .75, .75}, {"ミズノレナ", "ミズノレーナ", .75, .75},
		{"ミズノレナ", "ミズノレイナ", .75, .75}, {"サトウレナ", "サトウレーナ", .75, .75},
		{"サトウデビット", "サトウデービット", .25, .25}, {"devi^tomakki:", "de:vi^tomakki:", .25, .25},
		{"ニナフェドロフ", "ニーナフェドロフ", .25, .25}, {"ニコマリ", "ニコマリー", .125, .125},
		{"セオドアトマス", "セオドアトーマス", .25, .25}, {"イトマキコ", "イトーマキコ", .75, .75},
		{"ヤマモトヨハン", "ヤマモトヨーハン", .25, .25},
		{"マリニコ", "マリーニコ", .25, .25}, {"タナカミホ", "タナーーカミホ", 1.75, 1.75},
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
	for _, cap := range []float64{1, 1.75, 2, math.Inf(1)} {
		w := DefaultWeights()
		w.TwoMoraMaxCost = cap
		m := mustNew(t, w, Name{Text: "オノアキ"})
		for _, sample := range []struct {
			text string
			cost float64
		}{{"オノ", 1.75}, {"おの", 2}} {
			for _, limit := range []float64{1.5, 1.75, 2} {
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

	w.LongInsertion, w.RepeatedVowelPenalty = 1.5, 2
	// An ordinary insertion still costs 1 when the special length rule costs more.
	t.Run("higher-insertion-cost", func(t *testing.T) { checkTranscriptCorpus(t, w, nil) })
}
