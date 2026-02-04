package metrics

import (
	"log"
	"math"
	"sync"
	"time"
)

type Metrics struct {
	name string
	mu   sync.Mutex

	lastPTS      int64
	lastWallTime time.Time
	firstFrame   bool

	frameCount  int64
	ptsDeltas   []float64
	driftDeltas []float64

	ticker *time.Ticker
	stopCh chan struct{}
}

func New(name string, interval time.Duration) *Metrics {
	m := &Metrics{
		name:       name,
		firstFrame: true,
		stopCh:     make(chan struct{}),
	}
	m.ticker = time.NewTicker(interval)
	go m.reporter()
	return m
}

func (m *Metrics) RecordFrame(pts int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	m.frameCount++

	if m.firstFrame {
		m.lastPTS = pts
		m.lastWallTime = now
		m.firstFrame = false
		return
	}

	ptsDelta := float64(pts-m.lastPTS) / 90000.0
	wallDelta := now.Sub(m.lastWallTime).Seconds()

	if ptsDelta > 0 && ptsDelta < 5.0 {
		m.ptsDeltas = append(m.ptsDeltas, ptsDelta)
		drift := (wallDelta - ptsDelta) * 1000.0
		m.driftDeltas = append(m.driftDeltas, drift)
	}

	m.lastPTS = pts
	m.lastWallTime = now
}

func (m *Metrics) Stop() {
	close(m.stopCh)
}

func (m *Metrics) reporter() {
	for {
		select {
		case <-m.ticker.C:
			m.printReport()
		case <-m.stopCh:
			m.ticker.Stop()
			return
		}
	}
}

func (m *Metrics) printReport() {
	m.mu.Lock()
	defer m.mu.Unlock()

	n := len(m.ptsDeltas)
	if n < 10 {
		return
	}

	avgDelta := avg(m.ptsDeltas)
	fps := 1.0 / avgDelta
	jitter := stddev(m.ptsDeltas) * 1000.0
	avgDrift := avg(m.driftDeltas)
	maxDrift := maxAbs(m.driftDeltas)

	gaps := 0
	for _, d := range m.ptsDeltas {
		if d > avgDelta*2 {
			gaps++
		}
	}

	cv := stddev(m.ptsDeltas) / avgDelta
	smoothness := int(math.Max(0, math.Min(100, (1-cv)*100)))

	log.Printf("[%s] frames=%d fps=%.1f jitter=%.2fms drift=%.1fms drift_max=%.1fms gaps=%d smooth=%d/100",
		m.name, m.frameCount, fps, jitter, avgDrift, maxDrift, gaps, smoothness)

	if n > 300 {
		m.ptsDeltas = m.ptsDeltas[n-300:]
		m.driftDeltas = m.driftDeltas[n-300:]
	}
}

func avg(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	s := 0.0
	for _, v := range vals {
		s += v
	}
	return s / float64(len(vals))
}

func stddev(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	mean := avg(vals)
	variance := 0.0
	for _, v := range vals {
		d := v - mean
		variance += d * d
	}
	return math.Sqrt(variance / float64(len(vals)))
}

func maxAbs(vals []float64) float64 {
	m := 0.0
	for _, v := range vals {
		a := math.Abs(v)
		if a > m {
			m = a
		}
	}
	return m
}
