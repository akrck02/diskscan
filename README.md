<div align="center">

# diskscan

**Find out what is filling your disk.** Scan a folder, then click your way down through a sorted
list or a treemap.

[![CI](https://github.com/akrck02/diskscan/actions/workflows/ci.yml/badge.svg)](https://github.com/akrck02/diskscan/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/akrck02/diskscan?sort=semver)](https://github.com/akrck02/diskscan/releases/latest)
[![Go](https://img.shields.io/github/go-mod/go-version/akrck02/diskscan)](go.mod)
![Platforms](https://img.shields.io/badge/platforms-macOS%20%7C%20Linux-informational)

</div>

## AI disclosure

I write most of my code by hand, so I want to be clear about this project: **it was written by an
AI assistant (Claude, by Anthropic) under my direction.** I decided what to build and how it should
behave; the code, the tests and the documentation were written by the AI. Every commit made this
way carries a `Co-Authored-By: Claude` line.

**Status: pending thorough review.** The code compiles and its tests pass, but I have not yet
reviewed it line by line. Treat it as a working draft, not as code I vouch for.

---

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/images/map-dark.png">
  <img alt="The treemap: each folder is a rectangle whose area is its size" src="docs/images/map-light.png">
</picture>

## Two ways to use it

<table>
<tr>
<th width="33%">Sorted list</th>
<th width="33%">Treemap</th>
<th width="33%">In the browser</th>
</tr>
<tr>
<td>
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/images/list-dark.png">
  <img alt="The list view" src="docs/images/list-light.png">
</picture>
</td>
<td>
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/images/map-dark.png">
  <img alt="The treemap view" src="docs/images/map-light.png">
</picture>
</td>
<td>
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/images/web-dark.png">
  <img alt="The web version" src="docs/images/web-light.png">
</picture>
</td>
</tr>
<tr>
<td>Biggest first, with each folder's share. Click to go in.</td>
<td>Area is size. Small items are grouped into one grey tile.</td>
<td>The same views, served on <code>127.0.0.1</code>.</td>
</tr>
<tr>
<td colspan="2" align="center"><code>diskscan-ui</code>: a desktop window</td>
<td align="center"><code>diskscan</code>: a local web server</td>
</tr>
</table>

## Quick start

Download the archive for your system from the
[latest release](https://github.com/akrck02/diskscan/releases/latest):

| | 🍎 macOS 13+ (Apple Silicon and Intel) | 🐧 Linux (amd64, arm64) |
|---|---|---|
| **Window** | `diskscan-ui-<version>-macos-universal.zip` | `diskscan-ui-<version>-linux-<arch>.tar.gz` |
| **Web** | `diskscan-<version>-macos-universal.zip` | `diskscan-<version>-linux-<arch>.tar.gz` |

```bash
# macOS
unzip diskscan-ui-v0.1.1-macos-universal.zip
xattr -d com.apple.quarantine diskscan-ui      # first run only, see below
./diskscan-ui ~

# Linux
tar -xzf diskscan-ui-v0.1.1-linux-amd64.tar.gz
./diskscan-ui ~
```

> [!NOTE]
> **macOS** blocks the binaries the first time with *"cannot be verified"*, because they are not
> signed with an Apple Developer ID. The `xattr` line above (or right-click → **Open** in Finder)
> takes care of it. **Linux:** the web version has no dependencies; the window needs OpenGL 4.1 and
> X11 (XWayland is fine), which any desktop already has.

Prefer to build it? You need Go 1.25+ (see [Building](#building) for the window's C dependencies):

```bash
git clone https://github.com/akrck02/diskscan.git && cd diskscan
go build -o diskscan-ui ./cmd/diskscan-ui      # Linux: add -tags x11
./diskscan-ui ~
```

## Using it

### The window

```bash
./diskscan-ui [flags] [folder]      # folder defaults to the current one
```

The window opens straight away and scans in the background, with a counter in the status bar. A
home folder with a few hundred thousand folders takes around 15 seconds.

| In the window | |
|---|---|
| Click a folder | Go into it. The path at the top takes you back up. |
| **List** / **Map** | Switch between the sorted list and the treemap. Hover a tile to see its name and size. |
| **Show in Finder** | Open the current folder in your file manager. |
| **Rescan** | Scan again, for example after deleting something. |
| ☀️ / 🌙 | Light or dark theme. |

| Flag | |
|---|---|
| `-map` | Start on the treemap. |
| `-dark` | Start in dark mode. |
| `-width N`, `-height N` | Window size. |
| `-capture file.png` | Scan, save one frame as a PNG and exit. Handy for screenshots and tests. |

### The web version

```bash
./diskscan [-port 8080] [-open] [folder]
```

It scans the folder, then serves the views at `http://127.0.0.1:8080`. `-open` opens the browser
when the scan is done. It only listens on your own machine, never on the network.

## How it works

<p align="center">
  <img alt="Your folder → scan/ walks the tree once → diskscan-ui (a desktop window) or diskscan (a local web page)" src="docs/images/how-it-works.svg" width="880">
</p>

Both programs share one scanner. It walks the folder once, in parallel, and keeps only folders in
memory (each with the total of the files directly inside it), so a few million files cost no more
than the folders holding them. Everything after the scan is instant, because it never touches the
disk again until you rescan.

> [!TIP]
> **Good to know**
> - Sizes are file sizes as `ls -l` shows them, not the space they take on disk. The two differ
>   for sparse files (virtual machine disks, some databases) and on compressed file systems.
> - Symbolic links are skipped, so nothing is counted twice and link loops cannot trap the scan.
> - Folders you cannot read are skipped silently.
> - The scan is a snapshot: use **Rescan** after you delete things.

## Building

| | Web version | Window |
|---|---|---|
| **Needs** | Go 1.25+ | Go 1.25+ and a C compiler (cgo) |
| **macOS** | `go build -o diskscan .` | Xcode command line tools, then `go build ./cmd/diskscan-ui` |
| **Linux** | `go build -o diskscan .` | `scripts/linux-deps.sh` (Debian/Ubuntu), then `go build -tags x11 ./cmd/diskscan-ui` |

On Linux, the `x11` tag builds GLFW for X11 only, which leaves out Wayland and its extra
dependencies. Wayland desktops run the result through XWayland.

```bash
go vet ./...
go test ./...      # Linux: add -tags x11
```

The tests open no window: they cover the scanner and the treemap layout.

<details>
<summary><b>About the go-ui dependency</b></summary>

The window uses [go-ui](https://github.com/akrck02/go-ui) and
[go-renderer](https://github.com/akrck02/go-renderer), which are still in development, so `go.mod`
pins them to commits on their feature branches instead of tagged versions. There is also a
`replace` line for go-renderer, because go-ui points at a local copy of it. To update them:

```bash
go get github.com/akrck02/go-ui@<branch>
go mod edit -replace github.com/akrck02/go-renderer=github.com/akrck02/go-renderer@<commit>
```

Once go-renderer's `draw2d` package lands on `main`, the `replace` can go.

</details>

<details>
<summary><b>Project layout</b></summary>

```
scan/              the scanner, shared by both programs (with tests)
cmd/diskscan-ui/   the desktop window: main.go (list, toolbar, scanning) and treemap.go
main.go            the web server
index.html         the web page it serves
scripts/           release.sh, linux-deps.sh, screenshots.sh
docs/images/       the pictures in this README (scripts/screenshots.sh regenerates them)
.github/workflows/ CI and releases
```

</details>

<details>
<summary><b>Releasing</b></summary>

Push a tag and GitHub Actions does the Linux side:

```bash
git tag -a v0.2.0 -m "diskscan v0.2.0"
git push origin v0.2.0
```

The [release workflow](.github/workflows/release.yml) runs the tests, builds both programs on Linux
amd64 and arm64 machines, checks that the window really starts (it draws a frame on a virtual
display), and publishes a GitHub release with checksums.

macOS is not in the pipeline yet, because GitHub's macOS machines can take a long time to become
available. Build those archives on a Mac and attach them to the release:

```bash
scripts/release.sh v0.2.0
gh release upload v0.2.0 dist/*.zip
```

</details>
