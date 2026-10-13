// Package scan walks a directory tree once into memory and answers questions
// about it: the size of every folder, recursively, and what is inside each one.
// Both front-ends use it: the web UI (the root command) and the desktop window
// (cmd/diskscan-ui).
//
// Only directories are stored (files are aggregated per folder), so memory stays
// bounded by the number of directories, not the number of files.
package scan

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
)

// Node is one directory in the scanned tree.
type Node struct {
	Name      string           // base name (or full path for the root)
	Path      string           // absolute path
	Size      int64            // total bytes, recursive (files + all subdirs)
	FilesSize int64            // bytes of files directly in this dir (non-recursive)
	FileCount int64            // number of files directly in this dir
	dirs      map[string]*Node // child directories, keyed by base name
	mu        sync.Mutex       // guards dirs/FilesSize/FileCount during the scan
}

// scanner walks the filesystem concurrently, bounded by a semaphore so we
// don't open an unbounded number of directories at once.
type scanner struct {
	sem      chan struct{}
	wg       sync.WaitGroup
	dirCount *atomic.Int64
}

func newScanner(dirCount *atomic.Int64) *scanner {
	limit := runtime.NumCPU() * 4
	if limit < 8 {
		limit = 8
	}
	return &scanner{sem: make(chan struct{}, limit), dirCount: dirCount}
}

// Tree walks the tree rooted at the absolute path abs and returns a fully-sized
// root node. dirCount, when not nil, is bumped once per directory read, so a
// caller on another goroutine can show progress while the walk runs.
func Tree(abs string, dirCount *atomic.Int64) *Node {
	if dirCount == nil {
		dirCount = new(atomic.Int64)
	}
	root := &Node{Name: filepath.Base(abs), Path: abs}
	s := newScanner(dirCount)
	s.wg.Add(1)
	s.scan(root)
	s.wg.Wait()
	finalize(root)
	return root
}

// scan builds the subtree rooted at n by reading its entries, adding up the
// sizes of direct files and recursing (concurrently) into subdirectories.
func (s *scanner) scan(n *Node) {
	defer s.wg.Done()
	s.dirCount.Add(1)

	entries, err := os.ReadDir(n.Path)
	if err != nil {
		return // unreadable dir (permissions, races) — skip, don't crash
	}

	for _, e := range entries {
		if e.IsDir() {
			// Skip symlinked dirs: ReadDir reports the link's type, and
			// following them risks double-counting or cycles.
			child := &Node{Name: e.Name(), Path: filepath.Join(n.Path, e.Name())}
			n.mu.Lock()
			if n.dirs == nil {
				n.dirs = make(map[string]*Node)
			}
			n.dirs[e.Name()] = child
			n.mu.Unlock()

			s.wg.Add(1)
			select {
			case s.sem <- struct{}{}:
				go func() {
					defer func() { <-s.sem }()
					s.scan(child)
				}()
			default:
				// Pool full: recurse inline rather than block a worker.
				s.scan(child)
			}
			continue
		}

		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			continue // don't count symlinked files
		}
		n.mu.Lock()
		n.FilesSize += info.Size()
		n.FileCount++
		n.mu.Unlock()
	}
}

// finalize computes recursive Size totals bottom-up after the walk is done.
func finalize(n *Node) int64 {
	total := n.FilesSize
	for _, d := range n.dirs {
		total += finalize(d)
	}
	n.Size = total
	return total
}

// Lookup finds the node for an absolute path within the tree.
func Lookup(root *Node, path string) *Node {
	if path == root.Path {
		return root
	}
	rel, err := filepath.Rel(root.Path, path)
	if err != nil || rel == "." || rel == ".." || filepath.IsAbs(rel) {
		return nil
	}
	cur := root
	for _, part := range SplitPath(rel) {
		next, ok := cur.dirs[part]
		if !ok {
			return nil
		}
		cur = next
	}
	return cur
}

// SplitPath breaks a relative path into its parts, outermost first.
func SplitPath(rel string) []string {
	var parts []string
	dir := rel
	for dir != "." && dir != string(filepath.Separator) && dir != "" {
		parts = append([]string{filepath.Base(dir)}, parts...)
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return parts
}

// Entry is one line of a folder's listing: a child directory, or the bucket of
// files that live directly in the folder.
type Entry struct {
	Name     string
	Path     string
	Size     int64
	IsDir    bool
	NumFiles int64 // only for the files bucket
}

// FilesBucketName names the synthetic entry for a folder's own files.
const FilesBucketName = "(files in this folder)"

// Entries lists n's child directories plus its files bucket, biggest first.
func (n *Node) Entries() []Entry {
	var entries []Entry
	for _, d := range n.dirs {
		entries = append(entries, Entry{Name: d.Name, Path: d.Path, Size: d.Size, IsDir: true})
	}
	if n.FilesSize > 0 {
		entries = append(entries, Entry{
			Name:     FilesBucketName,
			Path:     n.Path,
			Size:     n.FilesSize,
			NumFiles: n.FileCount,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Size > entries[j].Size
	})
	return entries
}

// Crumb is one step of the path from the root down to a folder.
type Crumb struct {
	Name string
	Path string
}

// Crumbs lists the folders from root down to n, root first.
func Crumbs(root, n *Node) []Crumb {
	crumbs := []Crumb{{Name: DisplayName(root), Path: root.Path}}
	rel, err := filepath.Rel(root.Path, n.Path)
	if err != nil || rel == "." {
		return crumbs
	}
	cur := root.Path
	for _, part := range SplitPath(rel) {
		cur = filepath.Join(cur, part)
		crumbs = append(crumbs, Crumb{Name: part, Path: cur})
	}
	return crumbs
}

// DisplayName is the root's short name, or its full path for "/".
func DisplayName(root *Node) string {
	base := filepath.Base(root.Path)
	if base == "/" || base == "." || base == "" {
		return root.Path
	}
	return base
}

// HumanBytes formats a byte count in binary units (KiB, MiB, …).
func HumanBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

// RevealInFinder shows the given path in the OS file manager, selected.
func RevealInFinder(path string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", "-R", path)
	case "windows":
		cmd = exec.Command("explorer", "/select,", path)
	default:
		cmd = exec.Command("xdg-open", filepath.Dir(path)) // open the parent folder
	}
	_ = cmd.Start()
}
