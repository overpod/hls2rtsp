package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/overpod/hls2rtsp/internal/bridge"
	"github.com/overpod/hls2rtsp/internal/config"
	"github.com/overpod/hls2rtsp/internal/server"
)

var version = "dev"

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	showVersion := flag.Bool("version", false, "show version and exit")
	flag.Parse()

	if *showVersion {
		log.Printf("hls2rtsp %s", version)
		return
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	log.Printf("hls2rtsp %s starting (%d streams)", version, len(cfg.Streams))

	srv := server.New(
		cfg.Server.Port,
		cfg.Auth.Enabled,
		cfg.Auth.Username,
		cfg.Auth.Password,
	)

	if err := srv.Start(); err != nil {
		log.Fatalf("RTSP server: %v", err)
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
			log.Printf("  /%s → rtsp://%s:%s@localhost:%s/%s",
				name, cfg.Auth.Username, cfg.Auth.Password, cfg.Server.Port, name)
		} else {
			log.Printf("  /%s → rtsp://localhost:%s/%s", name, cfg.Server.Port, name)
		}
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("hls2rtsp running, press Ctrl+C to stop")

	<-sigCh
	log.Printf("shutting down...")

	for _, b := range bridges {
		b.Close()
	}
}
