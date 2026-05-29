package main

import (
	"bytes"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/ingridhq/zebrash"
	"github.com/ingridhq/zebrash/drawers"
)

func TestBallHexNotEmpty(t *testing.T) {
	if ballHex == "" {
		t.Fatal("ballHex is empty")
	}
}

func TestZPLTemplateRenders(t *testing.T) {
	tests := []struct {
		name string
		y    int
	}{
		{"top", 20},
		{"middle", 160},
		{"bottom", 300},
	}

	parser := zebrash.NewParser()
	drawer := zebrash.NewDrawer()
	opts := drawers.DrawerOptions{
		LabelWidthMm:         101.6,
		LabelHeightMm:        101.6,
		Dpmm:                 8,
		GrayscaleOutput:      true,
		EnableInvertedLabels: false,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zpl := fmt.Sprintf(labelZPL, 36, tt.y, ballHex)
			labels, err := parser.Parse([]byte(zpl))
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			if len(labels) == 0 {
				t.Fatal("no labels parsed")
			}

			var buf bytes.Buffer
			if err := drawer.DrawLabelAsPng(labels[0], &buf, opts); err != nil {
				t.Fatalf("render error: %v", err)
			}

			png := buf.Bytes()
			if len(png) == 0 {
				t.Fatal("rendered PNG is empty")
			}
			if !bytes.HasPrefix(png, []byte{0x89, 0x50, 0x4E, 0x47}) {
				t.Fatal("output is not a valid PNG (missing PNG header)")
			}

			t.Logf("rendered PNG size: %d bytes at y=%d", len(png), tt.y)
		})
	}
}

func TestBallOscillation(t *testing.T) {
	now := float64(time.Now().UnixMilli()) / 1000.0

	y1 := int(160 + math.Sin(now*4)*140)
	y2 := int(160 + math.Sin((now+0.25)*4)*140)
	y3 := int(160 + math.Sin((now+0.5)*4)*140)

	if y1 == y2 || y2 == y3 || y1 == y3 {
		t.Errorf("ball y should oscillate, got y1=%d y2=%d y3=%d", y1, y2, y3)
	}
}

func TestRenderProducesDifferentFrames(t *testing.T) {
	parser := zebrash.NewParser()
	drawer := zebrash.NewDrawer()
	opts := drawers.DrawerOptions{
		LabelWidthMm:         101.6,
		LabelHeightMm:        101.6,
		Dpmm:                 8,
		GrayscaleOutput:      true,
		EnableInvertedLabels: false,
	}

	render := func(y int) []byte {
		zpl := fmt.Sprintf(labelZPL, 36, y, ballHex)
		labels, err := parser.Parse([]byte(zpl))
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		var buf bytes.Buffer
		if err := drawer.DrawLabelAsPng(labels[0], &buf, opts); err != nil {
			t.Fatalf("render error: %v", err)
		}
		return buf.Bytes()
	}

	top := render(20)
	bottom := render(300)

	if bytes.Equal(top, bottom) {
		t.Error("frames at different Y positions should produce different PNGs")
	}
}
