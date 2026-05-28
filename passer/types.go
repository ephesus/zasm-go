package passer

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

type Token struct {
    Type  TokenType
    Value string
}

type Lexer struct {
    input        string
    position     int  // current position in input (points to current char)
    readPosition int  // next reading position in input (after current char)
    ch           byte // current char under examination
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

type Line struct {
    Label      string
    Mnemonic   string
    Operands   []Operand
    Directive  string
    Assignment string
    Value      string // for assignment or simple directive args
    Tokens     []Token // optionally keep tokens for debugging
}

type SymbolTable map[string]int

type Parser struct {
    lexer        *Lexer
    currentToken Token
    peekToken    Token
    SymbolTable  SymbolTable
    PC           int // Program Counter
}
