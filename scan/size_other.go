//go:build !unix

package scan

import "io/fs"

// diskUsage falls back to the file size where block counts are not available.
func diskUsage(info fs.FileInfo) int64 { return info.Size() }

type fileKey struct{}

// hardLinkKey reports no hard links where the scan cannot tell them apart.
func hardLinkKey(fs.FileInfo) (fileKey, bool) { return fileKey{}, false }
