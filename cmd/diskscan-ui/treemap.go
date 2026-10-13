package main

// The map view: the folder's contents as rectangles whose area is their size,
// laid out with the squarified algorithm (Bruls, Huizing and van Wijk) so the
// rectangles stay close to squares and are easy to compare.
//
// go-ui has no absolute positioning, and needs none here: the algorithm cuts
// the area into strips, and a strip is just a box whose children share its
// length by weight (Share), next to a box with what is left.

import (
	"fmt"

	"diskscan/scan"

	"github.com/akrck02/go-renderer/draw2d"
	ui "github.com/akrck02/go-ui"
	"github.com/akrck02/go-ui/window"
)

// mapScope names the scope the tiles live in, so their areas can be looked up.
const mapScope = "treemap"

// maximumTiles caps the rectangles drawn; the rest share one "other" tile.
const maximumTiles = 80

// smallestShare is the fraction of the folder below which an entry joins the
// "other" tile instead of getting a sliver of its own.
const smallestShare = 0.002

// tileGap is the space between two rectangles.
const tileGap = 3

// tileColours are the fills for folders, one after another, biggest first.
var tileColours = []draw2d.Color{
	draw2d.Hex("#3b6fd8"), draw2d.Hex("#2f9e8f"), draw2d.Hex("#8a5cd0"), draw2d.Hex("#d0753b"),
	draw2d.Hex("#c2477a"), draw2d.Hex("#4f9a3f"), draw2d.Hex("#b8902a"), draw2d.Hex("#3f8fb8"),
}

// quietColour fills the tiles that lead nowhere: the folder's own files and "other".
var quietColour = draw2d.Hex("#7d8696")

// tile is one rectangle of the map.
type tile struct {
	entry  scan.Entry
	weight float64 // its size, the only thing the layout looks at
	colour draw2d.Color
	other  int // for the "other" tile: how many entries it stands for
}

// tilesFor turns a folder's entries (biggest first) into the map's tiles.
func tilesFor(entries []scan.Entry, total int64) []tile {
	var tiles []tile
	var otherSize int64
	otherCount := 0
	for _, entry := range entries {
		if entry.Size <= 0 {
			continue
		}
		small := total > 0 && float64(entry.Size)/float64(total) < smallestShare
		if small || len(tiles) == maximumTiles {
			otherSize += entry.Size
			otherCount++
			continue
		}
		colour := quietColour
		if entry.IsDir {
			colour = tileColours[len(tiles)%len(tileColours)]
		}
		tiles = append(tiles, tile{entry: entry, weight: float64(entry.Size), colour: colour})
	}
	if otherSize > 0 {
		other := scan.Entry{Name: fmt.Sprintf("%d smaller items", otherCount), Size: otherSize}
		tiles = append(tiles, tile{entry: other, weight: float64(otherSize), colour: quietColour, other: otherCount})
	}
	return tiles
}

// treemap draws the folder as a map, filling the space it is given.
func (app *app) treemap(context *ui.Context, node *scan.Node) {
	tiles := tilesFor(app.entries, node.Size)
	if len(tiles) == 0 {
		empty := ui.Layout{Width: ui.Grow(), Height: ui.Grow(), AlignX: ui.Center, AlignY: ui.Center}
		context.Box(empty, func() { context.Caption("This folder is empty") })
		return
	}
	width, height := app.mapSize(context, tiles)
	context.Push(mapScope)
	context.Box(ui.Layout{Width: ui.Grow(), Height: ui.Grow()}, func() {
		app.squarify(context, tiles, width, height)
	})
	context.Pop()
}

// mapSize is the map's size in the last frame: the box round all its tiles.
// The layout only needs its proportions, so the first frame of a folder makes
// do with a guess, and asks for one more frame once the real size is known.
func (app *app) mapSize(context *ui.Context, tiles []tile) (float64, float64) {
	var bounds draw2d.Rect
	for _, tile := range tiles {
		area, found := context.Area(mapScope, tileName(tile))
		if !found {
			continue
		}
		bounds = union(bounds, area)
	}
	if bounds.IsEmpty() {
		window.Wake() // drawn once, the tiles have areas: lay them out again with the real proportions
		return 16, 9
	}
	return float64(bounds.Width), float64(bounds.Height)
}

// union is the smallest rectangle holding both.
func union(first, second draw2d.Rect) draw2d.Rect {
	if first.IsEmpty() {
		return second
	}
	left, top := min(first.X, second.X), min(first.Y, second.Y)
	right, bottom := max(first.Right(), second.Right()), max(first.Bottom(), second.Bottom())
	return draw2d.Rect{X: left, Y: top, Width: right - left, Height: bottom - top}
}

// squarify lays tiles (biggest first) into a width × height space: one strip
// (cutStrip), then the same again with what is left in the rest of the space.
func (app *app) squarify(context *ui.Context, tiles []tile, width, height float64) {
	if len(tiles) == 0 {
		return
	}
	next := cutStrip(tiles, width, height)
	strip, rest := tiles[:next.count], tiles[next.count:]
	stripWeight := sumOf(strip)

	outer := ui.Layout{Width: ui.Grow(), Height: ui.Grow(), Gap: tileGap}
	stripLayout := ui.Layout{Direction: ui.Vertical, Width: ui.Share(float32(next.share)), Height: ui.Grow(), Gap: tileGap}
	restLayout := ui.Layout{Width: ui.Share(float32(1 - next.share)), Height: ui.Grow()}
	if !next.wide {
		outer.Direction = ui.Vertical
		stripLayout = ui.Layout{Width: ui.Grow(), Height: ui.Share(float32(next.share)), Gap: tileGap}
		restLayout = ui.Layout{Width: ui.Grow(), Height: ui.Share(float32(1 - next.share))}
	}
	context.Box(outer, func() {
		context.Box(stripLayout, func() {
			for _, tile := range strip {
				app.drawTile(context, tile, float32(tile.weight/stripWeight), next.wide)
			}
		})
		if len(rest) > 0 {
			context.Box(restLayout, func() { app.squarify(context, rest, next.restWidth, next.restHeight) })
		}
	})
}

// strip is one cut of the squarified layout: the first count tiles go in a
// strip taking share of the space, and the rest fill restWidth × restHeight.
type strip struct {
	count                 int
	wide                  bool    // the space is wider than tall: the strip is a column on its left (else a row on top)
	share                 float64 // the strip's fraction of the space, along its width (wide) or height
	restWidth, restHeight float64
}

// cutStrip takes the tiles that make the squarest strip along the shorter side
// of a width × height space. It only measures, so the layout can be tested
// without a window.
func cutStrip(tiles []tile, width, height float64) strip {
	total := sumOf(tiles)
	count := stripLength(tiles, total, width*height, min(width, height))
	share := sumOf(tiles[:count]) / total
	if width >= height {
		return strip{count: count, wide: true, share: share, restWidth: width * (1 - share), restHeight: height}
	}
	return strip{count: count, share: share, restWidth: width, restHeight: height * (1 - share)}
}

// stripLength is how many of the first tiles go in the next strip: tiles are
// added while they make the strip's worst rectangle squarer.
func stripLength(tiles []tile, total, area, side float64) int {
	scale := area / total // space per unit of weight
	best := worstRatio(tiles[:1], scale, side)
	count := 1
	for count < len(tiles) {
		next := worstRatio(tiles[:count+1], scale, side)
		if next > best {
			break
		}
		best, count = next, count+1
	}
	return count
}

// worstRatio is how far from a square the most stretched rectangle of a strip
// is (1 = a square), for a strip laid along a side of the given length.
func worstRatio(strip []tile, scale, side float64) float64 {
	stripArea := sumOf(strip) * scale
	thickness := stripArea / side
	worst := 1.0
	for _, tile := range strip {
		length := tile.weight * scale / thickness
		ratio := max(length/thickness, thickness/length)
		worst = max(worst, ratio)
	}
	return worst
}

func sumOf(tiles []tile) float64 {
	sum := 0.0
	for _, tile := range tiles {
		sum += tile.weight
	}
	return sum
}

// tileName is a tile's id: its name, unique within a folder (the files bucket
// and the "other" tile have names no folder can share a listing with).
func tileName(tile tile) string { return tile.entry.Name + "##tile" }

// drawTile is one rectangle: its name and size when they fit, all of it in the
// tooltip, and a click on a folder goes into it.
func (app *app) drawTile(context *ui.Context, tile tile, share float32, wide bool) {
	layout := ui.Layout{Direction: ui.Vertical, Width: ui.Grow(), Height: ui.Share(share), Padding: ui.PadXY(8, 6), Gap: 2}
	if !wide {
		layout.Width, layout.Height = ui.Share(share), ui.Grow()
	}
	normal := ui.Look{Fill: tile.colour, Radius: 4, Clip: true}
	looks := ui.Looks{Normal: normal, Hovered: normal, Pressed: normal}
	if tile.entry.IsDir {
		looks.Hovered = ui.Look{Fill: tile.colour.Mix(draw2d.Hex("#ffffff"), 0.18), Radius: 4, Clip: true}
		looks.Pressed = ui.Look{Fill: tile.colour.Mix(draw2d.Hex("#000000"), 0.15), Radius: 4, Clip: true}
	}
	white := draw2d.Hex("#ffffff")
	clicked := context.Clickable(tileName(tile), layout, looks, func() {
		context.Text(tile.entry.Name, ui.TextStyle{Size: 13, Weight: draw2d.Medium, Color: white})
		context.Text(scan.HumanBytes(tile.entry.Size), ui.TextStyle{Size: 11, Color: white.WithAlpha(0.8)})
	})
	context.Tooltip(fmt.Sprintf("%s · %s", tile.entry.Name, scan.HumanBytes(tile.entry.Size)))
	if clicked && tile.entry.IsDir {
		app.open(tile.entry.Path)
	}
}
