package passer

import (
	"os"
	"path/filepath"
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

	l := NewLexer(input, "", "")

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

// writeTempAsm writes content to a temp file and returns the path and cleanup func.
func writeTempAsm(t *testing.T, dir, name, content string) (string, func()) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write temp file %s: %v", path, err)
	}
	return path, func() { os.Remove(path) }
}

func TestIncludeBareFilename(t *testing.T) {
	dir := t.TempDir()

	// Included file content
	const includedContent = `ld a, b
ret
`
	writeTempAsm(t, dir, "foo.asm", includedContent)

	input := ".include foo.asm\n"
	l := NewLexer(input, "", dir)

	tok := l.NextToken()
	if tok.Type != TokenIdentifier || tok.Value != "ld" {
		t.Fatalf("expected TokenIdentifier(ld), got %d(%q)", tok.Type, tok.Value)
	}
	tok = l.NextToken()
	if tok.Type != TokenIdentifier || tok.Value != "a" {
		t.Fatalf("expected TokenIdentifier(a), got %d(%q)", tok.Type, tok.Value)
	}
	tok = l.NextToken()
	if tok.Type != TokenComma {
		t.Fatalf("expected TokenComma, got %d(%q)", tok.Type, tok.Value)
	}
	tok = l.NextToken()
	if tok.Type != TokenIdentifier || tok.Value != "b" {
		t.Fatalf("expected TokenIdentifier(b), got %d(%q)", tok.Type, tok.Value)
	}
	tok = l.NextToken()
	if tok.Type != TokenNewline {
		t.Fatalf("expected TokenNewline, got %d(%q)", tok.Type, tok.Value)
	}
	tok = l.NextToken()
	if tok.Type != TokenIdentifier || tok.Value != "ret" {
		t.Fatalf("expected TokenIdentifier(ret), got %d(%q)", tok.Type, tok.Value)
	}
	tok = l.NextToken()
	if tok.Type != TokenNewline {
		t.Fatalf("expected TokenNewline, got %d(%q)", tok.Type, tok.Value)
	}
	// After the included file content, the lexer pops back to the parent file
	// and emits the newline that ended the include line, then EOF.
	tok = l.NextToken()
	if tok.Type != TokenNewline {
		t.Fatalf("expected TokenNewline (from include line), got %d(%q)", tok.Type, tok.Value)
	}
	tok = l.NextToken()
	if tok.Type != TokenEOF {
		t.Fatalf("expected TokenEOF, got %d(%q)", tok.Type, tok.Value)
	}
}

func TestIncludeBareFilenameHash(t *testing.T) {
	dir := t.TempDir()

	const includedContent = `nop
`
	writeTempAsm(t, dir, "bar.asm", includedContent)

	input := "#include bar.asm\n"
	l := NewLexer(input, "", dir)

	tok := l.NextToken()
	if tok.Type != TokenIdentifier || tok.Value != "nop" {
		t.Fatalf("expected TokenIdentifier(nop), got %d(%q)", tok.Type, tok.Value)
	}
	tok = l.NextToken()
	if tok.Type != TokenNewline {
		t.Fatalf("expected TokenNewline, got %d(%q)", tok.Type, tok.Value)
	}
}

func TestIncludeBareFilenameWithComment(t *testing.T) {
	dir := t.TempDir()

	const includedContent = `nop
`
	writeTempAsm(t, dir, "baz.asm", includedContent)

	input := ".include baz.asm ; this is a comment\n"
	l := NewLexer(input, "", dir)

	tok := l.NextToken()
	if tok.Type != TokenIdentifier || tok.Value != "nop" {
		t.Fatalf("expected TokenIdentifier(nop), got %d(%q)", tok.Type, tok.Value)
	}
}

func TestIncludeBareFilenameSubdir(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	const includedContent = `ret
`
	writeTempAsm(t, subdir, "inc.asm", includedContent)

	input := ".include sub/inc.asm\n"
	l := NewLexer(input, "", dir)

	tok := l.NextToken()
	if tok.Type != TokenIdentifier || tok.Value != "ret" {
		t.Fatalf("expected TokenIdentifier(ret), got %d(%q)", tok.Type, tok.Value)
	}
}

func TestIncludeQuotedStillWorks(t *testing.T) {
	dir := t.TempDir()

	const includedContent = `xor a
`
	writeTempAsm(t, dir, "quoted.asm", includedContent)

	input := ".include \"quoted.asm\"\n"
	l := NewLexer(input, "", dir)

	tok := l.NextToken()
	if tok.Type != TokenIdentifier || tok.Value != "xor" {
		t.Fatalf("expected TokenIdentifier(xor), got %d(%q)", tok.Type, tok.Value)
	}
}

func TestIncludeAngleBracketStillWorks(t *testing.T) {
	dir := t.TempDir()

	const includedContent = `inc b
`
	writeTempAsm(t, dir, "angle.asm", includedContent)

	input := ".include <angle.asm>\n"
	l := NewLexer(input, "", dir)

	tok := l.NextToken()
	if tok.Type != TokenIdentifier || tok.Value != "inc" {
		t.Fatalf("expected TokenIdentifier(inc), got %d(%q)", tok.Type, tok.Value)
	}
}

func TestIncludeFileNotFound(t *testing.T) {
	dir := t.TempDir()

	input := ".include nonexistent.asm\nld a, b\n"
	l := NewLexer(input, "", dir)

	// First token is the '.' dot token since tryConsumeInclude returned false
	tok := l.NextToken()
	if tok.Type != TokenDot {
		t.Fatalf("expected TokenDot, got %d(%q)", tok.Type, tok.Value)
	}
	// Next token surfaces the file-not-found error
	tok = l.NextToken()
	if tok.Type != TokenError {
		t.Fatalf("expected TokenError, got %d(%q)", tok.Type, tok.Value)
	}
}
