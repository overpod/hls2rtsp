package main

import (
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/overpod/hls2rtsp/internal/bridge"
	"github.com/overpod/hls2rtsp/internal/config"
	"github.com/overpod/hls2rtsp/internal/server"
)

var version = "dev"

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	showVersion := flag.Bool("version", false, "show version and exit")
	logJSON := flag.Bool("log-json", false, "output logs in JSON format")
	flag.Parse()

	if *logJSON {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	}

	if *showVersion {
		slog.Info("hls2rtsp", "version", version)
		return
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("config load failed", "error", err)
		os.Exit(1)
	}

	slog.Info("hls2rtsp starting", "version", version, "streams", len(cfg.Streams))

	srv := server.New(
		cfg.Server.Port,
		cfg.Auth.Enabled,
		cfg.Auth.Username,
		cfg.Auth.Password,
	)

	if err := srv.Start(); err != nil {
		slog.Error("RTSP server failed to start", "error", err)
		os.Exit(1)
	}
	defer srv.Close()

	var bridges []*bridge.Bridge

	for name, streamCfg := range cfg.Streams {
		b := bridge.New(
			name,
			streamCfg.URL,
			srv,
			cfg.Metrics.Enabled,
			cfg.Metrics.Interval,
		)
		b.Start()
		bridges = append(bridges, b)

		if cfg.Auth.Enabled {
			slog.Info("stream configured",
				"path", "/"+name,
				"rtsp", "rtsp://"+cfg.Auth.Username+":***@localhost:"+cfg.Server.Port+"/"+name)
		} else {
			slog.Info("stream configured",
				"path", "/"+name,
				"rtsp", "rtsp://localhost:"+cfg.Server.Port+"/"+name)
		}
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	slog.Info("hls2rtsp running, press Ctrl+C to stop")

	<-sigCh
	slog.Info("shutting down...")

	var wg sync.WaitGroup
	for _, b := range bridges {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.Close()
		}()
	}
	wg.Wait()
}
