//go:build unix

package scan

import (
	"io/fs"
	"syscall"
)

// diskUsage is the space a file takes on disk: its allocated 512-byte blocks,
// as du counts them. A sparse file, such as a virtual machine disk, takes far
// less than its size says.
func diskUsage(info fs.FileInfo) int64 {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return int64(st.Blocks) * 512
	}
	return info.Size()
}

// fileKey identifies a file across all its names.
type fileKey struct {
	dev uint64
	ino uint64
}

// hardLinkKey returns the file's identity when it has more than one hard link,
// so the scan can count its blocks once.
func hardLinkKey(info fs.FileInfo) (fileKey, bool) {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || st.Nlink < 2 {
		return fileKey{}, false
	}
	return fileKey{dev: uint64(st.Dev), ino: uint64(st.Ino)}, true
}
