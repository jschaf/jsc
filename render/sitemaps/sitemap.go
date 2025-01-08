package sitemaps

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

type Sitemap struct {
	urls []URL
}

type URL struct {
	// Loc is the URL of the page.
	Loc string
	// LastMod is the last modification date of the page. This date should be in
	// W3C Datetime format. This format allows you to omit the time portion, if
	// desired, and use YYYY-MM-DD.
	LastMod time.Time
	// ChangFreq is how frequently the page is likely to change. This value
	// provides general information to search engines and may not correlate
	// exactly to how often they crawl the page. Valid values are:
	//
	//  - always
	//  - hourly
	//  - daily
	//  - weekly
	//  - monthly
	//  - yearly
	//  - never
	ChangeFreq string
}

func New() *Sitemap {
	return &Sitemap{}
}

func (s *Sitemap) Add(url URL) {
	s.urls = append(s.urls, url)
}

func (s *Sitemap) Build() (string, error) {
	sb := &strings.Builder{}
	avgXMLSize := 180
	sb.Grow(avgXMLSize * len(s.urls))
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	//goland:noinspection HttpUrlsUsage
	sb.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	for _, url := range s.urls {
		sb.WriteString(`<url>`)
		if url.Loc == "" {
			return "", fmt.Errorf("missing url loc")
		}
		sb.WriteString(`<loc>`)
		writeEscaped(sb, url.Loc)
		sb.WriteString(`</loc>`)
		if !url.LastMod.IsZero() {
			sb.WriteString(`<lastmod>`)
			writeEscaped(sb, url.LastMod.Format("2006-01-02"))
			sb.WriteString(`</lastmod>`)
		}
		if url.ChangeFreq != "" {
			sb.WriteString(`<changefreq>`)
			writeEscaped(sb, url.ChangeFreq)
			sb.WriteString(`</changefreq>`)
		}
		sb.WriteString(`</url>`)
	}
	sb.WriteString(`</urlset>`)
	return sb.String(), nil
}

func writeEscaped(sb *strings.Builder, s string) {
	isSafe := true
	for _, r := range s {
		if r < 0x20 || r > 0x7E {
			isSafe = false
			break
		}
		switch r {
		case '<', '>', '&', '\'', '"':
			isSafe = false
		}
	}
	if isSafe {
		sb.WriteString(s)
		return
	}

	_ = xml.EscapeText(sb, []byte(s))
}
