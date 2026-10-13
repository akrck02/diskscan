# 📦 Installing

Every [release](https://github.com/akrck02/diskscan/releases/latest) ships both programs for macOS
and Linux. Pick the window (`diskscan-ui`), the web version (`diskscan`), or both.

| | macOS 13+ (Apple Silicon and Intel) | Linux (amd64, arm64) |
|---|---|---|
| **Window** | `diskscan-ui-<version>-macos-universal.zip` | `diskscan-ui-<version>-linux-<arch>.tar.gz` |
| **Web** | `diskscan-<version>-macos-universal.zip` | `diskscan-<version>-linux-<arch>.tar.gz` |

Each archive holds a single binary. `checksums.txt` in the release has their SHA-256 sums.

## macOS

```bash
unzip diskscan-ui-v0.1.1-macos-universal.zip
xattr -d com.apple.quarantine diskscan-ui
./diskscan-ui ~
```

The macOS builds are universal binaries, so the same file runs on Apple Silicon and Intel Macs.

> [!IMPORTANT]
> The binaries are not signed with an Apple Developer ID, so the first time you open one macOS says
> it *"cannot be verified"* and refuses. The `xattr` line removes the quarantine flag the browser put
> on the download. Right-clicking the file in Finder and choosing **Open** does the same. If you
> would rather not do either, [build it yourself](building.md).

## Linux

```bash
tar -xzf diskscan-ui-v0.1.1-linux-amd64.tar.gz      # or linux-arm64
./diskscan-ui ~
```

- **`diskscan`** (web) is a static binary with no dependencies. It runs anywhere, servers included.
- **`diskscan-ui`** (window) needs OpenGL 4.1 and X11. Any desktop has them. On Wayland it runs
  through XWayland.

## Putting it on your PATH

Both programs are single files, so installing means copying them somewhere on your `PATH`:

```bash
sudo mv diskscan-ui diskscan /usr/local/bin/
```
