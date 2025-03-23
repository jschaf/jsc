package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jschaf/jsc/pkg/dirs"
	"github.com/jschaf/jsc/pkg/errs"
	"github.com/jschaf/jsc/pkg/git"
	"github.com/jschaf/jsc/pkg/livereload"
	"github.com/jschaf/jsc/pkg/log"
	"github.com/jschaf/jsc/pkg/net/srv"
	"github.com/jschaf/jsc/pkg/process"
	"github.com/jschaf/jsc/pkg/sites"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

const port = "2222"

// HTTPS server flags.
var (
	tlsCertPath = flag.String("tls-cert-path", "private/cert/localhost_cert.pem", "path to the TLS certificate file; if set, server uses https")
	tlsKeyPath  = flag.String("tls-key-path", "private/cert/localhost_key.pem", "path to the TLS key file; if set, server uses https")
)

type Server struct {
	// Lifecycle context.
	// Calling serverCancel causes all background goroutines to stop. To stop the
	// HTTP server, call Shutdown.
	serverCtx    context.Context
	serverCancel context.CancelFunc
	// TLS
	tlsCertPath string
	tlsKeyPath  string
	// Servers
	httpSrv       *http.Server
	liveReloadSrv *livereload.LiveReload
	// Locks
	mu sync.Mutex
}

type ServerOpts struct {
	DistDir     string
	Cancel      context.CancelFunc
	TLSCertPath string
	TLSKeyPath  string
}

func InitServer(ctx context.Context, opts ServerOpts) (*Server, error) {
	// Validate TLS.
	if (opts.TLSCertPath == "") != (opts.TLSKeyPath == "") {
		return nil, errors.New("tls-cert-path and tls-key-path must be set together")
	}
	if opts.TLSCertPath != "" {
		if _, err := os.Stat(opts.TLSCertPath); err != nil {
			return nil, fmt.Errorf("stat tls-cert-path: %w", err)
		}
		if _, err := os.Stat(opts.TLSKeyPath); err != nil {
			return nil, fmt.Errorf("stat tls-key-path: %w", err)
		}
	}

	if err := dirs.CleanDir(opts.DistDir); err != nil {
		return nil, fmt.Errorf("clean public dir: %w", err)
	}

	// Rebuild in case content changed since last run.
	if err := sites.Rebuild(opts.DistDir); err != nil {
		return nil, fmt.Errorf("rebuild site: %w", err)
	}

	// Live reload.
	lr := livereload.NewServer()
	go lr.Start(ctx)

	// File system watcher.
	watcher := NewFSWatcher(opts.DistDir, lr)
	root := git.RootDir()
	if err := watcher.watchDirs(
		filepath.Join(root, dirs.Cmd),
		filepath.Join(root, dirs.Pkg),
		filepath.Join(root, dirs.Posts),
		filepath.Join(root, dirs.Static),
		filepath.Join(root, dirs.Style),
		filepath.Join(root, dirs.TIL),
	); err != nil {
		return nil, fmt.Errorf("watch dirs: %w", err)
	}
	go func() {
		if err := watcher.Start(); err != nil {
			slog.Error("watcher error", "error", err)
			opts.Cancel()
		}
	}()

	// HTTP server.
	routeHandler := buildRoutes(buildRoutesOpts{
		distDir: opts.DistDir,
		lr:      lr,
	})
	h2s := &http2.Server{}
	httpSrv := &http.Server{
		Handler: h2c.NewHandler(routeHandler, h2s),
	}

	return &Server{
		serverCtx:    ctx,
		serverCancel: opts.Cancel,
		// TLS
		tlsCertPath: opts.TLSCertPath,
		tlsKeyPath:  opts.TLSKeyPath,
		// Servers
		httpSrv:       httpSrv,
		liveReloadSrv: lr,
		// Locks
		mu: sync.Mutex{},
	}, nil
}

func (s *Server) ListenAndServe() (mErr error) {
	if err := s.serverCtx.Err(); err != nil {
		return fmt.Errorf("server context error: %w", err)
	}

	ln, err := net.Listen("tcp", "0.0.0.0:"+port)
	if err != nil {
		return fmt.Errorf("listen to http port: %w", err)
	}
	defer errs.Capture(&mErr, srv.NewListenerCloser(ln), "close http listener")

	isTLS := s.tlsCertPath != ""
	url := "http://localhost:" + port
	if isTLS {
		url = "https://localhost:" + port
	}
	slog.Info("ready: dev server listening", slog.String("url", url), slog.Bool("tls", isTLS))

	// Serve
	if isTLS {
		err = s.httpSrv.ServeTLS(ln, s.tlsCertPath, s.tlsKeyPath)
	} else {
		err = s.httpSrv.Serve(ln)
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("prod http serve: %w", err)
	}
	slog.Info("dev server shut down")
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// First, gracefully shutdown the HTTP server.
	err := s.httpSrv.Shutdown(ctx)
	// Then cancel the server context, which should trigger everything else to
	// stop.
	s.serverCancel()
	return err
}

func main() {
	process.RunMain(runMain)
}

func runMain(ctx context.Context) (mErr error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	fset := flag.CommandLine
	logLevel := log.DefineFlags(fset)
	if err := fset.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	slog.SetDefault(slog.New(log.NewDevHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})))

	slog.Info("start dev server", "process.args", os.Args[1:])

	devSrv, err := InitServer(ctx, ServerOpts{
		DistDir:     dirs.Dist,
		Cancel:      cancel,
		TLSCertPath: *tlsCertPath,
		TLSKeyPath:  *tlsKeyPath,
	})
	if err != nil {
		return fmt.Errorf("init server: %w", err)
	}

	// If context is done, shutdown.
	go func() {
		<-ctx.Done()
		slog.Debug("stop dev server")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer shutdownCancel()
		err := devSrv.Shutdown(shutdownCtx)
		if err != nil {
			slog.Error("shutdown dev server", slog.String("error", err.Error()))
		}
	}()

	if err := devSrv.ListenAndServe(); err != nil {
		return fmt.Errorf("listen and serve dev server: %w", err)
	}

	return nil
}
