FROM golang:1.25-alpine AS builder

ARG VERSION=development

RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w -X github.com/Photon-Panel/Photon-Daemon/system.Version=$VERSION" \
    -o photon-daemon \
    main.go

FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/photon-daemon /usr/bin/

CMD [ "/usr/bin/photon-daemon", "--config", "/etc/photon/config.yml" ]
