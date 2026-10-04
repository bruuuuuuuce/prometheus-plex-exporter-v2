# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src
COPY . .

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -mod=vendor -trimpath -o /prometheus-plex-exporter ./cmd/prometheus-plex-exporter

FROM alpine:3.22

RUN apk --update --no-cache add ca-certificates tzdata

COPY --from=build /prometheus-plex-exporter /prometheus-plex-exporter

USER nobody:nobody
EXPOSE 9000

ENTRYPOINT ["/prometheus-plex-exporter"]
