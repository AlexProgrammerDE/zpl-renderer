# zpl-renderer

Animated ZPL label preview server with live browser refresh. Renders a bouncing ball in ZPL via [zebrash](https://github.com/ingridhq/zebrash) and serves it over HTTP.

## Setup

```bash
go mod download
```

## Run

```bash
go run .
```

Then open <http://localhost:8080>. The browser refetches a freshly rendered PNG every 100ms, showing the ball bounce in real time.

## Test

```bash
go test ./...
```
