package yomiyose

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Units retain their source byte range even when several characters form one mora.
type unit struct {
	sound      string
	start, end int
	hira       bool
}

type letter struct {
	r          rune
	start, end int
	hira       bool
}

var kana = map[string]string{}
var latin = map[string]string{}

func init() {
	// One column per mora; the roman spelling also identifies the sound internally.
	for _, row := range []string{
		"アイウエオ a i u e o", "カキクケコ ka ki ku ke ko", "ガギグゲゴ ga gi gu ge go",
		"サシスセソ sa shi su se so", "ザジズゼゾ za ji zu ze zo", "タチツテト ta chi tsu te to",
		"ダヂヅデド da ji zu de do", "ナニヌネノ na ni nu ne no", "ハヒフヘホ ha hi fu he ho",
		"バビブベボ ba bi bu be bo", "パピプペポ pa pi pu pe po", "マミムメモ ma mi mu me mo",
		"ヤユヨ ya yu yo", "ラリルレロ ra ri ru re ro", "ワヰヱヲンヴ wa wi we wo n vu",
		"ァィゥェォャュョヮ a i u e o ya yu yo wa", "ッー ^ :",
	} {
		fields := strings.Fields(row)
		for i, r := range []rune(fields[0]) {
			kana[string(r)] = fields[i+1]
			latin[fields[i+1]] = fields[i+1]
		}
	}
	for _, pair := range strings.Fields("キ:ky ギ:gy シ:sh ジ:j チ:ch ヂ:j ニ:ny ヒ:hy ビ:by ピ:py ミ:my リ:ry") {
		p := strings.Split(pair, ":")
		for i, r := range []rune("ャュョ") {
			s := p[1] + string("auo"[i])
			kana[p[0]+string(r)], latin[s] = s, s
		}
	}
	for _, pair := range strings.Fields("ヴァ:va ヴィ:vi ヴェ:ve ヴォ:vo ヴュ:vyu ファ:fa フィ:fi フェ:fe フォ:fo フュ:fyu ティ:ti ディ:di テュ:tyu デュ:dyu トゥ:tu ドゥ:du ウィ:wi ウェ:we ウォ:wo シェ:she ジェ:je チェ:che ツァ:tsa ツィ:tsi ツェ:tse ツォ:tso イェ:ye クァ:kwa グァ:gwa") {
		p := strings.Split(pair, ":")
		kana[p[0]], latin[p[1]] = p[1], p[1]
	}
	// Kunrei aliases; thi and twu disambiguate foreign ティ and トゥ.
	for _, pair := range strings.Fields("si:shi zi:ji ti:chi tu:tsu hu:fu sya:sha syu:shu syo:sho tya:cha tyo:cho zya:ja zyu:ju zyo:jo thi:ti twu:tu") {
		p := strings.Split(pair, ":")
		latin[p[0]] = p[1]
	}
}

const halfwidth = "ヲァィゥェォャュョッーアイウエオカキクケコサシスセソタチツテトナニヌネノハヒフヘホマミムメモヤユヨラリルレロワン"

var wideKana = []rune(halfwidth)

func kanaVoice(r, mark rune) rune {
	switch mark {
	case 'ﾞ', '\u3099':
		if strings.ContainsRune("カキクケコサシスセソタチツテトハヒフヘホ", r) {
			return r + 1
		}
		if r == 'ウ' {
			return 'ヴ'
		}
	case 'ﾟ', '\u309a':
		if strings.ContainsRune("ハヒフヘホ", r) {
			return r + 2
		}
	}
	return 0
}

func letters(s string) []letter {
	out := make([]letter, 0, utf8.RuneCountInString(s))
	for start, r := range s {
		_, size := utf8.DecodeRuneInString(s[start:])
		end := start + size
		if r == '・' || r == '･' {
			continue
		}
		if len(out) > 0 {
			prev := &out[len(out)-1]
			if voiced := kanaVoice(prev.r, r); voiced != 0 && prev.end == start {
				prev.r, prev.end = voiced, end
				continue
			}
		}
		hira := r >= 'ぁ' && r <= 'ゖ' || r == 'ゝ' || r == 'ゞ'
		switch {
		case hira:
			r += 'ア' - 'あ'
		case r >= 'ｦ' && r <= 'ﾝ':
			r = wideKana[r-'ｦ']
		case r >= '！' && r <= '～':
			r -= 0xfee0
		}
		out = append(out, letter{unicode.ToLower(r), start, end, hira})
	}
	return out
}

func asciiLetter(r rune) bool { return r >= 'a' && r <= 'z' }
func vowel(s string) byte {
	if len(s) > 0 && strings.ContainsRune("aiueo", rune(s[len(s)-1])) {
		return s[len(s)-1]
	}
	return 0
}

// romanMora consumes one romanized mora. The caller supplies only ASCII letters.
func romanMora(word string) (string, int) {
	if len(word) >= 2 && word[0] == word[1] && word[0] != 'n' && vowel(word[:1]) == 0 {
		return "^", 1
	}
	for n := min(4, len(word)); n > 0; n-- {
		if sound, ok := latin[word[:n]]; ok {
			return sound, n
		}
	}
	return word[:1], 1
}

func tokenize(s string) []unit {
	chars := letters(s)
	out := make([]unit, 0, len(chars))
	for i := 0; i < len(chars); {
		begin := i
		r := chars[i].r
		if asciiLetter(r) {
			var word strings.Builder
			for i < len(chars) && asciiLetter(chars[i].r) {
				word.WriteByte(byte(chars[i].r))
				i++
			}
			w := word.String()
			for len(w) > 0 {
				sound, n := romanMora(w)
				out = append(out, unit{sound, chars[begin].start, chars[begin+n-1].end, false})
				w, begin = w[n:], begin+n
			}
			continue
		}
		i++
		sound := kana[string(r)]
		if sound != "" && i < len(chars) {
			if pair, ok := kana[string([]rune{r, chars[i].r})]; ok {
				sound = pair
				i++
			}
		}
		switch {
		case r == ':' || r == '^':
			sound = string(r)
		case r == '-' && begin > 0 && asciiLetter(chars[begin-1].r):
			sound = ":"
		case r == ' ' || r == '\t' || r == '　':
			sound = " "
			for i < len(chars) && (chars[i].r == ' ' || chars[i].r == '\t' || chars[i].r == '　') {
				i++
			}
		case r == '’' || r == '\'':
			sound = "'"
		case sound == "" && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			sound = string(r) // literal spelling, including kanji and accented letters
		}
		out = append(out, unit{sound, chars[begin].start, chars[i-1].end, chars[begin].hira || chars[i-1].hira})
	}
	return out
}
