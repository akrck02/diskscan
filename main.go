// Command diskscan scans a directory tree, then serves an interactive
// web UI for exploring which folders use the most space.
//
// Usage:
//
//	diskscan [flags] [PATH]
//
//	PATH   directory to scan (default: current directory)
//
// Flags:
//
//	-port N     HTTP port to serve on (default 8080)
//	-open       open the UI in your default browser when ready
//
// The whole tree is scanned once into memory at startup, so drilling
// through folders in the UI is instant. Only directories are stored
// (files are aggregated per folder), so memory stays bounded by the
// number of directories, not the number of files.
package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"diskscan/scan"
)

//go:embed index.html
var assets embed.FS

// --- JSON payloads for the UI ------------------------------------------------

type childJSON struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	IsDir    bool   `json:"isDir"`
	NumFiles int64  `json:"numFiles,omitempty"`
}

type crumbJSON struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type viewJSON struct {
	Path     string      `json:"path"`
	Size     int64       `json:"size"`
	Crumbs   []crumbJSON `json:"crumbs"`
	Children []childJSON `json:"children"`
}

func buildView(root, n *scan.Node) viewJSON {
	v := viewJSON{Path: n.Path, Size: n.Size}
	for _, c := range scan.Crumbs(root, n) {
		v.Crumbs = append(v.Crumbs, crumbJSON{Name: c.Name, Path: c.Path})
	}
	for _, e := range n.Entries() {
		v.Children = append(v.Children, childJSON{Name: e.Name, Path: e.Path, Size: e.Size, IsDir: e.IsDir, NumFiles: e.NumFiles})
	}
	return v
}

// scanTree walks the tree rooted at abs and returns a fully-sized root node,
// printing a live progress line to stdout. Used at startup and on rescans.
func scanTree(abs string) *scan.Node {
	var dirCount atomic.Int64
	done := make(chan struct{})
	start := time.Now()
	go func() {
		ticker := time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				fmt.Printf("\r  scanning… %d directories", dirCount.Load())
			}
		}
	}()
	root := scan.Tree(abs, &dirCount)
	close(done)
	fmt.Printf("\r  scanned %d directories in %s — total %s%s\n",
		dirCount.Load(), time.Since(start).Round(time.Millisecond),
		scan.HumanBytes(root.Size), "                ")
	return root
}

func main() {
	port := flag.Int("port", 8080, "HTTP port to serve on")
	open := flag.Bool("open", false, "open the UI in your browser when ready")
	flag.Parse()

	target := "."
	if flag.NArg() > 0 {
		target = flag.Arg(0)
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		log.Fatalf("resolving path: %v", err)
	}
	fi, err := os.Stat(abs)
	if err != nil || !fi.IsDir() {
		log.Fatalf("%q is not a directory", abs)
	}

	// --- Scan -----------------------------------------------------------
	// The current tree lives behind an atomic pointer so a rescan can swap in
	// a fresh tree without locking the read handlers.
	var rootPtr atomic.Pointer[scan.Node]
	rootPtr.Store(scanTree(abs))
	var rescanMu sync.Mutex // serialize rescans (one walk of the disk at a time)

	// --- Serve ----------------------------------------------------------
	mux := http.NewServeMux()
	mux.HandleFunc("/api/view", func(w http.ResponseWriter, r *http.Request) {
		root := rootPtr.Load()
		p := r.URL.Query().Get("path")
		if p == "" {
			p = root.Path
		}
		n := scan.Lookup(root, p)
		if n == nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(buildView(root, n))
	})
	mux.HandleFunc("/api/reveal", func(w http.ResponseWriter, r *http.Request) {
		root := rootPtr.Load()
		p := r.URL.Query().Get("path")
		// Only reveal a path that exists in the scanned tree — this bounds
		// the endpoint to real subfolders and blocks arbitrary paths.
		if p == "" || scan.Lookup(root, p) == nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		scan.RevealInFinder(p)
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/api/rescan", func(w http.ResponseWriter, r *http.Request) {
		rescanMu.Lock()
		defer rescanMu.Unlock()
		fresh := scanTree(abs)
		rootPtr.Store(fresh)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"path": fresh.Path,
			"size": fresh.Size,
		})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		b, _ := assets.ReadFile("index.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(b)
	})

	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("cannot listen on %s: %v", addr, err)
	}
	url := fmt.Sprintf("http://%s/", addr)
	fmt.Printf("  serving on %s  (Ctrl-C to stop)\n", url)
	if *open {
		openBrowser(url)
	}

	srv := &http.Server{Handler: mux}
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
