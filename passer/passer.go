//Package passer parses the source and matches lines with
//generic lines in the .TAB file, after matches are found for all
//instructions, each instruction can be converted to the final binary form
//Using a "two-pass" strategy, the first pass finds .TAB matches and sizes
//and a second pass backfills addresses, finishing preparation for the binary generation
package passer

func NewParser(l *Lexer) *Parser {
	p := &Parser{
		lexer:       l,
		SymbolTable: make(SymbolTable),
	}

	// Read two tokens, so currentToken and peekToken are both set
	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) nextToken() {
	p.currentToken = p.peekToken
	p.peekToken = p.lexer.NextToken()
}

func (p *Parser) Parse() []Line {
	var lines []Line

	for p.currentToken.Type != TokenEOF {
		line := p.parseLine()
		if line != nil {
			lines = append(lines, *line)
		}
		p.skipNewlines()
	}

	return lines
}

func (p *Parser) parseLine() *Line {
	line := &Line{}

	// Skip initial newlines
	p.skipNewlines()
	if p.currentToken.Type == TokenEOF {
		return nil
	}

	// Handle Label or Assignment
	// Not using a switch statement because of lines like "mylabel: ld a, b" get complicated
	if p.currentToken.Type == TokenIdentifier {
		if p.peekToken.Type == TokenColon {
			line.Label = p.currentToken.Value
			p.nextToken() // consume identifier
			p.nextToken() // consume colon
		} else if p.peekToken.Type == TokenEqual {
			line.Assignment = p.currentToken.Value
			p.nextToken() // consume identifier
			p.nextToken() // consume equal
			line.Value = p.parseExpression()
			p.skipUntilNewline()
			return line
		}
	}

	// Handle Directive
	if p.currentToken.Type == TokenDot {
		p.nextToken() // consume dot
		if p.currentToken.Type == TokenIdentifier {
			line.Directive = p.currentToken.Value
			p.nextToken()
			// Parse directive args if any
			if p.currentToken.Type != TokenNewline && p.currentToken.Type != TokenEOF {
				line.Value = p.parseExpression()
			}
		}
	} else if p.currentToken.Type == TokenHash {
		p.nextToken() // consume hash
		if p.currentToken.Type == TokenIdentifier {
			line.Directive = p.currentToken.Value
			p.nextToken()
			// Handle #directive = value or #directive value
			if p.currentToken.Type == TokenEqual {
				p.nextToken()
			}
			if p.currentToken.Type != TokenNewline && p.currentToken.Type != TokenEOF {
				line.Value = p.parseExpression()
			}
		}
	}

	// Handle Mnemonic
	if p.currentToken.Type == TokenIdentifier && line.Directive == "" && line.Assignment == "" {
		line.Mnemonic = p.currentToken.Value
		p.nextToken()
		line.Operands = p.parseOperands()
	}

	p.skipUntilNewline()
	return line
}

func (p *Parser) parseOperands() []Operand {
	var operands []Operand

	if p.currentToken.Type == TokenNewline || p.currentToken.Type == TokenEOF {
		return operands
	}

	for {
		op := p.parseOperand()
		operands = append(operands, op)

		if p.currentToken.Type == TokenComma {
			p.nextToken()
		} else {
			break
		}
	}

	return operands
}

func (p *Parser) parseOperand() Operand {
	var op Operand

	if p.currentToken.Type == TokenLParen {
		p.nextToken()
		op.Type = OpMemory
		op.Value = p.parseExpression()
		if p.currentToken.Type == TokenRParen {
			p.nextToken()
		}
	} else {
		// Simple operand for now
		op.Value = p.currentToken.Value
		if p.currentToken.Type == TokenNumber {
			op.Type = OpImmediate
		} else {
			op.Type = OpIdentifier // could be register or symbol
		}
		p.nextToken()
	}

	return op
}

func (p *Parser) parseExpression() string {
	// Very simple expression parser for now: just take the value
	val := p.currentToken.Value
	p.nextToken()
	return val
}

func (p *Parser) skipNewlines() {
	for p.currentToken.Type == TokenNewline {
		p.nextToken()
	}
}

func (p *Parser) skipUntilNewline() {
	for p.currentToken.Type != TokenNewline && p.currentToken.Type != TokenEOF {
		p.nextToken()
	}
}

func Pass() {
	// This will be called from main
}
