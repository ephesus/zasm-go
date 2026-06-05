package passer

func NewLexer(input string) *Lexer {
    l := &Lexer{input: input}
    l.readChar()
    return l
}

func (l *Lexer) readChar() {
    if l.readPosition >= len(l.input) {
        l.ch = 0
    } else {
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
		tok = l.newToken(TokenHash, l.ch)
	case '.':
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

func (l *Lexer) newToken(tokenType TokenType, ch byte) Token {
	return Token{Type: tokenType, Value: string(ch)}
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
