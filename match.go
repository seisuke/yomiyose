// Package yomiyose finds Japanese personal names in transcripts and returns
// byte ranges in the original text. A Matcher can be reused concurrently.
package yomiyose

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Weights assigns a nonnegative edit cost to each rule. Positive infinity
// disables a rule. Start with DefaultWeights to override individual costs.
type Weights struct {
	LongSpelling         float64 // alternative ways of writing the same long vowel
	EiSpelling           float64 // ei vs an e long vowel
	OuSpelling           float64 // ou vs an o long vowel
	LongLoss             float64 // medial length difference
	FinalLongLoss        float64 // final length difference
	LongInsertion        float64
	LongDeletion         float64
	RepeatedVowelPenalty float64
	VowelLoss            float64 // ai vs a
	Geminate             float64 // contextual presence/absence of ッ
	BV                   float64 // corresponding b/v moras
	TiChi                float64 // ティ/チ
	MedialVoicing        float64 // voiced/unvoiced pairs after the first mora
	InitialKS            float64 // same vowel with k/s in the first mora
	TailLoss             float64 // one missing final ordinary mora of the reference
	Other                float64 // other mora insertions, deletions and substitutions
	Hiragana             float64 // per observed mora originally written in hiragana
}

// DefaultWeights returns initial ranking costs, not estimated probabilities.
func DefaultWeights() Weights {
	return Weights{LongSpelling: 0, LongLoss: 0.25, FinalLongLoss: 0.125, LongInsertion: 2, LongDeletion: 1.25, RepeatedVowelPenalty: 2, TiChi: 0.25, VowelLoss: 0.25, Geminate: 0.25,
		BV: 0.25, MedialVoicing: 0.25, InitialKS: 0.5, TailLoss: 0.75, Other: 2, Hiragana: 0.125}
}

// Name groups a primary spelling with optional aliases. All spellings are searched.
// For a list of readings without aliases, use NewReadings.
type Name struct {
	Text     string
	Readings []string
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
}

// Matcher holds compiled name readings and their comparison weights.
type Matcher struct {
	names []entry
	w     Weights
}

// NewReadings compiles kana or romaji readings, one per name. Match.Name is
// the original registered reading, and Match.NameIndex is its index in readings.
func NewReadings(readings []string, w Weights) (*Matcher, error) {
	names := make([]Name, len(readings))
	for i, reading := range readings {
		names[i].Text = reading
	}
	return New(names, w)
}

// New compiles the name list. Weights are copied and the input slices can be reused.
func New(names []Name, w Weights) (*Matcher, error) {
	for _, v := range []float64{w.LongSpelling, w.EiSpelling, w.OuSpelling, w.LongLoss, w.FinalLongLoss, w.LongInsertion, w.LongDeletion, w.RepeatedVowelPenalty, w.TiChi, w.VowelLoss, w.Geminate, w.BV, w.MedialVoicing, w.InitialKS, w.TailLoss, w.Other, w.Hiragana} {
		if v < 0 || math.IsNaN(v) {
			return nil, fmt.Errorf("yomiyose: weights must be nonnegative")
		}
	}
	m := &Matcher{w: w}
	for index, name := range names {
		e := entry{text: name.Text}
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
func (w Weights) lengthCost(before, after []unit, deletion bool) float64 {
	prev := before[len(before)-1].sound
	cost := w.LongInsertion
	if deletion {
		cost = w.LongDeletion
	}
	if w.RepeatedVowelPenalty != 0 && inRepeatedVowelRun(before, after) {
		cost += w.RepeatedVowelPenalty
	}
	if prev == "o" && len(after) > 0 && after[0].sound == "no" {
		cost = math.Min(cost, w.LongLoss)
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
func (w Weights) gap(u []unit, i int, other []unit, end int, reference bool) float64 {
	s := u[i].sound
	if !phonetic(s) {
		return math.Inf(1)
	}
	cost := w.Other
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
			if next != "" && next == opposite && (vowel(prev) == 'i' || vowel(prev) == 'u') && strings.ContainsRune("kstpchf", rune(next[0])) {
				cost = math.Min(cost, w.Geminate)
			}
		}
	}
	if reference && i > 0 && i == len(u)-1 && vowel(s) != 0 && end > 0 && !isLong(u[i-1].sound, s) {
		cost = math.Min(cost, w.TailLoss)
	}
	return cost
}

func phonetic(s string) bool {
	return len(s) > 0 && (s[0] >= 'a' && s[0] <= 'z' || s == "^" || s == ":")
}

func (w Weights) substitution(a, b []unit, i, j int) float64 {
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
	if i > 0 && j > 0 && voicingPartner[x] == y {
		cost = math.Min(cost, w.MedialVoicing)
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
var ksPartner = partners("ka/sa ki/shi ku/su ke/se ko/so kya/sha kyu/shu kyo/sho")

// align compares one name to an observation beginning at a fixed position.
// In prefix mode it remembers the cheapest endpoint and stops as soon as all
// live paths cost more. Future edits have nonnegative costs.
func (w Weights) align(a, b []unit, prev, curr []float64, limit float64, prefix bool) (int, float64) {
	prev[0] = 0
	for i := range a {
		prev[i+1] = prev[i] + w.gap(a, i, b, 0, true)
	}
	best, end := math.Inf(1), 0
	for j, t := range b {
		if t.sound == "" {
			break
		}
		script := 0.0
		if prefix && t.hira {
			script = w.Hiragana
		}
		curr[0] = prev[0] + w.gap(b, j, a, 0, false) + script
		low := curr[0]
		for i := range a {
			diag := prev[i] + w.substitution(a, b, i, j) + script
			del := curr[i] + w.gap(a, i, b, j+1, true)
			ins := prev[i+1] + w.gap(b, j, a, i+1, false) + script
			curr[i+1] = math.Min(diag, math.Min(del, ins))
			low = math.Min(low, curr[i+1])
		}
		prev, curr = curr, prev
		if prefix {
			if low > limit {
				break
			}
			cost := prev[len(a)]
			if cost <= limit {
				best, end, limit = cost, j+1, cost
			}
		}
	}
	if prefix {
		return end, best
	}
	return len(b), prev[len(a)]
}

// Distance is the phonetic cost of a whole reference and observation. It omits
// the hiragana penalty used by Find. TailLoss is directional.
func (m *Matcher) Distance(reference, observed string) float64 {
	a, b := tokenize(reference), tokenize(observed)
	for _, units := range [][]unit{a, b} {
		for _, t := range units {
			if t.sound == "" {
				return math.Inf(1)
			}
		}
	}
	_, cost := m.w.align(a, b, make([]float64, len(a)+1), make([]float64, len(a)+1), math.Inf(1), false)
	return cost
}

// Find returns occurrences with cost <= maxCost, in source order. For each
// name, better matches suppress overlapping worse matches. For each start/reading
// the longest of its cheapest prefixes wins; equal-cost overlaps from other
// starts are retained unless one contains the other. Different names are independent.
func (m *Matcher) Find(text string, maxCost float64) []Match {
	if len(m.names) == 0 || maxCost < 0 || math.IsNaN(maxCost) {
		return nil
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
		var hits []Match
		for _, form := range name.forms {
			prev, curr := make([]float64, len(form)+1), make([]float64, len(form)+1)
			check := func(i int) {
				end, c := m.w.align(form, tokens[i:], prev, curr, maxCost, true)
				if end > 0 && !math.IsInf(c, 1) {
					hits = append(hits, Match{name.text, index, tokens[i].start, tokens[i+end-1].end, c})
				}
			}
			// Below Other, the first mora must match, or use one of the
			// permitted initial substitutions. No initial deletion is discounted.
			if maxCost < m.w.Other {
				first := form[0].sound
				for _, i := range positions[first] {
					check(i)
				}
				if m.w.BV <= maxCost {
					for _, i := range positions[bvPartner[first]] {
						check(i)
					}
				}
				if m.w.TiChi <= maxCost {
					for _, i := range positions[tiChiPartner[first]] {
						check(i)
					}
				}
				if m.w.InitialKS <= maxCost {
					for _, i := range positions[ksPartner[first]] {
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
		}
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
		first := len(result)
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

// Mask replaces every matched span, merging overlapping spans across names.
func (m *Matcher) Mask(text string, maxCost float64, replacement string) string {
	hits := m.Find(text, maxCost)
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
