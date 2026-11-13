// Package braille converts text to uncontracted Kazakh Braille as Unicode
// Braille patterns (U+2800-U+28FF).
//
// Table: Wikipedia "Kazakh Braille", based on UNESCO "World Braille Usage".
// The Russian letters have the same cells, so one table serves kk and ru.
// There is no capital sign (sources disagree on it): text is lowercased.
package braille

import (
	"slices"
	"strings"
	"unicode"
)

// NumberSign is written once before each run of digits.
const NumberSign = '⠼'

type pair struct {
	char rune
	cell rune
}

// The tables in alphabet order (used as is by Table).
var (
	letters = []pair{
		{'а', '⠁'}, {'ә', '⠜'}, {'б', '⠃'}, {'в', '⠺'}, {'г', '⠛'}, {'ғ', '⠻'}, {'д', '⠙'},
		{'е', '⠑'}, {'ё', '⠡'}, {'ж', '⠚'}, {'з', '⠵'}, {'и', '⠊'}, {'й', '⠯'}, {'к', '⠅'},
		{'қ', '⠹'}, {'л', '⠇'}, {'м', '⠍'}, {'н', '⠝'}, {'ң', '⠩'}, {'о', '⠕'}, {'ө', '⠣'},
		{'п', '⠏'}, {'р', '⠗'}, {'с', '⠎'}, {'т', '⠞'}, {'у', '⠥'}, {'ұ', '⠬'}, {'ү', '⠌'},
		{'ф', '⠋'}, {'х', '⠓'}, {'һ', '⠧'}, {'ц', '⠉'}, {'ч', '⠟'}, {'ш', '⠱'}, {'щ', '⠭'},
		{'ъ', '⠷'}, {'ы', '⠮'}, {'і', '⠽'}, {'ь', '⠾'}, {'э', '⠪'}, {'ю', '⠳'}, {'я', '⠫'},
	}
	punctuation = []pair{
		{',', '⠂'}, {'.', '⠲'}, {'?', '⠢'}, {'!', '⠖'}, {';', '⠆'}, {':', '⠒'},
	}
	digits = []pair{
		{'1', '⠁'}, {'2', '⠃'}, {'3', '⠉'}, {'4', '⠙'}, {'5', '⠑'},
		{'6', '⠋'}, {'7', '⠛'}, {'8', '⠓'}, {'9', '⠊'}, {'0', '⠚'},
	}
)

var (
	letterCells = toMap(letters, punctuation)
	digitCells  = toMap(digits)
)

func toMap(tables ...[]pair) map[rune]rune {
	m := map[rune]rune{}
	for _, t := range tables {
		for _, p := range t {
			m[p.char] = p.cell
		}
	}
	return m
}

// ToBraille converts text to Braille. Letters are lowercased first. Spaces and
// line breaks stay; every other character that is not in the table is left out
// and returned in skipped (each character once, in order of appearance).
func ToBraille(text string) (braille string, skipped []string) {
	var b strings.Builder
	skipped = []string{}
	inNumber := false
	for _, r := range text {
		if cell, ok := digitCells[r]; ok {
			if !inNumber {
				b.WriteRune(NumberSign)
				inNumber = true
			}
			b.WriteRune(cell)
			continue
		}
		inNumber = false
		switch {
		case r == ' ' || r == '\n' || r == '\r':
			b.WriteRune(r)
		case letterCells[unicode.ToLower(r)] != 0:
			b.WriteRune(letterCells[unicode.ToLower(r)])
		default:
			if s := string(r); !slices.Contains(skipped, s) {
				skipped = append(skipped, s)
			}
		}
	}
	return b.String(), skipped
}

// Dots returns the raised dots of a Braille cell, e.g. "1-4-5-6" for ⠹.
// In Unicode, bit n of (cell - U+2800) is dot n+1.
func Dots(cell rune) string {
	bits := cell - 0x2800
	var dots []string
	for d := 0; d < 8; d++ {
		if bits&(1<<d) != 0 {
			dots = append(dots, string(rune('1'+d)))
		}
	}
	return strings.Join(dots, "-")
}

// Entry is one row of the Braille table.
type Entry struct {
	Char    string `json:"char" example:"қ"`
	Braille string `json:"braille" example:"⠹"`
	Dots    string `json:"dots" example:"1-4-5-6"`
	Kind    string `json:"kind" example:"letter" enums:"letter,punctuation,digit,sign"`
}

// Table returns all cells: letters in alphabet order, punctuation, the number
// sign and the digits.
func Table() []Entry {
	var t []Entry
	add := func(kind string, ps []pair) {
		for _, p := range ps {
			t = append(t, Entry{Char: string(p.char), Braille: string(p.cell), Dots: Dots(p.cell), Kind: kind})
		}
	}
	add("letter", letters)
	add("punctuation", punctuation)
	t = append(t, Entry{Char: "number sign", Braille: string(NumberSign), Dots: Dots(NumberSign), Kind: "sign"})
	add("digit", digits)
	return t
}
