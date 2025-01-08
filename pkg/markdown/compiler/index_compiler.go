package compiler

import (
	"bytes"
	"fmt"
	"github.com/jschaf/jsc/pkg/errs"
	"github.com/jschaf/jsc/render/sitemaps"
	"html/template"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"

	"github.com/jschaf/jsc/pkg/dirs"
	"github.com/jschaf/jsc/pkg/markdown"
	"github.com/jschaf/jsc/pkg/markdown/html"
	"github.com/jschaf/jsc/pkg/markdown/mdctx"
	"github.com/jschaf/jsc/pkg/markdown/mdext"
	"github.com/jschaf/jsc/pkg/paths"
)

const rootURL = "https://joe.schafer.dev"

// IndexCompiler compiles the / path, the main homepage.
type IndexCompiler struct {
	md      *markdown.Markdown
	distDir string
}

func NewIndexCompiler(distDir string) *IndexCompiler {
	md := markdown.New(markdown.WithExtender(mdext.NewContinueReadingExt()))
	return &IndexCompiler{md: md, distDir: distDir}
}

func (ic *IndexCompiler) parseDirs(dirs ...string) ([]*markdown.AST, error) {
	asts := make([]*markdown.AST, 0, len(dirs)*8)
	for _, dir := range dirs {
		as, err := ic.collectASTs(dir)
		if err != nil {
			return nil, fmt.Errorf("collectASTs for dir %s: %w", dir, err)
		}
		asts = append(asts, as...)
	}
	sort.Slice(asts, func(i, j int) bool {
		return asts[i].Meta.Date.After(asts[j].Meta.Date)
	})
	return asts, nil
}

func (ic *IndexCompiler) collectASTs(dir string) ([]*markdown.AST, error) {
	asts, err := paths.WalkCollect(dir, func(path string, dirent fs.DirEntry) ([]*markdown.AST, error) {
		if !dirent.Type().IsRegular() || filepath.Ext(path) != ".md" {
			return nil, nil
		}
		slog.Debug("compiling for index", "path", path)
		bs, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read post at path %s: %w", path, err)
		}
		ast, err := ic.md.Parse(path, bytes.NewReader(bs))
		if err != nil {
			return nil, fmt.Errorf("parseFile markdown for root index: %w", err)
		}
		return []*markdown.AST{ast}, nil
	})
	return asts, err
}

func (ic *IndexCompiler) renderASTs(asts []*markdown.AST) ([]html.IndexPostParams, error) {
	posts := make([]html.IndexPostParams, 0, len(asts))
	for _, ast := range asts {
		if ast.Meta.Visibility != mdext.VisibilityPublished {
			continue
		}
		b := new(bytes.Buffer)
		if err := ic.md.Render(b, ast.Source, ast); err != nil {
			return nil, fmt.Errorf("render markdown for index: %w", err)
		}
		titleHTML, err := ic.renderTitle(ast)
		if err != nil {
			return nil, fmt.Errorf("render index title: %w", err)
		}
		posts = append(posts, html.IndexPostParams{
			Title:     ast.Meta.Title,
			TitleHTML: titleHTML,
			Slug:      ast.Meta.Slug,
			Date:      ast.Meta.Date,
			Body:      template.HTML(b.String()),
		})
	}
	sort.Slice(posts, func(i, j int) bool { return posts[i].Date.After(posts[j].Date) })
	return posts, nil
}

func (ic *IndexCompiler) renderTitle(ast *markdown.AST) (template.HTML, error) {
	b := new(bytes.Buffer)
	r := ic.md.Renderer()

	// Don't render the element, which is a link. The gohtml chooses how to
	// build the link.
	for c := ast.Meta.TitleNode.FirstChild(); c != nil; c = c.NextSibling() {
		err := r.Render(b, ast.Source, c)
		if err != nil {
			return "", fmt.Errorf("render title node child: %w", err)
		}
	}

	return template.HTML(b.String()), nil
}

func (ic *IndexCompiler) Compile() error {
	asts, err := ic.parseDirs(dirs.Posts, dirs.TIL)
	if err != nil {
		return err
	}

	featureSet := mdctx.NewFeatureSet()
	for _, ast := range asts {
		featureSet.AddAll(ast.Features)
	}
	featureSet.Add(mdctx.FeatureKatex)

	posts, err := ic.renderASTs(asts)
	if err != nil {
		return fmt.Errorf("compileAST asts for index: %w", err)
	}

	if err := os.MkdirAll(ic.distDir, 0o755); err != nil {
		return fmt.Errorf("make dir for index: %w", err)
	}
	dest := filepath.Join(ic.distDir, "index.html")

	destFile, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("open index.html file for write: %w", err)
	}
	data := html.IndexParams{
		Title:    "Joe Schafer's Blog",
		Posts:    posts,
		Features: featureSet,
	}
	if err := html.RenderIndex(destFile, data); err != nil {
		return fmt.Errorf("execute index template: %w", err)
	}

	err = writeSitemap(ic.distDir, asts)
	if err != nil {
		return fmt.Errorf("write sitemap: %w", err)
	}

	return nil
}

func writeSitemap(distDir string, ast []*markdown.AST) (mErr error) {
	destFile, err := os.OpenFile(filepath.Join(distDir, "sitemap.xml"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("open sitemap.xml file for write: %w", err)
	}
	defer errs.Capture(&mErr, destFile.Close, "close sitemap.xml file")

	sitemap := sitemaps.New()
	for _, a := range ast {
		if a.Meta.Visibility != "published" {
			continue
		}
		url := sitemaps.URL{
			Loc:        rootURL + "/" + a.Meta.Slug,
			LastMod:    a.Meta.Date,
			ChangeFreq: "monthly",
		}
		sitemap.Add(url)
	}
	sm, err := sitemap.Build()
	if err != nil {
		return fmt.Errorf("build sitemap: %w", err)
	}
	if _, err := destFile.WriteString(sm); err != nil {
		return fmt.Errorf("write sitemap.xml: %w", err)
	}
	return nil
}
