package main

import (
	"math"
	"testing"

	"diskscan/scan"
)

// rectangle is where the layout puts one tile, worked out without a window.
type rectangle struct {
	x, y, width, height float64
	tile                tile
}

// layOut follows cutStrip all the way down, the same cuts squarify turns into boxes.
func layOut(tiles []tile, x, y, width, height float64) []rectangle {
	if len(tiles) == 0 {
		return nil
	}
	next := cutStrip(tiles, width, height)
	strip, rest := tiles[:next.count], tiles[next.count:]
	stripWeight := sumOf(strip)
	var placed []rectangle
	if next.wide {
		stripWidth := width * next.share
		top := y
		for _, tile := range strip {
			tileHeight := height * tile.weight / stripWeight
			placed = append(placed, rectangle{x, top, stripWidth, tileHeight, tile})
			top += tileHeight
		}
		return append(placed, layOut(rest, x+stripWidth, y, next.restWidth, next.restHeight)...)
	}
	stripHeight := height * next.share
	left := x
	for _, tile := range strip {
		tileWidth := width * tile.weight / stripWeight
		placed = append(placed, rectangle{left, y, tileWidth, stripHeight, tile})
		left += tileWidth
	}
	return append(placed, layOut(rest, x, y+stripHeight, next.restWidth, next.restHeight)...)
}

func weighted(weights ...float64) []tile {
	tiles := make([]tile, len(weights))
	for index, weight := range weights {
		tiles[index] = tile{weight: weight}
	}
	return tiles
}

func aspectRatio(r rectangle) float64 { return max(r.width/r.height, r.height/r.width) }

func TestEveryTileGetsAnAreaInProportionToItsSize(t *testing.T) {
	layouts := []struct {
		name          string
		weights       []float64
		width, height float64
	}{
		{"the paper's example", []float64{6, 6, 4, 3, 2, 2, 1}, 6, 4},
		{"a wide window", []float64{1000, 400, 300, 120, 80, 50, 30, 10, 5, 1}, 1600, 900},
		{"a tall window", []float64{50, 40, 30, 20, 10}, 300, 1000},
		{"one tile", []float64{7}, 800, 600},
		{"one giant and crumbs", []float64{10000, 3, 2, 1}, 1200, 800},
	}
	for _, layout := range layouts {
		t.Run(layout.name, func(t *testing.T) {
			tiles := weighted(layout.weights...)
			placed := layOut(tiles, 0, 0, layout.width, layout.height)
			if len(placed) != len(tiles) {
				t.Fatalf("placed %d tiles, want %d", len(placed), len(tiles))
			}
			total, space := sumOf(tiles), layout.width*layout.height
			covered := 0.0
			for _, r := range placed {
				want := space * r.tile.weight / total
				if got := r.width * r.height; math.Abs(got-want) > 1e-6*space {
					t.Errorf("tile of weight %v has area %.3f, want %.3f", r.tile.weight, got, want)
				}
				if r.x < -1e-9 || r.y < -1e-9 || r.x+r.width > layout.width+1e-6 || r.y+r.height > layout.height+1e-6 {
					t.Errorf("tile of weight %v falls outside the space: %+v", r.tile.weight, r)
				}
				covered += r.width * r.height
			}
			if math.Abs(covered-space) > 1e-6*space {
				t.Errorf("tiles cover %.3f of %.3f", covered, space)
			}
		})
	}
}

func TestTheFirstStripIsTheSquarest(t *testing.T) {
	// From the squarified treemap paper: in a 6 × 4 space the two 6s make a
	// column; adding the 4 would stretch the strip, so it starts the next one.
	next := cutStrip(weighted(6, 6, 4, 3, 2, 2, 1), 6, 4)
	if next.count != 2 || !next.wide {
		t.Errorf("first strip = %+v, want the two 6s in a column", next)
	}
	if math.Abs(next.share-0.5) > 1e-9 || math.Abs(next.restWidth-3) > 1e-9 || next.restHeight != 4 {
		t.Errorf("first strip = %+v, want half the width, leaving 3 × 4", next)
	}
}

func TestEqualTilesComeOutNearlySquare(t *testing.T) {
	weights := make([]float64, 16)
	for index := range weights {
		weights[index] = 1
	}
	for _, r := range layOut(weighted(weights...), 0, 0, 400, 400) {
		if ratio := aspectRatio(r); ratio > 2 {
			t.Errorf("a tile is %.2f times longer than wide: %+v", ratio, r)
		}
	}
}

func TestTallSpacesGetRowsOnTop(t *testing.T) {
	next := cutStrip(weighted(5, 3, 2), 100, 400)
	if next.wide {
		t.Errorf("a space taller than wide should start with a row: %+v", next)
	}
	if next.restWidth != 100 {
		t.Errorf("a row keeps the full width for the rest, got %v", next.restWidth)
	}
}

func TestTilesForGroupsTheSmallOnesAndSkipsTheEmpty(t *testing.T) {
	entries := []scan.Entry{
		{Name: "big", Size: 9000, IsDir: true},
		{Name: scan.FilesBucketName, Size: 980},
		{Name: "tiny-a", Size: 15, IsDir: true}, // 0.15 % of 10000: below smallestShare
		{Name: "tiny-b", Size: 5, IsDir: true},
		{Name: "empty", Size: 0, IsDir: true},
	}
	tiles := tilesFor(entries, 10000)

	if len(tiles) != 3 {
		t.Fatalf("got %d tiles, want big, the files and one for the rest: %+v", len(tiles), tiles)
	}
	if tiles[0].entry.Name != "big" || tiles[0].colour == quietColour {
		t.Errorf("first tile = %+v, want big in a colour of its own", tiles[0])
	}
	if tiles[1].colour != quietColour {
		t.Error("the files bucket leads nowhere, so it should be quiet")
	}
	other := tiles[2]
	if other.other != 2 || other.entry.Size != 20 || other.entry.IsDir || other.colour != quietColour {
		t.Errorf("other tile = %+v, want the two tiny folders (20 bytes), quiet and not clickable", other)
	}
}

func TestTilesForCapsTheCount(t *testing.T) {
	var entries []scan.Entry
	for index := 0; index < maximumTiles+20; index++ {
		entries = append(entries, scan.Entry{Name: string(rune('a'+index%26)) + string(rune('0'+index/26)), Size: 100, IsDir: true})
	}
	tiles := tilesFor(entries, int64(100*len(entries)))
	if len(tiles) != maximumTiles+1 {
		t.Fatalf("got %d tiles, want %d plus one for the rest", len(tiles), maximumTiles)
	}
	if last := tiles[len(tiles)-1]; last.other != 20 || last.entry.Size != 2000 {
		t.Errorf("other tile = %+v, want the 20 left over", last)
	}
}

func TestTilesForAnEmptyFolder(t *testing.T) {
	if tiles := tilesFor(nil, 0); len(tiles) != 0 {
		t.Errorf("an empty folder got tiles: %+v", tiles)
	}
	if tiles := tilesFor([]scan.Entry{{Name: "zero", IsDir: true}}, 0); len(tiles) != 0 {
		t.Errorf("a folder of empty folders got tiles: %+v", tiles)
	}
}
