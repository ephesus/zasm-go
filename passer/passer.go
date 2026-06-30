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
		case lines[i].Assignment != "":
			v, err := p.evalValue(lines[i].Value)
			if err != nil {
				return fmt.Errorf("%s: %w\n    line: %s", loc, err, lines[i].String())
			}
			p.SymbolTable[lines[i].Assignment] = v

		// Directives may set or advance PC (.org, .db, .ds, ...).
		case lines[i].Directive != "":
			if err := p.applyDirective(lines[i]); err != nil {
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

// evalValue resolves a directive/assignment value: an existing symbol, a
// $-prefixed hex literal, or a decimal/0x literal. Expression support (math,
// shifts) can be layered on later.
func (p *Parser) evalValue(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	if v, ok := p.SymbolTable[s]; ok {
		return v, nil
	}
	if strings.HasPrefix(s, "$") {
		v, err := strconv.ParseInt(s[1:], 16, 32)
		return int(v), err
	}
	// Z80 convention: trailing "h" / "H" indicates hex (e.g. "0C0F9h")
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
			return 0, fmt.Errorf("cannot resolve value %q", s)
		}
		return int(v), nil
	}
	return int(v), nil
}

// applyDirective handles the subset of directives that affect addressing in
// Pass 1. The parser collapses both `.name` and `#name` into Line.Directive
// and drops any `=`
func (p *Parser) applyDirective(line Line) error {
	switch strings.ToLower(line.Directive) {
	case "org":
		if len(line.Operands) == 0 {
			return fmt.Errorf(".org requires a value")
		}
		v, err := p.evalValue(line.Operands[0].Value)
		if err != nil {
			return err
		}
		p.PC = v
	case "db":
		for _, op := range line.Operands {
			v, err := p.evalValue(op.Value)
			if err != nil {
				return err
			}
			if v < -128 || v > 255 {
				return fmt.Errorf("byte value %d out of range", v)
			}
			p.PC++
		}
	case "dw":
		for _, op := range line.Operands {
			v, err := p.evalValue(op.Value)
			if err != nil {
				return err
			}
			if v < -32768 || v > 65535 {
				return fmt.Errorf("word value %d out of range", v)
			}
			p.PC += 2
		}
	case "ds":
		if len(line.Operands) > 0 {
			v, err := p.evalValue(line.Operands[0].Value)
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
