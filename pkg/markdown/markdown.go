package markdown

import (
	"fmt"
	"github.com/yuin/goldmark/renderer/html"
	"io"

	"github.com/jschaf/jsc/pkg/cite"
	"github.com/jschaf/jsc/pkg/markdown/assets"
	"github.com/jschaf/jsc/pkg/markdown/mdctx"
	"github.com/jschaf/jsc/pkg/markdown/mdext"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
)

type AST struct {
	Node   ast.Node
	Meta   mdext.PostMeta
	Source []byte
	Assets []assets.Blob
	// The full path to the Markdown file that this AST represents.
	Path     string
	Features *mdctx.FeatureSet
}

// Options are global configuration options for parsing and rendering Markdown.
type Options struct {
	CiteStyle    cite.Style
	CiteAttacher mdext.CitationReferencesAttacher
	// TOCStyle determines how to show the TOC. Defaults to not showing a TOC.
	TOCStyle           mdext.TOCStyle
	Extenders          []goldmark.Extender
	HeadingAnchorStyle mdext.HeadingAnchorStyle
}

type Markdown struct {
	gm   goldmark.Markdown
	opts Options
}

// Option is a functional option that manipulates the Markdown struct.
type Option func(*Markdown)

// WithTOCStyle overrides the default TOC style. The default is to show no TOC.
func WithTOCStyle(s mdext.TOCStyle) Option {
	return func(m *Markdown) {
		m.opts.TOCStyle = s
	}
}

// WithHeadingAnchorStyle overrides the default content shown on hovering over
// a heading. The default is to show nothing on hover.
func WithHeadingAnchorStyle(s mdext.HeadingAnchorStyle) Option {
	return func(m *Markdown) {
		m.opts.HeadingAnchorStyle = s
	}
}

func WithExtender(e goldmark.Extender) Option {
	parser.WithAutoHeadingID()
	return func(m *Markdown) {
		m.opts.Extenders = append(m.opts.Extenders, e)
	}
}

func defaultExtensions(opts Options) []goldmark.Extender {
	return []goldmark.Extender{
		mdext.NewArticleExt(),
		mdext.NewCodeBlockExt(),
		mdext.NewColonBlockExt(),
		mdext.NewColonLineExt(),
		mdext.NewCustomExt(),
		mdext.NewEmbedExt(),
		mdext.NewFootnoteExt(opts.CiteStyle, opts.CiteAttacher),
		mdext.NewHeaderExt(),
		mdext.NewHeadingExt(opts.HeadingAnchorStyle),
		mdext.NewHeadingIDExt(),
		mdext.NewImageExt(),
		mdext.NewKatexExt(),
		mdext.NewLinkExt(),
		mdext.NewParagraphExt(),
		mdext.NewSmallCapsExt(),
		mdext.NewTableExt(),
		mdext.NewTOCExt(opts.TOCStyle),
		mdext.NewTOMLExt(),
		mdext.NewTimeExt(),
		mdext.NewTypographyExt(),
		mdext.NewFigureExt(), // TODO: must come last, why?
	}
}

// New creates a new Markdown parser and renderer allowing additional options
// beyond the defaults.
func New(opts ...Option) *Markdown {
	m := &Markdown{
		opts: Options{
			CiteStyle:          cite.IEEE,
			CiteAttacher:       mdext.NewCitationArticleAttacher(),
			TOCStyle:           mdext.TOCStyleNone,
			HeadingAnchorStyle: mdext.HeadingAnchorStyleNone,
		},
	}
	for _, opt := range opts {
		opt(m)
	}

	m.gm = goldmark.New(
		goldmark.WithExtensions(m.opts.Extenders...),
		goldmark.WithExtensions(defaultExtensions(m.opts)...),
		goldmark.WithRendererOptions(html.WithUnsafe()),
	)
	return m
}

func (m *Markdown) Parse(path string, r io.Reader) (*AST, error) {
	bs, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	ctx := parser.NewContext()
	mdctx.SetFilePath(ctx, path)
	mdctx.SetRenderer(ctx, m.gm.Renderer())

	node := m.gm.Parser().Parse(text.NewReader(bs), parser.WithContext(ctx))
	if parseErrs := mdctx.PopErrors(ctx); len(parseErrs) == 1 {
		return nil, fmt.Errorf("parse errors in context: %w", parseErrs[0])
	} else if len(parseErrs) > 1 {
		return nil, fmt.Errorf("parse errors in context: %v", parseErrs)
	}
	meta := mdext.GetTOMLMeta(ctx)
	meta.Title = mdctx.GetTitle(ctx).Text
	meta.TitleNode = mdctx.GetTitle(ctx).Node
	mdAssets := mdctx.GetAssets(ctx)
	mdFeats := mdctx.GetFeatures(ctx)
	return &AST{
		Node:     node,
		Meta:     meta,
		Assets:   mdAssets,
		Path:     path,
		Source:   bs,
		Features: mdFeats,
	}, nil
}

// Renderer returns a Goldmark renderer.
func (m *Markdown) Renderer() renderer.Renderer {
	return m.gm.Renderer()
}

func (m *Markdown) Render(w io.Writer, source []byte, p *AST) error {
	return m.Renderer().Render(w, source, p.Node)
}
