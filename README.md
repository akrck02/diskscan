# diskscan

## AI disclosure

I write most of my code by hand, so I want to be clear about this project: **it was written by an
AI assistant (Claude, by Anthropic) under my direction.** I decided what to build and how it should
behave; the code, the tests and the documentation were written by the AI. Every commit made this
way carries a `Co-Authored-By: Claude` line.

**Status: pending thorough review.** The code compiles and its tests pass, but I have not yet
reviewed it line by line. Treat it as a working draft, not as code I vouch for.

## What it does

Your disk is full and you want to know why. Point `diskscan` at a folder, wait a few seconds while
it adds everything up, then click your way down to whatever is eating the space.

![The treemap view](docs/images/map-light.png)

There are two ways to use it, both built on the same scanner:

- **`diskscan-ui`**: a desktop window with a sorted list and a treemap.
- **`diskscan`**: a tiny web server, so you can browse the results in your browser.

It runs on macOS and Linux.

## Quick start

On Linux, grab the archive for your machine from
[Releases](https://github.com/akrck02/diskscan/releases):

```bash
tar -xzf diskscan-ui-v0.1.1-linux-amd64.tar.gz
./diskscan-ui ~
```

On macOS, or if you would rather build it yourself, you need Go 1.25 or newer:

```bash
git clone https://github.com/akrck02/diskscan.git
cd diskscan
go build -o diskscan-ui ./cmd/diskscan-ui     # on Linux, add: -tags x11 (see Building)
./diskscan-ui ~
```

## Using the window

```bash
./diskscan-ui [flags] [folder]      # folder defaults to the current one
```

The window opens straight away and scans in the background, with a counter in the status bar.
On a home folder with a few hundred thousand folders that takes around 15 seconds.

![The list view](docs/images/list-light.png)

- **Click a folder** to go into it. The path at the top takes you back up.
- **List / Map** switches between the sorted list and the treemap. In the treemap, the area of each
  rectangle is its size. Tiny items are lumped into one grey rectangle. Hover over any rectangle to
  see its full name and size.
- **Show in Finder** opens the current folder in your file manager.
- **Rescan** scans everything again, for example after you have deleted something.
- The sun/moon button switches between light and dark.

![The treemap in the dark theme](docs/images/map-dark.png)

| Flag | What it does |
|---|---|
| `-map` | Start on the treemap instead of the list. |
| `-dark` | Start in dark mode. |
| `-width N`, `-height N` | Window size. |
| `-capture file.png` | Scan, save one frame as a PNG and exit. Handy for screenshots and tests. |

## Using the web version

```bash
./diskscan [-port 8080] [-open] [folder]
```

It scans the folder, then serves the same views at `http://127.0.0.1:8080`. `-open` opens your
browser when the scan is done. It only listens on your own machine, never on the network.

![The web version](docs/images/web.png)

## Good to know

- **Sizes are file sizes, as `ls -l` shows them**, not the space they take on disk. They can differ
  for sparse files (virtual machine disks, some databases) and on compressed file systems.
- **Symbolic links are skipped**, so nothing is counted twice and link loops cannot trap the scan.
- **Folders you cannot read are skipped silently.** Run it with more permissions if you need
  everything counted.
- **The scan is a snapshot.** Use Rescan after you delete things.
- **Memory stays small**: only folders are kept, not individual files.

## Installing a release

| | Linux amd64 / arm64 | macOS |
|---|---|---|
| Window | `diskscan-ui-<version>-linux-<arch>.tar.gz` | build from source for now |
| Web | `diskscan-<version>-linux-<arch>.tar.gz` | build from source for now |

On Linux, the web version has no dependencies. The window needs OpenGL 4.1 and X11 (XWayland is
fine), which any desktop already has.

macOS binaries can be made locally with `scripts/release.sh` (see below). They are not signed with
an Apple Developer ID, so if you share them, the person downloading needs to run
`xattr -d com.apple.quarantine diskscan-ui` once (or right-click it and choose **Open**).

## Building

The web version is plain Go: `go build -o diskscan .` works anywhere.

The window draws with OpenGL (Metal on macOS) through cgo, so it needs a C compiler:

- **macOS:** the Xcode command line tools (`xcode-select --install`). Then `go build ./cmd/diskscan-ui`.
- **Linux:** a C compiler plus the X11 and OpenGL headers. On Debian or Ubuntu,
  `scripts/linux-deps.sh` installs them. Then build with the `x11` tag, which leaves out GLFW's
  Wayland support (and its extra dependencies):

  ```bash
  scripts/linux-deps.sh
  go build -tags x11 -o diskscan-ui ./cmd/diskscan-ui
  ```

Run the checks with:

```bash
go vet ./...
go test ./...      # on Linux: go test -tags x11 ./...
```

The tests do not open any window. They cover the scanner and the treemap layout.

### About the go-ui dependency

The window uses [go-ui](https://github.com/akrck02/go-ui) and
[go-renderer](https://github.com/akrck02/go-renderer), which are still in development, so `go.mod`
pins them to commits on their feature branches instead of tagged versions. There is also a
`replace` line for go-renderer, because go-ui points at a local copy of it. To update them:

```bash
go get github.com/akrck02/go-ui@<branch>
go mod edit -replace github.com/akrck02/go-renderer=github.com/akrck02/go-renderer@<commit>
```

Once go-renderer's `draw2d` package lands on `main`, the `replace` can go.

## Project layout

```
scan/              the scanner, shared by both programs (with tests)
cmd/diskscan-ui/   the desktop window: main.go (list, toolbar, scanning) and treemap.go
main.go            the web server
index.html         the web page it serves
scripts/           release.sh (macOS and Linux web archives), linux-deps.sh
.github/workflows/ CI and releases
```

## Releasing

Push a tag and GitHub Actions does the rest:

```bash
git tag -a v0.2.0 -m "diskscan v0.2.0"
git push origin v0.2.0
```

The [release workflow](.github/workflows/release.yml) runs the tests, builds both programs for
Linux on amd64 and arm64 machines, checks that the window really starts (it draws a frame on a
virtual display), and publishes a GitHub release with checksums.

macOS is not in the pipeline yet, because GitHub's macOS machines can take a long time to become
available. Build those archives locally with `scripts/release.sh v0.2.0` and attach them to the
release by hand.
