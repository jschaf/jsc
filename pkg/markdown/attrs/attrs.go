package attrs

import (
	"strconv"

	"github.com/jschaf/jsc/pkg/texts"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/util"
)

const (
	// CustomTagAttr supports using a different tag for an element. Must be
	// implemented by a custom renderer.
	CustomTagAttr = "custom-tag"
)

// AddClass adds additional classes to the node attributes, preserving existing
// class attributes.
func AddClass(n ast.Node, class ...string) {
	size := 0
	for _, c := range class {
		size += len(c)
	}
	size += len(class) // spaces in between old

	var old []byte
	raw, ok := n.Attribute([]byte("class"))
	if ok {
		old = raw.([]byte)
	}

	newer := old
	if cap(old) < len(old)+size {
		// The existing slice capacity can't hold everything,
		// so allocate a new slice.
		newer = make([]byte, len(old), len(old)+size)
		copy(newer, old)
	}

	if len(newer) > 0 {
		newer = append(newer, ' ')
	}

	for i, cls := range class {
		newer = append(newer, cls...)
		if i < len(class)-1 {
			newer = append(newer, ' ')
		}
	}
	n.SetAttribute([]byte("class"), newer)
}

func GetStringAttr(n ast.Node, k string) string {
	a, ok := n.AttributeString(k)
	if !ok {
		return ""
	}
	switch s := a.(type) {
	case []byte:
		return string(s)
	case string:
		return s
	default:
		return ""
	}
}

// RenderAll renders all of a given node's attributes.
func RenderAll(w util.BufWriter, node ast.Node) {
	for _, attr := range node.Attributes() {
		_ = w.WriteByte(' ')
		_, _ = w.Write(attr.Name)
		_, _ = w.WriteString(`="`)
		switch v := attr.Value.(type) {
		case []byte:
			_, _ = w.Write(util.EscapeHTML(v))
		case string:
			_, _ = w.Write(util.EscapeHTML(texts.ReadOnlyBytes(v)))
		case int:
			_, _ = w.WriteString(strconv.Itoa(v))
		case int64:
			_, _ = w.WriteString(strconv.FormatInt(v, 10))
		case uint64:
			_, _ = w.WriteString(strconv.FormatUint(v, 10))
		case uint:
			_, _ = w.WriteString(strconv.FormatUint(uint64(v), 10))
		}
		_ = w.WriteByte('"')
	}
}
