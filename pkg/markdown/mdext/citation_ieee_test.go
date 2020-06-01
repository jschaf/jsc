package mdext

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jschaf/b2/pkg/cite"
	"github.com/jschaf/b2/pkg/cite/bibtex"
	"github.com/jschaf/b2/pkg/htmls/tags"
	"github.com/yuin/goldmark/ast"
)

func newCiteIEEE(key bibtex.Key, order string) string {
	attrs := fmt.Sprintf(`id=%s`, "cite_"+key)
	aAttrs := fmt.Sprintf(
		`href="%s" class=preview-target data-link-type=citation`,
		"#cite_ref_"+key)
	return tags.AAttrs(aAttrs, tags.CiteAttrs(attrs, order))
}

func newCiteRefIEEE(key bibtex.Key, order string, content ...string) string {
	attrs := fmt.Sprintf(`id=%s class=cite-reference`, "cite_ref_"+key)
	return tags.DivAttrs(attrs, tags.Cite(order), strings.Join(content, ""))
}

func TestNewCitationExt_IEEE(t *testing.T) {
	style := cite.IEEE
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			"ignores prefix and suffix",
			"[**qux**, @bib_foo *bar*]",
			tags.P(newCiteIEEE("bib_foo", "[1]")),
		},
		{
			"keeps surrounding text",
			"alpha [@bib_foo] bravo",
			tags.P("alpha ", newCiteIEEE("bib_foo", "[1]"), " bravo"),
		},
		{
			"numbers different citations",
			"alpha [@bib_foo] bravo [@bib_bar]",
			tags.P("alpha ", newCiteIEEE("bib_foo", "[1]"), " bravo ", newCiteIEEE("bib_bar", "[2]")),
		},
		{
			"re-use citation numbers",
			"alpha [@bib_foo] bravo [@bib_bar] charlie [@bib_foo] delta [@bib_bar]",
			tags.P(
				"alpha ", newCiteIEEE("bib_foo", "[1]"),
				" bravo ", newCiteIEEE("bib_bar", "[2]"),
				" charlie ", newCiteIEEE("bib_foo", "[1]"),
				" delta ", newCiteIEEE("bib_bar", "[2]"),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md, ctx := newMdTester(t, NewCitationExt(style, NewCitationNopAttacher()))
			SetTOMLMeta(ctx, PostMeta{
				BibPaths: []string{"./testdata/citation_test.bib"},
			})
			assertNoRenderDiff(t, md, ctx, tt.src, tt.want)
		})
	}
}

type citeDocAttacher struct{}

func (c citeDocAttacher) Attach(doc *ast.Document, refs *CitationReferences) error {
	doc.AppendChild(doc, refs)
	return nil
}

func newCiteRefsIEEE(ts ...string) string {
	return tags.DivAttrs("class=cite-references",
		tags.H2("References"),
		strings.Join(ts, ""))
}

func newJournal(ts ...string) string {
	return tags.EmAttrs("class=cite-journal", ts...)
}

func TestNewCitationExt_IEEE_References(t *testing.T) {
	style := cite.IEEE
	tests := []struct {
		name     string
		src      string
		wantBody string
		wantRefs string
	}{
		{
			"2 references",
			"alpha [@bib_foo] bravo [@bib_bar] charlie [@bib_foo] delta [@bib_bar]",
			tags.P(
				"alpha ", newCiteIEEE("bib_foo", "[1]"),
				" bravo ", newCiteIEEE("bib_bar", "[2]"),
				" charlie ", newCiteIEEE("bib_foo", "[1]"),
				" delta ", newCiteIEEE("bib_bar", "[2]"),
			),
			newCiteRefsIEEE(
				newCiteRefIEEE("bib_foo", "[1]",
					"Fred Q. Bloggs, John P. Doe, Another Idiot, ",
					`"Turtles in the time continum," in`,
					newJournal("Turtles in the Applied Sciences"),
					", Vol. 3, 2016.",
				),
				newCiteRefIEEE("bib_bar", "[2]",
					"Orti, E., Bredas, J.L., Clarisse, C., ",
					`"Turtles in the time continum," in`,
					newJournal("Nature"),
					", Vol. 3, 2019.",
				),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md, ctx := newMdTester(t, NewCitationExt(style, citeDocAttacher{}))
			SetTOMLMeta(ctx, PostMeta{
				BibPaths: []string{"./testdata/citation_test.bib"},
			})
			assertNoRenderDiff(t, md, ctx, tt.src, tt.wantBody+"\n"+tt.wantRefs)
		})
	}
}