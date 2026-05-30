package passer

import (
	"strings"
	"testing"
)

func TestLoadTabFile(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    map[string]int
		wantErr bool
	}{
		{
			name: "basic entries",
			input: "ADC  A,(HL)  8E   1 NOP 1\nADC  A,A     8F   1 NOP 1\nADD  A,B     80   1 NOP 1",
			want: map[string]int{"ADC": 2, "ADD": 1},
		},
		{
			name: "skip blank lines",
			input: "RET        C9   1 NOP 1\n\nNOP        00   1 NOP 1",
			want: map[string]int{"RET": 1, "NOP": 1},
		},
		{
			name:  "quoted empty operand",
			input: "IND  \"\"      AAED 2 NOP 1",
			want:  map[string]int{"IND": 1},
		},
		{
			name:    "empty input",
			input:   "",
			want:    map[string]int{},
		},
		{
			name: "COMBINE encoding entry",
			input: "LD   (IX*),* 36DD 4 COMBINE 1",
			want: map[string]int{"LD": 1},
		},
		{
			name: "entry size parsed correctly",
			input: "JP   *       C3   3 NOP 1",
			want: map[string]int{"JP": 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table, err := LoadTabFile(strings.NewReader(tt.input))
			if (err != nil) != tt.wantErr {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(table) != len(tt.want) {
				t.Fatalf("expected %d mnemonics, got %d", len(tt.want), len(table))
			}
			for mnemonic, count := range tt.want {
				entries := table[mnemonic]
				if len(entries) != count {
					t.Errorf("mnemonic %s: expected %d entries, got %d", mnemonic, count, len(entries))
				}
			}
		})
	}
}

func TestLoadTabFileEntryFields(t *testing.T) {
	input := "ADC  A,(HL)  8E   1 NOP 1"
	table, err := LoadTabFile(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	entries := table["ADC"]
	if len(entries) != 1 {
		t.Fatalf("expected 1 ADC entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Operands != "A,(HL)" {
		t.Errorf("expected operands %q, got %q", "A,(HL)", e.Operands)
	}
	if e.Opcode != "8E" {
		t.Errorf("expected opcode %q, got %q", "8E", e.Opcode)
	}
	if e.Size != 1 {
		t.Errorf("expected size 1, got %d", e.Size)
	}
	if e.Encoding != "NOP" {
		t.Errorf("expected encoding %q, got %q", "NOP", e.Encoding)
	}
}
