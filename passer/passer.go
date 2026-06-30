//Package passer parses the source and matches lines with
//generic lines in the .TAB file, after matches are found for all
//instructions, each instruction can be converted to the final binary form
//Using a "two-pass" strategy, the first pass finds .TAB matches and sizes
//and a second pass backfills addresses, finishing preparation for the binary generation
package passer

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

func NewParser(l *Lexer, encoding EncodingTable) *Parser {
	p := &Parser{
		lexer:             l,
		SymbolTable:       make(SymbolTable),
		Encoding:          encoding,
		currentLineTokens: []Token{},
	}

	// Read two tokens, so currentToken and peekToken are both set
	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) nextToken() {
	p.currentLineTokens = append(p.currentLineTokens, p.currentToken)
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

//parseLine parses one line
func (p *Parser) parseLine() *Line {
	line := &Line{}
	p.currentLineTokens = nil

	// Skip initial newlines
	p.skipNewlines()
	if p.currentToken.Type == TokenEOF {
		return nil
	}

	// Capture source location from the first meaningful token
	line.Filename = p.currentToken.Location.Filename
	line.LineNum = p.currentToken.Location.Line

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
			line.Tokens = p.currentLineTokens
			return line
		} else if p.peekToken.Type == TokenIdentifier && strings.ToLower(p.peekToken.Value) == "equ" {
			line.Assignment = p.currentToken.Value
			p.nextToken() // consume identifier
			p.nextToken() // consume "equ"
			line.Value = p.parseExpression()
			p.skipUntilNewline()
			line.Tokens = p.currentLineTokens
			return line
		}
	}

	// Handle Directive
	if p.currentToken.Type == TokenDot {
		p.nextToken() // consume dot
		if p.currentToken.Type == TokenIdentifier {
			line.Directive = p.currentToken.Value
			p.nextToken()
			if p.currentToken.Type != TokenNewline && p.currentToken.Type != TokenEOF {
				line.Operands = p.parseOperands()
			}
		}
	} else if p.currentToken.Type == TokenHash {
		p.nextToken() // consume hash
		if p.currentToken.Type == TokenIdentifier {
			line.Directive = p.currentToken.Value
			p.nextToken()
			if p.currentToken.Type == TokenEqual {
				p.nextToken()
			}
			if p.currentToken.Type != TokenNewline && p.currentToken.Type != TokenEOF {
				line.Operands = p.parseOperands()
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
	line.Tokens = p.currentLineTokens
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
	var parts []string
	for p.currentToken.Type != TokenNewline && p.currentToken.Type != TokenEOF &&
		p.currentToken.Type != TokenComma && p.currentToken.Type != TokenRParen {
		parts = append(parts, p.currentToken.Value)
		p.nextToken()
	}
	return strings.Join(parts, " ")
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

// Pass1 walks the parsed lines, assigning an address (PC) to every label
// and symbol, and advancing PC by each instruction's size from the encoding
// table. It does not emit anything — that's Pass 2's job once all symbols are
// resolved. Instruction operand values (jump targets, etc.) are deliberately
// left unresolved here because forward references aren't known until the whole
// SymbolTable is built; Pass 1 only needs sizes, which come from operand types.
func (p *Parser) Pass1(lines []Line) error {
	p.PC = 0 // origin; overridden by .org / #origin below

		for i := range lines {
		lines[i].Address = p.PC
		loc := p.lineLoc(lines[i])

		switch {
		// A label points at the CURRENT address, before any size advance.
		case lines[i].Label != "":
			if _, exists := p.SymbolTable[lines[i].Label]; exists {
				return fmt.Errorf("%s: duplicate label %q\n    line: %s", loc, lines[i].Label, lines[i].String())
			}
			p.SymbolTable[lines[i].Label] = p.PC

		// An assignment binds a name to a value; it does NOT advance PC.
		// If the value references a forward label, defer resolution to Pass2.
		case lines[i].Assignment != "":
			v, err := p.evalExpr(lines[i].Value, lines[i].Address)
			if err != nil {
				p.deferredAssignments = append(p.deferredAssignments, i)
			} else {
				p.SymbolTable[lines[i].Assignment] = v
			}

		// Directives may set or advance PC (.org, .db, .ds, ...).
		case lines[i].Directive != "":
			if err := p.applyDirective(&lines[i]); err != nil {
				return fmt.Errorf("%s: %w\n    line: %s", loc, err, lines[i].String())
			}
		}

		// A mnemonic can share a line with a label, so size it independently
		// of the switch above, after the label has been recorded.
		if lines[i].Mnemonic != "" {
			size, err := p.sizeOf(lines[i])
			if err != nil {
				return fmt.Errorf("%s: %w\n    line: %s", loc, err, lines[i].String())
			}
			lines[i].Size = size
			p.PC += size

			// Try to resolve instruction operand values; mark forward refs as unresolved.
			for j := range lines[i].Operands {
				p.resolveOperand(&lines[i].Operands[j], lines[i].Address)
			}
		}
	}
	return nil
}

// lineLoc returns a "filename:line" string for error messages.
func (p *Parser) lineLoc(line Line) string {
	if line.Filename != "" {
		return fmt.Sprintf("%s:%d", line.Filename, line.LineNum)
	}
	return fmt.Sprintf("line %d", line.LineNum)
}

// sizeOf finds the TabEntry whose operand pattern matches this line and
// returns its Size.
func (p *Parser) sizeOf(line Line) (int, error) {
	entries, ok := p.Encoding[strings.ToUpper(line.Mnemonic)]
	if !ok {
		return 0, fmt.Errorf("unknown mnemonic %q", line.Mnemonic)
	}
	for _, e := range entries {
		if matchOperands(e.Operands, line.Operands) {
			return e.Size, nil
		}
	}
	fmt.Println(line)
	ops := make([]string, len(line.Operands))
	for i, op := range line.Operands {
		ops[i] = op.Value
	}
	return 0, fmt.Errorf("no encoding for %q with operands %q", line.Mnemonic, strings.Join(ops, ","))
}

// matchOperands checks whether a TAB entry's operand string (e.g. "A,*", "2")
// matches the parsed operands. In the TAB format, "*" is a wildcard that
// matches any immediate or address operand.
func matchOperands(tabOperands string, parsed []Operand) bool {
	if tabOperands == "" {
		return len(parsed) == 0
	}
	parts := strings.Split(tabOperands, ",")
	if len(parts) != len(parsed) {
		return false
	}
	for i, part := range parts {
		if part == "*" {
			continue // wildcard matches any operand
		}
		if !strings.EqualFold(part, parsed[i].Value) {
			return false
		}
	}
	return true
}

type exprTokenType int

const (
	exprNumber exprTokenType = iota
	exprIdent
	exprPlus
	exprMinus
	exprStar
	exprSlash
	exprLParen
	exprRParen
	exprDollar
)

type exprToken struct {
	typ exprTokenType
	val string
}

// tokenizeExpr breaks an expression string into tokens for the evaluator.
func tokenizeExpr(s string) ([]exprToken, error) {
	var toks []exprToken
	runes := []rune(s)
	i := 0
	for i < len(runes) {
		ch := runes[i]
		if ch == ' ' || ch == '\t' {
			i++
			continue
		}
		switch {
		case ch == '+':
			toks = append(toks, exprToken{typ: exprPlus, val: "+"})
			i++
		case ch == '-':
			toks = append(toks, exprToken{typ: exprMinus, val: "-"})
			i++
		case ch == '*':
			toks = append(toks, exprToken{typ: exprStar, val: "*"})
			i++
		case ch == '/':
			toks = append(toks, exprToken{typ: exprSlash, val: "/"})
			i++
		case ch == '(':
			toks = append(toks, exprToken{typ: exprLParen, val: "("})
			i++
		case ch == ')':
			toks = append(toks, exprToken{typ: exprRParen, val: ")"})
			i++
		case ch == '$':
			// $ as hex prefix: consume following hex digits as a number
			if i+1 < len(runes) && isHexRune(runes[i+1]) {
				i++ // skip $
				start := i
				for i < len(runes) && isHexRune(runes[i]) {
					i++
				}
				toks = append(toks, exprToken{typ: exprNumber, val: "$" + string(runes[start:i])})
			} else {
				toks = append(toks, exprToken{typ: exprDollar, val: "$"})
				i++
			}
		case ch == '_' || unicode.IsLetter(ch):
			start := i
			for i < len(runes) && (runes[i] == '_' || unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i])) {
				i++
			}
			toks = append(toks, exprToken{typ: exprIdent, val: string(runes[start:i])})
		case unicode.IsDigit(ch):
			start := i
			// hex prefix 0x or 0X
			if ch == '0' && i+1 < len(runes) && (runes[i+1] == 'x' || runes[i+1] == 'X') {
				i += 2
				for i < len(runes) && isHexRune(runes[i]) {
					i++
				}
			} else {
				for i < len(runes) && isHexRune(runes[i]) {
					i++
				}
				if i < len(runes) && (runes[i] == 'h' || runes[i] == 'H') {
					i++
				}
			}
			toks = append(toks, exprToken{typ: exprNumber, val: string(runes[start:i])})
		default:
			return nil, fmt.Errorf("unexpected character %q in expression", ch)
		}
	}
	return toks, nil
}

func isHexRune(r rune) bool {
	return unicode.IsDigit(r) || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

// evalExpr evaluates a Z80 expression string (supports +, -, parens, $, symbols, numbers).
// addr is the current line's address, used for resolving $.
func (p *Parser) evalExpr(s string, addr int) (int, error) {
	if s == "" {
		return 0, nil
	}
	toks, err := tokenizeExpr(s)
	if err != nil {
		return 0, err
	}
	if len(toks) == 0 {
		return 0, fmt.Errorf("empty expression")
	}
	pos := 0
	result, err := p.parseAddExpr(toks, &pos, addr)
	if err != nil {
		return 0, err
	}
	if pos < len(toks) {
		return 0, fmt.Errorf("unexpected token %q after expression", toks[pos].val)
	}
	return result, nil
}

// parseAddExpr handles + and - (lowest precedence).
func (p *Parser) parseAddExpr(toks []exprToken, pos *int, addr int) (int, error) {
	left, err := p.parseMulExpr(toks, pos, addr)
	if err != nil {
		return 0, err
	}
	for *pos < len(toks) {
		switch toks[*pos].typ {
		case exprPlus:
			*pos++
			right, err := p.parseMulExpr(toks, pos, addr)
			if err != nil {
				return 0, err
			}
			left += right
		case exprMinus:
			*pos++
			right, err := p.parseMulExpr(toks, pos, addr)
			if err != nil {
				return 0, err
			}
			left -= right
		default:
			return left, nil
		}
	}
	return left, nil
}

// parseMulExpr handles * and / (medium precedence).
func (p *Parser) parseMulExpr(toks []exprToken, pos *int, addr int) (int, error) {
	left, err := p.parseUnary(toks, pos, addr)
	if err != nil {
		return 0, err
	}
	for *pos < len(toks) {
		switch toks[*pos].typ {
		case exprStar:
			*pos++
			right, err := p.parseUnary(toks, pos, addr)
			if err != nil {
				return 0, err
			}
			left *= right
		case exprSlash:
			*pos++
			right, err := p.parseUnary(toks, pos, addr)
			if err != nil {
				return 0, err
			}
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			left /= right
		default:
			return left, nil
		}
	}
	return left, nil
}

// parseUnary handles unary + and -.
func (p *Parser) parseUnary(toks []exprToken, pos *int, addr int) (int, error) {
	if *pos >= len(toks) {
		return 0, fmt.Errorf("unexpected end of expression")
	}
	switch toks[*pos].typ {
	case exprPlus:
		*pos++
		return p.parseUnary(toks, pos, addr)
	case exprMinus:
		*pos++
		val, err := p.parseUnary(toks, pos, addr)
		if err != nil {
			return 0, err
		}
		return -val, nil
	default:
		return p.parsePrimary(toks, pos, addr)
	}
}

// parsePrimary handles numbers, identifiers, $, and parenthesized expressions.
func (p *Parser) parsePrimary(toks []exprToken, pos *int, addr int) (int, error) {
	if *pos >= len(toks) {
		return 0, fmt.Errorf("unexpected end of expression")
	}
	tok := toks[*pos]
	*pos++

	switch tok.typ {
	case exprNumber:
		return parseNumber(tok.val)
	case exprIdent:
		if v, ok := p.SymbolTable[tok.val]; ok {
			return v, nil
		}
		return 0, fmt.Errorf("undefined symbol %q", tok.val)
	case exprDollar:
		return addr, nil
	case exprLParen:
		val, err := p.parseAddExpr(toks, pos, addr)
		if err != nil {
			return 0, err
		}
		if *pos >= len(toks) || toks[*pos].typ != exprRParen {
			return 0, fmt.Errorf("missing closing parenthesis")
		}
		*pos++
		return val, nil
	default:
		return 0, fmt.Errorf("unexpected token %q in expression", tok.val)
	}
}

// parseNumber converts a numeric token string to an int.
// Supports: $hex, 0xhex, trailing h/H, decimal.
func parseNumber(s string) (int, error) {
	if strings.HasPrefix(s, "$") {
		v, err := strconv.ParseInt(s[1:], 16, 32)
		return int(v), err
	}
	if len(s) > 1 && (s[len(s)-1] == 'h' || s[len(s)-1] == 'H') {
		v, err := strconv.ParseInt(s[:len(s)-1], 16, 32)
		if err == nil {
			return int(v), nil
		}
	}
	v, err := strconv.ParseInt(s, 0, 32) // handles 0x.. and decimal
	if err != nil {
		v, err2 := strconv.ParseInt(s, 16, 32) // fallback: bare hex like "0C0F9"
		if err2 != nil {
			return 0, fmt.Errorf("cannot parse number %q", s)
		}
		return int(v), nil
	}
	return int(v), nil
}

// applyDirective handles the subset of directives that affect addressing in
// Pass 1. The parser collapses both `.name` and `#name` into Line.Directive
// and drops any `=`
func (p *Parser) applyDirective(line *Line) error {
	switch strings.ToLower(line.Directive) {
	case "org":
		if len(line.Operands) == 0 {
			return fmt.Errorf(".org requires a value")
		}
		v, err := p.evalExpr(line.Operands[0].Value, line.Address)
		if err != nil {
			return err
		}
		line.Operands[0].Resolved = true
		line.Operands[0].IntValue = v
		p.PC = v
	case "db":
		for i := range line.Operands {
			op := &line.Operands[i]
			v, err := p.evalExpr(op.Value, line.Address)
			if err == nil {
				op.Resolved = true
				op.IntValue = v
				if v < -128 || v > 255 {
					return fmt.Errorf("byte value %d out of range", v)
				}
			}
			p.PC++
		}
	case "dw":
		for i := range line.Operands {
			op := &line.Operands[i]
			v, err := p.evalExpr(op.Value, line.Address)
			if err == nil {
				op.Resolved = true
				op.IntValue = v
				if v < -32768 || v > 65535 {
					return fmt.Errorf("word value %d out of range", v)
				}
			}
			p.PC += 2
		}
	case "ds":
		if len(line.Operands) > 0 {
			v, err := p.evalExpr(line.Operands[0].Value, line.Address)
			if err != nil {
				return err
			}
			p.PC += v
		}
	//this "plugin" format is left over from asmStudio (old Ti-calculator assembler)
	case "plugin":
	//.plugin asm86
		
	default:
		return fmt.Errorf("unknown directive %q", line.Directive)
	}
	return nil
}

// resolveOperand tries to evaluate an operand's value expression and sets the
// Resolved / IntValue fields. It is a no-op for register operands.
func (p *Parser) resolveOperand(op *Operand, addr int) {
	switch op.Type {
	case OpImmediate:
		if v, err := p.evalExpr(op.Value, addr); err == nil {
			op.Resolved = true
			op.IntValue = v
		}
	case OpIdentifier:
		if !registerOrCondition[strings.ToUpper(op.Value)] {
			if v, err := p.evalExpr(op.Value, addr); err == nil {
				op.Resolved = true
				op.IntValue = v
			}
		}
	case OpMemory:
		inner := strings.ToUpper(op.Value)
		if !registerOrCondition[inner] {
			if v, err := p.evalExpr(op.Value, addr); err == nil {
				op.Resolved = true
				op.IntValue = v
			}
		}
	}
}

// ResolveDeferred re-evaluates any assignments that were deferred in Pass1
// because they referenced forward labels. All explicit labels are now in the
// SymbolTable, so these should succeed. Returns an error if any still fail.
func (p *Parser) ResolveDeferred(lines []Line) error {
	for _, idx := range p.deferredAssignments {
		v, err := p.evalExpr(lines[idx].Value, lines[idx].Address)
		if err != nil {
			return fmt.Errorf("%s: %w\n    line: %s",
				p.lineLoc(lines[idx]), err, lines[idx].String())
		}
		p.SymbolTable[lines[idx].Assignment] = v
	}
	p.deferredAssignments = nil
	return nil
}

// Pass2 walks all lines and resolves any remaining unresolved operand values
// using the now-complete SymbolTable. After Pass2 every operand that can be
// resolved will have Resolved=true. Binary emission is not yet implemented;
// this is a placeholder that validates all references resolve.
func (p *Parser) Pass2(lines []Line) error {
	for i := range lines {
		loc := p.lineLoc(lines[i])

		// Resolve instruction operands
		if lines[i].Mnemonic != "" {
			for j := range lines[i].Operands {
				p.resolveOperand(&lines[i].Operands[j], lines[i].Address)
			}
		}

		// Resolve directive operands (db, dw)
		if lines[i].Directive != "" {
			for j := range lines[i].Operands {
				if !lines[i].Operands[j].Resolved {
					p.resolveOperand(&lines[i].Operands[j], lines[i].Address)
				}
			}
		}

		// Check for any still-unresolved operands (should not happen after ResolveDeferred)
		for _, op := range lines[i].Operands {
			if op.Resolved {
				continue
			}
			// Registers are expected to be unresolved; only flag non-register identifiers.
			if op.Type == OpIdentifier && !registerOrCondition[strings.ToUpper(op.Value)] {
				return fmt.Errorf("%s: unresolved forward reference %q\n    line: %s",
					loc, op.Value, lines[i].String())
			}
			if op.Type == OpMemory {
				inner := strings.ToUpper(op.Value)
				if !registerOrCondition[inner] {
					return fmt.Errorf("%s: unresolved forward reference %q\n    line: %s",
						loc, op.Value, lines[i].String())
				}
			}
		}
	}
	return nil
}

// registerOrCondition is the set of Z80 register, register-pair, and condition
// names. An identifier operand in this set is emitted literally in a pattern;
// anything else is treated as a symbol/address and rendered as the TAB
// wildcard "*".
var registerOrCondition = map[string]bool{
	// 8-bit registers
	"A": true, "B": true, "C": true, "D": true, "E": true, "H": true, "L": true,
	"I": true, "R": true,
	// 16-bit pairs
	"AF": true, "BC": true, "DE": true, "HL": true, "SP": true, "IX": true, "IY": true,
	"IXH": true, "IXL": true, "IYH": true, "IYL": true,
	// condition codes ("C" above doubles as carry)
	"NZ": true, "Z": true, "NC": true, "PO": true, "PE": true, "P": true, "M": true,
}

// operandPattern renders parsed operands into the exact string form the .TAB
// uses, e.g. "A,B", "(HL),A", "A,*". Registers and condition codes are emitted
// literally (uppercased); immediates and symbol/address operands become "*".
func operandPattern(operands []Operand) string {
	parts := make([]string, 0, len(operands))
	for _, op := range operands {
		parts = append(parts, operandToken(op))
	}
	return strings.Join(parts, ",")
}

func operandToken(op Operand) string {
	switch op.Type {
	case OpImmediate:
		return "*"
	case OpMemory:
		// A register/pair inside parens is a real addressing mode, e.g. (HL).
		// Anything else is a memory address operand, rendered as (*).
		inner := strings.ToUpper(op.Value)
		if registerOrCondition[inner] {
			return "(" + inner + ")"
		}
		return "(*)"
	default: // OpRegister / OpIdentifier
		upper := strings.ToUpper(op.Value)
		if registerOrCondition[upper] {
			return upper
		}
		// A bare symbol/label used as an operand is an address: "*".
		return "*"
	}
}

func tokenTypeName(t TokenType) string {
	switch t {
	case TokenEOF:
		return "EOF"
	case TokenError:
		return "Err"
	case TokenIdentifier:
		return "Id"
	case TokenNumber:
		return "Num"
	case TokenString:
		return "Str"
	case TokenComma:
		return "Comma"
	case TokenPlus:
		return "+"
	case TokenMinus:
		return "-"
	case TokenStar:
		return "*"
	case TokenSlash:
		return "/"
	case TokenPercent:
		return "%"
	case TokenLParen:
		return "("
	case TokenRParen:
		return ")"
	case TokenColon:
		return "Colon"
	case TokenSemicolon:
		return ";"
	case TokenHash:
		return "#"
	case TokenDot:
		return "."
	case TokenEqual:
		return "="
	case TokenNewline:
		return "NL"
	case TokenLeftShift:
		return "<<"
	case TokenRightShift:
		return ">>"
	default:
		return "?"
	}
}

func formatTokens(ts []Token) string {
	var b strings.Builder
	for i, t := range ts {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(tokenTypeName(t.Type))
		b.WriteByte('(')
		b.WriteString(strings.ReplaceAll(t.Value, "\n", `\n`)) //don't print the literal newline because it messes up the table
		b.WriteByte(')')
	}
	return b.String()
}

// PrintLines prints the results of Pass1: line index, address, size, label,
// mnemonic+operands, and directive/assignment info.
func PrintLines(lines []Line, symTable SymbolTable) {
	fmt.Println("\nresults of Pass1:")
	fmt.Printf("%-4s %-8s %-4s  %-16s %-10s %-20s %-10s %-12s %-10s %s\n",
		"Lnum", "Address", "Size", "Label", "Mnemonic", "Operands", "Directive", "Assignment", "Value", "Tokens")
	for i, line := range lines {
		addr := fmt.Sprintf("$%04X", line.Address)
		label := ""
		if line.Label != "" {
			label = line.Label + ":"
		}

		ops := make([]string, len(line.Operands))
		for j, op := range line.Operands {
			ops[j] = op.Value
		}

		fmt.Printf("%-4d %-8s %-4d  %-16s %-10s %-20s %-10s %-12s %-10s %s\n",
			i, addr, line.Size, label,
			line.Mnemonic,
			strings.Join(ops, ", "),
			line.Directive,
			line.Assignment,
			line.Value,
			formatTokens(line.Tokens))
	}

	// Print symbol table
	fmt.Println("\nSymbol table:")
	if len(symTable) == 0 {
		fmt.Println("  (empty)")
	}
	symbols := make([]string, 0, len(symTable))
	for sym := range symTable {
		symbols = append(symbols, sym)
	}
	sort.Strings(symbols)
	for _, sym := range symbols {
		fmt.Printf("  %-16s = $%04X (%d)\n", sym, symTable[sym], symTable[sym])
	}
}
