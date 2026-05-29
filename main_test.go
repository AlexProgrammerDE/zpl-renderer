package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ingridhq/zebrash"
	"github.com/ingridhq/zebrash/drawers"
)

func TestFramesLoaded(t *testing.T) {
	if len(badAppleFrames) == 0 {
		t.Fatal("no bad apple frames loaded")
	}
	t.Logf("loaded %d frames", len(badAppleFrames))
}

func TestFrameRenders(t *testing.T) {
	if len(badAppleFrames) < 10 {
		t.Skip("not enough frames to test")
	}

	parser := zebrash.NewParser()
	drawer := zebrash.NewDrawer()
	opts := drawers.DrawerOptions{
		LabelWidthMm:         50,
		LabelHeightMm:        50,
		Dpmm:                 12,
		GrayscaleOutput:      true,
		EnableInvertedLabels: false,
	}

	for _, idx := range []int{0, len(badAppleFrames) / 2, len(badAppleFrames) - 1} {
		frame := badAppleFrames[idx]
		zpl := "^XA\n^LL600\n^PW600\n" + frame + "\n^FO10,370^BQN,2,6^FDM,,https://github.com/AlexProgrammerDE/zpl-renderer^FS\n^XZ"

		labels, err := parser.Parse([]byte(zpl))
		if err != nil {
			t.Fatalf("parse error at frame %d: %v", idx, err)
		}
		if len(labels) == 0 {
			t.Fatalf("no labels at frame %d", idx)
		}

		var buf bytes.Buffer
		if err := drawer.DrawLabelAsPng(labels[0], &buf, opts); err != nil {
			t.Fatalf("render error at frame %d: %v", idx, err)
		}

		png := buf.Bytes()
		if len(png) == 0 {
			t.Fatalf("empty PNG at frame %d", idx)
		}
		if !bytes.HasPrefix(png, []byte{0x89, 0x50, 0x4E, 0x47}) {
			t.Fatalf("invalid PNG at frame %d", idx)
		}

		t.Logf("frame %d: %d bytes", idx, len(png))
	}
}

func TestFramesDiffer(t *testing.T) {
	if len(badAppleFrames) < 100 {
		t.Skip("not enough frames to test")
	}

	unique := make(map[string]bool)
	for i := 0; i < 100 && i < len(badAppleFrames); i++ {
		unique[badAppleFrames[i]] = true
	}
	if len(unique) < 2 {
		t.Error("expected at least 2 distinct frames in the first 100")
	}

	mid := len(badAppleFrames) / 2
	uniqueMid := make(map[string]bool)
	for i := mid; i < mid+100 && i < len(badAppleFrames); i++ {
		uniqueMid[badAppleFrames[i]] = true
	}
	if len(uniqueMid) < 2 {
		t.Error("expected at least 2 distinct frames around the middle")
	}
}

func TestQRCodeRenders(t *testing.T) {
	if len(badAppleFrames) == 0 {
		t.Skip("no frames loaded")
	}

	parser := zebrash.NewParser()
	drawer := zebrash.NewDrawer()
	opts := drawers.DrawerOptions{
		LabelWidthMm:         50,
		LabelHeightMm:        50,
		Dpmm:                 12,
		GrayscaleOutput:      true,
		EnableInvertedLabels: false,
	}

	zpl := currentZPL("badapple")
	if !strings.Contains(zpl, "https") {
		t.Fatal("QR code data missing https URL")
	}
	labels, err := parser.Parse([]byte(zpl))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(labels) == 0 {
		t.Fatal("no labels")
	}

	var buf bytes.Buffer
	if err := drawer.DrawLabelAsPng(labels[0], &buf, opts); err != nil {
		t.Fatalf("render error: %v", err)
	}

	png := buf.Bytes()
	if len(png) < 1000 {
		t.Error("PNG with QR code seems too small (QR may be cut off)")
	}
	if !bytes.HasPrefix(png, []byte{0x89, 0x50, 0x4E, 0x47}) {
		t.Fatal("invalid PNG")
	}

	t.Logf("full label with QR: %d bytes", len(png))
}

func TestDoomFrames(t *testing.T) {
	loadDoomFrames()
	if len(doomFrames) < 5 {
		t.Skip("not enough doom frames to test")
	}

	parser := zebrash.NewParser()
	drawer := zebrash.NewDrawer()
	opts := drawers.DrawerOptions{
		LabelWidthMm:         50,
		LabelHeightMm:        50,
		Dpmm:                 12,
		GrayscaleOutput:      true,
		EnableInvertedLabels: false,
	}

	for _, idx := range []int{0, 60, len(doomFrames) - 1} {
		frame := doomFrames[idx]
		zpl := "^XA\n^LL600\n^PW600\n" + frame + "\n^FO10,370^BQN,2,6^FDM,,https://github.com/AlexProgrammerDE/zpl-renderer^FS\n^XZ"

		labels, err := parser.Parse([]byte(zpl))
		if err != nil {
			t.Fatalf("parse error at doom frame %d: %v", idx, err)
		}
		if len(labels) == 0 {
			t.Fatalf("no labels at doom frame %d", idx)
		}

		var buf bytes.Buffer
		if err := drawer.DrawLabelAsPng(labels[0], &buf, opts); err != nil {
			t.Fatalf("render error at doom frame %d: %v", idx, err)
		}

		png := buf.Bytes()
		if len(png) == 0 {
			t.Fatalf("empty PNG at doom frame %d", idx)
		}
		if !bytes.HasPrefix(png, []byte{0x89, 0x50, 0x4E, 0x47}) {
			t.Fatalf("invalid PNG at doom frame %d", idx)
		}

		t.Logf("doom frame %d: %d bytes", idx, len(png))
	}
}
