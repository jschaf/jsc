// This extension parses TOML metadata blocks and stores metadata in
// parser.Context.
package mdext

import (
	"bytes"
	"path/filepath"
	"strings"
	"time"

	"github.com/jschaf/jsc/pkg/dirs"
	"github.com/jschaf/jsc/pkg/git"
	"github.com/jschaf/jsc/pkg/markdown/extenders"
	"github.com/jschaf/jsc/pkg/markdown/mdctx"
	"github.com/jschaf/jsc/pkg/markdown/ord"

	"github.com/BurntSushi/toml"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

const tomlSep = '+'

const (
	VisibilityPublished = "published"
)

// PostMeta is the TOML metadata of a post.
type PostMeta struct {
	// The slug from the markdown frontmatter.
	Slug string
	// The absolute URL path for this post, e.g. "/foo-bar". Has trailing slash.
	Path string
	// The title extracted from the first header.
	Title     string
	TitleNode ast.Node
	// The date from the markdown frontmatter.
	Date time.Time
	// Either draft or published.
	Visibility string
	// Paths (relative or absolute) to bibtex files to resolve references.
	BibPaths []string `toml:"bib_paths"`
}

var tomlCtxKey = parser.NewContextKey()

// GetTOMLMeta returns a TOML metadata.
func GetTOMLMeta(pc parser.Context) PostMeta {
	v := pc.Get(tomlCtxKey)
	if v == nil {
		return PostMeta{}
	}
	return v.(PostMeta)
}

func SetTOMLMeta(pc parser.Context, m PostMeta) {
	pc.Set(tomlCtxKey, m)
}

// tomlParser is a block parser for toml frontmatter.
type tomlParser struct{}

var defaultTOMLMetaParser = &tomlParser{}

// newTOMLParser returns a BlockParser that can parse TOML metadata blocks.
func newTOMLParser() parser.BlockParser {
	return defaultTOMLMetaParser
}

func isTOMLSep(line []byte) bool {
	line = util.TrimRightSpace(util.TrimLeftSpace(line))
	for i := 0; i < len(line); i++ {
		if line[i] != tomlSep {
			return false
		}
	}
	return true
}

func (t *tomlParser) Trigger() []byte {
	return []byte{tomlSep}
}

func (t *tomlParser) Open(_ ast.Node, reader text.Reader, _ parser.Context) (ast.Node, parser.State) {
	lineNum, _ := reader.Position()
	if lineNum != 0 {
		return nil, parser.NoChildren
	}
	line, _ := reader.PeekLine()
	if isTOMLSep(line) {
		return ast.NewTextBlock(), parser.NoChildren
	}
	return nil, parser.NoChildren
}

func (t *tomlParser) Continue(node ast.Node, reader text.Reader, _ parser.Context) parser.State {
	line, segment := reader.PeekLine()
	if isTOMLSep(line) {
		reader.Advance(segment.Len())
		return parser.Close
	}
	node.Lines().Append(segment)
	return parser.Continue | parser.NoChildren
}

func (t *tomlParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	lines := node.Lines()
	var buf bytes.Buffer
	for i := 0; i < lines.Len(); i++ {
		segment := lines.At(i)
		buf.Write(segment.Value(reader.Source()))
	}
	meta := &PostMeta{}
	if err := toml.Unmarshal(buf.Bytes(), &meta); err != nil {
		panic(err)
	}
	switch {
	case strings.Contains(mdctx.GetFilePath(pc), `/`+dirs.TIL+`/`):
		meta.Path = "/til/" + meta.Slug + "/"
	default:
		meta.Path = "/" + meta.Slug + "/"
	}

	postPath := mdctx.GetFilePath(pc)
	root := git.RootDir()
	for i, bib := range meta.BibPaths {
		if filepath.IsAbs(bib) {
			// Absolute starts from the root of the repository.
			meta.BibPaths[i] = filepath.Join(root, bib[1:])
		} else {
			// Relative starts from the post dir.
			meta.BibPaths[i] = filepath.Join(filepath.Dir(postPath), bib)
		}
	}

	SetTOMLMeta(pc, *meta)

	node.Parent().RemoveChild(node.Parent(), node)
}

func (t *tomlParser) CanInterruptParagraph() bool {
	return false
}

func (t *tomlParser) CanAcceptIndentedLine() bool {
	return false
}

type tomlFront struct{}

// New returns a new TOMLFrontmatter extension.
func NewTOMLExt() goldmark.Extender {
	return &tomlFront{}
}

func (t *tomlFront) Extend(m goldmark.Markdown) {
	extenders.AddBlockParser(m, newTOMLParser(), ord.TOMLParser)
}
