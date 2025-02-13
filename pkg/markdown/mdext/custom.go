package mdext

import (
	"github.com/jschaf/jsc/pkg/markdown/attrs"
	"github.com/jschaf/jsc/pkg/markdown/extenders"
	"github.com/jschaf/jsc/pkg/markdown/ord"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

var KindCustomInline = ast.NewNodeKind("CustomInline")

type CustomInline struct {
	ast.BaseInline
	Tag string
}

func NewCustomInline(tag string) *CustomInline {
	return &CustomInline{
		Tag: tag,
	}
}

func (c *CustomInline) Kind() ast.NodeKind {
	return KindCustomInline
}

func (c *CustomInline) Dump(source []byte, level int) {
	ast.DumpHelper(c, source, level, nil, nil)
}

type customInlineRenderer struct{}

func (cir customInlineRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindCustomInline, cir.renderCustom)
}

func (cir customInlineRenderer) renderCustom(w util.BufWriter, _ []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	c := n.(*CustomInline)
	if entering {
		_ = w.WriteByte('<')
		_, _ = w.WriteString(c.Tag)
		attrs.RenderAll(w, c)
		_ = w.WriteByte('>')
	} else {
		_, _ = w.WriteString("</")
		_, _ = w.WriteString(c.Tag)
		_ = w.WriteByte('>')
	}
	return ast.WalkContinue, nil
}

// CustomExt extends Markdown with the custom tag renderers.
type CustomExt struct{}

func NewCustomExt() CustomExt {
	return CustomExt{}
}

func (c CustomExt) Extend(m goldmark.Markdown) {
	extenders.AddRenderer(m, customInlineRenderer{}, ord.CustomRenderer)
}
