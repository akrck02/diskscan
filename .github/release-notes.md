`diskscan` shows what is filling your disk: scan a folder, then click through it as a sorted list
or a treemap. It comes as a desktop window (`diskscan-ui`) and as a local web page (`diskscan`).

| | macOS 13+ (Apple Silicon and Intel) | Linux (amd64, arm64) |
|---|---|---|
| **Window** | `diskscan-ui-<version>-macos-universal.zip` | `diskscan-ui-<version>-linux-<arch>.tar.gz` |
| **Web** | `diskscan-<version>-macos-universal.zip` | `diskscan-<version>-linux-<arch>.tar.gz` |

- **macOS:** the binaries are not signed with an Apple Developer ID, so on first run use
  `xattr -d com.apple.quarantine diskscan-ui` (or right-click → Open in Finder).
- **Linux:** the web version is a static binary; the window needs OpenGL 4.1 and X11 (XWayland
  works), which any desktop has.

Checksums are in `checksums.txt`. This project was written with an AI assistant and is pending
review; see the README.
