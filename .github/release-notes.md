A disk usage explorer: a native window (`diskscan-ui`, with list and treemap views) and the same
views as a local web page (`diskscan`, served on `127.0.0.1` only).

| | macOS 13+ (Apple Silicon + Intel) | Linux amd64 / arm64 |
|---|---|---|
| Window | `diskscan-ui-…-macos-universal.zip` | `diskscan-ui-…-linux-<arch>.tar.gz` |
| Web | `diskscan-…-macos-universal.zip` | `diskscan-…-linux-<arch>.tar.gz` |

**macOS:** the binaries are not signed with an Apple Developer ID, so the first time run
`xattr -d com.apple.quarantine diskscan-ui` (or right-click → Open in Finder).

**Linux:** the web version is a static binary. The window needs OpenGL 4.1 and X11 (or XWayland),
which any desktop has.

SHA-256 checksums are in `checksums.txt`. Written with an AI assistant and pending thorough
review — see the AI disclosure in the README.
