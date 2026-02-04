#!/bin/bash
# Test RTSP stream quality by measuring frame timing consistency
# Usage: ./test-stream.sh <rtsp_url> [duration_sec] [label]

RTSP_URL="${1:-rtsp://localhost:8554/stream}"
DURATION="${2:-15}"
LABEL="${3:-test}"
TMPFILE="/tmp/stream-test-${LABEL}.csv"

echo "=== Stream Quality Test: $LABEL ==="
echo "  URL: $RTSP_URL"
echo "  Duration: ${DURATION}s"
echo ""

# Capture frame timestamps using ffprobe
echo "Capturing frames..."
ffprobe -v quiet \
  -rtsp_transport tcp \
  -select_streams v:0 \
  -show_entries frame=pkt_pts_time,pkt_dts_time,pkt_size,key_frame \
  -of csv=print_section=0 \
  -read_intervals "%+${DURATION}" \
  -i "$RTSP_URL" 2>/dev/null > "$TMPFILE"

TOTAL_FRAMES=$(wc -l < "$TMPFILE" | tr -d ' ')

if [ "$TOTAL_FRAMES" -lt 10 ]; then
  echo "ERROR: Only $TOTAL_FRAMES frames captured. Stream may not be working."
  exit 1
fi

echo "Captured $TOTAL_FRAMES frames in ${DURATION}s"
echo ""

# Analyze with python
TMPFILE="$TMPFILE" python3 << 'PYEOF'
import sys, csv, os

tmpfile = os.environ["TMPFILE"]
frames = []
with open(tmpfile, "r") as f:
    for row in csv.reader(f):
        if len(row) < 3:
            continue
        try:
            # ffprobe CSV column order is alphabetical: key_frame, pkt_pts_time, pkt_size
            # (pkt_dts_time is omitted when not available, e.g. RTP streams)
            if len(row) == 3:
                key = int(row[0]) if row[0] else 0
                pts = float(row[1]) if row[1] else None
                size = int(row[2]) if row[2] else 0
            else:
                # 4 columns: key_frame, pkt_dts_time, pkt_pts_time, pkt_size
                key = int(row[0]) if row[0] else 0
                dts = float(row[1]) if row[1] else None
                pts = float(row[2]) if row[2] else None
                size = int(row[3]) if row[3] else 0
                if pts is None:
                    pts = dts
            frames.append({"pts": pts, "size": size, "key": key})
        except (ValueError, IndexError):
            pass

if len(frames) < 10:
    print("Not enough frames for analysis")
    sys.exit(1)

# Calculate frame deltas
deltas = []
for i in range(1, len(frames)):
    if frames[i]["pts"] is not None and frames[i-1]["pts"] is not None:
        d = frames[i]["pts"] - frames[i-1]["pts"]
        if d > 0:
            deltas.append(d)

if not deltas:
    print("No valid frame deltas")
    sys.exit(1)

avg_delta = sum(deltas) / len(deltas)
fps = 1.0 / avg_delta if avg_delta > 0 else 0

# Jitter = standard deviation of deltas
import math
variance = sum((d - avg_delta) ** 2 for d in deltas) / len(deltas)
jitter = math.sqrt(variance)

# Find gaps (delta > 2x average)
gaps = [d for d in deltas if d > avg_delta * 2]
big_gaps = [d for d in deltas if d > avg_delta * 3]

# Bitrate
total_bytes = sum(f["size"] for f in frames)
duration = frames[-1]["pts"] - frames[0]["pts"] if frames[-1]["pts"] and frames[0]["pts"] else 1
bitrate_kbps = (total_bytes * 8 / duration / 1000) if duration > 0 else 0

# Smoothness score (0-100, higher = smoother)
# Based on coefficient of variation of deltas
cv = jitter / avg_delta if avg_delta > 0 else 1
smoothness = max(0, min(100, int((1 - cv) * 100)))

print(f"--- Results ---")
print(f"Frames:       {len(frames)}")
print(f"FPS:          {fps:.1f}")
print(f"Avg delta:    {avg_delta*1000:.1f} ms")
print(f"Jitter (σ):   {jitter*1000:.2f} ms")
print(f"Gaps (>2x):   {len(gaps)}")
print(f"Big gaps(>3x):{len(big_gaps)}")
print(f"Bitrate:      {bitrate_kbps:.0f} kbps")
print(f"Smoothness:   {smoothness}/100")
print(f"")
if gaps:
    print(f"Largest gap:  {max(deltas)*1000:.1f} ms")
PYEOF

rm -f "$TMPFILE"
