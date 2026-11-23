// Package yomiyose finds and masks known names in Japanese speech-to-text transcripts.
// It matches registered readings by weighted phonetic edit distance and returns
// byte ranges in the original UTF-8 text. A Matcher can be reused concurrently.
package yomiyose

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
)

// Weights configures edit costs and short-candidate search limits.
// Start with DefaultWeights to override individual settings.
type Weights struct {
	LongSpelling         float64 // alternative ways of writing the same long vowel
	EiSpelling           float64 // ei vs an e long vowel
	OuSpelling           float64 // ou vs an o long vowel
	LongLoss             float64 // length difference in a supported phonetic context
	FinalLongLoss        float64 // supported length difference at the end of the reading
	LongInsertion        float64 // other short vowels lengthened in the observation
	LongDeletion         float64 // other registered long vowels shortened in the observation
	ForeignLongInsertion float64 // lengthening cost for readings containing designated loanword moras
	ForeignLongDeletion  float64 // shortening cost for readings containing designated loanword moras
	RepeatedVowelPenalty float64 // extra length-change cost in a run of three consonant-led moras with the same vowel
	VowelLoss            float64 // ai vs a
	GlideAfterOU         float64 // medial ア/ワ after a matching o/u mora and before a matching sound
	HaAsWa               float64 // cost to read an observed hiragana は as ワ in the same context
	Geminate             float64 // contextual presence/absence of ッ
	KuGemination         float64 // medial ク before a k mora realized as ッ (キクコ -> キッコ)
	BV                   float64 // corresponding b/v moras
	TiChi                float64 // ティ/チ
	TuToWa               float64 // registered トゥ heard as ト or トワ
	PuaContraction       float64 // registered プア heard as パ or ッパ in one edit
	KanaSize             float64 // small/full-size vowel in ウィ, ウェ or シェ
	ShiShu               float64 // シ/シュ in Japanese or foreign names
	RiRu                 float64 // リ/ル under foreign-name rules
	RD                   float64 // contextual ラ/ダ, リ/ディ, レ/デ and ロ/ド
	MedialVoicing        float64 // voiced/unvoiced pairs after the first mora
	InitialKS            float64 // same vowel with k/s in the first mora
	MedialLoss           float64 // one missing interior mora of the registered reading
	TailLoss             float64 // one missing final ordinary mora of the reference
	HeadLoss             float64 // Find/Mask: omit the first mora if six vowel-bearing moras remain
	Other                float64 // other mora insertions, deletions and substitutions
	Hiragana             float64 // per observed mora originally written in hiragana
	TwoMoraMaxCost       float64 // Find/Mask ceiling for two moras; +Inf uses the caller's limit
	ExactSingleMora      bool    // require identical normalized sounds for a one-mora candidate
	LimitByMoraCount     bool    // apply MoraLimits to Find/Mask; enabled by DefaultWeights

	MoraLimits *MoraLimits // nil uses the default length bands; copied when creating a matcher

	MatchPrefixes          bool    // also search for proper prefixes of a registered reading
	MinPrefixChars         int     // minimum normalized kana characters on both sides; 3, 4 or 5
	PrefixMaxCost          float64 // overall prefix ceiling, including five-character matches
	ThreeCharPrefixMaxCost float64 // additional cap when the shorter side has three characters
	FourCharPrefixMaxCost  float64 // additional cap when the shorter side has four characters
}

const maxPrefixChars = 5

// MoraLimits sets search costs by the shorter of the registered and matched readings.
// The one- and two-mora restrictions in Weights also apply.
type MoraLimits struct {
	UpTo6    float64
	From7To9 float64
	From10   float64
}

func (l MoraLimits) maximum() float64 {
	return max(l.UpTo6, l.From7To9, l.From10)
}

// DefaultWeights returns initial ranking costs, not estimated probabilities.
func DefaultWeights() Weights {
	return Weights{
		LongLoss: 0.25, FinalLongLoss: 0.125, LongInsertion: 0.75, LongDeletion: 0.75,
		ForeignLongInsertion: 0.5, ForeignLongDeletion: 0.5,
		TiChi: 0.25, TuToWa: 0.25, PuaContraction: 0.25, KanaSize: 0.25, ShiShu: 0.5, RiRu: 0.5, VowelLoss: 0.25, Geminate: 0.25,
		GlideAfterOU: 0.25, HaAsWa: 0.125, KuGemination: 0.5, RD: 0.5,
		BV: 0.25, MedialVoicing: 0.25, InitialKS: 0.5, MedialLoss: 1, TailLoss: 0.75, HeadLoss: 1, Other: 1,
		Hiragana: 0, TwoMoraMaxCost: 1, ExactSingleMora: true,
		LimitByMoraCount: true, MoraLimits: &MoraLimits{UpTo6: 1.5, From7To9: 1.75, From10: 2},
		MinPrefixChars: 3, ThreeCharPrefixMaxCost: 0.75, FourCharPrefixMaxCost: 1, PrefixMaxCost: 1.25,
	}
}

// Name groups a primary spelling with optional aliases. All spellings are searched.
// For a list of readings without aliases, use NewReadings.
type Name struct {
	Text     string
	Readings []string
}

// RuleSet selects which name-specific rules to use alongside the common rules.
type RuleSet uint8

const (
	AllRules      RuleSet = iota // both groups, including for mixed or unknown readings
	JapaneseRules                // rules based on Japanese name examples
	ForeignRules                 // rules based on foreign names written in Japanese
)

// ScopedName applies Rules to the primary reading and all its aliases.
type ScopedName struct {
	Name
	Rules RuleSet
}

type ruleWeights struct {
	Weights
	scope RuleSet
}

func (w Weights) forRuleSet(scope RuleSet) ruleWeights {
	off := math.Inf(1)
	switch scope {
	case JapaneseRules:
		w.FinalLongLoss, w.Geminate, w.BV = off, off, off
		w.TiChi, w.KanaSize, w.GlideAfterOU = off, off, off
		w.TuToWa, w.PuaContraction = off, off
		w.RiRu, w.RD = off, off
	case ForeignRules:
		w.VowelLoss, w.InitialKS, w.KuGemination = off, off, off
		w.RepeatedVowelPenalty = 0
	}
	return ruleWeights{Weights: w, scope: scope}
}

// Only the registered reading supplies evidence for changing vowel-length costs.
func (w ruleWeights) forReading(form []unit) ruleWeights {
	if w.scope != JapaneseRules {
		for _, mora := range form {
			switch mora.sound {
			case "tu", "du", "ti", "di", "va", "vi", "ve", "vo", "fi", "fe", "fo":
				w.LongInsertion, w.LongDeletion = w.ForeignLongInsertion, w.ForeignLongDeletion
				return w
			}
		}
	}
	return w
}

// Match identifies text[Start:End] in the original UTF-8 input.
type Match struct {
	Name       string
	NameIndex  int // index in the input list
	Start, End int // byte offsets, End exclusive
	Cost       float64
}

type entry struct {
	text  string
	forms [][]unit
	rules RuleSet
}

// Matcher holds compiled name readings and their comparison weights.
type Matcher struct {
	names []entry
	rules [ForeignRules + 1]ruleWeights
}

// NewReadings compiles kana or romaji readings, one per name. Match.Name is
// the original registered reading, and Match.NameIndex is its index in readings.
func NewReadings(readings []string, w Weights) (*Matcher, error) {
	names := make([]ScopedName, len(readings))
	for i, reading := range readings {
		names[i].Text = reading
	}
	return NewScoped(names, w)
}

// New compiles the name list with AllRules. Weights are copied and the input slices can be reused.
func New(names []Name, w Weights) (*Matcher, error) {
	scoped := make([]ScopedName, len(names))
	for i, name := range names {
		scoped[i].Name = name
	}
	return NewScoped(scoped, w)
}

// NewScoped compiles a list with an explicit rule set per name. The zero rule
// set is AllRules. The matcher never guesses a rule set from the spelling.
func NewScoped(names []ScopedName, w Weights) (*Matcher, error) {
	if w.MatchPrefixes && (w.MinPrefixChars < 3 || w.MinPrefixChars > maxPrefixChars) {
		return nil, fmt.Errorf("yomiyose: MinPrefixChars must be between 3 and 5 when matching prefixes")
	}
	if w.MoraLimits == nil {
		w.MoraLimits = DefaultWeights().MoraLimits
	} else {
		limits := *w.MoraLimits
		w.MoraLimits = &limits // the compiled matcher owns its settings
	}
	for _, v := range []float64{w.LongSpelling, w.EiSpelling, w.OuSpelling, w.LongLoss, w.FinalLongLoss,
		w.MoraLimits.UpTo6, w.MoraLimits.From7To9, w.MoraLimits.From10, w.PrefixMaxCost, w.ThreeCharPrefixMaxCost, w.FourCharPrefixMaxCost,
		w.TuToWa, w.PuaContraction, w.ForeignLongInsertion, w.ForeignLongDeletion,
		w.LongInsertion, w.LongDeletion, w.RepeatedVowelPenalty, w.TiChi, w.KanaSize, w.VowelLoss, w.GlideAfterOU, w.HaAsWa, w.Geminate,
		w.BV, w.ShiShu, w.RiRu, w.RD, w.MedialVoicing, w.InitialKS, w.KuGemination, w.MedialLoss, w.TailLoss, w.HeadLoss, w.Other, w.Hiragana, w.TwoMoraMaxCost} {
		if v < 0 || math.IsNaN(v) {
			return nil, fmt.Errorf("yomiyose: weights must be nonnegative")
		}
	}
	m := &Matcher{names: make([]entry, 0, len(names))}
	for scope := range m.rules {
		m.rules[scope] = w.forRuleSet(RuleSet(scope))
	}
	for index, name := range names {
		if name.Rules > ForeignRules {
			return nil, fmt.Errorf("yomiyose: unknown rule set %d at index %d", name.Rules, index)
		}
		e := entry{text: name.Text, rules: name.Rules}
		for _, reading := range append([]string{name.Text}, name.Readings...) {
			u := tokenize(strings.TrimSpace(reading))
			if len(u) == 0 {
				return nil, fmt.Errorf("yomiyose: empty name or reading at index %d", index)
			}
			for _, t := range u {
				if t.sound == "" {
					return nil, fmt.Errorf("yomiyose: unsupported separator in %q", reading)
				}
			}
			e.forms = append(e.forms, u)
		}
		m.names = append(m.names, e)
	}
	return m, nil
}

func isLong(prev, s string) bool {
	v := vowel(prev)
	return v != 0 && (s == ":" || len(s) == 1 && (s[0] == v || v == 'e' && s == "i" || v == 'o' && s == "u"))
}

// consonantLedVowel returns zero for standalone vowels, length marks and
// literal tokens. The phonetic representation stores one mora per token.
func consonantLedVowel(sound string) byte {
	if len(sound) < 2 || !asciiLetter(rune(sound[0])) {
		return 0
	}
	return vowel(sound)
}

// left ends at the mora being lengthened. Both slices come from the registered
// name, with the length marker under comparison omitted if it was present.
func inRepeatedVowelRun(left, right []unit) bool {
	v := consonantLedVowel(left[len(left)-1].sound)
	if v == 0 {
		return false
	}
	count := 1
	for i := len(left) - 2; i >= 0 && count < 3 && consonantLedVowel(left[i].sound) == v; i-- {
		count++
	}
	for i := 0; i < len(right) && count < 3 && consonantLedVowel(right[i].sound) == v; i++ {
		count++
	}
	return count == 3
}

// before and after are parts of the registered reading. Look only at that
// reading so that surrounding prose cannot enable a discounted edit.
func (w *ruleWeights) lengthCost(before, after []unit, deletion bool) float64 {
	prev := before[len(before)-1].sound
	cost := w.LongInsertion
	if deletion {
		cost = w.LongDeletion
	}
	if w.RepeatedVowelPenalty != 0 && inRepeatedVowelRun(before, after) {
		cost += w.RepeatedVowelPenalty
	}
	if w.scope != ForeignRules && prev == "o" && len(after) > 0 && after[0].sound == "no" {
		cost = math.Min(cost, w.LongLoss)
	}
	if w.scope == JapaneseRules {
		return cost
	}
	afterMari := prev == "ri" && len(before) >= 2 && before[len(before)-2].sound == "ma"
	beforeMasu := prev == "to" && len(after) >= 2 && after[0].sound == "ma" && after[1].sound == "su"
	if len(after) == 0 && (prev == "di" || prev == "ti" || afterMari) {
		cost = math.Min(cost, w.FinalLongLoss)
	} else if len(after) > 0 && (afterMari || beforeMasu || foreignLengthBefore[[2]string{prev, after[0].sound}]) {
		cost = math.Min(cost, w.LongLoss)
	}
	return cost
}

// Mora on either side of the length marker, in kana or romaji alike.
var foreignLengthBefore = map[[2]string]bool{
	{"de", "bi"}: true, {"de", "vi"}: true, {"she", "ku"}: true,
	{"ni", "na"}: true, {"yo", "ha"}: true, {"ru", "chi"}: true,
}

// gap scores omitting u[i]. The matching other-side prefix and suffix give
// context; e.g., ai/a must keep the preceding a on both sides of the alignment.
func (w *ruleWeights) gap(u []unit, i int, other []unit, end int, reference bool) float64 {
	s := u[i].sound
	if !phonetic(s) {
		return math.Inf(1)
	}
	cost := w.Other
	if reference && i > 0 && i+1 < len(u) {
		cost = math.Min(cost, w.MedialLoss)
	}
	if i > 0 && end > 0 {
		prev := u[i-1].sound
		aligned := other[end-1].sound
		if vowel(prev) != 0 && vowel(prev) == vowel(aligned) {
			if isLong(prev, s) {
				left, right := other[:end], other[end:]
				if reference {
					left, right = u[:i], u[i+1:]
				}
				cost = math.Min(cost, w.lengthCost(left, right, reference))
			}
			if vowel(prev) == 'a' && s == "i" {
				cost = math.Min(cost, w.VowelLoss)
			}
		}
		if s == "^" && (prev == aligned || bvPartner[prev] == aligned) && i+1 < len(u) && end < len(other) {
			// Conservative seed for foreign names: a high vowel before a voiceless
			// stop, fricative, or affricate. Other contexts keep the ordinary cost.
			next, opposite := u[i+1].sound, other[end].sound
			// The same alignment can also alternate ト/ド after the geminate.
			if next == "do" {
				next = "to"
			}
			if opposite == "do" {
				opposite = "to"
			}
			if next != "" && next == opposite {
				highVowel := vowel(prev) == 'i' || vowel(prev) == 'u'
				if (highVowel && strings.ContainsRune("kstpchf", rune(next[0]))) || (prev == "ba" && next == "pe") {
					cost = math.Min(cost, w.Geminate)
				}
			}
		}
	}
	return cost
}

func phonetic(s string) bool {
	return len(s) > 0 && (s[0] >= 'a' && s[0] <= 'z' || s == "^" || s == ":")
}

// Compare the two neighboring moras, using two on the same side at a name boundary.
func matchingRDContext(a, b []unit, i, j int) bool {
	if len(a) < 3 {
		return false
	}
	// A missing length mark after initial リ/ディ still leaves two following
	// moras to check. The alignment charges the deletion separately.
	if i == 0 && j == 0 && (a[0].sound == "ri" || a[0].sound == "di") && len(a) >= 4 && a[1].sound == ":" &&
		j+2 < len(b) && phonetic(a[2].sound) && phonetic(a[3].sound) &&
		a[2].sound == b[j+1].sound && a[3].sound == b[j+2].sound {
		return true
	}
	start := min(max(0, i-1), len(a)-3)
	observed := j - (i - start)
	if observed < 0 || observed+3 > len(b) {
		return false
	}
	for k := range 3 {
		if start+k != i && (!phonetic(a[start+k].sound) || a[start+k].sound != b[observed+k].sound) {
			return false
		}
	}
	return true
}

func (w *ruleWeights) substitution(a, b []unit, i, j int) float64 {
	x, y := a[i].sound, b[j].sound
	if x == y {
		return 0
	}
	if !phonetic(x) || !phonetic(y) {
		return math.Inf(1)
	}
	cost := w.Other
	if i > 0 && j > 0 && vowel(a[i-1].sound) == vowel(b[j-1].sound) &&
		isLong(a[i-1].sound, x) && isLong(b[j-1].sound, y) {
		switch {
		case vowel(a[i-1].sound) == 'e' && (x == "i" || y == "i"):
			cost = math.Min(cost, w.EiSpelling)
		case vowel(a[i-1].sound) == 'o' && (x == "u" || y == "u"):
			cost = math.Min(cost, w.OuSpelling)
		default:
			cost = math.Min(cost, w.LongSpelling)
		}
	}
	if bvPartner[x] == y {
		cost = math.Min(cost, w.BV)
	}
	if i == 0 && j == 0 && ksPartner[x] == y {
		cost = math.Min(cost, w.InitialKS)
	}
	if tiChiPartner[x] == y {
		cost = math.Min(cost, w.TiChi)
	}
	if x == "tu" && y == "to" {
		cost = math.Min(cost, w.TuToWa)
	}
	if shiShuPartner[x] == y {
		cost = math.Min(cost, w.ShiShu)
	}
	if riRuPartner[x] == y {
		cost = math.Min(cost, w.RiRu)
	}
	if rdPartner[x] == y && matchingRDContext(a, b, i, j) {
		cost = math.Min(cost, w.RD)
	}
	if i > 0 && j > 0 {
		if voicingPartner[x] == y || w.scope != JapaneseRules && foreignVoicingPartner[x] == y {
			cost = math.Min(cost, w.MedialVoicing)
		}
		// シュ/ジ changes both the vowel (シュ/シ) and voicing (シ/ジ) in one mora.
		if w.scope != JapaneseRules && (x == "shu" && y == "ji" || x == "ji" && y == "shu") {
			cost = math.Min(cost, w.ShiShu+w.MedialVoicing)
		}
	}
	var glide float64
	switch {
	case x == "a" && y == "wa" || x == "wa" && y == "a":
		glide = w.GlideAfterOU
	case x == "wa" && y == "ha" && b[j].hira:
		glide = w.HaAsWa
	case x == "a" && y == "ha" && b[j].hira:
		glide = w.GlideAfterOU + w.HaAsWa
	default:
		return cost
	}
	if i > 0 && j > 0 && i+1 < len(a) && j+1 < len(b) &&
		a[i-1].sound == b[j-1].sound && a[i+1].sound == b[j+1].sound && phonetic(a[i+1].sound) {
		v := vowel(a[i-1].sound)
		if v == 'o' || v == 'u' {
			cost = math.Min(cost, glide)
		}
	}
	return cost
}

// Pairs drive both candidate selection and the corresponding scoring rules.
func partners(pairs string) map[string]string {
	table := map[string]string{}
	for _, pair := range strings.Fields(pairs) {
		a, b, _ := strings.Cut(pair, "/")
		table[a], table[b] = b, a
	}
	return table
}

var bvPartner = partners("ba/va bi/vi bu/vu be/ve bo/vo byu/vyu")
var tiChiPartner = partners("ti/chi")
var voicingPartner = partners("ka/ga ki/gi ku/gu ke/ge ko/go sa/za shi/ji su/zu se/ze so/zo ta/da te/de to/do ha/ba hi/bi fu/bu he/be ho/bo")
var foreignVoicingPartner = partners("je/she ba/pa")
var shiShuPartner = partners("shi/shu")
var riRuPartner = partners("ri/ru")
var rdPartner = partners("ra/da ri/di re/de ro/do")
var ksPartner = partners("ka/sa ki/shi ku/su ke/se ko/so kya/sha kyu/shu kyo/sho")

// One compact mora aligns to both sounds of its full-size spelling in one edit.
var kanaSizePairs = [...]struct{ compact, first, second string }{
	{"wi", "u", "i"}, {"we", "u", "e"}, {"she", "shi", "e"},
}

func compactKana(u []unit, start int) string {
	if start < 0 || start+1 >= len(u) {
		return ""
	}
	for _, p := range kanaSizePairs {
		if u[start].sound == p.first && u[start+1].sound == p.second {
			return p.compact
		}
	}
	return ""
}

func fullSizeKana(compact string) (string, string) {
	for _, p := range kanaSizePairs {
		if p.compact == compact {
			return p.first, p.second
		}
	}
	return "", ""
}

// A medial ッ and the following k mora must both fall inside the match.
func startsKuGemination(b []unit, j int) bool {
	if j <= 0 || j+1 >= len(b) || b[j].sound != "^" || vowel(b[j-1].sound) == 0 {
		return false
	}
	next := b[j+1].sound
	return strings.HasPrefix(next, "k") && vowel(next) != 0
}

// Tokenization keeps separators for alignment, but they add no spoken length.
func countsTowardLength(t unit) bool {
	return t.sound != " " && t.sound != "'" && t.sound != ""
}

func (t unit) kanaChars() int {
	if !countsTowardLength(t) {
		return 0
	}
	if t.kanaPair {
		return 2
	}
	return 1
}

func (l MoraLimits) costLimit(moras int) float64 {
	switch {
	case moras <= 6:
		return l.UpTo6
	case moras <= 9:
		return l.From7To9
	default:
		return l.From10
	}
}

// Count only independent vowel-bearing moras. A long vowel spelled with a
// second kana must give the same evidence as one spelled with ー.
func canOmitHead(a []unit) bool {
	if len(a) < 7 || vowel(a[0].sound) == 0 || vowel(a[1].sound) == 0 || isLong(a[0].sound, a[1].sound) {
		return false
	}
	count := 0
	for i := 1; i < len(a); i++ {
		if vowel(a[i].sound) != 0 && !isLong(a[i-1].sound, a[i].sound) {
			count++
		}
	}
	return count >= 6
}

// At this prefix boundary the "missing" last mora is actually visible just
// after an extra length or gemination mark. Score that insertion normally.
func tailAfterMarker(b []unit, end int, last string) bool {
	return end+1 < len(b) && b[end+1].sound == last &&
		(b[end].sound == "^" || isLong(b[end-1].sound, b[end].sound))
}

type alignmentMode int

const (
	wholeReading alignmentMode = iota
	searchFullName
	searchNamePrefix
)

func (w *ruleWeights) withinLengthLimit(moras int, exact bool, cost float64) bool {
	if moras == 1 && w.ExactSingleMora && !exact {
		return false
	}
	if moras == 2 && cost > w.TwoMoraMaxCost {
		return false
	}
	return !w.LimitByMoraCount || cost <= w.MoraLimits.costLimit(moras)
}

// Each column is the cost of matching a prefix of the same registered reading.
// The unconsumed suffix costs nothing. Full readings are scored separately.
func (w *ruleWeights) namePrefixCost(a []unit, row []float64, observedChars, observedMoras int) float64 {
	best := math.Inf(1)
	chars, moras := 0, 0
	for i, t := range a[:len(a)-1] {
		width := t.kanaChars()
		if width == 0 {
			continue
		}
		chars += width
		if chars > maxPrefixChars {
			break
		}
		moras++
		limit := w.PrefixMaxCost
		switch min(chars, observedChars) {
		case 3:
			limit = w.ThreeCharPrefixMaxCost
		case 4:
			limit = w.FourCharPrefixMaxCost
		}
		cost := row[i+1]
		if chars >= w.MinPrefixChars && cost <= limit && cost < best &&
			w.withinLengthLimit(min(moras, observedMoras), false, cost) {
			best = cost
		}
	}
	return best
}

// align compares one reading to text beginning at a fixed position. Full names
// prefer the cheapest endpoint; partial names prefer the longest within budget.
func (w *ruleWeights) align(a, b []unit, rows []float64, limit float64, mode alignmentMode, omitHead bool) (int, float64) {
	width := len(a) + 1
	older, prev, curr := rows[:width], rows[width:2*width], rows[2*width:]
	prev[0] = 0
	referenceLength, observedLength, observedChars := 0, 0, 0
	sameSounds := true
	last := len(a) - 1
	canDropTail := last > 0 && vowel(a[last].sound) != 0 && !isLong(a[last-1].sound, a[last].sound)
	for i := range a {
		prev[i+1] = prev[i] + w.gap(a, i, b, 0, true)
		if mode != wholeReading && countsTowardLength(a[i]) {
			referenceLength++
		}
	}
	best, end := math.Inf(1), 0
	previousScript := 0.0
	for j, t := range b {
		if t.sound == "" {
			break
		}
		script := 0.0
		if mode != wholeReading && t.hira {
			script = w.Hiragana
		}
		curr[0] = prev[0] + w.gap(b, j, a, 0, false) + script
		low := curr[0]
		compact := compactKana(b, j-1)
		first, second := fullSizeKana(t.sound)
		kuGemination := startsKuGemination(b, j-1)
		for i := range a {
			diag := prev[i] + w.substitution(a, b, i, j) + script
			if omitHead && j == 0 && i == 1 && a[i].sound == t.sound {
				diag = math.Min(diag, w.HeadLoss+script)
			}
			del := curr[i] + w.gap(a, i, b, j+1, true)
			ins := prev[i+1] + w.gap(b, j, a, i+1, false) + script
			curr[i+1] = math.Min(diag, math.Min(del, ins))
			if j > 0 && compact == a[i].sound {
				curr[i+1] = math.Min(curr[i+1], older[i]+w.KanaSize+previousScript+script)
			}
			if i > 0 && a[i].sound == second && a[i-1].sound == first {
				curr[i+1] = math.Min(curr[i+1], prev[i-1]+w.KanaSize+script)
			}
			if t.sound == "wa" && j > 0 && b[j-1].sound == "to" && a[i].sound == "tu" {
				curr[i+1] = math.Min(curr[i+1], older[i]+w.TuToWa+previousScript+script)
			}
			if t.sound == "pa" && i > 0 && a[i-1].sound == "pu" && a[i].sound == "a" {
				curr[i+1] = math.Min(curr[i+1], prev[i-1]+w.PuaContraction+script)
				if j > 0 && b[j-1].sound == "^" {
					curr[i+1] = math.Min(curr[i+1], older[i-1]+w.PuaContraction+previousScript+script)
				}
			}
			if kuGemination && i >= 2 && a[i-1].sound == "ku" && a[i].sound == t.sound && a[i-2].sound == b[j-2].sound {
				curr[i+1] = math.Min(curr[i+1], older[i-1]+w.KuGemination+previousScript+script)
			}
			low = math.Min(low, curr[i+1])
		}
		if canDropTail && mode != searchNamePrefix && (mode == wholeReading || !tailAfterMarker(b, j+1, a[last].sound)) {
			curr[last+1] = math.Min(curr[last+1], curr[last]+w.TailLoss)
			low = math.Min(low, curr[last+1])
		}
		older, prev, curr = prev, curr, older
		previousScript = script
		if mode != wholeReading {
			// A two-mora edit can still skip this row to finish at the next one.
			if low > limit {
				canFinishKanaPair := w.KanaSize <= limit && compactKana(b, j) != ""
				canFinishKuPair := w.KuGemination <= limit && startsKuGemination(b, j)
				canFinishPua := w.PuaContraction <= limit && t.sound == "^" && j+1 < len(b) && b[j+1].sound == "pa"
				if !canFinishKanaPair && !canFinishKuPair && !canFinishPua {
					break
				}
			}
			if countsTowardLength(t) {
				observedLength++
			}
			sameSounds = sameSounds && j < len(a) && a[j].sound == t.sound
			// Test each endpoint independently. A rejected two-mora prefix
			// can still lead to an eligible three-mora match.
			if mode == searchNamePrefix {
				width := t.kanaChars()
				observedChars += width
				if observedChars > maxPrefixChars {
					break
				}
				if observedChars >= w.MinPrefixChars && width > 0 {
					cost := w.namePrefixCost(a, prev, observedChars, observedLength)
					if cost <= limit && !math.IsInf(cost, 1) {
						best, end = cost, j+1
					}
				}
				continue
			}
			shorter := min(referenceLength, observedLength)
			cost := prev[len(a)]
			if cost <= limit && w.withinLengthLimit(shorter, sameSounds && j+1 == len(a), cost) {
				best, end, limit = cost, j+1, cost
			}
		}
	}
	if mode != wholeReading {
		return end, best
	}
	return len(b), prev[len(a)]
}

// Distance is the AllRules phonetic cost of a whole reference and observation. It omits
// search limits and the hiragana penalty used by Find. Some edits are directional.
func (m *Matcher) Distance(reference, observed string) float64 {
	return m.DistanceWithRules(reference, observed, AllRules)
}

// DistanceWithRules compares two whole readings with the specified rule set.
// An unknown set or unsupported separator gives positive infinity.
func (m *Matcher) DistanceWithRules(reference, observed string, rules RuleSet) float64 {
	if rules > ForeignRules {
		return math.Inf(1)
	}
	a, b := tokenize(reference), tokenize(observed)
	for _, units := range [][]unit{a, b} {
		for _, t := range units {
			if t.sound == "" {
				return math.Inf(1)
			}
		}
	}
	w := m.rules[rules].forReading(a)
	_, cost := w.align(a, b, make([]float64, 3*(len(a)+1)), math.Inf(1), wholeReading, false)
	return cost
}

// Below Other, every eligible alignment must start with one of these sounds.
func (w Weights) initialSounds(form []unit, limit float64, omitHead bool) []string {
	first := form[0].sound
	sounds := []string{first}
	add := func(sound string) {
		if sound != "" && !slices.Contains(sounds, sound) {
			sounds = append(sounds, sound)
		}
	}
	if omitHead {
		add(form[1].sound)
	}
	if w.KanaSize <= limit {
		add(compactKana(form, 0))
		expanded, _ := fullSizeKana(first)
		add(expanded)
	}
	if w.BV <= limit {
		add(bvPartner[first])
	}
	if w.TiChi <= limit {
		add(tiChiPartner[first])
	}
	if first == "tu" && w.TuToWa <= limit {
		add("to")
	}
	if first == "pu" && len(form) > 1 && form[1].sound == "a" && w.PuaContraction <= limit {
		add("pa")
		add("^")
	}
	if w.ShiShu <= limit {
		add(shiShuPartner[first])
	}
	if w.RiRu <= limit {
		add(riRuPartner[first])
	}
	if w.RD <= limit && len(form) >= 3 {
		add(rdPartner[first])
	}
	if w.InitialKS <= limit {
		add(ksPartner[first])
	}
	return sounds
}

// FindDefault uses the configured length limits. When LimitByMoraCount is off,
// it uses the largest of the three limits for every length.
func (m *Matcher) FindDefault(text string) []Match {
	if len(m.names) == 0 {
		return nil
	}
	return m.Find(text, m.rules[AllRules].MoraLimits.maximum())
}

// Find returns occurrences with cost <= maxCost, in source order. For each
// name, full-name matches take precedence over overlapping prefix matches.
// At a given start the cheapest full-name span wins (longest if tied); prefix
// matching instead takes the longest span within its limit. Names are independent.
func (m *Matcher) Find(text string, maxCost float64) []Match {
	if len(m.names) == 0 || maxCost < 0 || math.IsNaN(maxCost) {
		return nil
	}
	if w := &m.rules[AllRules]; w.LimitByMoraCount {
		// Even a long name can match a shorter candidate with a higher custom limit.
		maxCost = min(maxCost, w.MoraLimits.maximum())
	}
	tokens := tokenize(text)
	positions := make(map[string][]int)
	for i, t := range tokens {
		if t.sound != "" {
			positions[t.sound] = append(positions[t.sound], i)
		}
	}
	var result []Match
	// Bytes are marked only after all candidates at the same cost were considered.
	occupied := make([]bool, len(text))
	for index, name := range m.names {
		var full, partial []Match
		for _, form := range name.forms {
			w := m.rules[name.rules].forReading(form)
			rows := make([]float64, 3*(len(form)+1))
			find := func(hits []Match, limit float64, mode alignmentMode) []Match {
				omitHead := mode == searchFullName && w.HeadLoss <= limit && canOmitHead(form)
				check := func(i int) {
					end, c := w.align(form, tokens[i:], rows, limit, mode, omitHead)
					if end > 0 && !math.IsInf(c, 1) {
						hits = append(hits, Match{name.text, index, tokens[i].start, tokens[i+end-1].end, c})
					}
				}
				if limit < w.Other {
					for _, sound := range w.initialSounds(form, limit, omitHead) {
						for _, i := range positions[sound] {
							check(i)
						}
					}
				} else {
					for i, t := range tokens {
						if t.sound != "" {
							check(i)
						}
					}
				}
				return hits
			}
			full = find(full, maxCost, searchFullName)
			if w.MatchPrefixes && len(form) > 1 {
				partial = find(partial, min(maxCost, w.PrefixMaxCost), searchNamePrefix)
			}
		}
		first := len(result)
		result = appendUnoccupied(result, full, occupied)
		if len(partial) > 0 {
			result = appendUnoccupied(result, partial, occupied)
		}
		mark(occupied, result[first:], false)
	}
	sort.SliceStable(result, func(i, j int) bool {
		a, b := result[i], result[j]
		if a.Start != b.Start {
			return a.Start < b.Start
		}
		if a.End != b.End {
			return a.End > b.End
		}
		return a.Cost < b.Cost
	})
	return result
}

// Lower-cost hits occupy their ranges first. The caller also uses this to
// reserve full-name hits before considering partial hits for the same name.
func appendUnoccupied(result, hits []Match, occupied []bool) []Match {
	sort.Slice(hits, func(i, j int) bool {
		a, b := hits[i], hits[j]
		if a.Cost != b.Cost {
			return a.Cost < b.Cost
		}
		if a.Start != b.Start {
			return a.Start < b.Start
		}
		return a.End > b.End
	})
	for i := 0; i < len(hits); {
		group := len(result)
		cost := hits[i].Cost
		farthestEnd := -1
		for ; i < len(hits) && hits[i].Cost == cost; i++ {
			h := hits[i]
			if h.End <= farthestEnd { // contained in an equally good hit
				continue
			}
			if !overlaps(occupied[h.Start:h.End]) {
				result = append(result, h)
				farthestEnd = h.End
			}
		}
		mark(occupied, result[group:], true)
	}
	return result
}

func overlaps(bytes []bool) bool {
	for _, v := range bytes {
		if v {
			return true
		}
	}
	return false
}
func mark(bytes []bool, hits []Match, value bool) {
	for _, h := range hits {
		for i := h.Start; i < h.End; i++ {
			bytes[i] = value
		}
	}
}

// MaskDefault masks the occurrences returned by FindDefault.
func (m *Matcher) MaskDefault(text, replacement string) string {
	return maskMatches(text, replacement, m.FindDefault(text))
}

// Mask replaces every matched span, merging overlapping spans across names.
func (m *Matcher) Mask(text string, maxCost float64, replacement string) string {
	return maskMatches(text, replacement, m.Find(text, maxCost))
}

func maskMatches(text, replacement string, hits []Match) string {
	if len(hits) == 0 {
		return text
	}
	var out strings.Builder
	end := 0
	for _, h := range hits {
		if h.Start >= end {
			out.WriteString(text[end:h.Start])
			out.WriteString(replacement)
		}
		end = max(end, h.End)
	}
	out.WriteString(text[end:])
	return out.String()
}
