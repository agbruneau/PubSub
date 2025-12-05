# =============================================================================
# Dockerfile Multi-stage pour PubSub Kafka Demo
# =============================================================================
# Produit des images légères (~15-20 MB) contenant uniquement les binaires.
#
# Usage:
#   docker build --target producer -t pubsub-producer .
#   docker build --target tracker -t pubsub-tracker .
#   docker build --target monitor -t pubsub-monitor .
#
# Ou tout construire:
#   docker compose build
# =============================================================================

# -----------------------------------------------------------------------------
# Stage 1: Builder - Compile tous les binaires
# -----------------------------------------------------------------------------
FROM golang:1.22-alpine AS builder

# Installer les dépendances de build pour confluent-kafka-go (CGO)
RUN apk add --no-cache \
    gcc \
    musl-dev \
    librdkafka-dev \
    pkgconf

WORKDIR /app

# Copier les fichiers de dépendances d'abord (cache Docker)
COPY go.mod go.sum ./
RUN go mod download

# Copier le code source
COPY . .

# Compiler le producer (avec CGO pour kafka)
RUN CGO_ENABLED=1 GOOS=linux go build -tags kafka \
    -ldflags="-w -s" \
    -o /bin/producer ./cmd/producer

# Compiler le tracker (avec CGO pour kafka)
RUN CGO_ENABLED=1 GOOS=linux go build -tags kafka \
    -ldflags="-w -s" \
    -o /bin/tracker ./cmd/tracker

# Compiler le monitor (sans CGO)
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /bin/monitor ./cmd/monitor

# -----------------------------------------------------------------------------
# Stage 2: Producer - Image finale minimale
# -----------------------------------------------------------------------------
FROM alpine:3.19 AS producer

# Installer librdkafka runtime
RUN apk add --no-cache librdkafka

# Créer un utilisateur non-root
RUN adduser -D -u 1000 appuser
USER appuser

WORKDIR /app

# Copier le binaire depuis le builder
COPY --from=builder /bin/producer /app/producer

# Copier le fichier de config exemple (optionnel)
COPY --from=builder /app/config.yaml.example /app/config.yaml.example

# Variables d'environnement par défaut
ENV KAFKA_BROKER=kafka:9092 \
    KAFKA_TOPIC=orders \
    PRODUCER_INTERVAL_MS=2000

ENTRYPOINT ["/app/producer"]

# -----------------------------------------------------------------------------
# Stage 3: Tracker - Image finale minimale
# -----------------------------------------------------------------------------
FROM alpine:3.19 AS tracker

# Installer librdkafka runtime
RUN apk add --no-cache librdkafka

# Créer un utilisateur non-root
RUN adduser -D -u 1000 appuser
USER appuser

WORKDIR /app

# Copier le binaire depuis le builder
COPY --from=builder /bin/tracker /app/tracker

# Copier le fichier de config exemple (optionnel)
COPY --from=builder /app/config.yaml.example /app/config.yaml.example

# Variables d'environnement par défaut
ENV KAFKA_BROKER=kafka:9092 \
    KAFKA_TOPIC=orders \
    KAFKA_CONSUMER_GROUP=order-tracker-group

# Volume pour les logs
VOLUME ["/app/logs"]

ENTRYPOINT ["/app/tracker"]

# -----------------------------------------------------------------------------
# Stage 4: Monitor - Image finale ultra-légère (pas de kafka)
# -----------------------------------------------------------------------------
FROM alpine:3.19 AS monitor

# Créer un utilisateur non-root
RUN adduser -D -u 1000 appuser
USER appuser

WORKDIR /app

# Copier le binaire depuis le builder
COPY --from=builder /bin/monitor /app/monitor

# Le monitor lit les fichiers de logs, donc volume partagé avec tracker
VOLUME ["/app/logs"]

ENTRYPOINT ["/app/monitor"]
