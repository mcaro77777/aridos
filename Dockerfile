# syntax=docker/dockerfile:1.7

# -----------------------------------------------------------------------------
# 1. Compilación del frontend Angular
# -----------------------------------------------------------------------------
FROM node:22-alpine AS frontend-dependencias
WORKDIR /app/web

COPY web/package.json web/package-lock.json ./
RUN npm ci --no-audit --no-fund

FROM frontend-dependencias AS frontend-compilacion
COPY web/ ./
RUN npm run build \
    && test -f /app/ClientApp/index.html

# -----------------------------------------------------------------------------
# 2. Compilación de la API Go
# -----------------------------------------------------------------------------
FROM golang:1.24-alpine AS api-dependencias
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

FROM api-dependencias AS api-compilacion
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /salida/aridos-api ./cmd/api

# -----------------------------------------------------------------------------
# 3. Imagen final mínima y sin privilegios
# -----------------------------------------------------------------------------
FROM alpine:3.22 AS final

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S aridos \
    && adduser -S -G aridos aridos \
    && mkdir -p /app/ClientApp \
    && chown -R aridos:aridos /app

WORKDIR /app

COPY --from=api-compilacion --chown=aridos:aridos \
    /salida/aridos-api /app/aridos-api

COPY --from=frontend-compilacion --chown=aridos:aridos \
    /app/ClientApp/ /app/ClientApp/

USER aridos

ENV APP_ENV=production
ENV PORT=8080
ENV CLIENT_DIR=/app/ClientApp

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=5s --start-period=20s --retries=5 \
    CMD wget -q -O /dev/null http://127.0.0.1:8080/api/health || exit 1

ENTRYPOINT ["/app/aridos-api"]
