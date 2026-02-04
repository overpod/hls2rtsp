# CLAUDE.md

## Project

hls2rtsp — converts HLS streams to RTSP. Single Go binary, no FFmpeg, no MediaMTX.

## Stack

- Go 1.25+
- `gohlslib/v2` — HLS client (segment download, MPEG-TS demux, H.264 extraction, DTS pacing)
- `gortsplib/v4` — RTSP server (protocol, RTP packetization, RTCP)
- `mediacommon` — H.264 NAL unit types

## Structure

```
cmd/hls2rtsp/main.go        — entry point, config loading, signal handling
internal/config/config.go    — YAML config parsing
internal/bridge/bridge.go    — HLS→RTSP bridge (one per camera, auto-reconnect)
internal/server/server.go    — RTSP server with path routing and Basic auth
internal/metrics/metrics.go  — stream quality metrics (FPS, jitter, drift, smoothness)
```

## Build & Run

```bash
go build -o hls2rtsp ./cmd/hls2rtsp
./hls2rtsp --config config.yaml
```

## Test

```bash
# Stream quality test (requires ffprobe + python3)
./scripts/test-stream.sh "rtsp://admin:admin@localhost:8554/camera1" 15 "test"
```

## Key Design Decisions

- No jitter buffer needed — gohlslib handles DTS-based pacing internally
- No transcoding — H.264 passthrough preserves quality, zero CPU overhead
- One Bridge per camera — independent lifecycle, failure isolation
- Auto-reconnect on HLS errors with 5s backoff
- SPS/PPS extracted from first IDR frame (not from HLS track metadata, which can be empty)
- RTP timestamp = raw PTS from gohlslib (90kHz clock, already continuous)

## Config Format

```yaml
server:
  port: "8554"
auth:
  enabled: true
  username: admin
  password: admin
streams:
  camera_name:
    url: https://example.com/stream.m3u8
metrics:
  enabled: true
  interval: 30s
```

## Dependencies Note

- `gortsplib/v4` pinned to v4.11.0 (v4.16.3 has empty module)
- `mediacommon` v1 used for h264.NALUType constants (v2 is used internally by gohlslib)
