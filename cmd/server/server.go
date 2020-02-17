package main

import (
	"fmt"
	"github.com/jschaf/b2/pkg/git"
	"github.com/jschaf/b2/pkg/livereload"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

func main() {
	port := "8080"

	liveReload := livereload.NewWebsocketServer()
	lrJSPath := "/dev/livereload.js"
	lrPath := "/dev/livereload"
	http.HandleFunc(lrJSPath, livereload.ServeJSHandler)
	http.HandleFunc(lrPath, liveReload.WebSocketHandler)
	go liveReload.Start()

	root, err := git.FindRootDir()
	if err != nil {
		log.Fatalf("failed to find root dir: %s", err)
	}
	pubDir := filepath.Join(root, "public")
	log.Printf("Serving dir %s", pubDir)
	pubDirHandler := http.FileServer(http.Dir(pubDir))

	lrScript := strings.Join([]string{
		fmt.Sprintf("<script defer src=%s?port=%s&path=%s type='application/javascript'>",
			lrJSPath, port, strings.TrimLeft(lrPath, "/")),
		"</script>",
	}, "")
	http.Handle("/", livereload.NewHTMLInjector(lrScript, pubDirHandler))

	watcher := NewFSWatcher(liveReload)
	if err = watcher.AddRecursively(pubDir); err != nil {
		log.Fatalf("failed to watch path %s: %s", pubDir, err)
	}

	styleDir := filepath.Join(root, "style")
	if err = watcher.AddRecursively(styleDir); err != nil {
		log.Fatalf("failed to watch path %s: %s", styleDir, err)
	}

	postsDir := filepath.Join(root, "posts")
	if err = watcher.AddRecursively(postsDir); err != nil {
		log.Fatalf("failed to watch path %s: %s", styleDir, err)
	}

	go watcher.Start()

	log.Printf("Serving at port %s", port)
	log.Fatal(http.ListenAndServe("127.0.0.1:"+port, nil))
}
