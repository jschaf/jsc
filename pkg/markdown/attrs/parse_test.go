package attrs

import (
	"github.com/jschaf/jsc/pkg/testing/difftest"
	"github.com/jschaf/jsc/pkg/testing/require"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		expr string
		want Values
	}{
		{
			name: "both attributes with double quotes",
			expr: `{name="foo.go" description="bar"}`,
			want: Values{Name: "foo.go", Description: "bar"},
		},
		{
			name: "both attributes with single quotes",
			expr: `{description='single quotes' name='test.go'}`,
			want: Values{Name: "test.go", Description: "single quotes"},
		},
		{
			name: "only name attribute",
			expr: `{name="only name"}`,
			want: Values{Name: "only name", Description: ""},
		},
		{
			name: "only description attribute",
			expr: `{description="only description"}`,
			want: Values{Name: "", Description: "only description"},
		},
		{
			name: "Mixed quotes for attributes",
			expr: `{name="foo.go" description='bar'}`,
			want: Values{Name: "foo.go", Description: "bar"},
		},
		{
			name: "leading and trailing spaces",
			expr: `{ description="leading space" name="space.go" }`,
			want: Values{Name: "space.go", Description: "leading space"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseValues(tt.expr)
			require.NoError(t, err)
			difftest.AssertSame(t, tt.want, got)
		})
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name string
		expr string
		want string
	}{
		{
			name: "missing closing brace",
			expr: `{name="foo.go"`,
			want: "missing closing delimiter",
		},
		{
			name: "missing space between fields",
			expr: `{name="foo.go"description="bar"}`,
			want: "not separated",
		},
		{
			name: "missing quotes for name",
			expr: `{name=foo.go description="bar"}`,
			want: "missing start quote",
		},
		{
			name: "missing quotes for description",
			expr: `{name="foo.go" description=bar}`,
			want: "missing start quote",
		},
		{
			name: "invalid field name",
			expr: `{name="foo.go" desc="bar"}`,
			want: "unsupported field name",
		},
		{
			name: "missing closing quote",
			expr: `{name="foo.go" description="bar}`,
			want: "missing closing quote",
		},
		{
			name: "mismatched quotes",
			expr: `{description="bar'}`,
			want: "missing closing quote",
		},
		{
			name: "mismatched quotes reversed",
			expr: `{description="bar" name='foo.go"}`,
			want: "missing closing quote",
		},
		{
			name: "empty name value",
			expr: `{name= description="bar"}`,
			want: "missing start quote",
		},
		{
			name: "missing description value",
			expr: `{name="foo.go" description=}`,
			want: "missing start quote",
		},
		{
			name: "extra field",
			expr: `{name="foo.go" description="bar" extra="bad"}`,
			want: "unsupported field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseValues(tt.expr)
			if err == nil {
				t.Errorf("expected error for expr %q, got nil", tt.expr)
				return
			}
			got := err.Error()
			if !strings.Contains(got, tt.want) {
				t.Errorf("want error substring: %q;\ngot: %s", tt.want, got)
			}
		})
	}
}
