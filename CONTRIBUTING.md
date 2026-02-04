# Contributing to hls2rtsp

## Getting Started

```bash
git clone https://github.com/overpod/hls2rtsp.git
cd hls2rtsp
make build
```

## Development

### Prerequisites

- Go 1.25+
- golangci-lint (optional, for linting)

### Build & Test

```bash
make build        # Build binary
make test         # Run tests
make lint         # Run linter
make docker       # Build Docker image
```

### Project Structure

```
cmd/hls2rtsp/          Entry point
internal/bridge/       HLS-to-RTSP bridge (one per camera)
internal/config/       YAML config parsing
internal/server/       RTSP server with auth and path routing
internal/metrics/      Stream quality metrics
```

## Submitting Changes

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-change`
3. Make your changes
4. Run `make lint test` to verify
5. Commit with a clear message
6. Open a Pull Request

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Keep changes focused and minimal
- Add tests for new functionality

## Reporting Issues

Use [GitHub Issues](https://github.com/overpod/hls2rtsp/issues) with the provided templates.
