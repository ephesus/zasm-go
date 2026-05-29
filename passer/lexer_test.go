package passer

import (
	"testing"
)

func TestNextToken(t *testing.T) {
	input := `ld a, b
xor a ; xx
rl l
rla
_textShadow = $9f00
#origin = _textShadow
.org origin
label1:
  ld (hl), a
  add a, 15
  ret
  .db 0, 1, 2
`

	//try a table driven test 
	//labels are TokenIdentifier
	tests := []struct {
		expectedType  TokenType
		expectedValue string
	}{
		{TokenIdentifier, "ld"},
		{TokenIdentifier, "a"},
		{TokenComma, ","},
		{TokenIdentifier, "b"},
		{TokenNewline, "\n"},
		{TokenIdentifier, "xor"},
		{TokenIdentifier, "a"},
		{TokenNewline, "\n"},
		{TokenIdentifier, "rl"},
		{TokenIdentifier, "l"},
		{TokenNewline, "\n"},
		{TokenIdentifier, "rla"},
		{TokenNewline, "\n"},
		{TokenIdentifier, "_textShadow"},
		{TokenEqual, "="},
		{TokenNumber, "$9f00"},
		{TokenNewline, "\n"},
		{TokenHash, "#"},
		{TokenIdentifier, "origin"},
		{TokenEqual, "="},
		{TokenIdentifier, "_textShadow"},
		{TokenNewline, "\n"},
		{TokenDot, "."},
		{TokenIdentifier, "org"},
		{TokenIdentifier, "origin"},
		{TokenNewline, "\n"},
		{TokenIdentifier, "label1"},
		{TokenColon, ":"},
		{TokenNewline, "\n"},
		{TokenIdentifier, "ld"},
		{TokenLParen, "("},
		{TokenIdentifier, "hl"},
		{TokenRParen, ")"},
		{TokenComma, ","},
		{TokenIdentifier, "a"},
		{TokenNewline, "\n"},
		{TokenIdentifier, "add"},
		{TokenIdentifier, "a"},
		{TokenComma, ","},
		{TokenNumber, "15"},
		{TokenNewline, "\n"},
		{TokenIdentifier, "ret"},
		{TokenNewline, "\n"},
		{TokenDot, "."},
		{TokenIdentifier, "db"},
		{TokenNumber, "0"},
		{TokenComma, ","},
		{TokenNumber, "1"},
		{TokenComma, ","},
		{TokenNumber, "2"},
		{TokenNewline, "\n"},
		{TokenEOF, ""},
	}

	l := NewLexer(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%d, got=%d",
				i, tt.expectedType, tok.Type)
		}

		if tok.Value != tt.expectedValue {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedValue, tok.Value)
		}
	}
}
