package mdext

import (
	"testing"

	"github.com/jschaf/jsc/pkg/markdown/mdtest"
)

func TestCodeBlockExt(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "go func named",
			src: fenced(`go {name='foo.go'}
func foo() {}`),
			want: `
<div class="code-block-container">
	<pre class="code-block">
		<code-kw>func</code-kw> <code-fn>foo</code-fn>() {}
	</pre>
</div>
<div class="code-block-info"><div class="code-block-name">foo.go</div></div>
`,
		},
		{
			name: "go func highlight",
			src: fenced(`go {name='foo.go'}
func foo() {} // <HL>`),
			want: `
<div class="code-block-container">
	<pre class="code-block">
		<code-hl>
			<code-kw>func</code-kw> <code-fn>foo</code-fn>() {}
		</code-hl>
	</pre>
</div>
<div class="code-block-info"><div class="code-block-name">foo.go</div></div>
`,
		},
		{
			name: "go func highlight whitespace",
			src: fenced(`go {name='foo.go'}
func foo() {} //         <HL>`),
			want: `
<div class="code-block-container">
	<pre class="code-block">
		<code-hl>
			<code-kw>func</code-kw> <code-fn>foo</code-fn>() {}
		</code-hl>
	</pre>
</div>
<div class="code-block-info"><div class="code-block-name">foo.go</div></div>
`,
		},
		{
			name: "go func with percent",
			src: fenced(` go
Foo 28%`),
			want: `
<div class="code-block-container">
	<pre class="code-block">
		Foo 28%
	</pre>
</div>`,
		},
		{
			name: "go func receiver",
			src: fenced(` go
func (t *T) foo() {}
`),
			want: `
<div class="code-block-container">
	<pre class="code-block">
		<code-kw>func</code-kw> (t *T) <code-fn>foo</code-fn>() {}
	</pre>
</div>`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md, ctx := mdtest.NewTester(t, NewCodeBlockExt())
			doc := mdtest.MustParseMarkdown(t, md, ctx, tt.src)
			mdtest.AssertNoRenderDiff(t, doc, md, tt.src, tt.want)
		})
	}
}

func fenced(s string) string {
	return "```" + s + "\n```"
}
