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
	x, y   float64
	angle  float64
	sector int
	moveFB int32
	moveLR int32
	turnLR int32
}

type edgeData struct {
	x1, y1, x2, y2 float64
}

type doomGame struct {
	level      *level.Level
	player     player
	frameGFA   string
	frameReady atomic.Bool
	edges      []edgeData
}

var doomGameState *doomGame

func loadDoomFrames() {}

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

	for _, ld := range lvl.LinesDefs {
		v1 := lvl.Vert(uint32(ld.Start))
		v2 := lvl.Vert(uint32(ld.End))
		g.edges = append(g.edges, edgeData{
			x1: float64(v1.X()), y1: float64(v1.Y()),
			x2: float64(v2.X()), y2: float64(v2.Y()),
		})
	}

	for _, thing := range lvl.Things {
		if thing.Type == 1 {
			g.player.x = float64(thing.X)
			g.player.y = float64(thing.Y)
			g.player.angle = float64(thing.Angle) * math.Pi / 180.0
			fmt.Printf("doom: player start at (%.0f, %.0f) facing %.0f deg\n", g.player.x, g.player.y, float64(thing.Angle))
			break
		}
	}

	doomGameState = g
	go g.loop()
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
		mx := float64(fb)*math.Cos(g.player.angle) + float64(lr)*math.Cos(g.player.angle+math.Pi/2)
		my := float64(fb)*math.Sin(g.player.angle) + float64(lr)*math.Sin(g.player.angle+math.Pi/2)
		dist := moveSpeed * float64(frameInterval) / float64(time.Second)
		nx := g.player.x + mx*dist
		ny := g.player.y + my*dist
		if !g.collides(nx, ny) {
			g.player.x = nx
			g.player.y = ny
		}
	}
}

func (g *doomGame) collides(x, y float64) bool {
	const margin = 15.0
	for _, e := range g.edges {
		dx := e.x2 - e.x1
		dy := e.y2 - e.y1
		lenSq := dx*dx + dy*dy
		if lenSq < 0.01 {
			continue
		}
		t := ((x-e.x1)*dx + (y-e.y1)*dy) / lenSq
		t = math.Max(0, math.Min(1, t))
		cx := e.x1 + t*dx
		cy := e.y1 + t*dy
		dist := math.Sqrt((x-cx)*(x-cx) + (y-cy)*(y-cy))
		if dist < margin {
			return true
		}
	}
	return false
}

func (g *doomGame) render() {
	w := screenW
	h := screenH
	raw := make([]byte, (w*h+7)/8)

	// floor: checkerboard
	floorStart := (w*h/2 + 7) / 8
	for i := floorStart; i < len(raw); i++ {
		if (i/5)%2 == 0 {
			raw[i] = 0x55
		}
	}

	// crosshair
	cx, cy := w/2, h/2
	for i := cx - 10; i <= cx+10; i++ {
		setPixelRaw(raw, w, i, cy)
	}
	for i := cy - 10; i <= cy+10; i++ {
		setPixelRaw(raw, w, cx, i)
	}

	numRays := w
	for col := 0; col < numRays; col++ {
		rayAngle := g.player.angle - fov/2 + (float64(col)/float64(numRays))*fov
		cosA := math.Cos(rayAngle)
		sinA := math.Sin(rayAngle)

		minDist := math.MaxFloat64

		for _, e := range g.edges {
			dx := e.x2 - e.x1
			dy := e.y2 - e.y1

			// ray-line intersection: t = cross(A-O, B-A) / cross(D, B-A)
			// cross(v,w) = v_x*w_y - v_y*w_x
			crossAO_BA := (e.x1-g.player.x)*dy - (e.y1-g.player.y)*dx
			crossD_BA := cosA*dy - sinA*dx

			if math.Abs(crossD_BA) < 0.0001 {
				continue
			}

			t := crossAO_BA / crossD_BA
			if t <= 0 {
				continue
			}

			ix := g.player.x + t*cosA
			iy := g.player.y + t*sinA

			// check if intersection is on the segment
			dot := (ix-e.x1)*dx + (iy-e.y1)*dy
			lenSq := dx*dx + dy*dy
			if dot < 0 || dot > lenSq {
				continue
			}

			if t < minDist {
				minDist = t
			}
		}

		if minDist < math.MaxFloat64 {
			correctedDist := minDist * math.Cos(rayAngle-g.player.angle)
			if correctedDist < 0.1 {
				correctedDist = 0.1
			}
			wallHeight := int(float64(h) / correctedDist)
			if wallHeight > h {
				wallHeight = h
			}
			top := (h - wallHeight) / 2
			bottom := top + wallHeight
			for row := top; row < bottom; row++ {
				setPixelRaw(raw, w, col, row)
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

func setPixelRaw(raw []byte, stride, x, y int) {
	if x < 0 || y < 0 || x >= stride || y*stride/8+x/8 >= len(raw) {
		return
	}
	byteIdx := y*(stride/8) + x/8
	bitIdx := 7 - (x % 8)
	raw[byteIdx] |= 1 << bitIdx
}
