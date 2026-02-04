# hls2rtsp

[![CI](https://github.com/overpod/hls2rtsp/actions/workflows/ci.yml/badge.svg)](https://github.com/overpod/hls2rtsp/actions/workflows/ci.yml)
[![Release](https://github.com/overpod/hls2rtsp/actions/workflows/release.yml/badge.svg)](https://github.com/overpod/hls2rtsp/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/overpod/hls2rtsp)](https://goreportcard.com/report/github.com/overpod/hls2rtsp)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/overpod/hls2rtsp)](go.mod)

Convert HLS streams to RTSP. Single binary, zero external dependencies (no FFmpeg, no MediaMTX).

Built with [gohlslib](https://github.com/bluenviron/gohlslib) and [gortsplib](https://github.com/bluenviron/gortsplib).

## Features

- Multiple HLS sources → multiple RTSP paths
- RTSP Basic authentication
- H.264 passthrough (no transcoding)
- Auto-reconnect on HLS failures
- Built-in stream quality metrics (FPS, jitter, drift, smoothness)
- Single static binary, ~11MB

## Quick Start

### Binary

```bash
# Build
go build -o hls2rtsp ./cmd/hls2rtsp

# Create config
cp config.example.yaml config.yaml
# Edit config.yaml with your HLS URLs

# Run
./hls2rtsp --config config.yaml
```

### Docker (prebuilt image)

```bash
# Create config
cp config.example.yaml config.yaml
# Edit config.yaml

docker run -d --name hls2rtsp \
  -v $(pwd)/config.yaml:/etc/hls2rtsp/config.yaml:ro \
  -p 8554:8554 \
  ghcr.io/overpod/hls2rtsp:latest
```

### Docker Compose

```yaml
services:
  hls2rtsp:
    image: ghcr.io/overpod/hls2rtsp:latest
    container_name: hls2rtsp
    restart: unless-stopped
    volumes:
      - ./config.yaml:/etc/hls2rtsp/config.yaml:ro
    ports:
      - "8554:8554"         # RTSP (TCP)
      - "8000:8000/udp"     # RTP (optional, for UDP transport)
      - "8001:8001/udp"     # RTCP (optional, for UDP transport)
```

> **Note:** UDP ports 8000/8001 are only needed if RTSP clients request UDP transport.
> Most clients (VLC, ffplay) use TCP interleaved by default — in that case only port 8554 is required.

## Configuration

```yaml
server:
  port: "8554"

auth:
  enabled: true
  username: admin
  password: admin

streams:
  camera1:
    url: https://example.com/stream1/mono.m3u8

  camera2:
    url: https://example.com/stream2/mono.m3u8

metrics:
  enabled: true
  interval: 30s
```

## Usage

After starting, streams are available at:

```
rtsp://admin:admin@localhost:8554/camera1
rtsp://admin:admin@localhost:8554/camera2
```

Open in VLC: **Media → Open Network Stream** → paste the RTSP URL.

## How It Works

```
HLS source 1 ──► gohlslib.Client ──► Bridge 1 ──► /camera1 ──► RTSP clients
HLS source 2 ──► gohlslib.Client ──► Bridge 2 ──► /camera2 ──► RTSP clients
                                                      │
                                              gortsplib.Server
                                                 :8554
```

1. Each HLS source is handled by a separate bridge
2. `gohlslib` downloads HLS segments, demuxes MPEG-TS, extracts H.264 NAL units with DTS-based pacing
3. NAL units are packetized into RTP (RFC 6184) by `gortsplib` and sent to RTSP clients
4. No transcoding — original H.264 quality is preserved

## Metrics

When `metrics.enabled: true`, the log shows periodic quality reports:

```
[camera1] frames=600 fps=20.0 jitter=0.08ms drift=0.0ms drift_max=4.9ms gaps=0 smooth=99/100
```

- **fps** — actual framerate from PTS timestamps
- **jitter** — standard deviation of frame intervals (lower = smoother)
- **drift** — difference between expected and actual delivery timing
- **gaps** — count of frame intervals > 2x average
- **smooth** — overall smoothness score (0–100)

## License

MIT
