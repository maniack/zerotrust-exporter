# syntax=docker/dockerfile:1.6
FROM golang:1.24-alpine AS builder
RUN apk add --no-cache build-base git
WORKDIR /src
COPY . .
ENV CGO_ENABLED=0 GOOS=linux
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go mod tidy
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go build -trimpath -ldflags "-s -w" -o /bin/zerotrust-exporter .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
RUN addgroup -S app && adduser -S app -G app
COPY --from=builder /bin/zerotrust-exporter /bin/zerotrust-exporter
USER app
EXPOSE 9184
ENTRYPOINT ["/bin/zerotrust-exporter"]
