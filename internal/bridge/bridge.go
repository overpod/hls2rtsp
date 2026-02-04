package bridge

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/bluenviron/gohlslib/v2"
	"github.com/bluenviron/gohlslib/v2/pkg/codecs"
	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/bluenviron/gortsplib/v4/pkg/format/rtph264"
	"github.com/bluenviron/mediacommon/pkg/codecs/h264"

	"github.com/overpod/hls2rtsp/internal/metrics"
	"github.com/overpod/hls2rtsp/internal/server"
)

type Bridge struct {
	name    string
	hlsURL  string
	server  *server.Server
	metrics *metrics.Metrics
	log     *slog.Logger

	metricsEnabled  bool
	metricsInterval time.Duration

	stream     *gortsplib.ServerStream
	media      *description.Media
	h264Format *format.H264
	encoder    *rtph264.Encoder

	mu          sync.Mutex
	initialized bool
	cancel      context.CancelFunc
	done        chan struct{}
}

func New(name, hlsURL string, srv *server.Server, metricsEnabled bool, metricsInterval time.Duration) *Bridge {
	return &Bridge{
		name:            name,
		hlsURL:          hlsURL,
		server:          srv,
		log:             slog.Default().With("stream", name),
		metricsEnabled:  metricsEnabled,
		metricsInterval: metricsInterval,
		done:            make(chan struct{}),
	}
}

func (b *Bridge) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	b.cancel = cancel
	go b.runLoop(ctx)
}

func (b *Bridge) Close() {
	if b.cancel != nil {
		b.cancel()
	}
	<-b.done
	if b.metrics != nil {
		b.metrics.Stop()
	}
}

func (b *Bridge) runLoop(ctx context.Context) {
	defer close(b.done)

	for {
		b.log.Info("connecting to HLS", "url", b.hlsURL)

		err := b.run(ctx)
		if ctx.Err() != nil {
			return
		}

		b.log.Warn("HLS disconnected, reconnecting in 5s", "error", err)

		select {
		case <-time.After(5 * time.Second):
		case <-ctx.Done():
			return
		}
	}
}

func (b *Bridge) run(ctx context.Context) error {
	b.mu.Lock()
	b.initialized = false
	b.stream = nil
	b.encoder = nil
	b.mu.Unlock()

	if b.metricsEnabled && b.metrics == nil {
		b.metrics = metrics.New(b.name, b.metricsInterval)
	}

	var client *gohlslib.Client
	client = &gohlslib.Client{
		URI: b.hlsURL,
		OnTracks: func(tracks []*gohlslib.Track) error {
			return b.onTracks(tracks, client)
		},
		OnDecodeError: func(err error) {
			b.log.Warn("decode error", "error", err)
		},
	}

	if err := client.Start(); err != nil {
		return fmt.Errorf("HLS start: %w", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- client.Wait2()
	}()

	select {
	case err := <-errCh:
		client.Close()
		return err
	case <-ctx.Done():
		client.Close()
		return ctx.Err()
	}
}

func (b *Bridge) onTracks(tracks []*gohlslib.Track, client *gohlslib.Client) error {
	for _, track := range tracks {
		if _, ok := track.Codec.(*codecs.H264); !ok {
			continue
		}

		b.log.Info("found H264 track, waiting for SPS/PPS")

		ctrack := track
		client.OnDataH26x(ctrack, func(pts int64, dts int64, au [][]byte) {
			b.onH264Data(pts, dts, au)
		})

		return nil
	}

	return fmt.Errorf("no H264 track found")
}

func (b *Bridge) onH264Data(pts int64, dts int64, au [][]byte) {
	b.mu.Lock()

	if !b.initialized {
		sps, pps := extractSPSPPS(au)
		if sps == nil || pps == nil {
			b.mu.Unlock()
			return
		}

		b.log.Info("got SPS/PPS", "sps_bytes", len(sps), "pps_bytes", len(pps))

		b.h264Format = &format.H264{
			PayloadTyp:        96,
			PacketizationMode: 1,
			SPS:               sps,
			PPS:               pps,
		}

		b.media = &description.Media{
			Type:    description.MediaTypeVideo,
			Formats: []format.Format{b.h264Format},
		}

		encoder, err := b.h264Format.CreateEncoder()
		if err != nil {
			b.log.Error("create encoder failed", "error", err)
			b.mu.Unlock()
			return
		}
		b.encoder = encoder

		desc := &description.Session{Medias: []*description.Media{b.media}}
		b.stream = b.server.AddStream(b.name, desc)
		b.initialized = true

		b.log.Info("RTSP stream ready", "path", "/"+b.name)
	}

	b.mu.Unlock()

	if b.metrics != nil {
		b.metrics.RecordFrame(pts)
	}

	b.deliverFrame(pts, au)
}

func (b *Bridge) deliverFrame(pts int64, nals [][]byte) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.stream == nil {
		return
	}

	packets, err := b.encoder.Encode(nals)
	if err != nil {
		return
	}

	for _, pkt := range packets {
		pkt.Timestamp = uint32(pts)
		if err := b.stream.WritePacketRTP(b.media, pkt); err != nil {
			return
		}
	}
}

func extractSPSPPS(au [][]byte) (sps, pps []byte) {
	for _, nalu := range au {
		if len(nalu) == 0 {
			continue
		}
		typ := h264.NALUType(nalu[0] & 0x1F)
		switch typ {
		case h264.NALUTypeSPS:
			sps = make([]byte, len(nalu))
			copy(sps, nalu)
		case h264.NALUTypePPS:
			pps = make([]byte, len(nalu))
			copy(pps, nalu)
		}
	}
	return
}
