package compiler

import (
	"bytes"
	"fmt"
	"github.com/jschaf/b2/pkg/dirs"
	"github.com/jschaf/b2/pkg/git"
	"github.com/jschaf/b2/pkg/markdown"
	"github.com/jschaf/b2/pkg/markdown/assets"
	"github.com/jschaf/b2/pkg/markdown/html"
	"github.com/jschaf/b2/pkg/markdown/mdext"
	"github.com/jschaf/b2/pkg/paths"
	"github.com/karrick/godirwalk"
	"go.uber.org/zap"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
)

// TILIndexCompiler compiles the /til/ path, an index of all TIL posts.
type TILPostCompiler struct {
	md     *markdown.Markdown
	l      *zap.SugaredLogger
	pubDir string
}

func NewForTILPost(pubDir string, l *zap.Logger) *TILPostCompiler {
	md := markdown.New(l)
	return &TILPostCompiler{md: md, pubDir: pubDir, l: l.Sugar()}
}

func (c *TILPostCompiler) parse(path string) (*markdown.PostAST, error) {
	c.l.Debugf("compiling til %s", path)
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open TIL post %s: %w", path, err)
	}
	src, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("read TIL filepath %s: %w", path, err)
	}
	ast, err := c.md.Parse(path, bytes.NewReader(src))
	if err != nil {
		return nil, fmt.Errorf("parse TIL markdown at path %s: %w", path, err)
	}
	return ast, nil
}

func (c *TILPostCompiler) createDestFile(ast *markdown.PostAST) (*os.File, error) {
	slug := ast.Meta.Slug
	if slug == "" {
		return nil, fmt.Errorf("empty slug for path: %s", ast.Path)
	}
	slugDir := filepath.Join(c.pubDir, dirs.TIL, slug)
	if err := os.MkdirAll(slugDir, 0755); err != nil {
		return nil, fmt.Errorf("make dir for slug %s: %w", slug, err)
	}
	dest := filepath.Join(slugDir, "index.html")
	destFile, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("create dest file %q for TIL post: %w", dest, err)
	}
	return destFile, nil
}

func (c *TILPostCompiler) compile(ast *markdown.PostAST, w io.Writer) error {
	if ast.Meta.Visibility != mdext.VisibilityPublished {
		return nil
	}
	if err := assets.CopyAll(c.pubDir, ast.Assets); err != nil {
		return err
	}
	b := new(bytes.Buffer)
	if err := c.md.Render(b, ast.Source, ast); err != nil {
		return fmt.Errorf("failed to markdown for index: %w", err)
	}
	data := html.TILPostTemplateData{
		Title:    "TIL - Joe Schafer's Blog",
		Content:  template.HTML(b.String()),
		Features: ast.Features,
	}
	if err := html.RenderTILPost(w, data); err != nil {
		return fmt.Errorf("render TIL: %w", err)
	}
	return nil
}

func (c *TILPostCompiler) CompileAll() error {
	tilDir := filepath.Join(git.MustFindRootDir(), dirs.TIL)
	err := paths.WalkConcurrent(tilDir, runtime.NumCPU(), func(path string, dirent *godirwalk.Dirent) error {
		if !dirent.IsRegular() || filepath.Ext(path) != ".md" {
			return nil
		}
		ast, err := c.parse(path)
		if err != nil {
			return fmt.Errorf("parse TIL post into AST at path %s: %w", path, err)
		}
		dest, err := c.createDestFile(ast)
		if err != nil {
			return err
		}
		if err := c.compile(ast, dest); err != nil {
			return fmt.Errorf("compile TIL post at path: %s: %w", path, err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("compile all TILs: %w", err)
	}
	return nil
}
