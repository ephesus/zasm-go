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
		lexer:       l,
		SymbolTable: make(SymbolTable),
		Encoding:    encoding,
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

//parseLine parses one line
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

		switch {
		// A label points at the CURRENT address, before any size advance.
		case lines[i].Label != "":
			if _, exists := p.SymbolTable[lines[i].Label]; exists {
				return fmt.Errorf("line %d: duplicate label %q", i, lines[i].Label)
			}
			p.SymbolTable[lines[i].Label] = p.PC

		// An assignment binds a name to a value; it does NOT advance PC.
		case lines[i].Assignment != "":
			v, err := p.evalValue(lines[i].Value)
			if err != nil {
				return fmt.Errorf("line %d: %w", i, err)
			}
			p.SymbolTable[lines[i].Assignment] = v

		// Directives may set or advance PC (.org, .db, .ds, ...).
		case lines[i].Directive != "":
			if err := p.applyDirective(lines[i]); err != nil {
				return fmt.Errorf("line %d: %w", i, err)
			}
		}

		// A mnemonic can share a line with a label, so size it independently
		// of the switch above, after the label has been recorded.
		if lines[i].Mnemonic != "" {
			size, err := p.sizeOf(lines[i])
			if err != nil {
				return fmt.Errorf("line %d: %w", i, err)
			}
			lines[i].Size = size
			p.PC += size
		}
	}
	return nil
}

// sizeOf finds the TabEntry whose operand pattern matches this line and
// returns its Size. This is the step that needs operands classified well
// enough to disambiguate same-mnemonic variants (see operandPattern).
func (p *Parser) sizeOf(line Line) (int, error) {
	entries, ok := p.Encoding[strings.ToUpper(line.Mnemonic)]
	if !ok {
		return 0, fmt.Errorf("unknown mnemonic %q", line.Mnemonic)
	}
	pattern := operandPattern(line.Operands)
	for _, e := range entries {
		if e.Operands == pattern {
			return e.Size, nil
		}
	}
	return 0, fmt.Errorf("no encoding for %q with operands %q", line.Mnemonic, pattern)
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
	v, err := strconv.ParseInt(s, 0, 32) // handles 0x.. and decimal
	if err != nil {
		return 0, fmt.Errorf("cannot resolve value %q", s)
	}
	return int(v), nil
}

// applyDirective handles the subset of directives that affect addressing in
// Pass 1. The parser collapses both `.name` and `#name` into Line.Directive
// and drops any `=`, so a `#name = value` form (e.g. "#origin = _textShadow")
// is indistinguishable from a defining directive — we treat any unrecognized
// directive that carries a value as a symbol definition.
func (p *Parser) applyDirective(line Line) error {
	switch strings.ToLower(line.Directive) {
	case "org":
		v, err := p.evalValue(line.Value)
		if err != nil {
			return err
		}
		p.PC = v
	// case "db", "dw", "ds": advance PC by the data size (TODO)
	default:
		// `#name = value` defines a symbol; valueless directives are no-ops.
		if line.Value != "" {
			v, err := p.evalValue(line.Value)
			if err != nil {
				return err
			}
			p.SymbolTable[line.Directive] = v
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

// PrintLines prints the results of Pass1: line index, address, size, label,
// mnemonic+operands, and directive/assignment info.
func PrintLines(lines []Line, symTable SymbolTable) {
	fmt.Println("\nresults of Pass1:")
	fmt.Printf("%-4s %-8s %-4s  %-16s %s\n", "Lnum", "Address", "Size", "Label", "Instruction")
	for i, line := range lines {
		addr := fmt.Sprintf("$%04X", line.Address)
		label := ""
		if line.Label != "" {
			label = line.Label + ":"
		}

		var inst string
		switch {
		case line.Mnemonic != "":
			ops := make([]string, len(line.Operands))
			for j, op := range line.Operands {
				ops[j] = op.Value
			}
			inst = line.Mnemonic
			if len(ops) > 0 {
				inst += " " + strings.Join(ops, ", ")
			}
		case line.Directive != "":
			inst = "." + line.Directive
			if line.Value != "" {
				inst += " " + line.Value
			}
		case line.Assignment != "":
			inst = line.Assignment + " = " + line.Value
		}

		fmt.Printf("%-4d %-8s %-4d  %-16s %s\n", i, addr, line.Size, label, inst)
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
