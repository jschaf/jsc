package mdtest

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jschaf/jsc/pkg/htmls"
	"github.com/jschaf/jsc/pkg/markdown/mdctx"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

const PostPath = "/md/test/path.md"

// NewTester creates a new Markdown with the given extensions. We can't use
// our top level Markdown because it would create a circular dependency.
func NewTester(_ *testing.T, exts ...goldmark.Extender) (goldmark.Markdown, parser.Context) {
	md := goldmark.New(goldmark.WithExtensions(exts...))
	pc := parser.NewContext()
	mdctx.SetFilePath(pc, PostPath)
	mdctx.SetRenderer(pc, md.Renderer())

	return md, pc
}

// MustParseMarkdown parses Markdown into a document node.
func MustParseMarkdown(t *testing.T, md goldmark.Markdown, ctx parser.Context, src string) ast.Node {
	t.Helper()
	reader := text.NewReader([]byte(src))
	doc := md.Parser().Parse(reader, parser.WithContext(ctx))
	if errs := mdctx.PopErrors(ctx); len(errs) > 0 {
		t.Fatalf("errors while parsing: %v", errs)
	}
	return doc
}

// AssertNoRenderDiff asserts the Markdown instance renders src into the wanted
// string.
func AssertNoRenderDiff(t *testing.T, doc ast.Node, md goldmark.Markdown, src, want string, opts ...cmp.Option) {
	t.Helper()
	bufW := &bytes.Buffer{}
	if err := md.Renderer().Render(bufW, []byte(src), doc); err != nil {
		t.Fatal(err)
	}

	if diff, err := htmls.Diff(strings.NewReader(want), bufW, opts...); err != nil {
		t.Fatal(err)
	} else if diff != "" {
		t.Errorf("Render mismatch (-want +got):\n%s", diff)
	}
}
