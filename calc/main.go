package main

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"text/scanner"
)

//go:generate goyacc -o eval.go eval.y

const prompt = ">>> "

var symbolTable = make([]*symEntry, 0)

type symEntry struct {
	id  string
	num *big.Float
	fn  func(...*big.Float) (*big.Float, error)
}

func main() {
	lex := &lexer{sc: new(scanner.Scanner)}
	lex.Init(os.Stdin)
	addFunc("sum", sum)
	yyParse(lex)
}

type lexer struct {
	sc *scanner.Scanner
}

func (l *lexer) Init(src io.Reader) {
	l.sc.Init(src)
	l.sc.Whitespace = 1<<'\t' | 1<<' '
	fmt.Printf(prompt)
}

func (l *lexer) Lex(lval *yySymType) int {
	for {
		switch tok, ctx := l.sc.Scan(), l.sc.TokenText(); tok {
		case scanner.EOF:
			return 0
		case scanner.Ident:
			var sym *symEntry
			for _, e := range symbolTable {
				if e.id == ctx {
					sym = e
					break
				}
			}
			if sym == nil {
				sym = new(symEntry)
				sym.id = ctx
				symbolTable = append(symbolTable, sym)
			}
			lval.sym = sym
			return IDENTIFIER
		case scanner.Int, scanner.Float:
			lval.num, _, _ = big.ParseFloat(ctx, 0, 1000, big.ToNearestEven)
			return CONSTANT
		case '\r':
			fmt.Printf(prompt)
			continue
		case '\n':
			continue
		default:
			return int(tok)
		}
	}
}

func (l *lexer) Error(e string) {
	p := len(prompt) + l.sc.Pos().Column
	fmt.Printf("%s%s\n", nstr(p-1, "~"), "^")
	fmt.Printf("%s%s\n", nstr(p, " "), e)
	os.Exit(0)
}

func wrap1(x *big.Float, act func(z, x *big.Float) *big.Float) *big.Float {
	var temp = new(big.Float)
	return act(temp, x)
}

func wrap2(x *big.Float, y *big.Float, act func(z, x, y *big.Float) *big.Float) *big.Float {
	var temp = new(big.Float)
	return act(temp, x, y)
}

func format(x *big.Float) string {
	const maxp = 8
	for acc := 0; acc < maxp; acc++ {
		r := big.NewFloat(math.Pow(10, float64(acc)))
		if r.Mul(x, r).IsInt() {
			return x.Text('f', acc)
		}
	}
	return x.Text('f', maxp)
}

func nstr(n int, s string) string {
	var buf bytes.Buffer
	for i := 0; i < n; i++ {
		buf.WriteString(s)
	}
	return buf.String()
}

func addFunc(id string, fn func(n ...*big.Float) (*big.Float, error)) {
	symbolTable = append(symbolTable, &symEntry{id: id, fn: fn})
}

func sum(ns ...*big.Float) (*big.Float, error) {
	total := new(big.Float)
	for _, n := range ns {
		total.Add(total, n)
	}
	return total, nil
}
