package server

import (
	"encoding/base64"
	"log/slog"
	"strings"
	"sync"

	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/description"
)

type streamEntry struct {
	stream *gortsplib.ServerStream
}

type Server struct {
	server *gortsplib.Server

	authEnabled  bool
	authUsername string
	authPassword string

	mu      sync.RWMutex
	streams map[string]*streamEntry
}

func New(port string, authEnabled bool, username, password string) *Server {
	s := &Server{
		authEnabled:  authEnabled,
		authUsername: username,
		authPassword: password,
		streams:      make(map[string]*streamEntry),
	}

	s.server = &gortsplib.Server{
		Handler:        s,
		RTSPAddress:    ":" + port,
		UDPRTPAddress:  ":8000",
		UDPRTCPAddress: ":8001",
	}

	return s
}

func (s *Server) Start() error {
	slog.Info("RTSP server starting", "address", s.server.RTSPAddress)
	if s.authEnabled {
		slog.Info("authentication enabled", "username", s.authUsername)
	}
	return s.server.Start()
}

func (s *Server) Close() {
	s.mu.Lock()
	for _, entry := range s.streams {
		entry.stream.Close()
	}
	s.mu.Unlock()
	s.server.Close()
}

func (s *Server) AddStream(path string, desc *description.Session) *gortsplib.ServerStream {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := "/" + path
	if old, ok := s.streams[key]; ok {
		old.stream.Close()
		slog.Info("replaced existing stream", "path", key)
	}

	stream := gortsplib.NewServerStream(s.server, desc)
	s.streams[key] = &streamEntry{stream: stream}
	slog.Info("stream registered", "path", key)
	return stream
}

func (s *Server) GetStream(path string) *gortsplib.ServerStream {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.streams[path]
	if !ok {
		return nil
	}
	return entry.stream
}

func (s *Server) checkAuth(req *base.Request) bool {
	if !s.authEnabled {
		return true
	}

	auth := req.Header["Authorization"]
	if auth == nil {
		return false
	}

	for _, val := range auth {
		if strings.HasPrefix(val, "Basic ") {
			decoded, err := base64.StdEncoding.DecodeString(val[6:])
			if err != nil {
				continue
			}
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) == 2 && parts[0] == s.authUsername && parts[1] == s.authPassword {
				return true
			}
		}
	}

	return false
}

func (s *Server) unauthorizedResponse() *base.Response {
	return &base.Response{
		StatusCode: base.StatusUnauthorized,
		Header: base.Header{
			"WWW-Authenticate": base.HeaderValue{"Basic realm=\"hls2rtsp\""},
		},
	}
}

// --- gortsplib ServerHandler interfaces ---

func (s *Server) OnConnOpen(ctx *gortsplib.ServerHandlerOnConnOpenCtx) {
	slog.Info("client connected", "remote", ctx.Conn.NetConn().RemoteAddr())
}

func (s *Server) OnConnClose(ctx *gortsplib.ServerHandlerOnConnCloseCtx) {
	slog.Info("client disconnected", "error", ctx.Error)
}

func (s *Server) OnSessionOpen(_ *gortsplib.ServerHandlerOnSessionOpenCtx) {}

func (s *Server) OnSessionClose(_ *gortsplib.ServerHandlerOnSessionCloseCtx) {}

func (s *Server) OnDescribe(ctx *gortsplib.ServerHandlerOnDescribeCtx) (*base.Response, *gortsplib.ServerStream, error) {
	if !s.checkAuth(ctx.Request) {
		return s.unauthorizedResponse(), nil, nil
	}

	stream := s.GetStream(ctx.Path)
	if stream == nil {
		return &base.Response{StatusCode: base.StatusNotFound}, nil, nil
	}

	return &base.Response{StatusCode: base.StatusOK}, stream, nil
}

func (s *Server) OnSetup(ctx *gortsplib.ServerHandlerOnSetupCtx) (*base.Response, *gortsplib.ServerStream, error) {
	if !s.checkAuth(ctx.Request) {
		return s.unauthorizedResponse(), nil, nil
	}

	stream := s.GetStream(ctx.Path)
	if stream == nil {
		return &base.Response{StatusCode: base.StatusNotFound}, nil, nil
	}

	return &base.Response{StatusCode: base.StatusOK}, stream, nil
}

func (s *Server) OnPlay(ctx *gortsplib.ServerHandlerOnPlayCtx) (*base.Response, error) {
	slog.Info("playback started", "path", ctx.Path)
	return &base.Response{StatusCode: base.StatusOK}, nil
}
