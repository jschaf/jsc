package mdext

import (
	"github.com/jschaf/b2/pkg/markdown/mdtest"
	"github.com/jschaf/b2/pkg/texts"
	"testing"
)

func TestNewHeadingIDExt(t *testing.T) {
	tests := []struct {
		src  string
		want string
	}{
		{
			texts.Dedent(`
				# h1
				# h1 dupe
				# h1 dupe
				## h2 dupe
				## h2
				## h2 dupe
			`),
			texts.Dedent(`
				<h1 id="h1">h1</h1>
				<h1 id="h1-dupe">h1 dupe</h1>
				<h1 id="h1-dupe-1">h1 dupe</h1>
				<h2 id="h2-dupe">h2 dupe</h2>
				<h2 id="h2">h2</h2>
				<h2 id="h2-dupe-1">h2 dupe</h2>
		`),
		},
		{
			`# h1--   joe`,
			`<h1 id="h1-joe">h1--   joe</h1>`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			md, ctx := mdtest.NewTester(t, NewHeadingIDExt())
			doc := mdtest.MustParseMarkdown(t, md, ctx, tt.src)
			mdtest.AssertNoRenderDiff(t, doc, md, tt.src, tt.want)
		})
	}
}
