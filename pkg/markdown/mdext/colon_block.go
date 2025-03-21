package mdext

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/jschaf/jsc/pkg/markdown/asts"
	"github.com/jschaf/jsc/pkg/markdown/extenders"
	"github.com/jschaf/jsc/pkg/markdown/mdctx"
	"github.com/jschaf/jsc/pkg/markdown/ord"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var KindColonBlock = ast.NewNodeKind("ColonBlock")

type ColonBlockName string

const (
	ColonBlockPreview  ColonBlockName = "preview"
	ColonBlockFootnote ColonBlockName = "footnote"
)

// Preview is a link preview.
type Preview struct {
	// The URL defined as an argument to this colon block, e.g:
	//    ::: preview http://example.com
	URL string
	// The parent colon block that holds the preview content.
	Parent *ColonBlock
}

var previewCtxKey = parser.NewContextKey()

// AddPreview adds a preview to the context so that it can be rendered into
// the corresponding link.
func AddPreview(pc parser.Context, p Preview) {
	if existing := pc.Get(previewCtxKey); existing == nil {
		pc.Set(previewCtxKey, make(map[string]Preview))
	}
	previews := pc.Get(previewCtxKey).(map[string]Preview)
	previews[p.URL] = p
}

// GetPreview returns the preview, if any, for the URL. Returns an empty Preview
// and false if no preview exists for the URL.
func GetPreview(pc parser.Context, url string) (Preview, bool) {
	previews, ok := pc.Get(previewCtxKey).(map[string]Preview)
	if !ok {
		return Preview{}, false
	}
	p, ok := previews[url]
	return p, ok
}

var (
	footnoteLinkCtxKey = parser.NewContextKey()
	footnoteBodyCtxKey = parser.NewContextKey()
)

func AddFootnoteLink(pc parser.Context, f *FootnoteLink) {
	existing := GetFootnoteLinks(pc)
	if existing == nil {
		existing = make([]*FootnoteLink, 0, 4)
	}
	pc.Set(footnoteLinkCtxKey, append(existing, f))
}

func GetFootnoteLinks(pc parser.Context) []*FootnoteLink {
	existing := pc.Get(footnoteLinkCtxKey)
	if existing == nil {
		return nil
	}
	return existing.([]*FootnoteLink)
}

func AddFootnoteBody(pc parser.Context, f *FootnoteBody) {
	if existing := pc.Get(footnoteBodyCtxKey); existing == nil {
		pc.Set(footnoteBodyCtxKey, make(map[FootnoteName]*FootnoteBody))
	}
	notes := pc.Get(footnoteBodyCtxKey).(map[FootnoteName]*FootnoteBody)
	notes[f.Name] = f
}

func GetFootnoteBodies(pc parser.Context) map[FootnoteName]*FootnoteBody {
	if existing := pc.Get(footnoteBodyCtxKey); existing == nil {
		pc.Set(footnoteBodyCtxKey, make(map[FootnoteName]*FootnoteBody))
	}
	return pc.Get(footnoteBodyCtxKey).(map[FootnoteName]*FootnoteBody)
}

// ColonBlock parses colon delimited structures inspired by
// https://pandoc.org/MANUAL.html#extension-fenced_divs.
// For example:
//
//	::: preview http://example.com
//	# heading
//	Some *content*
//	:::
type ColonBlock struct {
	ast.BaseBlock

	Name ColonBlockName
	Args string
}

func NewColonBlock() *ColonBlock {
	return &ColonBlock{
		BaseBlock: ast.BaseBlock{},
	}
}

func (c *ColonBlock) Kind() ast.NodeKind {
	return KindColonBlock
}

func (c *ColonBlock) Dump(source []byte, level int) {
	ast.DumpHelper(c, source, level, nil, nil)
}

// colonBlockParser parsers colon blocks.
type colonBlockParser struct{}

const (
	colonBlockDelim = ":::"
)

func (cbp colonBlockParser) Trigger() []byte {
	return []byte{':'}
}

func (cbp colonBlockParser) Open(_ ast.Node, reader text.Reader, _ parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	if !bytes.HasPrefix(line, []byte(colonBlockDelim)) {
		return nil, parser.NoChildren
	}
	reader.AdvanceLine()
	rest := bytes.Trim(line[len(colonBlockDelim):], " \t\n")
	nameArgs := bytes.SplitN(rest, []byte{' '}, 2)
	block := NewColonBlock()
	if len(nameArgs) >= 1 {
		block.Name = ColonBlockName(strings.Trim(string(nameArgs[0]), " "))
	}
	if len(nameArgs) == 2 {
		block.Args = strings.Trim(string(nameArgs[1]), " \n")
	}
	return block, parser.HasChildren
}

func (cbp colonBlockParser) Continue(_ ast.Node, reader text.Reader, _ parser.Context) parser.State {
	line, _ := reader.PeekLine()
	if bytes.HasPrefix(line, []byte(colonBlockDelim)) {
		reader.AdvanceLine()
		return parser.Close
	}
	return parser.Continue | parser.HasChildren
}

func (cbp colonBlockParser) Close(node ast.Node, _ text.Reader, pc parser.Context) {
	block := node.(*ColonBlock)
	switch block.Name {
	case ColonBlockPreview:
		url := block.Args
		preview := Preview{
			URL:    url,
			Parent: block,
		}
		AddPreview(pc, preview)
	case ColonBlockFootnote:
		// Replace the ColonBlock with a FootnoteBody.
		name, variant, err := parseFootnoteName(block.Args)
		if err != nil {
			mdctx.PushError(pc, fmt.Errorf("close colon block footnote: %w", err))
		}
		body := NewFootnoteBody()
		body.Name = name
		body.Variant = variant
		asts.Reparent(body, node)
		parent := node.Parent()
		parent.ReplaceChild(parent, node, body)
		AddFootnoteBody(pc, body)

	default:
		mdctx.PushError(pc, fmt.Errorf("unknown colon block name %q", block.Name))
	}
}

func (cbp colonBlockParser) CanInterruptParagraph() bool {
	return false // No, the colon block must be delimited by a newline.
}

func (cbp colonBlockParser) CanAcceptIndentedLine() bool {
	return false // No, the colon block must not be indented.
}

// colonBlockRenderer renders colon block by omitting them from HTML.
type colonBlockRenderer struct{}

func newColonBlockRenderer() colonBlockRenderer {
	return colonBlockRenderer{}
}

func (cbr colonBlockRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindColonBlock, cbr.renderColonBlock)
}

func (cbr colonBlockRenderer) renderColonBlock(_ util.BufWriter, _ []byte, n ast.Node, _ bool) (ast.WalkStatus, error) {
	c := n.(*ColonBlock)
	switch c.Name {
	case ColonBlockPreview:
		return ast.WalkSkipChildren, nil
	case ColonBlockFootnote:
		// Transformed into a FootnoteBody in the an AST transformer.
		return ast.WalkSkipChildren, nil
	default:
		return ast.WalkContinue, fmt.Errorf("render unknown colon block name %q", c.Name)
	}
}

// ColonBlockExt extends Markdown with support for colon blocks, like:
//
//	::: preview http://example.com
//	# header
//	:::
type ColonBlockExt struct{}

func NewColonBlockExt() goldmark.Extender {
	return ColonBlockExt{}
}

func (c ColonBlockExt) Extend(m goldmark.Markdown) {
	extenders.AddBlockParser(m, colonBlockParser{}, ord.ColonBlockParser)
	extenders.AddRenderer(m, newColonBlockRenderer(), ord.ColonBlockRenderer)
}
