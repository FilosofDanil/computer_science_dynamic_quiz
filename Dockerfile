# ── Stage 1: Build ───────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

# git is required for `go mod download` with VCS deps
RUN apk add --no-cache git

WORKDIR /src

# Cache dependency downloads separately from source compilation
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /bin/bot ./cmd/bot

# ── Stage 2: Runtime ─────────────────────────────────────────────────────────
FROM scratch

# Copy TLS certificates so the bot can reach api.telegram.org over HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the statically-linked binary and the topics data file
COPY --from=builder /bin/bot /bot
COPY --from=builder /src/data/topics.json /data/topics.json

# The bot reads these from the environment; set sensible defaults
ENV TOPICS_PATH=/data/topics.json \
    LOG_FORMAT=json \
    LOG_LEVEL=info

# TELEGRAM_BOT_TOKEN must be supplied at runtime (never bake it into the image)
# e.g. docker run -e TELEGRAM_BOT_TOKEN=<token> basics-bot

ENTRYPOINT ["/bot"]
