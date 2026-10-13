<div align="center">

# Disk scan

**Find out what is filling your disk.** Scan a folder, then click your way down through a sorted
list or a treemap.

[![CI](https://github.com/akrck02/diskscan/actions/workflows/ci.yml/badge.svg)](https://github.com/akrck02/diskscan/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/akrck02/diskscan?sort=semver)](https://github.com/akrck02/diskscan/releases/latest)
[![Go](https://img.shields.io/github/go-mod/go-version/akrck02/diskscan)](go.mod)
![Platforms](https://img.shields.io/badge/platforms-macOS%20%7C%20Linux-informational)

</div>

## 🤖 AI disclosure

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

## ✨ Two ways to use it

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

## 🚀 Quick start

Grab the archive for your system from the
[latest release](https://github.com/akrck02/diskscan/releases/latest) (macOS 13+ universal, or
Linux amd64 / arm64), then:

```bash
./diskscan-ui ~             # the desktop window
./diskscan -open ~          # or the web version, in your browser
```

On macOS, run `xattr -d com.apple.quarantine diskscan-ui` once first: the binaries are not signed
with an Apple Developer ID. [More on installing →](docs/install.md)

Or build it with Go 1.25+:

```bash
go build -o diskscan-ui ./cmd/diskscan-ui     # Linux: add -tags x11, see docs/building.md
```

## 🔍 How it works

<p align="center">
  <img alt="Your folder → scan/ walks the tree once → diskscan-ui (a desktop window) or diskscan (a local web page)" src="docs/images/how-it-works.svg" width="880">
</p>

One scanner walks the folder once, in parallel, and keeps only folders in memory. Both programs
read from it, so after the scan everything is instant.
[More on how it works →](docs/how-it-works.md)

## 📚 Documentation

| | |
|---|---|
| [Installing](docs/install.md) | Releases on macOS and Linux, and getting past Gatekeeper. |
| [Using diskscan](docs/usage.md) | Every button and flag, in the window and the web version. |
| [How it works](docs/how-it-works.md) | The scanner, the treemap layout, and what the sizes mean. |
| [Building and testing](docs/building.md) | Building on each system, the tests, the go-ui dependency. |
| [Releasing](docs/releasing.md) | Tags, the CI and release pipelines, macOS archives, screenshots. |
