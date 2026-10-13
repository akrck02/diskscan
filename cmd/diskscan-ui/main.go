// Command diskscan-ui is the desktop version of diskscan: it scans a directory
// tree and shows, in a native window drawn by go-ui, which folders use the most
// space, as a list or as a map of rectangles. Click a folder to go into it, a
// crumb at the top to go back up.
//
// Usage:
//
//	diskscan-ui [flags] [PATH]
//
//	PATH   directory to scan (default: current directory)
//
// Flags: the window's own (-dark, -width, -height, -capture shot.png, …).
// With -capture the scan finishes before the one frame is drawn.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"diskscan/scan"

	"github.com/akrck02/go-renderer/draw2d"
	ui "github.com/akrck02/go-ui"
	"github.com/akrck02/go-ui/window"
)

// rowHeight is the height of one line of the listing.
const rowHeight = 40

// barWidth is the width of the bar that shows each entry's share of the folder.
const barWidth = 160

// app is the window's state. The scan runs on its own goroutine, so the tree
// and the scan's progress are atomics the drawing reads every frame.
type app struct {
	path        string // the absolute directory scanned
	tree        atomic.Pointer[scan.Node]
	directories atomic.Int64 // directories read by the scan in progress
	scanning    atomic.Bool
	lastScan    atomic.Pointer[string] // the status line once a scan has finished
	rescans     sync.Mutex             // one walk of the disk at a time

	current string       // the folder on show
	entries []scan.Entry // its contents, listed once per folder and per scan
	listed  *scan.Node   // the node entries were listed from (a rescan makes new nodes)
	started bool         // the first scan has been asked for
	asMap   bool         // show the folder as a map of rectangles instead of a list
}

func main() {
	options := window.Options{Title: "Disk scan", Width: 1100, Height: 720, MinWidth: 720, MinHeight: 420}
	options.RegisterFlags(flag.CommandLine)
	asMap := flag.Bool("map", false, "start with the map instead of the list")
	flag.Parse()

	path, err := directoryToScan()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	application := &app{path: path, current: path, asMap: *asMap}
	if options.CapturePath != "" {
		application.rescan() // a capture is a single frame: show the finished scan in it
		application.started = true
	}
	if err := window.Run(options, application.draw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// directoryToScan is the command's argument (or the current directory), made absolute.
func directoryToScan() (string, error) {
	target := "."
	if flag.NArg() > 0 {
		target = flag.Arg(0)
	}
	path, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("%q is not a directory", path)
	}
	return path, nil
}

// startScan walks the disk in the background, waking the window a few times a
// second so the progress shows, and once more when the walk is done.
func (app *app) startScan() {
	if !app.scanning.CompareAndSwap(false, true) {
		return // already scanning
	}
	go func() {
		done := make(chan struct{})
		go wakeUntil(done)
		app.rescan()
		close(done)
		window.Wake()
	}()
}

// wakeUntil redraws the window every so often until done closes.
func wakeUntil(done chan struct{}) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			window.Wake()
		}
	}
}

// rescan walks the whole tree again and swaps it in.
func (app *app) rescan() {
	app.rescans.Lock()
	defer app.rescans.Unlock()
	app.scanning.Store(true)
	defer app.scanning.Store(false)

	app.directories.Store(0)
	start := time.Now()
	tree := scan.Tree(app.path, &app.directories)
	status := fmt.Sprintf("Scanned %d directories in %s", app.directories.Load(), roundDuration(time.Since(start)))
	app.lastScan.Store(&status)
	app.tree.Store(tree)
}

// roundDuration keeps a duration short to read: milliseconds, or microseconds
// for a scan so small it would otherwise read "0s".
func roundDuration(duration time.Duration) time.Duration {
	if duration < time.Millisecond {
		return duration.Round(time.Microsecond)
	}
	return duration.Round(time.Millisecond)
}

// currentNode is the folder on show in the latest tree. A rescan can lose the
// folder (deleted in the meantime): then the listing goes back to the root.
func (app *app) currentNode(tree *scan.Node) *scan.Node {
	node := scan.Lookup(tree, app.current)
	if node != nil {
		return node
	}
	app.current = tree.Path
	return tree
}

// open shows another folder.
func (app *app) open(path string) {
	app.current = path
}

// draw describes the whole window, top to bottom.
func (app *app) draw(context *ui.Context) {
	if !app.started {
		// The first scan starts here, not in main: it wakes the window, which
		// only works once the window exists.
		app.started = true
		app.startScan()
	}
	context.Box(ui.Layout{Direction: ui.Vertical, Width: ui.Grow(), Height: ui.Grow()}, func() {
		tree := app.tree.Load()
		app.toolbar(context, tree)
		context.Separator()
		if tree == nil {
			app.scanningPlaceholder(context)
		} else {
			app.listing(context, tree)
		}
		context.Separator()
		app.statusBar(context)
	})
}

// bar is a full-width strip painted with the surface colour.
func bar(context *ui.Context, height, padding float32, content func()) {
	layout := ui.Layout{Width: ui.Grow(), Height: ui.Fixed(height), Padding: ui.PadXY(padding, 0), Gap: 2, AlignY: ui.Center}
	context.Frame(layout, ui.Look{Fill: context.Theme.Surface}, content)
}

func gap(context *ui.Context, pixels float32) { context.Box(ui.Layout{Width: ui.Fixed(pixels)}, nil) }

// toolbar holds the way back up (the crumbs) and the actions on the folder.
func (app *app) toolbar(context *ui.Context, tree *scan.Node) {
	bar(context, 44, 8, func() {
		context.IconSized("hard-drive", 18, context.Theme.Primary)
		gap(context, 8)
		if tree != nil {
			app.crumbs(context, tree)
		}
		context.Spacer()
		if context.Selectable("list", "List", !app.asMap) {
			app.asMap = false
		}
		if context.Selectable("layout-grid", "Map", app.asMap) {
			app.asMap = true
		}
		gap(context, 12)
		if tree != nil {
			if context.FlatIconButton("folder-open", "Show in Finder") {
				scan.RevealInFinder(app.currentNode(tree).Path)
			}
			context.Tooltip("Open this folder in the file manager")
			gap(context, 4)
		}
		app.rescanButton(context)
		gap(context, 4)
		app.themeButton(context)
	})
}

// crumbs is the path from the scanned folder down to the one on show; each
// step but the last goes back to that folder.
func (app *app) crumbs(context *ui.Context, tree *scan.Node) {
	crumbs := scan.Crumbs(tree, app.currentNode(tree))
	if len(crumbs) > 1 {
		if context.FlatIconButton("arrow-up", "") {
			app.open(crumbs[len(crumbs)-2].Path)
		}
		context.Tooltip("Go up one folder")
	}
	for index, crumb := range crumbs {
		if index > 0 {
			context.IconSized("chevron-right", 14, context.Theme.InkQuiet)
		}
		if index == len(crumbs)-1 {
			gap(context, 6)
			context.Text(crumb.Name, ui.TextStyle{Weight: draw2d.Bold})
			continue
		}
		if context.FlatButton(crumb.Name + "##crumb" + crumb.Path) {
			app.open(crumb.Path)
		}
	}
}

func (app *app) rescanButton(context *ui.Context) {
	if app.scanning.Load() {
		context.FlatIconButton("loader", "Scanning…")
		return
	}
	if context.PrimaryIconButton("refresh-cw", "Rescan") {
		app.startScan()
	}
	context.Tooltip("Scan the disk again from scratch")
}

// themeButton switches between the light and the dark theme.
func (app *app) themeButton(context *ui.Context) {
	dark := context.Theme.Background == ui.DarkTheme().Background
	icon, hint, next := "moon", "Dark theme", ui.DarkTheme()
	if dark {
		icon, hint, next = "sun", "Light theme", ui.LightTheme()
	}
	if context.FlatIconButton(icon, "") {
		context.Theme = next
	}
	context.Tooltip(hint)
}

// scanningPlaceholder fills the window while the first scan runs.
func (app *app) scanningPlaceholder(context *ui.Context) {
	layout := ui.Layout{Direction: ui.Vertical, Width: ui.Grow(), Height: ui.Grow(), Gap: 8, AlignX: ui.Center, AlignY: ui.Center}
	context.Box(layout, func() {
		context.IconSized("loader", 28, context.Theme.Primary)
		context.Heading("Scanning " + app.path)
		context.Caption(fmt.Sprintf("%d directories so far", app.directories.Load()))
	})
}

// columns are the listing's column widths: the name takes what is left (Share,
// not Grow, so the header and every row split it the same way).
var columns = []ui.Sizing{ui.Share(1), ui.Fixed(barWidth + 56), ui.Fixed(96), ui.Fixed(96)}

// listing is the folder's total and its contents, biggest first.
func (app *app) listing(context *ui.Context, tree *scan.Node) {
	node := app.currentNode(tree)
	if app.listed != node {
		app.entries = node.Entries()
		app.listed = node
	}
	theme := context.Theme
	layout := ui.Layout{Direction: ui.Vertical, Width: ui.Grow(), Height: ui.Grow(), Padding: ui.Pad(20), Gap: 14}
	context.Box(layout, func() {
		context.Row(func() {
			context.Text(scan.HumanBytes(node.Size), ui.TextStyle{Size: 24, Weight: draw2d.Bold})
			context.Caption(fmt.Sprintf("in %s", node.Path))
		})
		if app.asMap {
			app.treemap(context, node)
			return
		}
		look := ui.Look{Fill: theme.Surface, Border: theme.Line, BorderWidth: 1, Radius: 10, Clip: true}
		context.Frame(ui.Layout{Direction: ui.Vertical, Width: ui.Grow(), Height: ui.Grow()}, look, func() {
			header := ui.Layout{Width: ui.Grow(), Height: ui.Fixed(32), Padding: ui.PadXY(16, 0), Gap: 12, AlignY: ui.Center}
			context.Frame(header, ui.Look{Fill: theme.Sunken}, func() {
				for index, title := range []string{"NAME", "SHARE", "SIZE", "FILES"} {
					cell(context, columns[index], alignOf(index), func() { context.Caption(title) })
				}
			})
			context.Separator()
			if len(app.entries) == 0 {
				empty := ui.Layout{Direction: ui.Vertical, Width: ui.Grow(), Padding: ui.Pad(24), AlignX: ui.Center}
				context.Box(empty, func() { context.Caption("This folder is empty") })
				return
			}
			// each folder keeps its own scroll position; a List builds only the rows in sight
			context.Push(node.Path)
			context.List("rows", ui.Layout{Width: ui.Grow(), Height: ui.Grow()}, len(app.entries), rowHeight, func(row int) {
				app.entryRow(context, app.entries[row], node.Size)
			})
			context.Pop()
		})
	})
}

// alignOf puts numbers on the right and words on the left.
func alignOf(column int) ui.Align {
	if column >= 2 {
		return ui.End
	}
	return ui.Start
}

// cell is one column of a row. It clips, so a name too long ends in "…".
func cell(context *ui.Context, width ui.Sizing, align ui.Align, content func()) {
	context.Frame(ui.Layout{Width: width, AlignX: align, AlignY: ui.Center, Gap: 8}, ui.Look{Clip: true}, content)
}

// entryRow is one folder (click to go into it) or the folder's own files.
func (app *app) entryRow(context *ui.Context, entry scan.Entry, total int64) {
	theme := context.Theme
	looks := ui.Looks{Hovered: ui.Look{Fill: theme.Sunken}, Pressed: ui.Look{Fill: theme.Line}}
	if !entry.IsDir {
		looks = ui.Looks{} // the files bucket leads nowhere
	}
	layout := ui.Layout{Width: ui.Grow(), Height: ui.Fixed(rowHeight), Padding: ui.PadXY(16, 0), Gap: 12, AlignY: ui.Center}
	clicked := context.Clickable(entry.Name+"##row", layout, looks, func() {
		cell(context, columns[0], ui.Start, func() {
			icon, colour := "folder", theme.Primary
			if !entry.IsDir {
				icon, colour = "files", theme.InkQuiet
			}
			context.IconSized(icon, 16, colour)
			style := ui.TextStyle{Weight: draw2d.Medium}
			if !entry.IsDir {
				style = ui.TextStyle{Color: theme.InkSoft}
			}
			context.Text(entry.Name, style)
		})
		cell(context, columns[1], ui.Start, func() { shareBar(context, entry.Size, total) })
		cell(context, columns[2], ui.End, func() { context.Label(scan.HumanBytes(entry.Size)) })
		cell(context, columns[3], ui.End, func() {
			if entry.NumFiles > 0 {
				context.Text(fmt.Sprint(entry.NumFiles), ui.TextStyle{Color: theme.InkSoft})
			}
		})
	})
	if clicked && entry.IsDir {
		app.open(entry.Path)
	}
}

// shareBar is a track filled in proportion to size/total, with the percentage after it.
func shareBar(context *ui.Context, size, total int64) {
	theme := context.Theme
	fraction := float32(0)
	if total > 0 {
		fraction = float32(size) / float32(total)
	}
	track := ui.Layout{Width: ui.Fixed(barWidth), Height: ui.Fixed(8)}
	context.Frame(track, ui.Look{Fill: theme.Sunken, Radius: 4, Clip: true}, func() {
		if fraction <= 0 {
			return
		}
		fill := ui.Layout{Width: ui.Fixed(max(barWidth*fraction, 2)), Height: ui.Grow()}
		context.Frame(fill, ui.Look{Fill: theme.Primary, Radius: 4}, nil)
	})
	context.Text(fmt.Sprintf("%.0f%%", fraction*100), ui.TextStyle{Size: theme.SmallTextSize, Color: theme.InkSoft})
}

// statusBar says how the scan went, or how far along the one running is.
func (app *app) statusBar(context *ui.Context) {
	bar(context, 26, 12, func() {
		status, colour := "", context.Theme.Positive
		switch last := app.lastScan.Load(); {
		case app.scanning.Load():
			status, colour = fmt.Sprintf("Scanning… %d directories", app.directories.Load()), context.Theme.Warning
		case last != nil:
			status = *last
		}
		context.IconSized("circle", 8, colour)
		gap(context, 4)
		context.Caption(status)
		context.Spacer()
		if tree := app.tree.Load(); tree != nil {
			context.Caption(fmt.Sprintf("%s · total %s", tree.Path, scan.HumanBytes(tree.Size)))
		}
	})
}
