package passer

import (
	"os"
	"path/filepath"
	"strings"
)

func NewLexer(input, filename, baseDir string) *Lexer {
    l := &Lexer{
		input:    input,
		filename: filename,
		baseDir:  baseDir,
		line:     1,
	}
    l.readChar()
    return l
}

func (l *Lexer) readChar() {
    if l.readPosition >= len(l.input) {
        l.ch = 0
    } else {
        if l.ch == '\n' {
            l.line++
        }
        l.ch = l.input[l.readPosition]
    	l.position = l.readPosition
    	l.readPosition++
    }
}

func (l *Lexer) getChar() byte {
	return l.ch
}

// NextToken scans the next token
func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	if l.err != nil {
		err := l.err
		l.err = nil
		return Token{Type: TokenError, Value: err.Error()}
	}

	switch l.ch {
	case ',':
		tok = l.newToken(TokenComma, l.ch)
	case '+':
		tok = l.newToken(TokenPlus, l.ch)
	case '-':
		tok = l.newToken(TokenMinus, l.ch)
	case '*':
		tok = l.newToken(TokenStar, l.ch)
	case '/':
		tok = l.newToken(TokenSlash, l.ch)
	case '%':
		tok = l.newToken(TokenPercent, l.ch)
	case '(':
		tok = l.newToken(TokenLParen, l.ch)
	case ')':
		tok = l.newToken(TokenRParen, l.ch)
	case ':':
		tok = l.newToken(TokenColon, l.ch)
	case ';':
		// Skip comment until newline or EOF
		for l.ch != '\n' && l.ch != 0 {
			l.readChar()
		}
		return l.NextToken()
	case '#':
		if l.tryConsumeInclude() {
			return l.NextToken()
		}
		tok = l.newToken(TokenHash, l.ch)
	case '.':
		if l.tryConsumeInclude() {
			return l.NextToken()
		}
		tok = l.newToken(TokenDot, l.ch)
	case '=':
		tok = l.newToken(TokenEqual, l.ch)
	case '\n':
		tok = l.newToken(TokenNewline, l.ch)
	case '$':
		// Hex number starting with $
		l.readChar()
		tok.Type = TokenNumber
		tok.Value = "$" + l.readHexNumber()
		return tok
	case '<':
		if l.peekChar() == '<' {
			l.readChar()
			tok = Token{Type: TokenLeftShift, Value: "<<"}
		} else {
			tok = l.newToken(TokenError, l.ch)
		}
	case '>':
		if l.peekChar() == '>' {
			l.readChar()
			tok = Token{Type: TokenRightShift, Value: ">>"}
		} else {
			tok = l.newToken(TokenError, l.ch)
		}
	case '"':
		tok.Type = TokenString
		tok.Value = l.readString()
	case 0:
		if len(l.frames) > 0 {
			l.popFrame()
			return l.NextToken()
		}
		tok.Value = ""
		tok.Type = TokenEOF
	default:
		if isLetter(l.ch) || l.ch == '_' {
			tok.Value = l.readIdentifier()
			tok.Type = TokenIdentifier
			return tok
		} else if isDigit(l.ch) {
			tok.Value = l.readNumber()
			tok.Type = TokenNumber
			return tok
		} else {
			tok = l.newToken(TokenError, l.ch)
		}
	}

	l.readChar()
	return tok
}

// tryConsumeInclude checks if the current position starts an include directive
// (#include "file" / .include "file" / #include <file> / .include <file>),
// and if so, loads the file and pushes a lexer frame. Returns true if the
// include was consumed, false otherwise. On file-read errors it sets l.err.
func (l *Lexer) tryConsumeInclude() bool {
	idx := l.readPosition

	// Skip whitespace between #/. and "include"
	for idx < len(l.input) && (l.input[idx] == ' ' || l.input[idx] == '\t') {
		idx++
	}

	// Check for "include"
	if !strings.HasPrefix(l.input[idx:], "include") {
		return false
	}
	idx += 7 // skip "include"

	// Skip whitespace before filename
	for idx < len(l.input) && (l.input[idx] == ' ' || l.input[idx] == '\t') {
		idx++
	}
	if idx >= len(l.input) {
		return false
	}

	// Read filename: "..." or <...>
	var filename string
	switch l.input[idx] {
	case '"':
		idx++ // skip opening quote
		start := idx
		for idx < len(l.input) && l.input[idx] != '"' {
			idx++
		}
		if idx >= len(l.input) {
			return false
		}
		filename = l.input[start:idx]
		idx++ // skip closing quote
	case '<':
		idx++ // skip opening <
		start := idx
		for idx < len(l.input) && l.input[idx] != '>' {
			idx++
		}
		if idx >= len(l.input) {
			return false
		}
		filename = l.input[start:idx]
		idx++ // skip closing >
	default:
		return false
	}

	if filename == "" {
		return false
	}

	// Skip trailing whitespace
	for idx < len(l.input) && (l.input[idx] == ' ' || l.input[idx] == '\t') {
		idx++
	}

	// Handle optional ; comment
	for idx < len(l.input) && l.input[idx] == ';' {
		for idx < len(l.input) && l.input[idx] != '\n' {
			idx++
		}
	}

	// Must be at newline or EOF
	if idx < len(l.input) && l.input[idx] != '\n' {
		return false
	}

	// Resolve path
	includePath := filename
	if !filepath.IsAbs(filename) && l.baseDir != "" {
		includePath = filepath.Join(l.baseDir, filename)
	}

	// Read included file
	content, err := os.ReadFile(includePath)
	if err != nil {
		l.err = err
		return false
	}

	// Save current state as a frame, advanced past the include line
	var frame IncludeFrame
	if idx < len(l.input) && l.input[idx] == '\n' {
		frame = IncludeFrame{
			input:        l.input,
			position:     idx,
			readPosition: idx + 1,
			ch:           '\n',
			line:         l.line,
			filename:     l.filename,
		}
	} else {
		frame = IncludeFrame{
			input:        l.input,
			position:     idx,
			readPosition: idx,
			ch:           0,
			line:         l.line,
			filename:     l.filename,
		}
	}
	l.frames = append(l.frames, frame)

	// Set new state from included file
	l.input = string(content)
	l.position = 0
	l.readPosition = 0
	l.ch = 0
	l.line = 1
	l.filename = includePath
	l.readChar()

	return true
}

func (l *Lexer) newToken(tokenType TokenType, ch byte) Token {
	return Token{
		Type:  tokenType,
		Value: string(ch),
		Location: TokenLocation{
			Filename: l.filename,
			Line:     l.line,
		},
	}
}

func (l *Lexer) popFrame() {
	frame := l.frames[len(l.frames)-1]
	l.frames = l.frames[:len(l.frames)-1]
	l.input = frame.input
	l.position = frame.position
	l.readPosition = frame.readPosition
	l.ch = frame.ch
	l.line = frame.line
	l.filename = frame.filename
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readNumber() string {
	position := l.position
	for isDigit(l.ch) || l.ch == 'x' || (l.ch >= 'a' && l.ch <= 'f') || (l.ch >= 'A' && l.ch <= 'F') {
		// This is a bit loose, it will catch 0x123 and also 123
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readHexNumber() string {
	position := l.position
	for isDigit(l.ch) || (l.ch >= 'a' && l.ch <= 'f') || (l.ch >= 'A' && l.ch <= 'F') {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readString() string {
	l.readChar() // skip "
	position := l.position
	for l.ch != '"' && l.ch != 0 {
		l.readChar()
	}
	s := l.input[position:l.position]
	if l.ch == '"' {
		l.readChar()
	}
	return s
}

//view the next character without advancing the position pointer
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}
