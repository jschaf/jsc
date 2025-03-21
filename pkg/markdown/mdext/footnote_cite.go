package mdext

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/jschaf/bibtex"
	"github.com/yuin/goldmark/ast"
)

var KindCitation = ast.NewNodeKind("citation")

// Citation is an inline node representing a citation.
// See https://pandoc.org/MANUAL.html#citations.
type Citation struct {
	ast.BaseInline
	Key bibtex.CiteKey
	// The bibtex entry this citation points to.
	Bibtex bibtex.Entry
	// The prefix in a citation reference, i.e., the "foo" in `[^foo @qux]`.
	Prefix string
	// The suffix in a citation reference, i.e., the "bar" in `[^@qux bar]`.
	Suffix string
}

func NewCitation() *Citation {
	return &Citation{}
}

func (c *Citation) Kind() ast.NodeKind {
	return KindCitation
}

func (c *Citation) Dump(source []byte, level int) {
	ast.DumpHelper(c, source, level, nil, nil)
}

// CiteID returns the HTML CiteID that links to a citation.
func (c *Citation) CiteID(count int) string {
	if count == 0 {
		return "footnote-link-" + c.Key
	}
	return "footnote-link-" + c.Key + "-" + strconv.Itoa(count)
}

// ReferenceID returns the HTML ID that links to the full reference for a
// citation, displayed in the reference section, if any.
func (c *Citation) ReferenceID() string {
	return "cite_ref_" + c.Key
}

var (
	KindCitationRef        = ast.NewNodeKind("CitationRef")
	KindCitationReferences = ast.NewNodeKind("CitationReferences")
)

type CitationRef struct {
	ast.BaseInline
	Citation *Citation
	// Order the citation appeared in the doc. Not contiguous because sidenotes
	// are also ordered.
	Order int
	// The number of times the citation appeared in the doc. Useful for generating
	// backlinks from the reference to the citation link in the doc.
	Count int
}

func NewCitationRef() *CitationRef {
	return &CitationRef{}
}

func (c *CitationRef) Kind() ast.NodeKind {
	return KindCitationRef
}

func (c *CitationRef) Dump(source []byte, level int) {
	ast.DumpHelper(c, source, level, nil, nil)
}

// CitationReferences is a list of citations that appeared in the document.
type CitationReferences struct {
	ast.BaseBlock
	// Unique list of citations refs ordered by appearance.
	Refs []*CitationRef
}

func NewCitationReferences() *CitationReferences {
	return &CitationReferences{}
}

func (r *CitationReferences) Kind() ast.NodeKind {
	return KindCitationReferences
}

func (r *CitationReferences) Dump(source []byte, level int) {
	ast.DumpHelper(r, source, level, nil, nil)
}

// CitationReferencesAttacher determines how references are attached to the
// source document.
type CitationReferencesAttacher interface {
	Attach(doc *ast.Document, refs *CitationReferences) error
}

// CitationArticleAttacher attaches citation references to the end of an
// article.
type CitationArticleAttacher struct{}

func NewCitationArticleAttacher() CitationArticleAttacher {
	return CitationArticleAttacher{}
}

func (c CitationArticleAttacher) Attach(doc *ast.Document, refs *CitationReferences) error {
	var article *Article
	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkSkipChildren, nil
		}
		if _, ok := n.(*Article); !ok {
			return ast.WalkContinue, nil
		}
		article = n.(*Article)
		return ast.WalkStop, nil
	})
	if err != nil {
		return fmt.Errorf("citation: article attacher: %w", err)
	}
	if article == nil {
		return errors.New("citation: no article node found to attach citation references")
	}

	article.AppendChild(article, refs)
	return nil
}

func NewCitationNopAttacher() CitationReferencesAttacher {
	return nil
}
