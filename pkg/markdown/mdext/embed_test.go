package mdext

import (
	"testing"

	"github.com/jschaf/jsc/pkg/htmls/tags"
	"github.com/jschaf/jsc/pkg/markdown/mdtest"
	"github.com/jschaf/jsc/pkg/texts"
)

func TestNewEmbedExt(t *testing.T) {
	tests := []struct {
		src  string
		want string
	}{
		{
			texts.Dedent(`
				:embed: {name='testdata/embed.html'}
     `),
			tags.Join(
				tags.DivAttrs("class=embed",
					tags.DivAttrs("", "embed test")),
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			md, ctx := mdtest.NewTester(t,
				NewColonLineExt(),
				NewEmbedExt(),
				NewColonBlockExt(),
			)
			doc := mdtest.MustParseMarkdown(t, md, ctx, tt.src)
			mdtest.AssertNoRenderDiff(t, doc, md, tt.src, tt.want)
		})
	}
}
