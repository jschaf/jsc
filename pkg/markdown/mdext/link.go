package mdext

import (
	"bytes"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/jschaf/b2/pkg/markdown/asts"
	"github.com/jschaf/b2/pkg/markdown/attrs"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type LinkTransformer struct{}

type linkType = string

const (
	linkPDF  linkType = "pdf"
	linkWiki linkType = "wikipedia"
)

func (l *LinkTransformer) Transform(doc *ast.Document, _ text.Reader, pc parser.Context) {
	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkSkipChildren, nil
		}
		if n.Kind() != ast.KindLink {
			return ast.WalkContinue, nil
		}

		link := n.(*ast.Link)
		origDest := string(link.Destination)

		if filepath.IsAbs(origDest) || strings.HasPrefix(origDest, "http") {
			return ast.WalkContinue, nil
		}
		filePath := filepath.Dir(GetFilePath(pc))
		meta := GetTOMLMeta(pc)
		newDest := path.Join(meta.Path, origDest)
		link.Destination = []byte(newDest)
		localPath := filepath.Join(filePath, origDest)
		remotePath := filepath.Join(meta.Path, origDest)
		AddAsset(pc, remotePath, localPath)

		return ast.WalkSkipChildren, nil
	})
	if err != nil {
		panic(err)
	}
}

type linkDecorationTransform struct{}

func (l linkDecorationTransform) Transform(doc *ast.Document, reader text.Reader, pc parser.Context) {
	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkSkipChildren, nil
		}
		if n.Kind() != ast.KindLink {
			return ast.WalkContinue, nil
		}

		link := n.(*ast.Link)
		origDest := string(link.Destination)

		switch {
		case path.Ext(origDest) == ".pdf":
			link.SetAttribute([]byte("data-link-type"), []byte(linkPDF))

		case strings.HasPrefix(origDest, "https://en.wikipedia.org"):
			link.SetAttribute([]byte("data-link-type"), []byte(linkWiki))
		}

		renderPreview(pc, origDest, reader, link)

		return ast.WalkSkipChildren, nil
	})
	if err != nil {
		panic(err)
	}
}

func renderPreview(pc parser.Context, origDest string, reader text.Reader, link *ast.Link) {
	// If we have a preview, render it into the attributes.
	preview, ok := GetPreview(pc, origDest)
	if !ok {
		return
	}
	renderer, ok := GetRenderer(pc)
	if !ok {
		panic("link preview: no renderer")
	}

	colonBlock := preview.Parent
	if colonBlock == nil {
		return
	}
	// Assume title is first child.
	title := colonBlock.FirstChild()
	if title == nil {
		return
	}
	attrs.AddClass(title, "preview-title")
	titleLink := ast.NewLink()
	titleLink.Destination = []byte(origDest)
	asts.Reparent(titleLink, title)
	title.AppendChild(title, titleLink)
	titleHTML := &bytes.Buffer{}
	if err := renderer.Render(titleHTML, reader.Source(), title); err != nil {
		panic(fmt.Sprintf("render preview title to HTML for %s: %s", GetFilePath(pc), err.Error()))
	}
	link.SetAttribute([]byte("class"), []byte("preview-target"))
	link.SetAttribute([]byte("data-preview-title"), bytes.Trim(titleHTML.Bytes(), " \n"))

	// Assume the rest of the children are the body.
	snippetHTML := &bytes.Buffer{}
	snippetNode := title.NextSibling()
	for snippetNode != nil {
		if err := renderer.Render(snippetHTML, reader.Source(), snippetNode); err != nil {
			panic(fmt.Sprintf("render preview snippet to HTML for %s: %s", GetFilePath(pc), err.Error()))
		}
		snippetNode = snippetNode.NextSibling()
	}
	link.SetAttribute([]byte("data-preview-snippet"), bytes.Trim(snippetHTML.Bytes(), " \n"))
}

type LinkExt struct{}

func NewLinkExt() *LinkExt {
	return &LinkExt{}
}

func (l *LinkExt) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithASTTransformers(
			util.Prioritized(&linkDecorationTransform{}, 900),
			util.Prioritized(&LinkTransformer{}, 901)))
}
