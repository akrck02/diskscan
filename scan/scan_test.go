package scan

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

// writeFile creates a file of the given size, and the folders above it.
func writeFile(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
}

// sampleTree builds:
//
//	root/          a.bin (100)
//	root/big/      b.bin (1000), deep/c.bin (500)
//	root/small/    d.bin (10), e.bin (20)
//	root/empty/
func sampleTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.bin"), 100)
	writeFile(t, filepath.Join(root, "big", "b.bin"), 1000)
	writeFile(t, filepath.Join(root, "big", "deep", "c.bin"), 500)
	writeFile(t, filepath.Join(root, "small", "d.bin"), 10)
	writeFile(t, filepath.Join(root, "small", "e.bin"), 20)
	if err := os.Mkdir(filepath.Join(root, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestTreeAddsUpSizesRecursively(t *testing.T) {
	root := sampleTree(t)
	var directories atomic.Int64
	tree := Tree(root, &directories)

	if tree.Size != 1630 {
		t.Errorf("root size = %d, want 1630", tree.Size)
	}
	if tree.FilesSize != 100 || tree.FileCount != 1 {
		t.Errorf("root files = %d bytes in %d, want 100 in 1", tree.FilesSize, tree.FileCount)
	}
	big := Lookup(tree, filepath.Join(root, "big"))
	if big == nil || big.Size != 1500 || big.FilesSize != 1000 {
		t.Errorf("big = %+v, want size 1500 with 1000 of its own", big)
	}
	small := Lookup(tree, filepath.Join(root, "small"))
	if small == nil || small.Size != 30 || small.FileCount != 2 {
		t.Errorf("small = %+v, want size 30 in 2 files", small)
	}
	if got := directories.Load(); got != 5 { // root, big, deep, small, empty
		t.Errorf("directories counted = %d, want 5", got)
	}
}

func TestTreeAcceptsNoCounter(t *testing.T) {
	if tree := Tree(sampleTree(t), nil); tree.Size != 1630 {
		t.Errorf("size = %d, want 1630", tree.Size)
	}
}

func TestTreeSkipsSymbolicLinks(t *testing.T) {
	root := sampleTree(t)
	// A link to a folder would count big twice (or loop forever, pointing up);
	// a link to a file would count the file twice.
	if err := os.Symlink(filepath.Join(root, "big"), filepath.Join(root, "link-to-big")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(root, filepath.Join(root, "big", "link-up")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "a.bin"), filepath.Join(root, "link-to-a.bin")); err != nil {
		t.Fatal(err)
	}

	tree := Tree(root, nil)
	if tree.Size != 1630 {
		t.Errorf("size = %d, want 1630 (links not followed nor counted)", tree.Size)
	}
	if tree.FileCount != 1 {
		t.Errorf("root file count = %d, want 1 (the link is not a file)", tree.FileCount)
	}
}

func TestTreeSkipsUnreadableFolders(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads every folder")
	}
	root := sampleTree(t)
	locked := filepath.Join(root, "big", "deep")
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) }) // so TempDir can remove it

	tree := Tree(root, nil)
	if tree.Size != 1130 {
		t.Errorf("size = %d, want 1130 (the 500 behind the locked folder unseen)", tree.Size)
	}
	if Lookup(tree, locked) == nil {
		t.Error("the locked folder should still be listed, with nothing in it")
	}
}

func TestLookup(t *testing.T) {
	root := sampleTree(t)
	tree := Tree(root, nil)

	cases := []struct {
		name, path string
		found      bool
	}{
		{"the root itself", root, true},
		{"a child", filepath.Join(root, "big"), true},
		{"a grandchild", filepath.Join(root, "big", "deep"), true},
		{"a missing folder", filepath.Join(root, "nowhere"), false},
		{"a file, not a folder", filepath.Join(root, "a.bin"), false},
		{"the parent", filepath.Dir(root), false},
		{"somewhere else", "/", false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			node := Lookup(tree, test.path)
			if (node != nil) != test.found {
				t.Fatalf("Lookup(%q) found = %v, want %v", test.path, node != nil, test.found)
			}
			if node != nil && node.Path != test.path {
				t.Errorf("Lookup(%q) returned %q", test.path, node.Path)
			}
		})
	}
}

func TestEntriesAreBiggestFirstWithTheFilesBucket(t *testing.T) {
	tree := Tree(sampleTree(t), nil)
	entries := tree.Entries()

	want := []struct {
		name  string
		size  int64
		isDir bool
	}{
		{"big", 1500, true},
		{FilesBucketName, 100, false},
		{"small", 30, true},
		{"empty", 0, true},
	}
	if len(entries) != len(want) {
		t.Fatalf("got %d entries, want %d: %+v", len(entries), len(want), entries)
	}
	for index, expected := range want {
		entry := entries[index]
		if entry.Name != expected.name || entry.Size != expected.size || entry.IsDir != expected.isDir {
			t.Errorf("entry %d = %+v, want %+v", index, entry, expected)
		}
	}
	if bucket := entries[1]; bucket.NumFiles != 1 || bucket.Path != tree.Path {
		t.Errorf("files bucket = %+v, want 1 file at the folder's own path", bucket)
	}
}

func TestEntriesLeaveOutAnEmptyFilesBucket(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "only", "x.bin"), 5)
	for _, entry := range Tree(root, nil).Entries() {
		if !entry.IsDir {
			t.Errorf("a folder with no files of its own got a files bucket: %+v", entry)
		}
	}
}

func TestCrumbs(t *testing.T) {
	root := sampleTree(t)
	tree := Tree(root, nil)

	crumbs := Crumbs(tree, Lookup(tree, filepath.Join(root, "big", "deep")))
	want := []Crumb{
		{Name: filepath.Base(root), Path: root},
		{Name: "big", Path: filepath.Join(root, "big")},
		{Name: "deep", Path: filepath.Join(root, "big", "deep")},
	}
	if len(crumbs) != len(want) {
		t.Fatalf("crumbs = %+v, want %+v", crumbs, want)
	}
	for index := range want {
		if crumbs[index] != want[index] {
			t.Errorf("crumb %d = %+v, want %+v", index, crumbs[index], want[index])
		}
	}

	if atRoot := Crumbs(tree, tree); len(atRoot) != 1 || atRoot[0].Path != root {
		t.Errorf("crumbs at the root = %+v, want just the root", atRoot)
	}
}

func TestDisplayNameOfTheFilesystemRoot(t *testing.T) {
	if name := DisplayName(&Node{Path: "/"}); name != "/" {
		t.Errorf("DisplayName(/) = %q, want /", name)
	}
	if name := DisplayName(&Node{Path: "/Users/someone"}); name != "someone" {
		t.Errorf("DisplayName = %q, want someone", name)
	}
}

func TestSplitPath(t *testing.T) {
	cases := map[string][]string{
		".":     nil,
		"":      nil,
		"a":     {"a"},
		"a/b/c": {"a", "b", "c"},
	}
	for input, want := range cases {
		got := SplitPath(input)
		if len(got) != len(want) {
			t.Errorf("SplitPath(%q) = %q, want %q", input, got, want)
			continue
		}
		for index := range want {
			if got[index] != want[index] {
				t.Errorf("SplitPath(%q) = %q, want %q", input, got, want)
			}
		}
	}
}

func TestHumanBytes(t *testing.T) {
	cases := map[int64]string{
		0:                      "0 B",
		1023:                   "1023 B",
		1024:                   "1.0 KiB",
		1536:                   "1.5 KiB",
		1024 * 1024:            "1.0 MiB",
		5 * 1024 * 1024 * 1024: "5.0 GiB",
		3 << 40:                "3.0 TiB",
	}
	for bytes, want := range cases {
		if got := HumanBytes(bytes); got != want {
			t.Errorf("HumanBytes(%d) = %q, want %q", bytes, got, want)
		}
	}
}
