package main

import (
	"encoding/hex"
	"fmt"
	"math"
	"strings"

	"github.com/tinogoehlert/goom/level"
	"github.com/tinogoehlert/goom/wad"
)

var doomFrames []string

func loadDoomFrames() {
	w, err := wad.NewWADFromFile("goom/DOOM1.WAD")
	if err != nil {
		panic(fmt.Sprintf("failed to load DOOM1.WAD: %v", err))
	}

	store := level.NewStore()
	if err := store.LoadWAD(w); err != nil {
		panic(fmt.Sprintf("failed to load levels: %v", err))
	}

	lvl := store["E1M1"]
	if lvl == nil {
		panic("E1M1 not found in DOOM1.WAD")
	}

	minX, minY, maxX, maxY := float64(math.MaxFloat64), float64(math.MaxFloat64), float64(-math.MaxFloat64), float64(-math.MaxFloat64)
	type edge struct {
		x1, y1, x2, y2 float64
	}
	var edges []edge
	var vertices []struct{ x, y float64 }

	for _, ld := range lvl.LinesDefs {
		v1 := lvl.Vert(uint32(ld.Start))
		v2 := lvl.Vert(uint32(ld.End))
		x1, y1 := float64(v1.X()), float64(-v1.Y())
		x2, y2 := float64(v2.X()), float64(-v2.Y())
		vertices = append(vertices, struct{ x, y float64 }{x1, y1})
		vertices = append(vertices, struct{ x, y float64 }{x2, y2})
		edges = append(edges, edge{x1, y1, x2, y2})
		if x1 < minX {
			minX = x1
		}
		if y1 < minY {
			minY = y1
		}
		if x2 < minX {
			minX = x2
		}
		if y2 < minY {
			minY = y2
		}
		if x1 > maxX {
			maxX = x1
		}
		if y1 > maxY {
			maxY = y1
		}
		if x2 > maxX {
			maxX = x2
		}
		if y2 > maxY {
			maxY = y2
		}
	}

	mapW := maxX - minX
	mapH := maxY - minY
	scale := 600.0 / math.Max(mapW, mapH)
	imgW := int(mapW*scale) + 20
	imgH := int(mapH*scale) + 20

	for i := 0; i < len(vertices); i++ {
		vertices[i].x = (vertices[i].x-minX)*scale + 10
		vertices[i].y = (vertices[i].y-minY)*scale + 10
	}

	totalSteps := 120
	for step := 0; step < totalSteps; step++ {

		bytesPerRow := (imgW + 7) / 8
		totalBytes := bytesPerRow * imgH
		raw := make([]byte, totalBytes)

		for _, e := range edges {
			drawLineRaw(raw, bytesPerRow, int(e.x1), int(e.y1), int(e.x2), int(e.y2))
		}

		for _, v := range vertices {
			setPixel(raw, bytesPerRow, int(v.x), int(v.y))
			setPixel(raw, bytesPerRow, int(v.x)+1, int(v.y))
		}

		hexStr := strings.ToUpper(hex.EncodeToString(raw))
		doomFrames = append(doomFrames, fmt.Sprintf(`^FO0,0^GFA,%d,%d,%d,%s^FS`, totalBytes, totalBytes, bytesPerRow, hexStr))
	}
}

func drawLineRaw(raw []byte, stride, x1, y1, x2, y2 int) {
	dx := abs(x2 - x1)
	dy := -abs(y2 - y1)
	sx, sy := 1, 1
	if x1 > x2 {
		sx = -1
	}
	if y1 > y2 {
		sy = -1
	}
	err := dx + dy
	for {
		setPixel(raw, stride, x1, y1)
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x1 += sx
		}
		if e2 <= dx {
			err += dx
			y1 += sy
		}
	}
}

func setPixel(raw []byte, stride, x, y int) {
	if x < 0 || y < 0 || x >= stride*8 || y >= len(raw)/stride {
		return
	}
	byteIdx := y*stride + x/8
	bitIdx := 7 - (x % 8)
	raw[byteIdx] |= 1 << bitIdx
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
