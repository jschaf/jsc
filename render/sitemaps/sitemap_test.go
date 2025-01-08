package sitemaps

import (
	"github.com/jschaf/jsc/pkg/testing/difftest"
	"github.com/jschaf/jsc/pkg/testing/require"
	"testing"
	"time"
)

func TestSitemap_Build(t *testing.T) {
	tests := []struct {
		name string
		urls []URL
		want string
	}{
		{
			name: "single URL",
			urls: []URL{
				{
					Loc:        "https://example.com",
					LastMod:    time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC),
					ChangeFreq: "daily",
				},
			},
			want: `<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>https://example.com</loc><lastmod>2023-10-01</lastmod><changefreq>daily</changefreq></url></urlset>`,
		},
		{
			name: "single URL with lt sign",
			urls: []URL{
				{
					Loc: "https://example.com/foo<bar",
				},
			},
			want: `<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>https://example.com/foo&lt;bar</loc></url></urlset>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New()
			for _, url := range tt.urls {
				s.Add(url)
			}
			got, err := s.Build()
			require.NoError(t, err)

			difftest.AssertSame(t, tt.want, got)
		})
	}
}
