package main

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/ingridhq/zebrash"
	"github.com/ingridhq/zebrash/drawers"
)

const labelZPL = `^XA
^LL400
^PW400
^FO%d,%d^GFA,1024,1024,16,%s^FS
^XZ`

var ballHex = func() string {
	return stringsJoin(
		"0000000000000000,00000007FF000000,0000007FFFE00000,000001FFFFF80000",
		"000007FFFFFF0000,00000FFFFFFF8000,00001FFFFFFFC000,00003FFFFFFFE000",
		"00007FFFFFFFF000,00007FFFFFFFF800,0000FFFFFFFFF800,0000FFFFFFFFFC00",
		"0001FFFFFFFFFC00,0001FFFFFFFFFE00,0001FFFFFFFFFE00,0001FFFFFFFFFF00",
		"0001FFFFFFFFFF00,0001FFFFFFFFFF00,0001FFFFFFFFFF00,0001FFFFFFFFFF00",
		"0001FFFFFFFFFE00,0001FFFFFFFFFE00,0000FFFFFFFFFC00,0000FFFFFFFFFC00",
		"00007FFFFFFFF800,00007FFFFFFFF000,00003FFFFFFFE000,00001FFFFFFFC000",
		"00000FFFFFFF8000,000007FFFFFF0000,000001FFFFF80000,0000007FFFE00000",
		"00000007FF000000,0000000000000000",
	)
}()

func stringsJoin(s ...string) string {
	var result string
	for _, v := range s {
		result += v
	}
	return result
}

func main() {
	parser := zebrash.NewParser()
	drawer := zebrash.NewDrawer()

	opts := drawers.DrawerOptions{
		LabelWidthMm:         101.6,
		LabelHeightMm:        101.6,
		Dpmm:                 8,
		GrayscaleOutput:      true,
		EnableInvertedLabels: false,
	}

	http.HandleFunc("/label.png", func(w http.ResponseWriter, r *http.Request) {
		t := float64(time.Now().UnixMilli()) / 1000.0
		y := int(160 + math.Sin(t*4)*140)

		zpl := fmt.Sprintf(labelZPL, 36, y, ballHex)

		labels, err := parser.Parse([]byte(zpl))
		if err != nil {
			http.Error(w, fmt.Sprintf("parse error: %v", err), http.StatusInternalServerError)
			return
		}
		if len(labels) == 0 {
			http.Error(w, "no labels found", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		if err := drawer.DrawLabelAsPng(labels[0], w, opts); err != nil {
			http.Error(w, fmt.Sprintf("render error: %v", err), http.StatusInternalServerError)
			return
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(indexHTML))
	})

	log.Println("zpl-renderer running on http://localhost:8080")
	srv := &http.Server{
		Addr:              ":8080",
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

var indexHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>zpl-renderer</title>
<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    font-family: system-ui, -apple-system, sans-serif;
    background: #0d0d0d;
    color: #e0e0e0;
    min-height: 100dvh;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 24px;
    padding: 32px 16px;
  }
  h1 { font-size: 14px; font-weight: 500; letter-spacing: 0.05em; color: #888; text-transform: uppercase; }
  .container {
    background: #1a1a1a;
    border: 1px solid #2a2a2a;
    border-radius: 8px;
    padding: 24px;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 16px;
    max-width: 90vw;
  }
  img {
    display: block;
    max-width: 100%;
    height: auto;
    background: #fff;
    border-radius: 4px;
    image-rendering: auto;
  }
  .status {
    font-size: 12px;
    color: #666;
    font-variant-numeric: tabular-nums;
  }
</style>
</head>
<body>
<h1>zpl-renderer</h1>
<div class="container">
  <img id="label" src="/label.png" alt="ZPL label preview">
  <div class="status">
    Polling every 100ms &middot frame <span id="frame">0</span>
  </div>
</div>
<script>
  const img = document.getElementById("label");
  const frameEl = document.getElementById("frame");
  let frame = 0;

  function tick() {
    frame++;
    frameEl.textContent = frame;
    img.src = "/label.png?" + Date.now();
  }

  setInterval(tick, 100);
</script>
</body>
</html>`
