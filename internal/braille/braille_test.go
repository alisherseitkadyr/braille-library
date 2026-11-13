package braille

import (
	"slices"
	"strings"
	"testing"
)

// The table from the task, written out again on purpose: the test checks the
// code against an independent copy, not against itself.
const kazakhTable = "а⠁ ә⠜ б⠃ в⠺ г⠛ ғ⠻ д⠙ е⠑ ё⠡ ж⠚ з⠵ и⠊ й⠯ к⠅ қ⠹ л⠇ м⠍ н⠝ ң⠩ о⠕ ө⠣ п⠏ р⠗ " +
	"с⠎ т⠞ у⠥ ұ⠬ ү⠌ ф⠋ х⠓ һ⠧ ц⠉ ч⠟ ш⠱ щ⠭ ъ⠷ ы⠮ і⠽ ь⠾ э⠪ ю⠳ я⠫"

func TestEveryLetter(t *testing.T) {
	entries := strings.Fields(kazakhTable)
	if len(entries) != 42 {
		t.Fatalf("test table has %d letters, want 42", len(entries))
	}
	for _, e := range entries {
		r := []rune(e)
		letter, want := string(r[0]), string(r[1])
		t.Run(letter, func(t *testing.T) {
			if got, skipped := ToBraille(letter); got != want || len(skipped) != 0 {
				t.Errorf("ToBraille(%q) = %q, %v; want %q", letter, got, skipped, want)
			}
			upper := strings.ToUpper(letter)
			if got, _ := ToBraille(upper); got != want {
				t.Errorf("ToBraille(%q) = %q; want %q (lowercased)", upper, got, want)
			}
		})
	}
}

func TestToBraille(t *testing.T) {
	tests := []struct {
		in, want string
		skipped  []string
	}{
		{"Абай", "⠁⠃⠁⠯", nil},
		{"Қазақ", "⠹⠁⠵⠁⠹", nil},
		{"2026", "⠼⠃⠚⠃⠋", nil},
		{"1234567890", "⠼⠁⠃⠉⠙⠑⠋⠛⠓⠊⠚", nil},
		{"Абай жолы 2026!", "⠁⠃⠁⠯ ⠚⠕⠇⠮ ⠼⠃⠚⠃⠋⠖", nil},
		{", . ? ! ; :", "⠂ ⠲ ⠢ ⠖ ⠆ ⠒", nil},
		// The number sign is repeated after anything that ends a run of digits.
		{"1 2", "⠼⠁ ⠼⠃", nil},
		{"12,5", "⠼⠁⠃⠂⠼⠑", nil},
		// Spaces and line breaks stay.
		{"ақ  су\n\nкөл", "⠁⠹  ⠎⠥\n\n⠅⠣⠇", nil},
		// Anything else is left out and reported once, in order.
		{"бір-бірі «сөз» — ok-", "⠃⠽⠗⠃⠽⠗⠽ ⠎⠣⠵  ", []string{"-", "«", "»", "—", "o", "k"}},
		{"", "", nil},
	}
	for _, tt := range tests {
		got, skipped := ToBraille(tt.in)
		if tt.skipped == nil {
			tt.skipped = []string{}
		}
		if got != tt.want || !slices.Equal(skipped, tt.skipped) {
			t.Errorf("ToBraille(%q) = %q, %q; want %q, %q", tt.in, got, skipped, tt.want, tt.skipped)
		}
	}
}

func TestDots(t *testing.T) {
	tests := map[rune]string{'⠁': "1", '⠹': "1-4-5-6", '⠼': "3-4-5-6", '⠫': "1-2-4-6", '⠿': "1-2-3-4-5-6", '⠀': ""}
	for cell, want := range tests {
		if got := Dots(cell); got != want {
			t.Errorf("Dots(%q) = %q, want %q", cell, got, want)
		}
	}
}

// Letter and punctuation cells must be unique, so a cell can be read back as one
// character (the UI shows the letter of a cell). Digits reuse letter cells; the
// number sign tells them apart.
func TestCellsAreUnique(t *testing.T) {
	seen := map[string]string{}
	for _, e := range Table() {
		if e.Kind == "digit" {
			continue
		}
		if other, ok := seen[e.Braille]; ok {
			t.Errorf("cell %s is used by %q and %q", e.Braille, other, e.Char)
		}
		seen[e.Braille] = e.Char
	}
	if n := len(Table()); n != 42+6+1+10 {
		t.Errorf("Table has %d entries, want 59", n)
	}
}
