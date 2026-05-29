package passer

import (
	"testing"
)

func TestParse(t *testing.T) {
	input := `_textShadow = $9f00
#origin = _textShadow
.org origin
starting:
  ld a, b
  ld (hl), a
  add a, 15
  ret `

	zlexer := NewLexer(input)
	p := NewParser(zlexer)
	lines := p.Parse()

	expectedLines := []struct {
		Label      string
		Mnemonic   string
		Directive  string
		Assignment string
	}{
		{"", "", "", "_textShadow"},
		{"", "", "origin", ""},
		{"", "", "org", ""},
		{"starting", "", "", ""},
		{"", "ld", "", ""},
		{"", "ld", "", ""},
		{"", "add", "", ""},
		{"", "ret", "", ""},
	}

	//verify that labels, mnemonics, directives, and assignments are parsed correctly (operands are not included)
	if len(lines) != len(expectedLines) {
		t.Fatalf("expected %d lines, got %d", len(expectedLines), len(lines))
	}

	for i, expected := range expectedLines {
		if lines[i].Label != expected.Label {
			t.Errorf("line %d: expected label %q, got %q", i, expected.Label, lines[i].Label)
		}
		if lines[i].Mnemonic != expected.Mnemonic {
			t.Errorf("line %d: expected mnemonic %q, got %q", i, expected.Mnemonic, lines[i].Mnemonic)
		}
		if lines[i].Directive != expected.Directive {
			t.Errorf("line %d: expected directive %q, got %q", i, expected.Directive, lines[i].Directive)
		}
		if lines[i].Assignment != expected.Assignment {
			t.Errorf("line %d: expected assignment %q, got %q", i, expected.Assignment, lines[i].Assignment)
		}
	}
}
