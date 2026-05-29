package main

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"encoding/hex"
	"fmt"
	"image/jpeg"
	"log"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

//go:embed frames.zip
var framesZip []byte

var badAppleFrames []string

func init() {
	start := time.Now()
	reader, err := zip.NewReader(bytes.NewReader(framesZip), int64(len(framesZip)))
	if err != nil {
		panic(fmt.Sprintf("failed to read frames.zip: %v", err))
	}

	type frameEntry struct {
		index int
		data  []byte
	}

	var entries []frameEntry
	for _, f := range reader.File {
		if !strings.HasSuffix(f.Name, ".jpg") {
			continue
		}
		name := f.Name
		if idx := strings.LastIndex(name, "/"); idx >= 0 {
			name = name[idx+1:]
		}
		numStr := strings.TrimPrefix(strings.TrimSuffix(name, ".jpg"), "output_")
		num, err := strconv.Atoi(numStr)
		if err != nil {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			panic(fmt.Sprintf("failed to open %s: %v", f.Name, err))
		}
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(rc); err != nil {
			rc.Close()
			panic(fmt.Sprintf("failed to read %s: %v", f.Name, err))
		}
		rc.Close()
		entries = append(entries, frameEntry{index: num, data: buf.Bytes()})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].index < entries[j].index
	})

	log.Printf("read %d frames from zip (%v)", len(entries), time.Since(start).Round(time.Millisecond))

	badAppleFrames = make([]string, len(entries))

	workers := runtime.GOMAXPROCS(0)
	var wg sync.WaitGroup
	var progress atomic.Int64
	done := make(chan struct{})

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				p := progress.Load()
				log.Printf("decoding frames: %d/%d (%d%%)", p, len(entries), p*100/int64(len(entries)))
			}
		}
	}()

	jobs := make(chan int, len(entries))
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				badAppleFrames[i] = jpegToGFA(entries[i].data)
				progress.Add(1)
			}
		}()
	}

	for i := range entries {
		jobs <- i
	}
	close(jobs)

	wg.Wait()
	close(done)

	log.Printf("decoded %d frames to GFA (%v)", len(badAppleFrames), time.Since(start).Round(time.Millisecond))
}

func jpegToGFA(data []byte) string {
	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return ""
	}

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	bytesPerRow := (w + 7) / 8
	totalBytes := bytesPerRow * h

	raw := make([]byte, totalBytes)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, _ := img.At(x+bounds.Min.X, y+bounds.Min.Y).RGBA()
			lum := 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
			if lum > 128 {
				continue
			}
			byteIdx := y*bytesPerRow + x/8
			bitIdx := 7 - (x % 8)
			raw[byteIdx] |= 1 << bitIdx
		}
	}

	hexStr := strings.ToUpper(hex.EncodeToString(raw))
	return fmt.Sprintf(`^FO0,0^GFA,%d,%d,%d,%s^FS`, totalBytes, totalBytes, bytesPerRow, hexStr)
}
