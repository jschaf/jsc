package mdext

import (
	"fmt"
	"github.com/jschaf/jsc/pkg/markdown/attrs"
	"github.com/jschaf/jsc/pkg/markdown/extenders"
	"github.com/jschaf/jsc/pkg/markdown/ord"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"os"
	"path/filepath"
)

// KindEmbed represents an Embed node.
var KindEmbed = ast.NewNodeKind("Embed")

// Embed contains a path to a file to embed literally.
// Embed nodes are created from the ColonLine parser.
type Embed struct {
	ast.BaseBlock
	sourceDir string
	rawAttrs  string
}

func NewEmbed(sourceDir string, rawAttrs string) *Embed {
	return &Embed{
		sourceDir: sourceDir,
		rawAttrs:  rawAttrs,
	}
}

func (c *Embed) Kind() ast.NodeKind {
	return KindEmbed
}

func (c *Embed) Dump(source []byte, level int) {
	ast.DumpHelper(c, source, level, nil, nil)
}

// EmbedRenderer is the HTML renderer for an Embed node.
type EmbedRenderer struct{}

func newEmbedRenderer() EmbedRenderer { return EmbedRenderer{} }

func (tr EmbedRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindEmbed, renderEmbed)
}

func renderEmbed(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*Embed)
	vals, err := attrs.ParseValues(n.rawAttrs)
	if err != nil {
		return ast.WalkContinue, fmt.Errorf("parse embed attrs: %w", err)
	}
	if vals.Name == "" {
		return ast.WalkContinue, fmt.Errorf("embed directive missing name; :embed: {name='<path>'}")
	}
	embedPath := filepath.Join(n.sourceDir, vals.Name)
	bs, err := os.ReadFile(embedPath)
	if err != nil {
		return ast.WalkContinue, fmt.Errorf("read embed file: %w", err)
	}
	_ = w.WriteByte('\n')
	_, _ = w.Write(bs)
	_ = w.WriteByte('\n')
	return ast.WalkContinue, nil
}

type EmbedExt struct {
}

func NewEmbedExt() goldmark.Extender {
	return EmbedExt{}
}

func (t EmbedExt) Extend(m goldmark.Markdown) {
	extenders.AddRenderer(m, newEmbedRenderer(), ord.EmbedRenderer)
}
