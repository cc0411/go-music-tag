FROM golang:1.24-alpine AS builder

ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn,direct
ENV CGO_ENABLED=1
ENV GOOS=linux
ENV GOARCH=arm64

RUN apk add --no-cache gcc g++ musl-dev ca-certificates tzdata
ENV TZ=Asia/Shanghai

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 go build -ldflags="-s -w -extldflags '-static'" -o music ./cmd/server/

# ============================================
FROM alpine

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai

WORKDIR /app

COPY --from=builder /app/music .
COPY --from=builder /app/config.yaml .
COPY --from=builder /app/frontend ./frontend

RUN mkdir -p /app/data
RUN mkdir -p /app/data/lyrics /app/data/covers
RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser && \
    chown -R appuser:appgroup /app

USER appuser

VOLUME ["/app/data"]

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["./music"]