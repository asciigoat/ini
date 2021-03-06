package lexer

import (
	"asciigoat.org/ini/token"
	"fmt"
	"unicode/utf8"
)

const (
	EOF = -1
)

type Lexer struct {
	name  string
	input string

	start     uint
	line, col uint

	pos, runes uint

	nextState LexerStateFn
	tokens    chan *token.Token
}

type LexerStateFn func(*Lexer) LexerStateFn

func NewLexer(name string, input string, buf uint) (*Lexer, chan *token.Token) {
	l := &Lexer{
		name:  name,
		input: input,

		line: 1,
		col:  1,

		nextState: preambleLexer,
		tokens:    make(chan *token.Token, buf),
	}
	return l, l.tokens
}

// Run is intended as the execution loop in a Goroutine
func (l *Lexer) Run() {
	for l.nextState != nil {
		l.nextState = l.nextState(l)
	}
	close(l.tokens)
}

// NextToken on the other hand consumes tokens and moves the loop
// without the need of Goroutines
func (l *Lexer) NextToken() *token.Token {
	for {
		select {
		case t := <-l.tokens:
			return t
		default:
			if l.nextState != nil {
				l.nextState = l.nextState(l)
			} else {
				close(l.tokens)
				return nil
			}
		}
	}
}

// Token generates a new token.Token with proper context
func (l *Lexer) Token(typ token.TokenType, val string) *token.Token {
	return &token.Token{typ, val, l.name, l.line, l.col}
}

// Emit Token
func (l *Lexer) emitNotEmpty(typ token.TokenType) {
	if !l.empty() {
		l.emit(typ)
	}
}

func (l *Lexer) emitBackNotEmpty(runes, bytes uint, typ token.TokenType) {
	if !l.emptyBack(runes, bytes) {
		l.emitBack(runes, bytes, typ)
	}
}

func (l *Lexer) emitBack(runes, bytes uint, typ token.TokenType) {
	l.tokens <- l.Token(typ, l.input[l.start:l.pos-bytes])

	l.start = l.pos - bytes
	l.col += l.runes - runes
	l.runes = runes
}

func (l *Lexer) emit(typ token.TokenType) {
	l.tokens <- l.Token(typ, l.input[l.start:l.pos])

	l.start = l.pos
	l.col += l.runes
	l.runes = 0
}

func (l *Lexer) emitEOL() {
	l.emit(token.TokenEOL)
	l.line += 1
	l.col = 1
}

func (l *Lexer) emitEOF() {
	l.emit(token.TokenEOF)
}

func (l *Lexer) emitError(str string) {
	var s string
	s = l.input[l.start:l.pos]
	s = fmt.Sprintf("%s: %q (%v)", str, s, len(s))

	l.tokens <- l.Token(token.TokenError, s)

	l.start = l.pos
	l.col += l.runes
	l.runes = 0
}

// Helpers
func (l *Lexer) empty() bool {
	return l.start == l.pos
}
func (l *Lexer) emptyBack(_, bytes uint) bool {
	return (l.start + bytes) == l.pos
}

func (l *Lexer) next() (rune, uint) {
	if l.pos >= uint(len(l.input)) {
		return EOF, 0
	}
	r, size := utf8.DecodeRuneInString(l.input[l.pos:])
	w := uint(size)

	l.pos += w   // byte offset
	l.runes += 1 // column step
	return r, w
}

func (l *Lexer) back(runes, bytes uint) {
	l.runes -= runes
	l.pos -= bytes
}

func (l *Lexer) forth(runes, bytes uint) {
	l.runes += runes
	l.pos += bytes
}

func (l *Lexer) skip() {
	l.start = l.pos
	l.col += l.runes
	l.runes = 0
}
