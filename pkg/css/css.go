package css

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"github.com/jschaf/jsc/pkg/dirs"
	"github.com/jschaf/jsc/pkg/git"
	"github.com/karrick/godirwalk"

	"github.com/jschaf/jsc/pkg/paths"
)

// CopyAllCSS copies all CSS files into distDir/style.
func CopyAllCSS(distDir string) ([]string, error) {
	styleDir := filepath.Join(git.RootDir(), dirs.Style)
	destDir := filepath.Join(distDir, dirs.Style)
	if err := os.MkdirAll(filepath.Dir(destDir), 0o755); err != nil {
		return nil, fmt.Errorf("create public style dir: %w", err)
	}

	cssPaths, err := paths.WalkCollect(styleDir, func(path string, dirent fs.DirEntry) ([]string, error) {
		if !dirent.Type().IsRegular() || filepath.Ext(path) != ".css" {
			return nil, nil
		}
		rel, err := filepath.Rel(styleDir, path)
		if err != nil {
			return nil, fmt.Errorf("rel path for css: %w", err)
		}
		dest := filepath.Join(destDir, rel)
		if isSame, err := paths.CopyLazy(dest, path); err != nil {
			return nil, fmt.Errorf("copy lazy css file: %w", err)
		} else if isSame {
			return nil, nil
		}
		return []string{dest}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("copy css to public dir: %w", err)
	}
	return cssPaths, nil
}

// CopyAllFonts copies all font files into distDir/fonts.
func CopyAllFonts(distDir string) error {
	fontDir := filepath.Join(git.RootDir(), dirs.Style, dirs.Fonts)
	destDir := filepath.Join(distDir, dirs.Style, dirs.Fonts)
	if err := os.MkdirAll(filepath.Dir(destDir), 0o755); err != nil {
		return fmt.Errorf("create public font dir: %w", err)
	}

	cb := func(path string, dirent *godirwalk.Dirent) error {
		if !dirent.IsRegular() || filepath.Ext(path) != ".woff2" {
			return nil
		}
		rel, err := filepath.Rel(fontDir, path)
		if err != nil {
			return fmt.Errorf("rel path for font: %w", err)
		}
		dest := filepath.Join(destDir, rel)
		if _, err := paths.CopyLazy(dest, path); err != nil {
			return fmt.Errorf("copy lazy font file: %w", err)
		}
		return nil
	}
	err := paths.WalkConcurrent(fontDir, runtime.NumCPU(), cb)
	if err != nil {
		return fmt.Errorf("copy css to public dir: %w", err)
	}
	return nil
}
