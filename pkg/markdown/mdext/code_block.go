package mdext

import (
	"bytes"
	"fmt"
	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/lexers"
	"github.com/jschaf/jsc/pkg/markdown/attrs"
	"github.com/jschaf/jsc/pkg/markdown/extenders"
	"github.com/jschaf/jsc/pkg/markdown/ord"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"html"
	"io"
)

// codeBlockRenderer renders code blocks, replacing the default renderer.
type codeBlockRenderer struct{}

func (c codeBlockRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, c.render)
}

func (c codeBlockRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (status ast.WalkStatus, err error) {
	n := node.(*ast.FencedCodeBlock)

	if entering {
		info, err := parseCodeBlockInfo(n, source)
		if err != nil {
			return ast.WalkStop, fmt.Errorf("parse code block info: %w", err)
		}

		lexer := getLexer(info.lang)

		tokenIter, err := lexer.Tokenise(nil, readAllCodeBlockLines(n, source))
		if err != nil {
			panic(err)
		}
		if err := formatCodeBlock(w, tokenIter, info); err != nil {
			panic(err)
		}

	}
	return ast.WalkContinue, nil
}

type codeInfo struct {
	lang        string
	name        string
	description string
}

func parseCodeBlockInfo(n *ast.FencedCodeBlock, source []byte) (codeInfo, error) {
	if n.Info == nil {
		return codeInfo{}, nil
	}
	segment := n.Info.Segment
	info := bytes.TrimSpace(segment.Value(source))
	split := bytes.IndexByte(info, ' ')
	if split == -1 {
		return codeInfo{lang: string(info)}, nil
	}
	lang := string(info[:split])
	vals, err := attrs.ParseValues(string(info[split+1:]))
	if err != nil {
		return codeInfo{}, fmt.Errorf("parse extented attribute values: %w", err)
	}

	return codeInfo{
		lang:        lang,
		name:        vals.Name,
		description: vals.Description,
	}, nil
}

func readAllCodeBlockLines(n *ast.FencedCodeBlock, source []byte) string {
	var b bytes.Buffer
	l := n.Lines().Len()
	for i := 0; i < l; i++ {
		line := n.Lines().At(i)
		b.Write(line.Value(source))
	}
	return b.String()
}

func getLexer(language string) chroma.Lexer {
	lexer := lexers.Fallback
	if language != "" {
		lexer = lexers.Get(language)
	}
	lexer = chroma.Coalesce(lexer)
	return lexer
}

func formatCodeBlock(w io.Writer, iterator chroma.Iterator, info codeInfo) error {
	writeStrings(w, "<div class='code-block-container'>")
	lines := chroma.SplitTokensIntoLines(iterator.Tokens())
	minLineLegend := 3
	if info.lang != "" && info.lang != "text" && len(lines) > minLineLegend {
		writeStrings(w, "<div class='code-block-lang'>", info.lang, "</div>")
	}
	writeStrings(w, "<pre class='code-block'>")

	for _, tokens := range lines {
		for i, token := range tokens {
			h := html.EscapeString(token.String())
			switch token.Type {

			case chroma.Comment:
				fallthrough
			case chroma.CommentHashbang:
				fallthrough
			case chroma.CommentMultiline:
				fallthrough
			case chroma.CommentPreproc:
				fallthrough
			case chroma.CommentPreprocFile:
				fallthrough
			case chroma.CommentSingle:
				fallthrough
			case chroma.CommentSpecial:
				writeStrings(w, "<code-comment>", h, "</code-comment>")

			case chroma.Keyword:
				fallthrough
			case chroma.KeywordConstant:
				fallthrough
			case chroma.KeywordDeclaration:
				fallthrough
			case chroma.KeywordNamespace:
				fallthrough
			case chroma.KeywordPseudo:
				fallthrough
			case chroma.KeywordReserved:
				fallthrough
			case chroma.KeywordType:
				writeStrings(w, "<code-kw>", h, "</code-kw>")

			case chroma.NameFunction:
				switch info.lang {
				case "go":
					if i < 2 {
						writeStrings(w, h)
						continue
					}
					isFunc := tokens[i-2].Value == "func"
					isReceiver := tokens[i-2].Value == ")"
					if isFunc || isReceiver {
						writeStrings(w, "<code-fn>", h, "</code-fn>")
					} else {
						writeStrings(w, h)
					}

				default:
					writeStrings(w, "<code-fn>", h, "</code-fn>")
				}

			case chroma.String:
				fallthrough
			case chroma.StringAffix:
				fallthrough
			case chroma.StringBacktick:
				fallthrough
			case chroma.StringChar:
				fallthrough
			case chroma.StringDelimiter:
				fallthrough
			case chroma.StringDoc:
				fallthrough
			case chroma.StringDouble:
				fallthrough
			case chroma.StringEscape:
				fallthrough
			case chroma.StringHeredoc:
				fallthrough
			case chroma.StringInterpol:
				fallthrough
			case chroma.StringOther:
				fallthrough
			case chroma.StringRegex:
				fallthrough
			case chroma.StringSingle:
				fallthrough
			case chroma.StringSymbol:
				writeStrings(w, "<code-str>", h, "</code-str>")

			default:
				writeStrings(w, h)
			}
		}
	}

	writeStrings(w, "</pre>")
	writeStrings(w, "</div>")

	// Info block.
	if info.name != "" || info.description != "" {
		writeStrings(w, "<div class='code-block-info'>")
		if info.name != "" {
			writeStrings(w, "<div class='code-block-name'>", info.name, "</div>")
		}
		if info.description != "" {
			writeStrings(w, "<div class='code-block-description'>", info.description, "</div>")
		}
		writeStrings(w, "</div>")
	}
	return nil
}

func writeStrings(w io.Writer, ss ...string) {
	for _, s := range ss {
		_, _ = w.Write([]byte(s))
	}
}

// CodeBlockExt extends Markdown to better render code blocks with syntax
// highlighting.
type CodeBlockExt struct{}

func NewCodeBlockExt() CodeBlockExt {
	return CodeBlockExt{}
}

func (c CodeBlockExt) Extend(m goldmark.Markdown) {
	extenders.AddRenderer(m, codeBlockRenderer{}, ord.CodeBlockRenderer)
}
