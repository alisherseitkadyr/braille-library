package service

import (
	"testing"

	"github.com/alisherseitkadyr/braille-library/internal/domain"
)

func TestCheckCorrection(t *testing.T) {
	// Positions are in characters, not bytes: "Күз" is 3 characters but 6 bytes.
	content := "Күз келіп, дала сарғая бастады.\n\nАуыл шетінде кәрі терек тұрды."
	tests := []struct {
		name     string
		content  string
		position int
		original string
		ok       bool
	}{
		{"first word", content, 0, "Күз", true},
		{"word after multi-byte letters", content, 4, "келіп", true},
		{"second paragraph", content, 33, "Ауыл", true},
		{"byte offset is wrong", content, 7, "келіп", false},
		{"different text", content, 4, "кеоіп", false},
		{"past the end", content, 60, "тұрды", false},
		{"item without content", "", 999, "anything", true},
	}
	for _, tt := range tests {
		err := CheckCorrection(tt.content, domain.Correction{Position: tt.position, Original: tt.original, Suggested: "x"})
		if (err == nil) != tt.ok {
			t.Errorf("%s: CheckCorrection error = %v, want ok=%v", tt.name, err, tt.ok)
		}
	}
}
