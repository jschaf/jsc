package mdext

import (
	"github.com/jschaf/jsc/pkg/markdown/extenders"
	"github.com/jschaf/jsc/pkg/markdown/ord"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// typographerParser is an inline parser that replaces common ASCII typography
// abbreviations with the real thing, like -- to an en-dash, or ... to an
// ellipsis.
type typographerParser struct{}

func (t typographerParser) Trigger() []byte {
	return []byte{'-', '.'}
}

const (
	enDash   = "–"
	emDash   = "—"
	ellipsis = "…"
)

func (t typographerParser) Parse(_ ast.Node, block text.Reader, _ parser.Context) ast.Node {
	line, _ := block.PeekLine()
	c0 := line[0]
	c1, c2 := byte('\n'), byte('\n')
	if len(line) > 1 {
		c1 = line[1]
	}
	if len(line) > 2 {
		c2 = line[2]
	}

	switch c0 {
	case '-':
		if c1 == '-' && c2 == '-' {
			n := ast.NewString([]byte(emDash))
			n.SetCode(true)
			block.Advance(3)
			return n
		} else if c1 == '-' {
			n := ast.NewString([]byte(enDash))
			n.SetCode(true)
			block.Advance(2)
			return n
		}
	case '.':
		if c1 == '.' && c2 == '.' {
			n := ast.NewString([]byte(ellipsis))
			n.SetCode(true)
			block.Advance(3)
		}
	}

	return nil
}

type TypographyExt struct{}

func NewTypographyExt() TypographyExt {
	return TypographyExt{}
}

func (t TypographyExt) Extend(m goldmark.Markdown) {
	extenders.AddInlineParser(m, typographerParser{}, ord.TypographyParser)
}
