FROM golang:1.23-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /hls2rtsp ./cmd/hls2rtsp

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /hls2rtsp /usr/local/bin/hls2rtsp
ENTRYPOINT ["hls2rtsp"]
CMD ["--config", "/etc/hls2rtsp/config.yaml"]
