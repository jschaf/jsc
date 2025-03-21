package mdext

import "github.com/yuin/goldmark/ast"

// CloneNode creates a deep clone of the src node.
func CloneNode(src ast.Node) ast.Node {
	newN := newNode(src)
	for _, attr := range src.Attributes() {
		newN.SetAttribute(attr.Name, attr.Value)
	}
	for c := src.FirstChild(); c != nil; c = c.NextSibling() {
		newC := CloneNode(c)
		newN.AppendChild(newN, newC)
	}
	return newN
}

// newNode returns a new node that's the same type as src and copies all
// attributes of src except the ast.BaseBlock or ast.BaseInline node traits
// like children.
func newNode(src ast.Node) ast.Node {
	switch n := src.(type) {
	// Goldmark inline nodes.
	case *ast.Text:
		t := ast.NewTextSegment(n.Segment)
		t.SetSoftLineBreak(n.SoftLineBreak())
		t.SetHardLineBreak(n.HardLineBreak())
		t.SetRaw(n.IsRaw())
		return t
	case *ast.String:
		s := ast.NewString(n.Value)
		s.SetRaw(n.IsRaw())
		s.SetCode(n.IsCode())
		return s
	case *ast.CodeSpan:
		return ast.NewCodeSpan()
	case *ast.Emphasis:
		return ast.NewEmphasis(n.Level)
	case *ast.Link:
		l := ast.NewLink()
		l.Destination = n.Destination
		l.Title = n.Title
		return l
	case *ast.Image:
		img := &ast.Image{}
		img.Destination = n.Destination
		img.Title = n.Title
		return img
	case *ast.AutoLink:
		return &ast.AutoLink{
			BaseInline:   ast.BaseInline{},
			AutoLinkType: n.AutoLinkType,
			Protocol:     n.Protocol,
		}
	case *ast.RawHTML:
		return &ast.RawHTML{
			BaseInline: ast.BaseInline{},
			Segments:   n.Segments,
		}

	// Goldmark block nodes.
	case *ast.Document:
		return ast.NewDocument()
	case *ast.TextBlock:
		return ast.NewTextBlock()
	case *ast.Paragraph:
		return ast.NewParagraph()
	case *ast.Heading:
		return ast.NewHeading(n.Level)
	case *ast.ThematicBreak:
		return ast.NewThematicBreak()
	case *ast.CodeBlock:
		return ast.NewCodeBlock()
	case *ast.FencedCodeBlock:
		return ast.NewFencedCodeBlock(n.Info)
	case *ast.Blockquote:
		return ast.NewBlockquote()
	case *ast.List:
		l := ast.NewList(n.Marker)
		l.IsTight = n.IsTight
		l.Start = n.Start
		return l
	case *ast.ListItem:
		return ast.NewListItem(n.Offset)
	case *ast.HTMLBlock:
		h := ast.NewHTMLBlock(n.HTMLBlockType)
		h.ClosureLine = n.ClosureLine
		return h

	// Custom AST types.
	case *Article:
		return NewArticle()
	case *Citation:
		c := NewCitation()
		c.Key = n.Key
		c.Bibtex = n.Bibtex
		c.Prefix = n.Prefix
		c.Suffix = n.Suffix
		return c
	case *CitationRef:
		cr := NewCitationRef()
		cr.Citation = newNode(n.Citation).(*Citation)
		cr.Order = n.Order
		cr.Count = n.Count
		return cr
	case *CitationReferences:
		cr := NewCitationReferences()
		cr.Refs = make([]*CitationRef, len(n.Refs))
		for i, ref := range n.Refs {
			cr.Refs[i] = newNode(ref).(*CitationRef)
		}
		return cr
	case *ColonBlock:
		cb := NewColonBlock()
		cb.Name = n.Name
		cb.Args = n.Args
		return cb
	case *ColonLine:
		cl := NewColonLine()
		cl.Name = n.Name
		cl.RawAttrs = n.RawAttrs
		return cl
	case *ContinueReading:
		return NewContinueReading(n.Link)
	case *Figure:
		f := NewFigure()
		f.Destination = n.Destination
		f.Title = n.Title
		f.AltText = n.AltText
		return f
	case *FigCaption:
		return NewFigCaption()
	case *Header:
		return NewHeader()
	case *SmallCaps:
		sc := NewSmallCaps()
		sc.Segment = n.Segment
		return sc
	case *Time:
		return NewTime(n.Date)
	}

	panic("newNode: unrecognized node type")
}
