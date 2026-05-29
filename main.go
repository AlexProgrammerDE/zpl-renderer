package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/ingridhq/zebrash"
	"github.com/ingridhq/zebrash/drawers"
)

const frameRate = 30.0

var (
	startTime   = time.Now()
	renderCount atomic.Int64
	renderNanos atomic.Int64
	firstRender atomic.Bool
)

func currentZPL() string {
	if len(badAppleFrames) == 0 {
		return "^XA^XZ"
	}
	elapsed := time.Since(startTime).Seconds()
	idx := int(elapsed*frameRate) % len(badAppleFrames)
	return fmt.Sprintf("^XA\n^LL600\n^PW600\n%s\n^FO10,370^BQN,2,6^FDhttps://github.com/AlexProgrammerDE/zpl-renderer^FS\n^XZ", badAppleFrames[idx])
}

func currentFrameIndex() int {
	if len(badAppleFrames) == 0 {
		return 0
	}
	elapsed := time.Since(startTime).Seconds()
	return int(elapsed*frameRate) % len(badAppleFrames)
}

func printStats() {
	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		count := renderCount.Load()
		nanos := renderNanos.Load()
		avg := time.Duration(0)
		if count > 0 {
			avg = time.Duration(nanos / count)
		}
		log.Printf("stats: %d renders | avg %v | frame %d/%d",
			count, avg.Round(time.Microsecond), currentFrameIndex(), len(badAppleFrames))
	}
}

func main() {
	log.Printf("loaded %d bad apple frames", len(badAppleFrames))
	if len(badAppleFrames) == 0 {
		log.Fatal("no frames loaded — does frames.zip exist in the binary directory?")
	}

	go printStats()

	parser := zebrash.NewParser()
	drawer := zebrash.NewDrawer()

	opts := drawers.DrawerOptions{
		LabelWidthMm:         50,
		LabelHeightMm:        50,
		Dpmm:                 12,
		GrayscaleOutput:      true,
		EnableInvertedLabels: false,
	}

	http.HandleFunc("/label.png", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		zpl := currentZPL()

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

		elapsed := time.Since(start)
		renderCount.Add(1)
		renderNanos.Add(elapsed.Nanoseconds())

		if firstRender.CompareAndSwap(false, true) {
			log.Printf("first render: %v (frame %d)", elapsed.Round(time.Microsecond), currentFrameIndex())
		}
	})

	http.HandleFunc("/label.zpl", func(w http.ResponseWriter, r *http.Request) {
		zpl := currentZPL()
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Write([]byte(zpl))
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
  .grid {
    display: flex;
    gap: 24px;
    align-items: flex-start;
    flex-wrap: wrap;
    justify-content: center;
  }
  .panel {
    background: #1a1a1a;
    border: 1px solid #2a2a2a;
    border-radius: 8px;
    padding: 24px;
    display: flex;
    flex-direction: column;
    gap: 12px;
    min-width: 320px;
  }
  .panel-label {
    font-size: 11px;
    font-weight: 500;
    letter-spacing: 0.06em;
    color: #555;
    text-transform: uppercase;
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .copy-btn {
    font-family: inherit;
    font-size: 10px;
    font-weight: 500;
    color: #888;
    background: #252525;
    border: 1px solid #333;
    border-radius: 4px;
    padding: 2px 8px;
    cursor: pointer;
    transition: color 0.15s, background 0.15s;
  }
  .copy-btn:hover { color: #fff; background: #3a3a3a; }
  .copy-btn.copied { color: #4ade80; border-color: #4ade80; }
  img {
    display: block;
    max-width: 480px;
    height: auto;
    background: #fff;
    border-radius: 4px;
    image-rendering: auto;
  }
  pre {
    font-family: "SF Mono", "Fira Code", "JetBrains Mono", monospace;
    font-size: 10px;
    line-height: 1.3;
    color: #ccc;
    background: #0f0f0f;
    border-radius: 6px;
    padding: 16px;
    overflow-x: auto;
    white-space: pre-wrap;
    word-break: break-all;
    max-width: 520px;
    max-height: 80dvh;
    overflow-y: auto;
  }
  .status {
    font-size: 12px;
    color: #666;
    font-variant-numeric: tabular-nums;
  }
</style>
</head>
<body>
<h1>bad apple in zpl</h1>
<div class="status">
  <span id="fps">0</span> fps &middot; frame <span id="frame">0</span> / <span id="total">0</span>
</div>
<div class="grid">
  <div class="panel">
    <span class="panel-label">Preview</span>
    <img id="label" src="/label.png" alt="ZPL label preview">
  </div>
  <div class="panel">
    <span class="panel-label">ZPL Source (full hex) <button class="copy-btn" id="copyBtn">Copy</button></span>
    <pre id="zpl"></pre>
  </div>
</div>
<script>
  const img = document.getElementById("label");
  const pre = document.getElementById("zpl");
  const fpsEl = document.getElementById("fps");
  const frameEl = document.getElementById("frame");
  const totalEl = document.getElementById("total");
  const copyBtn = document.getElementById("copyBtn");
  let frame = 0;
  let last = performance.now();
  let frames = 0;
  let liveZPL = "";

  copyBtn.addEventListener("click", () => {
    navigator.clipboard.writeText(liveZPL).then(() => {
      copyBtn.textContent = "Copied";
      copyBtn.classList.add("copied");
      setTimeout(() => {
        copyBtn.textContent = "Copy";
        copyBtn.classList.remove("copied");
      }, 1200);
    });
  });

  function fmtZPL(text) {
    const lines = text.split("\n");
    const hexLine = lines.find(l => l.startsWith("^FO"));
    if (!hexLine) return text;
    const commaIdx = hexLine.indexOf(",", hexLine.indexOf(",", hexLine.indexOf(",") + 1) + 1);
    const hex = hexLine.slice(commaIdx + 1).replace(/\^FS$/, "");
    return lines.slice(0, 3).join("\n") + "\n" + hex + "\n" + lines.slice(-1).join("\n");
  }

  fetch("/label.zpl").then(r => r.text()).then(t => {
    totalEl.textContent = t.length.toLocaleString();
  });

  function tick() {
    const now = performance.now();
    frame++;
    frames++;
    frameEl.textContent = frame;
    img.src = "/label.png?" + Date.now();
    fetch("/label.zpl?" + Date.now())
      .then(r => r.text())
      .then(text => {
        liveZPL = text;
        pre.textContent = fmtZPL(text);
      })
      .catch(() => { pre.textContent = "(fetch failed)"; });
    if (now - last >= 1000) {
      fpsEl.textContent = frames;
      frames = 0;
      last = now;
    }
  }

  tick();
  setInterval(tick, 100);
</script>
</body>
</html>`
