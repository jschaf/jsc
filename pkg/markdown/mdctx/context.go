package mdctx

import (
	"github.com/jschaf/jsc/pkg/markdown/assets"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
)

var errorsCtxKey = parser.NewContextKey()

func PushError(pc parser.Context, err error) {
	var errs []error
	if e := pc.Get(errorsCtxKey); e != nil {
		errs = e.([]error)
	}
	errs = append(errs, err)
	pc.Set(errorsCtxKey, errs)
}

func PopErrors(pc parser.Context) []error {
	var errs []error
	if e := pc.Get(errorsCtxKey); e != nil {
		errs = e.([]error)
	}
	pc.Set(errorsCtxKey, nil)
	return errs
}

var assetsCtxKey = parser.NewContextKey()

// GetAssets returns all blobs associated with a post.
func GetAssets(pc parser.Context) []assets.Blob {
	m := pc.Get(assetsCtxKey)
	if _, ok := m.([]assets.Blob); m == nil || !ok {
		m = make([]assets.Blob, 0)
		pc.Set(assetsCtxKey, m)
	}
	return m.([]assets.Blob)
}

func AddAsset(pc parser.Context, b assets.Blob) {
	m := pc.Get(assetsCtxKey)
	if _, ok := m.([]assets.Blob); m == nil || !ok {
		m = make([]assets.Blob, 0, 4)
	}
	pc.Set(assetsCtxKey, append(m.([]assets.Blob), b))
}

var featuresCtxKey = parser.NewContextKey()

func GetFeatures(pc parser.Context) *FeatureSet {
	fs := pc.Get(featuresCtxKey)
	if _, ok := fs.(*FeatureSet); fs == nil || !ok {
		fs = NewFeatureSet()
		pc.Set(featuresCtxKey, fs)
	}
	return fs.(*FeatureSet)
}

func AddFeature(pc parser.Context, feat Feature) {
	fs := pc.Get(featuresCtxKey)
	if _, ok := fs.(*FeatureSet); fs == nil || !ok {
		fs = NewFeatureSet()
	}
	fs.(*FeatureSet).Add(feat)
	pc.Set(featuresCtxKey, fs)
}

var titleCtxKey = parser.NewContextKey()

type Title struct {
	Text string
	Node ast.Node
}

// GetTitle returns the title as parsed from the first H1 header in the
// document.
func GetTitle(pc parser.Context) Title {
	return pc.Get(titleCtxKey).(Title)
}

func SetTitle(pc parser.Context, title Title) {
	pc.Set(titleCtxKey, title)
}

var filePathCtxKey = parser.NewContextKey()

// GetFilePath returns the file path of the main Markdown file that initiated
// this parse. Helpful for error messages.
func GetFilePath(pc parser.Context) string {
	p := pc.Get(filePathCtxKey)
	if p == nil {
		return ""
	}
	return p.(string)
}

// SetFilePath sets the file path of the main Markdown path. Should only be
// called once per file.
func SetFilePath(pc parser.Context, val string) {
	pc.Set(filePathCtxKey, val)
}

var rendererCtxKey = parser.NewContextKey()

// GetRenderer returns the main goldmark renderer or nil if none exists. Useful
// for rendering Markdown into HTML that doesn't fit into the
// parse-transform-render model. For example, link preview data is rendered into
// HTML stored in the data attributes of <a> tags, like:
//
//	<a data-preview-title="<h1>foo</h1>">
func GetRenderer(pc parser.Context) (renderer.Renderer, bool) {
	r := pc.Get(rendererCtxKey)
	if r == nil {
		return nil, false
	}
	return r.(renderer.Renderer), true
}

// SetRenderer sets the goldmark renderer used to render Markdown. Should only
// be called once per file.
func SetRenderer(pc parser.Context, r renderer.Renderer) {
	pc.Set(rendererCtxKey, r)
}

var headingIDsCtxKey = parser.NewContextKey()

func HeadingIDs(pc parser.Context) map[string]struct{} {
	rawIDs := pc.Get(headingIDsCtxKey)
	if rawIDs == nil {
		rawIDs = make(map[string]struct{})
		pc.Set(headingIDsCtxKey, rawIDs)
	}
	return rawIDs.(map[string]struct{})
}
