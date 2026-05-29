package main

import (
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/tinogoehlert/goom/level"
	"github.com/tinogoehlert/goom/wad"
)

const (
	screenW       = 320
	screenH       = 200
	fov           = math.Pi / 3
	moveSpeed     = 100.0
	turnSpeed     = 2.5
	frameInterval = 33 * time.Millisecond
)

type player struct {
	x, y    float64
	angle   float64
	sector  int
	moveFB  int32 // -1 back, 0 stop, 1 forward
	moveLR  int32 // -1 left, 0 stop, 1 right
	turnLR  int32 // -1 left, 0 stop, 1 right
}

type doomGame struct {
	level      *level.Level
	player     player
	frameGFA   string
	frameReady atomic.Bool
	vertices   []struct{ x, y float64 }
	edges      []struct {
		x1, y1, x2, y2 float64
		sector          int
	}
	sectors []struct {
		floor, ceil float64
	}
}

var doomGameState *doomGame

func loadDoomFrames() {
	// kept for compatibility
}

func loadDoomGame() {
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

	g := &doomGame{level: lvl}

	// extract vertices and edges
	for _, ld := range lvl.LinesDefs {
		v1 := lvl.Vert(uint32(ld.Start))
		v2 := lvl.Vert(uint32(ld.End))
		x1, y1 := float64(v1.X()), float64(v1.Y())
		x2, y2 := float64(v2.X()), float64(v2.Y())
		g.vertices = append(g.vertices, struct{ x, y float64 }{x1, y1})
		g.vertices = append(g.vertices, struct{ x, y float64 }{x2, y2})

		sectorIdx := int(ld.Right)
		if sectorIdx < 0 || sectorIdx >= len(lvl.Sectors) {
			sectorIdx = 0
		}
		sector := lvl.Sectors[sectorIdx]
		g.edges = append(g.edges, struct {
			x1, y1, x2, y2 float64
			sector          int
		}{x1, y1, x2, y2, sectorIdx})
		g.sectors = append(g.sectors, struct{ floor, ceil float64 }{
			floor: float64(sector.FloorHeight()),
			ceil:  float64(sector.CeilHeight()),
		})
	}

	// find player start
	for _, thing := range lvl.Things {
		if thing.Type == 1 {
			g.player.x = float64(thing.X)
			g.player.y = float64(thing.Y)
			g.player.angle = float64(thing.Angle) * math.Pi / 180.0
			for i, ld := range lvl.LinesDefs {
				v1 := lvl.Vert(uint32(ld.Start))
				v2 := lvl.Vert(uint32(ld.End))
				if pointInSector(float64(thing.X), float64(thing.Y),
					float64(v1.X()), float64(v1.Y()),
					float64(v2.X()), float64(v2.Y())) {
					g.player.sector = i
					break
				}
			}
			break
		}
	}

	doomGameState = g
	go g.loop()
}

func pointInSector(px, py, x1, y1, x2, y2 float64) bool {
	return (x2-x1)*(py-y1)-(y2-y1)*(px-x1) > 0
}

func (g *doomGame) loop() {
	ticker := time.NewTicker(frameInterval)
	for range ticker.C {
		g.update()
		g.render()
	}
}

func (g *doomGame) update() {
	fb := atomic.LoadInt32(&g.player.moveFB)
	lr := atomic.LoadInt32(&g.player.moveLR)
	tr := atomic.LoadInt32(&g.player.turnLR)
	if tr != 0 {
		g.player.angle += float64(tr) * turnSpeed * float64(frameInterval) / float64(time.Second)
	}
	if fb != 0 || lr != 0 {
		mx := float64(fb) * math.Cos(g.player.angle)
		my := float64(fb) * math.Sin(g.player.angle)
		mx += float64(lr) * math.Cos(g.player.angle+math.Pi/2)
		my += float64(lr) * math.Sin(g.player.angle+math.Pi/2)
		dist := moveSpeed * float64(frameInterval) / float64(time.Second)
		if dist > 0 {
			nx := g.player.x + mx*dist
			ny := g.player.y + my*dist
			if !g.collides(nx, ny) {
				g.player.x = nx
				g.player.y = ny
			}
		}
	}
}

func (g *doomGame) collides(x, y float64) bool {
	for _, e := range g.edges {
		minDist := 20.0
		dx := e.x2 - e.x1
		dy := e.y2 - e.y1
		lenSq := dx*dx + dy*dy
		if lenSq < 1 {
			continue
		}
		t := ((x-e.x1)*dx + (y-e.y1)*dy) / lenSq
		t = math.Max(0, math.Min(1, t))
		cx := e.x1 + t*dx
		cy := e.y1 + t*dy
		dist := math.Sqrt((x-cx)*(x-cx) + (y-cy)*(y-cy))
		if dist < minDist {
			return true
		}
	}
	return false
}

func (g *doomGame) render() {
	w := screenW
	h := screenH
	raw := make([]byte, (w*h+7)/8)

	// ceiling - all white
	for i := 0; i < w*h/2/8; i++ {
		raw[i] = 0xFF
	}

	// floor pattern
	start := (w*h/2 + 7) / 8
	for i := start; i < len(raw); i++ {
		raw[i] = 0x55
	}

	// raycasting for walls
	numRays := w
	for col := 0; col < numRays; col++ {
		rayAngle := g.player.angle - fov/2 + (float64(col)/float64(numRays))*fov
		cosA := math.Cos(rayAngle)
		sinA := math.Sin(rayAngle)

		minDist := math.MaxFloat64
		var wallHeight int

		for _, e := range g.edges {
			ex1, ey1, ex2, ey2 := e.x1, e.y1, e.x2, e.y2

			dx := ex2 - ex1
			dy := ey2 - ey1
			denom := cosA*dy - sinA*dx
			if math.Abs(denom) < 0.001 {
				continue
			}
			t := ((ex1-g.player.x)*dy - (ey1-g.player.y)*dx) / denom
			u := ((g.player.x-ex1)*sinA - (g.player.y-ey1)*cosA) / denom
			if t > 0 && u >= -0.001 && u <= 1.001 && t < minDist {
				minDist = t
				wallHeight = int(float64(h) / (t * math.Cos(rayAngle-g.player.angle)))
			}
		}

		if wallHeight > h {
			wallHeight = h
		}
		top := (h - wallHeight) / 2
		bottom := top + wallHeight
		for row := top; row < bottom; row++ {
			byteIdx := row*(w/8) + col/8
			bitIdx := 7 - (col % 8)
			if byteIdx < len(raw) {
				raw[byteIdx] |= 1 << bitIdx
			}
		}
	}

	bytesPerRow := w / 8
	totalBytes := len(raw)
	hexStr := strings.ToUpper(hex.EncodeToString(raw))
	frame := fmt.Sprintf(`^FO0,0^GFA,%d,%d,%d,%s^FS`, totalBytes, totalBytes, bytesPerRow, hexStr)

	ptr := (*unsafe.Pointer)(unsafe.Pointer(&g.frameGFA))
	atomic.StorePointer(ptr, unsafe.Pointer(&frame))
	g.frameReady.Store(true)
}

func currentDoomZPL() string {
	if doomGameState == nil || !doomGameState.frameReady.Load() {
		return "^XA^XZ"
	}
	ptr := (*unsafe.Pointer)(unsafe.Pointer(&doomGameState.frameGFA))
	return *(*string)(atomic.LoadPointer(ptr))
}

func setDoomInput(fb, lr, tr int32) {
	if doomGameState != nil {
		atomic.StoreInt32(&doomGameState.player.moveFB, fb)
		atomic.StoreInt32(&doomGameState.player.moveLR, lr)
		atomic.StoreInt32(&doomGameState.player.turnLR, tr)
	}
}
