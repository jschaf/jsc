package main

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/jschaf/jsc/pkg/css"
	"github.com/jschaf/jsc/pkg/errs"
	"github.com/jschaf/jsc/pkg/git"
	"github.com/jschaf/jsc/pkg/livereload"
	"github.com/jschaf/jsc/pkg/sites"
	"github.com/jschaf/jsc/pkg/static"
)

// FSWatcher watches the filesystem for modifications and sends LiveReload
// commands to browser clients.
type FSWatcher struct {
	liveReload *livereload.LiveReload
	watcher    *fsnotify.Watcher
	distDir    string
	stopOnce   *sync.Once
	stopC      chan struct{}
}

func NewFSWatcher(distDir string, lr *livereload.LiveReload) *FSWatcher {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}
	return &FSWatcher{
		distDir:    distDir,
		liveReload: lr,
		watcher:    watcher,
		stopOnce:   &sync.Once{},
		stopC:      make(chan struct{}),
	}
}

func (f *FSWatcher) Start() (mErr error) {
	defer errs.Capture(&mErr, f.watcher.Close, "close FSWatcher")
	rootDir := git.RootDir()

	for {
		select {
		case <-f.stopC:
			return nil

		case event := <-f.watcher.Events:
			if strings.HasSuffix(event.Name, "~") {
				// Intellij temp file
				break
			}
			// Ignore everything except write calls.
			if event.Op&fsnotify.Write != fsnotify.Write {
				break
			}

			rel, err := filepath.Rel(rootDir, event.Name)
			if err != nil {
				slog.Info("get relative path", "error", err)
				break
			}

			switch {
			case rel == "style/main.css":
				f.reloadMainCSS()

			case strings.HasPrefix(rel, "static/"):
				slog.Info("static reload", "relative_path", rel)
				if err := static.CopyStaticFiles(f.distDir); err != nil {
					return fmt.Errorf("failed to copy static files: %w", err)
				}
				// Send the empty string which should reload all LiveReload
				// clients.
				f.liveReload.ReloadFile("")

			case filepath.Ext(rel) == ".md" || filepath.Ext(rel) == ".html":
				if err := f.compileReloadMd(); err != nil {
					return fmt.Errorf("failed to compiled changed markdown: %w", err)
				}
				f.liveReload.ReloadFile(event.Name)

			case strings.HasPrefix(rel, "pkg/markdown/html"):
				err := f.compileReloadMd()
				if err != nil {
					return fmt.Errorf("compile markdown for changed file %s: %w", rel, err)
				}
				f.liveReload.ReloadFile("")

			case strings.HasPrefix(rel, "pkg/markdown/"):
				// Skip recompiling since we don't have the server hot-reload
				// enabled.

			case filepath.Ext(rel) == ".go" && !strings.HasSuffix(rel, "_test.go"):
				// Rebuild the server to pick up any new changes.
				if err := f.rebuildServer(); err != nil {
					slog.Error("rebuild server", "error", err)
				}
			}
		case err := <-f.watcher.Errors:
			slog.Info("error", "error", err)
		}
	}
}

func (f *FSWatcher) Stop() {
	f.stopOnce.Do(func() {
		close(f.stopC)
	})
}

func (f *FSWatcher) compileMdWithGoRun() error {
	slog.Info("pkg/markdown changed, compiling all markdown")
	cmd := exec.Command("go", "run", "github.com/jschaf/jsc/cmd/compiler")
	buf := &bytes.Buffer{}
	cmd.Stdout = buf
	cmd.Stderr = buf
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start go run compiler: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("go run compiler failed: %w\n%s", err, buf.String())
	}
	return nil
}

func (f *FSWatcher) watchDirs(dirs ...string) error {
	for _, dir := range dirs {
		if err := f.AddRecursively(dir); err != nil {
			return fmt.Errorf("failed to watch dir: %w", err)
		}
	}
	return nil
}

func (f *FSWatcher) compileReloadMd() error {
	if err := sites.Rebuild(f.distDir); err != nil {
		return fmt.Errorf("rebuild for changed md: %w", err)
	}
	return nil
}

func (f *FSWatcher) reloadMainCSS() {
	stylePaths, err := css.CopyAllCSS(f.distDir)
	if err != nil {
		slog.Info("copy all css", "error", err)
	}
	for _, stylePath := range stylePaths {
		f.liveReload.ReloadFile(stylePath)
	}
}

func (f *FSWatcher) AddRecursively(name string) error {
	walk := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}

		err = f.watcher.Add(path)
		if err != nil {
			return fmt.Errorf("failed to watch directory: %w", err)
		}
		return nil
	}

	return filepath.Walk(name, walk)
}

func (f *FSWatcher) rebuildServer() error {
	slog.Info("hot swapping server because go file changed")
	out := os.Args[0]
	pkg := "github.com/jschaf/jsc/cmd/server"
	cmd := exec.Command("go", "build", "-o", out, pkg)
	buf := &bytes.Buffer{}
	cmd.Stdout = buf
	cmd.Stderr = buf
	now := time.Now()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start server rebuild: %w\n%s", err, buf.String())
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("wait for server rebuild: %w\n%s", err, buf.String())
	}
	slog.Info("completed server rebuild", "duration", time.Since(now))
	slog.Debug("sending SIGHUP")
	if err := sendSighup(); err != nil {
		return err
	}
	return nil
}

func sendSighup() error {
	pid := os.Getpid()
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to get process from PID: %w", err)
	}
	if err = process.Signal(syscall.SIGHUP); err != nil {
		return fmt.Errorf("failed to send SIGHUP: %w", err)
	}
	return nil
}
