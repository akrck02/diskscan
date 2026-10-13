# 🛠️ Building and testing

You need **Go 1.25 or newer**. The web version is plain Go. The window draws with Metal (macOS) or
OpenGL (Linux) through cgo, so it also needs a C compiler and, on Linux, some system headers.

| | Web version | Window |
|---|---|---|
| **Needs** | Go | Go and a C compiler (cgo) |
| **macOS** | `go build -o diskscan .` | `go build -o diskscan-ui ./cmd/diskscan-ui` |
| **Linux** | `go build -o diskscan .` | `go build -tags x11 -o diskscan-ui ./cmd/diskscan-ui` |

## macOS

Install the Xcode command line tools once, if you do not have them:

```bash
xcode-select --install
```

Then build both programs:

```bash
go build -o diskscan-ui ./cmd/diskscan-ui
go build -o diskscan .
```

## Linux

On Debian or Ubuntu, [`scripts/linux-deps.sh`](../scripts/linux-deps.sh) installs the X11 and
OpenGL headers GLFW needs, plus Xvfb and Mesa's software OpenGL for running the window without a
screen. Then build the window with the `x11` tag:

```bash
scripts/linux-deps.sh
go build -tags x11 -o diskscan-ui ./cmd/diskscan-ui
```

> [!NOTE]
> Without the tag, GLFW builds for both X11 and Wayland, which needs the Wayland development
> packages too. With `-tags x11` it builds for X11 only. Wayland desktops still run it through
> XWayland.

On other distributions, install a C compiler and the equivalents of `libgl1-mesa-dev`,
`libx11-dev`, `libxcursor-dev`, `libxrandr-dev`, `libxinerama-dev`, `libxi-dev` and
`libxxf86vm-dev`.

## Tests

```bash
go vet ./...
go test ./...      # Linux: add -tags x11
```

The tests never open a window:

- **`scan/`** builds small folder trees in a temporary directory and checks the sizes, symbolic
  links, unreadable folders, lookups, listings and size formatting.
- **`cmd/diskscan-ui/`** checks the treemap layout: areas in proportion to sizes, tiles covering the
  space, near-square tiles, and how small items are grouped.

To see the window without a screen, draw one frame to a PNG. CI does this on Linux:

```bash
./diskscan-ui -capture frame.png ~
# headless Linux:
LIBGL_ALWAYS_SOFTWARE=1 xvfb-run -a ./diskscan-ui -capture frame.png ~
```

## The go-ui dependency

The window uses [go-ui](https://github.com/akrck02/go-ui) and
[go-renderer](https://github.com/akrck02/go-renderer). Both are still in development, so `go.mod`
pins them to commits on their feature branches instead of tagged versions.

There is also a `replace` line for go-renderer. go-ui's own `go.mod` points go-renderer at a local
folder, and a dependency's `replace` is ignored by the projects that use it, so diskscan has to say
which go-renderer commit to use. To move to newer commits:

```bash
go get github.com/akrck02/go-ui@<branch>
go mod edit -replace github.com/akrck02/go-renderer=github.com/akrck02/go-renderer@<commit>
go mod tidy
```

Once go-renderer's `draw2d` package lands on its `main` branch with a tagged version, the `replace`
can go.
