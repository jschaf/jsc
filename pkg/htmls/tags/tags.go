package tags

import "strings"

func WrapAttrs(tag string, attrs string, contents ...string) string {
	startTagSize := len(tag) + 2
	endTagSize := startTagSize + 1
	attrsSize := 0
	if len(attrs) > 0 {
		attrsSize = len(attrs) + 1
	}
	size := startTagSize + endTagSize + attrsSize
	for _, content := range contents {
		size += len(content)
	}
	b := strings.Builder{}
	b.Grow(size)

	b.WriteString("<" + tag)
	if len(attrs) > 0 {
		b.WriteString(" ")
		b.WriteString(attrs)
	}
	b.WriteString(">")
	for _, t := range contents {
		b.WriteString(t)
	}
	b.WriteString("</" + tag + ">")
	return b.String()
}

func Wrap(tag string, contents ...string) string {
	return WrapAttrs(tag, "", contents...)
}

func AAttrs(attrs string, ts ...string) string {
	return WrapAttrs("a", attrs, ts...)
}

func Cite(ts ...string) string {
	return CiteAttrs("", ts...)
}

func CiteAttrs(attrs string, ts ...string) string {
	return WrapAttrs("cite", attrs, ts...)
}

func Code(ts ...string) string {
	return Wrap("code", ts...)
}

func DivAttrs(attrs string, ts ...string) string {
	return WrapAttrs("div", attrs, ts...)
}

func Em(ts ...string) string {
	return Wrap("em", ts...)
}

func EmAttrs(attrs string, ts ...string) string {
	return WrapAttrs("em", attrs, ts...)
}

func H2(ts ...string) string {
	return Wrap("h2", ts...)
}

func P(ts ...string) string {
	return Wrap("p", ts...)
}

func SC(ts ...string) string {
	return WrapAttrs("span", "class=small-caps", ts...)
}

func Strong(ts ...string) string {
	return Wrap("strong", ts...)
}