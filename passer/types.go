package passer

import "strings"

type TokenType int

const (
    TokenEOF TokenType = iota
    TokenError
    TokenIdentifier // ld, a, my_label
    TokenNumber     // 8, 15, 0x7F, $9F00
    TokenString     // "hello"
    TokenComma      // ,
    TokenPlus       // +
    TokenMinus      // -
    TokenStar       // *
    TokenSlash      // /
    TokenPercent    // %
    TokenLParen     // (
    TokenRParen     // )
    TokenColon      // :
    TokenSemicolon  // ;
    TokenHash       // #
    TokenDot        // .
    TokenEqual      // =
    TokenNewline    // \n
    TokenLeftShift  // <<
    TokenRightShift // >>
)

type TokenLocation struct {
	Filename string
	Line     int
}

type Token struct {
	Type     TokenType
	Value    string
	Location TokenLocation
}

type IncludeFrame struct {
	input        string
	position     int
	readPosition int
	ch           byte
	line         int
	filename     string
}

type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // next reading position in input (after current char)
	ch           byte // current char under examination
	line         int
	filename     string
	baseDir      string
	frames       []IncludeFrame
	err          error
}

type OperandType int

const (
    OpRegister OperandType = iota
    OpImmediate
    OpMemory
    OpIdentifier
)

type Operand struct {
    Type  OperandType
    Value string
}

//Line will only have a subset of these fields set, ex. a label will only have Label and Assignment
type Line struct {
	Label      string
	Mnemonic   string
	Operands   []Operand
	Directive  string
	Assignment string
	Value      string // for assignment or simple directive args
	Address    int    // PC assigned during Pass1
	Size       int    // instruction size from encoding table
	Tokens     []Token // optionally keep tokens for debugging
	Filename   string
	LineNum    int
}

func (l Line) String() string {
	var b strings.Builder
	if l.Label != "" {
		b.WriteString(l.Label)
		b.WriteString(": ")
	}
	if l.Mnemonic != "" {
		b.WriteString(l.Mnemonic)
		b.WriteByte(' ')
	}
	if len(l.Operands) > 0 {
		for i, op := range l.Operands {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(op.Value)
		}
		b.WriteByte(' ')
	}
	if l.Directive != "" {
		b.WriteString(".(")
		b.WriteString(l.Directive)
		b.WriteString(") ")
	}
	if l.Assignment != "" {
		b.WriteString(l.Assignment)
		b.WriteString(" = ")
	}
	if l.Value != "" {
		b.WriteString(l.Value)
	}
	return strings.TrimSpace(b.String())
}

type SymbolTable map[string]int //z80 is only 16 bit words

type Parser struct {
    lexer        *Lexer
    currentToken Token
    peekToken    Token
    SymbolTable  SymbolTable
    PC           int // Program Counter
	Encoding         EncodingTable
	currentLineTokens []Token
}
