FROM golang:1.23-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/go-offensive-fortune .

FROM debian:bookworm-slim
WORKDIR /app

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates wget \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /app/data /usr/share/games/fortunes/off

RUN wget -q https://deb.debian.org/debian/pool/main/f/fortune-mod/fortunes-off_1.99.1-7.1_all.deb -O /tmp/fortunes-off.deb \
    && dpkg -x /tmp/fortunes-off.deb /tmp/fortunes-off \
    && cp -r /tmp/fortunes-off/usr/share/games/fortunes/off/* /usr/share/games/fortunes/off/ || true

COPY --from=build /out/go-offensive-fortune /usr/local/bin/go-offensive-fortune

EXPOSE 6724

ENTRYPOINT ["/usr/local/bin/go-offensive-fortune", "-dir", "/usr/share/games/fortunes/off/", "-db", "/app/data/requests.db"]
