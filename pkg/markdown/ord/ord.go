// Package ord contains ordering used to run parser, renderer, and AST
// transformation extensions.
package ord

type (
	ParserPriority          int
	ParaTransformerPriority int
	ASTTransformerPriority  int
	RendererPriority        int
)

const (
	TOMLParser            ParserPriority = 0
	ColonBlockParser      ParserPriority = 10
	ColonLineParser       ParserPriority = 12
	FootnoteLinkParser    ParserPriority = 20
	KatexParser           ParserPriority = 150
	ContinueReadingParser ParserPriority = 800
	SmallCapsParser       ParserPriority = 999
	TypographyParser      ParserPriority = 999
)

const (
	TableParaTransformer ParaTransformerPriority = 200
)

const (
	HeadingIdTransformer       ASTTransformerPriority = 600
	ArticleTransformer         ASTTransformerPriority = 900
	LinkDecorationTransformer  ASTTransformerPriority = 900
	LinkAssetTransformer       ASTTransformerPriority = 901
	FigureTransformer          ASTTransformerPriority = 999
	ImageTransformer           ASTTransformerPriority = 999
	TableCaptionTransformer    ASTTransformerPriority = 999
	FootnoteBodyTransformer    ASTTransformerPriority = 1000
	TOCTransformer             ASTTransformerPriority = 1000
	ContinueReadingTransformer ASTTransformerPriority = 1001
	KatexFeatureTransformer    ASTTransformerPriority = 1200
)

const (
	HeadingRenderer         RendererPriority = 10
	ParagraphRenderer       RendererPriority = 10
	KatexRenderer           RendererPriority = 150
	ContinueReadingRenderer RendererPriority = 500
	TableCaptionRenderer                     = TableRenderer - 1
	TableRenderer           RendererPriority = 500
	TimeRenderer            RendererPriority = 500
	ArticleRenderer         RendererPriority = 999
	CitationRenderer        RendererPriority = 999
	CodeBlockRenderer       RendererPriority = 999
	CustomRenderer          RendererPriority = 999
	FigureRenderer          RendererPriority = 999
	HeaderRenderer          RendererPriority = 999
	SmallCapsRenderer       RendererPriority = 999
	ImageRenderer           RendererPriority = 500
	FootnoteRenderer        RendererPriority = 1000
	ColonBlockRenderer      RendererPriority = 1000
	ColonLineRenderer       RendererPriority = 1000
	TOCRenderer             RendererPriority = 1000
	EmbedRenderer           RendererPriority = 1000
)
